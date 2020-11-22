package imageutils

import (
	"encoding/binary"
	_ "fmt"
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

func JPGDimensions(body []byte) (int32, int32) {
	w, h := JPGHeadersQuick(body)
	if w <= 0 || h <= 0 {
		w, h = JPGHeaders(body)
	}
	return w, h
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
			block_length := int(data[i])*256 + int(data[i+1])
			for i < dataSize {
				i += block_length //Increase the file index to get to the next block

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
					i += 2                                           //Skip the block marker
					block_length = int(data[i])*256 + int(data[i+1]) //Go to the next block
				}
			}
		}
	}

	return 0, 0
}

func DetermineImageType(image *[]byte) string {

	bytes := make([]byte, 4, 4)
	copy(bytes, *image)

	if len(bytes) < 4 {
		return ""
	}

	if bytes[0] == 0x89 && bytes[1] == 0x50 && bytes[2] == 0x4E && bytes[3] == 0x47 {
		return "png"
	}
	if bytes[0] == 0xFF && bytes[1] == 0xD8 {
		return "jpg"
	}
	if bytes[0] == 0x47 && bytes[1] == 0x49 && bytes[2] == 0x46 && bytes[3] == 0x38 {
		return "gif"
	}
	if bytes[0] == 0x42 && bytes[1] == 0x4D {
		return "bmp"
	}
	if bytes[0] == 0x52 && bytes[1] == 0x49 {
		return "webp"
	}
	return ""
}
