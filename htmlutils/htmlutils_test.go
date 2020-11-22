package htmlutils

import (
	"bytes"
	"io/ioutil"
	"testing"
)

func TestBaseURL(t *testing.T) {
	data := "http://www.gex.pl/test.html"
	result := BaseURL(data)
	if result != "http://www.gex.pl/" {
		t.Errorf("Failed to extract BaseURL %s", result)
	}
}
func TestServerURL(t *testing.T) {
	data := "http://www.gex.pl/test.html"
	result := ServerURL(data)
	if result != "http://www.gex.pl" {
		t.Errorf("Failed to extract ServerURL %s", result)
	}
}

func TestGetBaseUrlString(t *testing.T) {
	src := "/files/assets/public/image-resources/haveyoursay/ccc_artscentre.jpg"
	url := "http://www.campbelltown.nsw.gov.au/WhatsOn/CampbelltownCommunityFacilitiesStrategy"
	result := GetBaseUrlString(src, url)

	if result != "http://www.campbelltown.nsw.gov.au/files/assets/public/image-resources/haveyoursay/ccc_artscentre.jpg" {
		t.Errorf("Failed to extract getBaseUrlString %s", result)
	}

	result = GetBaseUrlString("http://www.test.com/image.gif", url)

	if result != "http://www.test.com/image.gif" {
		t.Errorf("Failed to extract getBaseUrlString %s", result)
	}

}

func TestSearchForTitle(t *testing.T) {
	//	var fn ImageOp
	body, _ := ioutil.ReadFile("samples/body.html")

	result := SearchForTitle(bytes.NewReader(body))
	if result != "test" {
		t.Errorf("Failed to extract TestSearchForTitle %s", result)
	}
}
