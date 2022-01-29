package imageutils

import (
	"io/ioutil"
	"testing"
)

var testExt = []struct {
	ext, source string
}{
	{"png", "samples/file.png"},
	{"jpg", "samples/file.jpg"},
	{"webp", "samples/file.webp"},
	{"gif", "samples/file.gif"},
}

func TestDetermineImageType(t *testing.T) {
	for _, sample := range testExt {
		data, err := ioutil.ReadFile(sample.source)
		check(err)
		if v := DetermineImageType(&data); v != sample.ext {
			t.Errorf("GetImageType(%s) returned %v, expected %v", sample.source, v, sample.ext)
		}
	}
}

func TestPNGDimensions(t *testing.T) {
	data, err := ioutil.ReadFile("samples/file.png")
	check(err)
	if w, h := PNGDimensions(data); w != 521 || h != 450 {
		t.Errorf("PNGDimensions (samples/file.png) returned (%d, %d), expected %d, %d", w, h, 521, 450)
	}
}

func TestGIFDimensions(t *testing.T) {
	data, err := ioutil.ReadFile("samples/file.gif")
	check(err)
	if w, h := GIFDimensions(data); w != 251 || h != 201 {
		t.Errorf("PNGDimensions (samples/file.png) returned (%d, %d), expected %d, %d", w, h, 251, 201)
	}
}

func TestJPGHeaders(t *testing.T) {

	var samples = []struct {
		src  string
		w, h int32
	}{
		{"samples/file.jpg", 251, 201},
		{"samples/file2.jpg", 550, 449},
		{"samples/file3.jpg", 800, 598},
	}

	for _, sample := range samples {
		data, err := ioutil.ReadFile(sample.src)
		check(err)
		if w, h := JPGHeaders(data); w != sample.w || h != sample.h {
			t.Errorf("JPGHeaders (%s) returned (%d, %d), expected %d, %d", sample.src, w, h, sample.w, sample.h)
		}
	}
}

func TestJPGHeadersQuick(t *testing.T) {

	var samples = []struct {
		src  string
		w, h int32
	}{
		{"samples/file.jpg", 251, 201},
		{"samples/file2.jpg", 550, 449},
	}

	for _, sample := range samples {
		data, err := ioutil.ReadFile(sample.src)
		check(err)
		if w, h := JPGHeadersQuick(data); w != sample.w || h != sample.h {
			t.Errorf("JPGHeadersQuick (%s) returned (%d, %d), expected %d, %d", sample.src, w, h, sample.w, sample.h)
		}
	}
}

func TestJPGDimensions(t *testing.T) {

	var samples = []struct {
		src  string
		w, h int32
	}{
		{"samples/file.jpg", 251, 201},
		{"samples/file2.jpg", 550, 449},
		{"samples/file3.jpg", 800, 598},
	}

	for _, sample := range samples {
		data, err := ioutil.ReadFile(sample.src)
		check(err)
		if w, h := JPGDimensions(data); w != sample.w || h != sample.h {
			t.Errorf("JPGDimensions (%s) returned (%d, %d), expected %d, %d", sample.src, w, h, sample.w, sample.h)
		}
	}
}

func BenchmarkPrepare(b *testing.B) {
	//	var fn ImageOp
	data, err := ioutil.ReadFile("samples/file.jpg")
	check(err)
	for i := 0; i < b.N; i++ {
		_ = DetermineImageType(&data)
	}
}

func BenchmarkTestJPG(b *testing.B) {
	//	var fn ImageOp
	data, err := ioutil.ReadFile("samples/file4.jpg")
	check(err)
	for i := 0; i < b.N; i++ {
		_, _ = JPGHeaders(data)
	}
}

func BenchmarkTestJPGHeadersQuick(b *testing.B) {
	//	var fn ImageOp
	data, err := ioutil.ReadFile("samples/file4.jpg")
	check(err)
	for i := 0; i < b.N; i++ {
		_, _ = JPGHeadersQuick(data)
	}
}
func check(e error) {
	if e != nil {
		panic(e)
	}
}
