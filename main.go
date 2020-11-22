package prom

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"github.com/facebookgo/pidfile"
	"os"
	"time"

	"github.com/slav123/prom/htmlutils"
	"github.com/slav123/prom/imageutils"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"

	"github.com/denisbrodbeck/striphtmltags"
	_ "google.golang.org/appengine"
	_ "google.golang.org/appengine/urlfetch"
)

const maxWorkers = 5

var (
	promImage     string
	maxDimensions int
)

func GetDimensions(id int, jobs <-chan string, results chan<- int, r *http.Request) {

	var w, h int32

	for url := range jobs {

		w = 0
		h = 0

		fmt.Println("worker", id, "started  job", url)

		// header size to get
		min := 0
		max := 51200

		// get file - app engine
		/*
			ctx := appengine.NewContext(r)
			client := urlfetch.Client(ctx)

			req, err := http.NewRequest("GET", url, nil)
			rangeHeader := "bytes=" + strconv.Itoa(min) + "-" + strconv.Itoa(max-1) // Add the data for the Range header of the form "bytes=0-100"
			req.Header.Add("Range", rangeHeader)
			resp, err := client.Do(req)
		*/
		// end app engine

		// get file
		client := &http.Client{}
		req, err := http.NewRequest("GET", url, nil)
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

		body, err := ioutil.ReadAll(resp.Body)

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
	Title string `json:"title"`

	//  DatePublished time.Time `json:"date_published"`
	DatePublished string `json:"date_published"`
	LeadImageURL  string `json:"lead_image_url"`
	Dek           string `json:"dek"`
	URL           string `json:"url"`
	Domain        string `json:"domain"`
	Excerpt       string `json:"excerpt"`
	Content       string `json:"content"`
}

// generate pid file in /tmp
func init() {
	tempDir := os.TempDir()
	pidfile.SetPidfilePath(tempDir + "/prom.pid")
	err := pidfile.Write()
	if err != nil {
		log.Fatalf("Unable to create pid file %s\n", err)
	}
}

// keep minVersion for static builds
var minVersion string

func main() {
	log.Printf("Build: %s\n", minVersion)
	log.Println("Listening on port: 9090")

	http.HandleFunc("/status", handleStatus)

	http.HandleFunc("/url/", handleExtract)

	http.HandleFunc("/", handleStatus)

	//appengine.Main()

	err := http.ListenAndServe(":9090", nil)
	if err != nil {
		log.Fatal("ListenAndServe: ", err)
	}
}

// display status with version
func handleStatus(w http.ResponseWriter, r *http.Request) {
	// A very simple health check.
	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte("{ \"alive\" : true, \"version\" : \"" + minVersion + "\"}"))
}

// process extraction
func handleExtract(w http.ResponseWriter, r *http.Request) {
	url := r.URL.Query().Get("url")

	var result Output

	/* app engine
	ctx := appengine.NewContext(r)
	client := urlfetch.Client(ctx)
	resp, err := client.Get(url)
	*/

	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	client := &http.Client{
		Timeout:   time.Second * 10,
		Transport: tr,
	}

	// get page
	req, err := http.NewRequest("GET", url, nil)

	// pretend to be google bot ;)
	req.Header.Add("User-agent", "Googlebot")
	resp, err := client.Do(req)

	if err != nil {
		//w.WriteHeader(404)
		w.Write([]byte("I'm not ok"))
		log.Printf("Can't read page error: %s", err.Error())
	}
	defer resp.Body.Close()

	// we do try to extract image
	var urlStr string

	if resp != nil && resp.Request != nil {
		urlStr = resp.Request.URL.String()
	} else {
		urlStr = r.URL.String()
	}

	// get actual URL of page
	result.URL = urlStr

	// domain
	result.Domain = htmlutils.DomainURL(result.URL)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		fmt.Printf("failed to read body of : %s ", result.URL)
		w.Write([]byte("I'm not ok, can't read body "))
	}

	// copy body to process it again
	resp.Request.Body = ioutil.NopCloser(bytes.NewBuffer(body))

	// log for title
	result.Title = htmlutils.SearchForTitle(bytes.NewReader(body))

	// date
	result.DatePublished = htmlutils.SearchForDate(bytes.NewReader(body))

	// content
	result.Content = htmlutils.ReadBody(string(body))

	// trimed
	result.Dek = strings.Trim(striphtmltags.StripTags(result.Content), " ")

	// excerpt
	result.Excerpt = htmlutils.Excerpt(result.Dek)

	// lead image - first try tu get it form meta
	promImage = htmlutils.SearchForMeta(bytes.NewReader(body))

	if promImage == "" {
		maxDimensions = 0
		GetAllImages(bytes.NewReader(body), url, r)
	} else {
		promImage = htmlutils.GetBaseUrlString(promImage, url)
	}
	result.LeadImageURL = promImage

	buf := new(bytes.Buffer)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)

	if err := enc.Encode(&result); err != nil {
		log.Println(err)
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Origin, X-Requested-With, Content-Type, Accept, Authorization")
	w.WriteHeader(200)
	w.Write(buf.Bytes())
	//out.WriteTo(os.Stdout)

}
