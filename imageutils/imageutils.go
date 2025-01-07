package imageutils

import (
	"bytes"
	"encoding/binary"
	"encoding/xml"
	"fmt"
	"regexp"
)

// read PNG and return dimensions
func PNGDimensions(body []byte) (int32, int32) {
	const offset = 16
	return int32(binary.BigEndian.Uint32(body[offset : offset+4])), int32(binary.BigEndian.Uint32(body[(offset + 4) : offset+12]))
}

func GIFDimensions(body []byte) (int32, int32) {
	const offset = 6
	return int32(binary.LittleEndian.Uint16(body[offset : offset+2])), int32(binary.LittleEndian.Uint16(body[(offset + 2) : offset+4]))

}

// read JPG and return dimensions
func JPGDimensions(body []byte) (int32, int32) {
	w, h := JPGHeadersQuick(body)
	if w <= 0 || h <= 0 {
		w, h = JPGHeaders(body)
	}
	return w, h
}

// WEBPDimensions returns the width and height of a WebP image
func WEBPDimensions(header []byte) (int32, int32) {
	// Check if the file is a WebP
	if string(header[:4]) != "RIFF" || string(header[8:12]) != "WEBP" {
		return 0, 0
	}

	// Extract width and height from the header
	width := int(header[26]) | (int(header[27]) << 8)
	height := int(header[28]) | (int(header[29]) << 8)

	return int32(width), int32(height)
}

// read JPG headers and return dimensions look only for basic marker
func JPGHeaders(body []byte) (int32, int32) {
	offset := 0

	for i := range body {
		if body[i] == 0xFF && (body[i+1] == 0xC0 || body[i+1] == 0xC2) {
			offset = i + 5
			break
		}

	}

	const size = 2
	return int32(binary.BigEndian.Uint16(body[(offset + size):(offset + (2 * size))])), int32(binary.BigEndian.Uint16(body[offset : offset+size]))

}

// look for complete marker
func JPGHeadersQuick(data []byte) (int32, int32) {
	var width, height, i int

	dataSize := len(data)

	if data[i] == 0xFF && data[i+1] == 0xD8 && data[i+2] == 0xFF && data[i+3] == 0xE0 {
		i += 4
		if data[i+2] == 'J' && data[i+3] == 'F' && data[i+4] == 'I' && data[i+5] == 'F' && data[i+6] == 0x00 {
			blockLength := int(data[i])*256 + int(data[i+1])
			for i < dataSize {
				i += blockLength //Increase the file index to get to the next block

				if i >= dataSize {
					return -1, -1 //Check to protect against segmentation faults
				}

				if data[i] != 0xFF {
					return -1, -1 //Check that we are truly at the start of another block
				}

				if data[i+1] == 0xC0 || data[i+1] == 0xC2 {
					//0xFFC0 is the "Start of frame" marker which contains the file size
					//The structure of the 0xFFC0 block is quite simple [0xFFC0][ushort length][uchar precision][ushort x][ushort y]
					height = int(data[i+5])*256 + int(data[i+6])
					width = int(data[i+7])*256 + int(data[i+8])
					return int32(width), int32(height)
				} else {
					i += 2                                          //Skip the block marker
					blockLength = int(data[i])*256 + int(data[i+1]) //Go to the next block
				}
			}
		}
	}

	return 0, 0
}

// SVGDimensions reads SVG file header and returns dimensions
func SVGDimensions(body []byte) (int32, int32) {
	type SVG struct {
		Width  string `xml:"width,attr"`
		Height string `xml:"height,attr"`
	}

	var svg SVG
	if err := xml.NewDecoder(bytes.NewReader(body)).Decode(&svg); err != nil {
		return 0, 0
	}

	// Regular expression to extract numeric values
	re := regexp.MustCompile(`^(\d+)`)

	// Extract width
	widthMatch := re.FindString(svg.Width)
	var width int32
	if widthMatch != "" {
		var w int
		fmt.Sscanf(widthMatch, "%d", &w)
		width = int32(w)
	}

	// Extract height
	heightMatch := re.FindString(svg.Height)
	var height int32
	if heightMatch != "" {
		var h int
		fmt.Sscanf(heightMatch, "%d", &h)
		height = int32(h)
	}

	return width, height
}

// DetermineImageType returns the image type
func DetermineImageType(image *[]byte) string {

	img := make([]byte, 512) // Increased buffer to detect SVG
	copy(img, *image)

	if len(img) < 4 {
		return ""
	}

	if img[0] == 0x89 && img[1] == 0x50 && img[2] == 0x4E && img[3] == 0x47 {
		return "png"
	}
	if img[0] == 0xFF && img[1] == 0xD8 {
		return "jpg"
	}
	if img[0] == 0x47 && img[1] == 0x49 && img[2] == 0x46 && img[3] == 0x38 {
		return "gif"
	}

	// Check for SVG
	if bytes.Contains(img, []byte("<?xml")) && bytes.Contains(img, []byte("<svg")) {
		return "svg"
	}
	if bytes.HasPrefix(img, []byte("<svg")) {
		return "svg"
	}
	if img[0] == 0x42 && img[1] == 0x4D {
		return "bmp"
	}
	if img[0] == 0x52 && img[1] == 0x49 {
		return "webp"
	}
	return ""
}
