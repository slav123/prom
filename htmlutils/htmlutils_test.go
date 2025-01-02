package htmlutils

import (
	"bytes"
	"os"
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
	tests := []struct {
		name     string
		src      string
		baseURL  string
		expected string
	}{
		{
			name:     "relative path",
			src:      "/files/assets/public/image-resources/haveyoursay/ccc_artscentre.jpg",
			baseURL:  "http://www.campbelltown.nsw.gov.au/WhatsOn/CampbelltownCommunityFacilitiesStrategy",
			expected: "http://www.campbelltown.nsw.gov.au/files/assets/public/image-resources/haveyoursay/ccc_artscentre.jpg",
		},
		{
			name:     "absolute URL",
			src:      "http://www.test.com/image.gif",
			baseURL:  "http://www.campbelltown.nsw.gov.au/WhatsOn/CampbelltownCommunityFacilitiesStrategy",
			expected: "http://www.test.com/image.gif",
		},
		{
			name:     "relative path with dots",
			src:      "../images/photo.jpg",
			baseURL:  "http://example.com/blog/post/",
			expected: "http://example.com/blog/images/photo.jpg",
		},
		{
			name:     "protocol-relative URL",
			src:      "//cdn.example.com/image.png",
			baseURL:  "https://example.com/page",
			expected: "https://cdn.example.com/image.png",
		},
		{
			name:     "query parameters",
			src:      "/images/photo.jpg?size=large",
			baseURL:  "http://example.com/gallery",
			expected: "http://example.com/images/photo.jpg?size=large",
		},
		{
			name:     "no trailing slash",
			src:      "/images/photo.jpg",
			baseURL:  "http://example.com/gallery",
			expected: "http://example.com/images/photo.jpg",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetBaseUrlString(tt.src, tt.baseURL)
			if result != tt.expected {
				t.Errorf("GetBaseUrlString(%q, %q) = %q, want %q",
					tt.src, tt.baseURL, result, tt.expected)
			}
		})
	}
}

func TestSearchForTitle(t *testing.T) {
	//	var fn ImageOp
	body, _ := os.ReadFile("samples/body.html")

	result := SearchForTitle(bytes.NewReader(body))
	if result != "test" {
		t.Errorf("Failed to extract TestSearchForTitle %s", result)
	}
}
