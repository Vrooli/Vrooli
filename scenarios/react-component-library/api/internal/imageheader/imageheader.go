// Package imageheader validates uploaded reference images without decoding
// pixels: it recognises the five accepted image types, normalises a declared
// MIME type, and reads width/height from the signature each format places
// first. Unrecognised payloads report 0,0 rather than guessing.
//
// It is deliberately dependency-free and pure so the asset store, the upload
// validator and any future importer share one answer for "what is this file and
// how big is it".
package imageheader

import (
	"encoding/binary"
	"strings"
)

// MaxBytes bounds one decoded upload. Reference shots are screenshots; 8 MiB
// leaves room for a full-resolution capture without turning a table into a blob
// store. Uploads are base64 in JSON, so the request bound is ~4/3 of this plus
// the small metadata fields.
const MaxBytes = 8 << 20

// AllowedMimes are the image types a reference asset may declare.
var AllowedMimes = map[string]struct{}{
	"image/png":     {},
	"image/jpeg":    {},
	"image/webp":    {},
	"image/gif":     {},
	"image/svg+xml": {},
}

// NormalizeMime lowercases a declared type and drops any parameters. It does
// not decide whether the type is accepted; AllowedMimes owns that.
func NormalizeMime(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	if semi := strings.Index(value, ";"); semi >= 0 {
		value = strings.TrimSpace(value[:semi])
	}
	return value
}

// Dimensions reads width and height from the image header without a codec.
// SVG and unrecognised payloads return 0,0; the caller keeps the bytes and
// records unknown dimensions rather than inventing them.
func Dimensions(mime string, data []byte) (int, int) {
	switch NormalizeMime(mime) {
	case "image/png":
		return pngDimensions(data)
	case "image/gif":
		return gifDimensions(data)
	case "image/jpeg":
		return jpegDimensions(data)
	case "image/webp":
		return webpDimensions(data)
	default:
		return 0, 0
	}
}

func pngDimensions(data []byte) (int, int) {
	// 8-byte signature, then a length/type (8) before the IHDR payload.
	if len(data) < 24 || string(data[12:16]) != "IHDR" {
		return 0, 0
	}
	width := int(binary.BigEndian.Uint32(data[16:20]))
	height := int(binary.BigEndian.Uint32(data[20:24]))
	return width, height
}

func gifDimensions(data []byte) (int, int) {
	if len(data) < 10 || (string(data[0:6]) != "GIF87a" && string(data[0:6]) != "GIF89a") {
		return 0, 0
	}
	return int(binary.LittleEndian.Uint16(data[6:8])), int(binary.LittleEndian.Uint16(data[8:10]))
}

func isJPEGFrameMarker(marker byte) bool {
	switch marker {
	case 0xC0, 0xC1, 0xC2, 0xC3, 0xC5, 0xC6, 0xC7,
		0xC9, 0xCA, 0xCB, 0xCD, 0xCE, 0xCF:
		return true
	default:
		return false
	}
}

func jpegDimensions(data []byte) (int, int) {
	if len(data) < 4 || data[0] != 0xFF || data[1] != 0xD8 {
		return 0, 0
	}
	for index := 2; index+8 < len(data); {
		if data[index] != 0xFF {
			index++
			continue
		}
		marker := data[index+1]
		// Standalone markers carry no segment length.
		if marker == 0xD8 || marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7) {
			index += 2
			continue
		}
		if marker == 0xDA || marker == 0xD9 {
			return 0, 0 // start of scan: no frame header found
		}
		segmentLength := int(binary.BigEndian.Uint16(data[index+2 : index+4]))
		if segmentLength < 2 {
			return 0, 0
		}
		if isJPEGFrameMarker(marker) {
			if index+9 > len(data) {
				return 0, 0
			}
			height := int(binary.BigEndian.Uint16(data[index+5 : index+7]))
			width := int(binary.BigEndian.Uint16(data[index+7 : index+9]))
			return width, height
		}
		index += 2 + segmentLength
	}
	return 0, 0
}

func webpDimensions(data []byte) (int, int) {
	if len(data) < 30 || string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return 0, 0
	}
	switch string(data[12:16]) {
	case "VP8 ":
		// Lossy: 3-byte frame tag, 3-byte start code, then 14-bit dimensions.
		if data[23] != 0x9D || data[24] != 0x01 || data[25] != 0x2A {
			return 0, 0
		}
		width := int(binary.LittleEndian.Uint16(data[26:28]) & 0x3FFF)
		height := int(binary.LittleEndian.Uint16(data[28:30]) & 0x3FFF)
		return width, height
	case "VP8L":
		if data[20] != 0x2F {
			return 0, 0
		}
		bits := binary.LittleEndian.Uint32(data[21:25])
		return int(bits&0x3FFF) + 1, int((bits>>14)&0x3FFF) + 1
	case "VP8X":
		width := int(uint32(data[24])|uint32(data[25])<<8|uint32(data[26])<<16) + 1
		height := int(uint32(data[27])|uint32(data[28])<<8|uint32(data[29])<<16) + 1
		return width, height
	default:
		return 0, 0
	}
}
