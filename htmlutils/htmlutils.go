package htmlutils

import (
	"fmt"
	"io"
	"log"
	"net/url"

	"github.com/PuerkitoBio/goquery"
	"github.com/mauidude/go-readability"

	"strings"
)

// BaseURL return base url - string .html or last element
func BaseURL(url string) string {
	split := strings.Split(url, "/")
	return (strings.Join(split[0:len(split)-1], "/") + "/")
}

// ServerURL return server url - root domain
func ServerURL(url string) string {
	split := strings.Split(url, "/")
	return strings.Join(split[0:3], "/")
}

// DomainURL get's domain url
func DomainURL(iurl string) string {
	u, _ := url.Parse(iurl)
	return u.Scheme + "://" + u.Host
}

func cleanup(textcopy string) string {

	textcopy = strings.Replace(textcopy, "\r\n", " ", -1)
	textcopy = strings.Replace(textcopy, "\n", " ", -1)
	textcopy = strings.Replace(textcopy, "\t", "", -1)

	for strings.Contains(textcopy, "  ") {
		textcopy = strings.Replace(textcopy, "  ", " ", -1)
	}

	return textcopy
}

//  Excerpt generate excerpt
func Excerpt(textcopy string) string {

	textcopy = cleanup(textcopy)

	split := strings.Split(textcopy, " ")

	max := 70

	if len(split) < max {
		max = len(split)
	}

	return strings.Trim(strings.Join(split[0:max], " "), " ")
}

// ScrapeImg scrape all images from given copy
func ScrapeImg(r io.Reader, url string) []string {
	images := make([]string, 0)

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err)
	}
	// Find the review items
	doc.Find("img").Each(func(i int, s *goquery.Selection) {
		src := s.AttrOr("src", "")

		if strings.Contains(src, "data:image") {
			return
		}

		fullImageUrl := GetBaseUrlString(src, url)
		fmt.Println(src, url)

		images = append(images, fullImageUrl)

	})

	return images

}

func GetBaseUrlString(src, url string) string {

	if string(src[0]) == "/" {
		return ServerURL(url) + src
	} else if string(src[0:4]) == "http" {
		return src
	} else {
		return BaseURL(url) + src
	}
}

func SearchForTitle(r io.Reader) string {
	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err)
	}

	return doc.Find("title").Text()
}

func SearchForDate(r io.Reader) string {

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err)
	}

	var content string
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {

		hit := false

		itemprop, _ := s.Attr("itemprop")

		if itemprop == "datePublished" || itemprop == "article:published_time" {
			hit = true
		}

		property, _ := s.Attr("property")
		if property == "article:published_time" {
			hit = true
		}

		//<meta property="article:published_time" content="2018-03-19T09:17:23+10:00" />

		if hit {
			content = s.AttrOr("content", "")
			return
		}
	})
	return content
}

// serach for meta og image or sth
func SearchForMeta(r io.Reader) string {

	doc, err := goquery.NewDocumentFromReader(r)
	if err != nil {
		log.Fatal(err)
	}

	var content string
	doc.Find("meta").Each(func(i int, s *goquery.Selection) {

		hit := false

		name, _ := s.Attr("name")

		if name == "og:image" {
			hit = true
		}

		property, _ := s.Attr("property")

		if property == "og:image" {
			hit = true
		}

		itemprop, _ := s.Attr("itemprop")

		if itemprop == "image" {
			hit = true
		}

		if hit {

			content = s.AttrOr("content", "")
			return
		}
	})

	return content
}

func ReadBody(body string) string {

	doc, err := readability.NewDocument(body)
	if err != nil {
		// do something ...
	}

	content := doc.Content()
	// do something with my content

	content = strings.Replace(content, "<head></head>", "", -1)
	content = strings.Replace(content, "<html>", "", -1)
	content = strings.Replace(content, "</html>", "", -1)
	content = strings.Replace(content, "<body>", "", -1)
	content = strings.Replace(content, "</body>", "", -1)
	//content = strings.Trim(content, "<div>")
	//content = strings.Trim(content, "</div>")

	return cleanup(content)
}
