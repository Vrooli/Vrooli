package projects

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

var ErrUnsupportedImageMetadata = errors.New("unsupported image color metadata")

func sourceImageMetadata(data []byte, format string) (orientation int, hasProfile bool, profileStatus string, err error) {
	orientation = 1
	profileStatus = "unspecified"
	orientationSeen := false
	switch format {
	case "jpeg":
		if len(data) < 2 || data[0] != 0xff || data[1] != 0xd8 {
			return 0, false, "", fmt.Errorf("invalid JPEG signature")
		}
		for pos := 2; pos < len(data); {
			for pos < len(data) && data[pos] != 0xff {
				pos++
			}
			if pos >= len(data) {
				break
			}
			for pos < len(data) && data[pos] == 0xff {
				pos++
			}
			if pos >= len(data) {
				return 0, false, "", fmt.Errorf("truncated JPEG marker")
			}
			marker := data[pos]
			pos++
			if marker == 0xd9 {
				return orientation, hasProfile, profileStatus, nil
			}
			if marker == 0xda {
				if pos+2 > len(data) {
					return 0, false, "", fmt.Errorf("truncated JPEG scan header")
				}
				length := int(binary.BigEndian.Uint16(data[pos : pos+2]))
				if length < 2 || pos+length > len(data) {
					return 0, false, "", fmt.Errorf("invalid JPEG scan header")
				}
				pos += length
				for pos < len(data) {
					if data[pos] != 0xff {
						pos++
						continue
					}
					start := pos
					pos++
					for pos < len(data) && data[pos] == 0xff {
						pos++
					}
					if pos >= len(data) {
						return 0, false, "", fmt.Errorf("truncated JPEG entropy marker")
					}
					m := data[pos]
					if m == 0x00 || m >= 0xd0 && m <= 0xd7 {
						pos++
						continue
					}
					pos = start
					break
				}
				continue
			}
			if marker == 0x01 || marker >= 0xd0 && marker <= 0xd7 {
				continue
			}
			if pos+2 > len(data) {
				return 0, false, "", fmt.Errorf("truncated JPEG segment")
			}
			length := int(binary.BigEndian.Uint16(data[pos : pos+2]))
			if length < 2 || pos+length > len(data) {
				return 0, false, "", fmt.Errorf("invalid JPEG segment length")
			}
			payload := data[pos+2 : pos+length]
			if marker == 0xe1 && bytes.HasPrefix(payload, []byte("Exif\x00\x00")) {
				value, found, parseErr := parseTIFFOrientation(payload)
				if parseErr != nil {
					return 0, false, "", parseErr
				}
				if found {
					if orientationSeen && orientation != value {
						return 0, false, "", fmt.Errorf("conflicting EXIF orientation values")
					}
					orientation = value
					orientationSeen = true
				}
			}
			if marker == 0xe2 && bytes.HasPrefix(payload, []byte("ICC_PROFILE\x00")) {
				hasProfile = true
			}
			pos += length
		}
		return orientation, hasProfile, profileStatus, nil
	case "png":
		if len(data) < 8 || !bytes.Equal(data[:8], []byte{137, 80, 78, 71, 13, 10, 26, 10}) {
			return 0, false, "", fmt.Errorf("invalid PNG signature")
		}
		for pos := 8; pos+12 <= len(data); {
			length := uint64(binary.BigEndian.Uint32(data[pos : pos+4]))
			end := uint64(pos) + 12 + length
			if end > uint64(len(data)) {
				return 0, false, "", fmt.Errorf("truncated PNG chunk")
			}
			chunk := string(data[pos+4 : pos+8])
			payload := data[pos+8 : uint64(pos+8)+length]
			switch chunk {
			case "iCCP":
				hasProfile = true
			case "cICP", "gAMA", "cHRM":
				return 0, false, "", fmt.Errorf("%w: PNG %s", ErrUnsupportedImageMetadata, chunk)
			case "sRGB":
				if length != 1 || payload[0] > 3 || profileStatus == "srgb" {
					return 0, false, "", fmt.Errorf("invalid or duplicate PNG sRGB declaration")
				}
				profileStatus = "srgb"
			case "eXIf":
				value, found, parseErr := parseTIFFOrientation(payload)
				if parseErr != nil {
					return 0, false, "", parseErr
				}
				if found {
					if orientationSeen && orientation != value {
						return 0, false, "", fmt.Errorf("conflicting EXIF orientation values")
					}
					orientation = value
					orientationSeen = true
				}
			case "acTL":
				return 0, false, "", fmt.Errorf("animated PNG is unsupported")
			case "IEND":
				return orientation, hasProfile, profileStatus, nil
			}
			pos = int(end)
		}
		return orientation, hasProfile, profileStatus, nil
	default:
		return 0, false, "", fmt.Errorf("unsupported image format")
	}
}

func parseTIFFOrientation(exif []byte) (int, bool, error) {
	tiff := exif
	if bytes.HasPrefix(tiff, []byte("Exif\x00\x00")) {
		tiff = tiff[6:]
	}
	if len(tiff) < 8 {
		return 0, false, fmt.Errorf("truncated EXIF TIFF header")
	}
	var order binary.ByteOrder
	switch string(tiff[:2]) {
	case "II":
		order = binary.LittleEndian
	case "MM":
		order = binary.BigEndian
	default:
		return 0, false, fmt.Errorf("invalid EXIF byte order")
	}
	if order.Uint16(tiff[2:4]) != 42 {
		return 0, false, fmt.Errorf("invalid EXIF TIFF magic")
	}
	ifd := uint64(order.Uint32(tiff[4:8]))
	if ifd+2 > uint64(len(tiff)) {
		return 0, false, fmt.Errorf("EXIF IFD offset is out of bounds")
	}
	count := uint64(order.Uint16(tiff[ifd : ifd+2]))
	if ifd+2+count*12 > uint64(len(tiff)) {
		return 0, false, fmt.Errorf("truncated EXIF IFD")
	}
	valueFound := false
	orientation := 1
	for i := uint64(0); i < count; i++ {
		entry := tiff[ifd+2+i*12 : ifd+2+(i+1)*12]
		if order.Uint16(entry[:2]) != 0x0112 {
			continue
		}
		if valueFound {
			return 0, false, fmt.Errorf("duplicate EXIF orientation tags")
		}
		if order.Uint16(entry[2:4]) != 3 || order.Uint32(entry[4:8]) != 1 {
			return 0, false, fmt.Errorf("unsupported EXIF orientation value shape")
		}
		value := int(order.Uint16(entry[8:10]))
		if value < 1 || value > 8 {
			return 0, false, fmt.Errorf("invalid EXIF orientation value")
		}
		orientation, valueFound = value, true
	}
	return orientation, valueFound, nil
}
