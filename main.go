package main

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"io"

	"github.com/slav123/prom/htmlutils"
	"github.com/slav123/prom/imageutils"

	"log"
	"net/http"
	"strconv"
	"strings"

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

// GetDimensions get image dimensions
func GetDimensions(id int, jobs <-chan string, results chan<- int, r *http.Request) {

	var w, h int32

	for url := range jobs {

		w = 0
		h = 0

		fmt.Println("worker", id, "started  job", url)

		// header size to get
		min := 0
		max := 51200

		// get file
		client := &http.Client{}
		req, err := http.NewRequest(http.MethodGet, url, nil)
		if err != nil {
			log.Printf("error creating request for %s: %s", url, err.Error())
			results <- 0
			return
		}
		rangeHeader := "bytes=" + strconv.Itoa(min) + "-" + strconv.Itoa(max-1) // Add the data for the Range header of the form "bytes=0-100"
		req.Header.Add("Range", rangeHeader)
		req.Header.Add("User-agent", "Googlebot-Image/1.0")
		// req.Header.Add("Referer", base_url)
		resp, err := client.Do(req)

		if err != nil {
			log.Printf("error pulling %s: %s", url, err.Error())

			results <- 0
			return
		}

		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)

		if err != nil {
			log.Printf("error reading %s: %s", url, err.Error())
			results <- 0
			return
		}

		// display body type
		//fmt.Println(body[0:max])
		//fmt.Printf("%#X\n", body[0:max])

		// determine image type
		fileType := imageutils.DetermineImageType(&body)

		// get dimensions
		switch fileType {
		case "png":
			w, h = imageutils.PNGDimensions(body)
		case "jpg":
			w, h = imageutils.JPGDimensions(body)
		case "gif":
			w, h = imageutils.GIFDimensions(body)

		default:
			w = 0
			h = 0
		}

		z := int(w * h)

		fmt.Printf("url: %s, width: %d, height: %d, z: %d\n", url, w, h, z)

		if z > maxDimensions {
			promImage = url
			maxDimensions = z
			fmt.Println("hit", z)
		}

		results <- z
	}
}

// GetAllImages on the website
func GetAllImages(re io.Reader, url string, r *http.Request) {

	// get all images url
	images := htmlutils.ScrapeImg(re, url)

	// count images
	imagesCount := len(images)

	//fmt.Println("found ", imagesCount)

	// jobs & results feeds
	jobs := make(chan string, imagesCount)
	results := make(chan int, imagesCount)

	// spin up workers
	for w := 1; w <= maxWorkers; w++ {
		go GetDimensions(w, jobs, results, r)
	}

	// send jobs
	for j := 0; j < imagesCount; j++ {
		jobs <- images[j]
	}
	close(jobs)

	// get results
	for a := 0; a < imagesCount; a++ {
		<-results
	}

	fmt.Println("prom image", promImage)

}

type Output struct {
	Title         string `json:"title"`
	Success       bool   `json:"success"`
	Message       string `json:"message"`
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

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: tr,
	}

	// get page
	req, err := http.NewRequest(http.MethodGet, url, nil)
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
		log.Printf("Failed to read body of: %s, error: %v", result.URL, err)
		return
	}

	// Process the content
	result.Title = htmlutils.SearchForTitle(bytes.NewReader(body))

	result.DatePublished = htmlutils.SearchForDate(bytes.NewReader(body))

	if lastMod := resp.Header.Get("Last-Modified"); lastMod != "" {
		result.LastModified = lastMod
	}

	result.Content = htmlutils.ReadBody(string(body))
	result.Dek = strings.Trim(striphtmltags.StripTags(result.Content), " ")
	result.Excerpt = htmlutils.Excerpt(result.Dek)

	// lead image - first try to get it from meta
	promImage = htmlutils.SearchForMeta(bytes.NewReader(body))
	if promImage == "" {
		maxDimensions = 0
		GetAllImages(bytes.NewReader(body), url, r)
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
