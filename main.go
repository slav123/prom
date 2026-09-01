package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"os"
	"time"

	"io"

	"github.com/slav123/prom/htmlutils"
	"github.com/slav123/prom/imageutils"

	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/denisbrodbeck/striphtmltags"
)

const (
	maxWorkers = 5
	port       = 9999
)

var (
	promImage     string
	maxDimensions int
)

// sanitizeRequestURL parses and validates a raw URL and returns a normalized,
// absolute URL that is safe to request. It rejects non-http(s) schemes,
// relative URLs, URLs containing credentials, and URLs whose host resolves to a
// non-public IP address. The returned string is the canonical form produced by
// url.URL.String(), so it no longer contains the raw user-supplied value.
func sanitizeRequestURL(rawURL string) (string, error) {
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return "", fmt.Errorf("invalid URL: %w", err)
	}
	if !parsedURL.IsAbs() {
		return "", fmt.Errorf("URL must be absolute")
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return "", fmt.Errorf("unsupported URL scheme %q", parsedURL.Scheme)
	}
	if parsedURL.Hostname() == "" {
		return "", fmt.Errorf("URL must contain a host")
	}
	if parsedURL.User != nil {
		return "", fmt.Errorf("URL must not contain credentials")
	}

	host := parsedURL.Hostname()
	if ip := net.ParseIP(host); ip != nil {
		if !isPublicIP(ip) {
			return "", fmt.Errorf("URL resolves to a non-public address")
		}
		return parsedURL.String(), nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return "", fmt.Errorf("failed to resolve host: %w", err)
	}
	if len(ips) == 0 {
		return "", fmt.Errorf("host has no DNS records")
	}
	for _, ip := range ips {
		if !isPublicIP(ip) {
			return "", fmt.Errorf("URL resolves to a non-public address")
		}
	}
	return parsedURL.String(), nil
}

// validateRequestURL is a convenience wrapper around sanitizeRequestURL for
// callers that only need to check validity without using the canonical URL.
func validateRequestURL(rawURL string) error {
	_, err := sanitizeRequestURL(rawURL)
	return err
}

func isPublicIP(ip net.IP) bool {
	return ip.IsGlobalUnicast() && !ip.IsPrivate()
}

// sanitizeForLog removes line breaks from untrusted strings before they are
// written to logs, preventing a malicious value from forging new log entries.
func sanitizeForLog(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	return s
}

func safeDialContext(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return nil, fmt.Errorf("invalid request address: %w", err)
	}
	ips, err := net.DefaultResolver.LookupIP(ctx, "ip", host)
	if err != nil {
		return nil, fmt.Errorf("resolve request host: %w", err)
	}
	var dialer net.Dialer
	for _, ip := range ips {
		if !isPublicIP(ip) {
			continue
		}
		conn, dialErr := dialer.DialContext(ctx, network, net.JoinHostPort(ip.String(), port))
		if dialErr == nil {
			return conn, nil
		}
		err = dialErr
	}
	if err != nil {
		return nil, fmt.Errorf("connect to request host: %w", err)
	}
	return nil, fmt.Errorf("request host has no public IP addresses")
}

func newSafeHTTPClient(timeout time.Duration) *http.Client {
	transport := &http.Transport{
		DialContext:     safeDialContext,
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		return validateRequestURL(req.URL.String())
	}
	return client
}

// ImageResult holds information about processed image
type ImageResult struct {
	URL    string
	Width  int32
	Height int32
	Area   int
}

// GetDimensions get image dimensions
func GetDimensions(id int, jobs <-chan string, results chan<- ImageResult, r *http.Request) {
	for url := range jobs {
		result := ImageResult{
			URL: url,
		}

		fmt.Println("worker", id, "started job", sanitizeForLog(url))

		// header size to get
		min := 0
		max := 51200

		// get file
		safeURL, err := sanitizeRequestURL(url)
		if err != nil {
			log.Printf("refusing image URL %s: %s", sanitizeForLog(url), err.Error())
			results <- result
			continue
		}
		client := newSafeHTTPClient(10 * time.Second)
		req, err := http.NewRequest(http.MethodGet, safeURL, nil)
		if err != nil {
			log.Printf("error creating request for %s: %s", sanitizeForLog(url), err.Error())
			results <- result
			continue
		}

		rangeHeader := "bytes=" + strconv.Itoa(min) + "-" + strconv.Itoa(max-1)
		req.Header.Add("Range", rangeHeader)
		req.Header.Add("User-agent", "Googlebot-Image/1.0")
		resp, err := client.Do(req)

		if err != nil {
			log.Printf("error pulling %s: %s", sanitizeForLog(url), err.Error())
			results <- result
			continue
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			log.Printf("error reading %s: %s", sanitizeForLog(url), err.Error())
			results <- result
			continue
		}

		// determine image type
		fileType := imageutils.DetermineImageType(&body)

		// get dimensions
		switch fileType {
		case "png":
			result.Width, result.Height = imageutils.PNGDimensions(body)
		case "jpg":
			result.Width, result.Height = imageutils.JPGDimensions(body)
		case "gif":
			result.Width, result.Height = imageutils.GIFDimensions(body)
		case "webp":
			result.Width, result.Height = imageutils.WEBPDimensions(body)
		case "svg":
			result.Width, result.Height = imageutils.SVGDimensions(body)
		}

		result.Area = int(result.Width * result.Height)
		fmt.Printf("url: %s, width: %d, height: %d, area: %d\n",
			result.URL, result.Width, result.Height, result.Area)

		results <- result
	}
}

// GetAllImages on the website
func GetAllImages(re io.Reader, url string, r *http.Request) string {
	// get all images url
	images := htmlutils.ScrapeImg(re, url)

	// count images
	imagesCount := len(images)

	// jobs & results feeds
	jobs := make(chan string, imagesCount)
	results := make(chan ImageResult, imagesCount)

	// spin up workers
	for w := 1; w <= maxWorkers; w++ {
		go GetDimensions(w, jobs, results, r)
	}

	// send jobs
	for j := 0; j < imagesCount; j++ {
		jobs <- images[j]
	}
	close(jobs)

	// collect all results
	var largestImage ImageResult
	for a := 0; a < imagesCount; a++ {
		result := <-results
		if result.Area > largestImage.Area {
			largestImage = result
		}
	}

	if largestImage.URL != "" {
		fmt.Printf("Largest image found: %s (dimensions: %dx%d, area: %d)\n",
			largestImage.URL, largestImage.Width, largestImage.Height, largestImage.Area)
	}

	return largestImage.URL
}

type Output struct {
	Success       bool   `json:"success"`
	Message       string `json:"message"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Keywords      string `json:"keywords"`
	DatePublished string `json:"date_published"`
	LastModified  string `json:"last_modified"`
	LeadImageURL  string `json:"lead_image_url"`
	Dek           string `json:"dek"`
	URL           string `json:"url"`
	Domain        string `json:"domain"`
	Excerpt       string `json:"excerpt"`
	Content       string `json:"content"`
}

type StatusResponse struct {
	Alive   bool   `json:"alive"`
	Version string `json:"version"`
}

// keep minVersion for static builds
var minVersion string

func main() {
	log.Printf("Build: %s\n", minVersion)
	log.Printf("Listening on port: %d", port)

	http.HandleFunc("/status", handleStatus)
	http.HandleFunc("/url/", handleExtract)
	http.HandleFunc("/", handleStatus)

	err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// handleStatus display status with version
func handleStatus(w http.ResponseWriter, r *http.Request) {
	response := StatusResponse{
		Alive:   true,
		Version: minVersion,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// handleExtract process extraction
func handleExtract(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")

	var result Output
	result.Success = false // default to false

	url := r.URL.Query().Get("url")
	if url == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Output{
			Success: false,
			Message: "Can't work without url",
		})
		return
	}

	proxy := r.URL.Query().Get("proxy")
	if proxy == "own" {
		url = fmt.Sprintf("%s%s", os.Getenv("PROXY_OWN"), url)
	}

	safeURL, err := sanitizeRequestURL(url)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Output{
			Success: false,
			Message: fmt.Sprintf("Invalid request URL: %v", err),
		})
		return
	}

	client := newSafeHTTPClient(10 * time.Second)

	// get page
	req, err := http.NewRequest(http.MethodGet, safeURL, nil)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(Output{
			Success: false,
			Message: fmt.Sprintf("Failed to create request: %v", err),
		})
		log.Printf("Can't create request: %s", err.Error())
		return
	}

	// pretend to be google bot ;)
	req.Header.Add("User-agent", "Mozilla/5.0 (compatible; Googlebot/2.1; +http://www.google.com/bot.html)")
	req.Header.Add("Accept-Language", r.Header.Get("Accept-Language"))
	resp, err := client.Do(req)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(Output{
			Success: false,
			Message: fmt.Sprintf("Failed to fetch page: %v", err.Error()),
		})
		log.Printf("Can't read page error: %s", err.Error())
		return
	}

	if resp == nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(Output{
			Success: false,
			Message: "Empty response received",
		})
		return
	}
	defer resp.Body.Close()

	// we do try to extract image
	var urlStr string
	if resp.Request != nil {
		urlStr = resp.Request.URL.String()
	} else {
		urlStr = r.URL.String()
	}

	// get actual URL of page
	result.URL = urlStr
	result.Domain = htmlutils.DomainURL(result.URL)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusBadGateway)
		json.NewEncoder(w).Encode(Output{
			Success: false,
			Message: fmt.Sprintf("Failed to read body: %v", err),
		})
		log.Printf("Failed to read body of: %s, error: %v", sanitizeForLog(result.URL), err)
		return
	}

	// Process the content
	bodyReader := bytes.NewReader(body)
	doc, err := goquery.NewDocumentFromReader(bodyReader)
	if err != nil {
		log.Fatal(err)
	}

	result.Title = htmlutils.SearchForTitleFromDoc(doc)
	result.DatePublished = htmlutils.SearchForDateFromDoc(doc)
	result.Description, err = htmlutils.SearchForMetaTag(bytes.NewReader(body), "description")
	if err != nil {
		slog.Error(err.Error())
	}
	result.Keywords, err = htmlutils.SearchForMetaTag(bytes.NewReader(body), "keywords")
	if err != nil {
		slog.Error(err.Error())
	}

	if lastMod := resp.Header.Get("Last-Modified"); lastMod != "" {
		result.LastModified = lastMod
	}

	result.Content, err = htmlutils.ReadBodyFromDoc(doc)
	if err != nil {
		slog.Error(err.Error())
	}
	result.Dek = strings.Trim(striphtmltags.StripTags(result.Content), " ")
	result.Excerpt = htmlutils.Excerpt(result.Dek)

	// lead image - first try to get it from meta
	promImage, err = htmlutils.SearchForMetaImage(bytes.NewReader(body))
	if err != nil {
		slog.Error(err.Error())
	}

	if promImage == "" {
		promImage = GetAllImages(bytes.NewReader(body), url, r)
	} else {
		// remove proxy url from image
		if proxy == "own" {
			promImage = strings.Replace(url, os.Getenv("PROXY_OWN"), "", 1)
		}
		promImage = htmlutils.GetBaseUrlString(promImage, url)
	}
	result.LeadImageURL = promImage

	// If we got here, everything was successful
	result.Success = true
	result.Message = "Content extracted successfully"

	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(&result); err != nil {
		log.Printf("Failed to encode response: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(Output{
			Success: false,
			Message: "Failed to encode response",
		})
		return
	}
}
