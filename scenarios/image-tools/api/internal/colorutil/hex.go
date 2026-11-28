// Package colorutil owns shared parsing for image color literals.
package colorutil

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"
)

const hexDigits = "0123456789abcdefABCDEF"

// ParseHex parses #rgb, #rrggbb, or #rrggbbaa (the leading # is optional).
// Whitespace around a literal is ignored. The alpha channel defaults to 255.
func ParseHex(value string) (color.NRGBA, error) {
	value = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(value), "#"))
	if len(value) == 3 {
		value = string([]byte{value[0], value[0], value[1], value[1], value[2], value[2]})
	}
	if len(value) != 6 && len(value) != 8 {
		return color.NRGBA{}, fmt.Errorf("invalid hex color %q", value)
	}
	if strings.Trim(value, hexDigits) != "" {
		return color.NRGBA{}, fmt.Errorf("invalid hex color %q", value)
	}
	n, err := strconv.ParseUint(value, 16, 32)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("invalid hex color %q", value)
	}
	if len(value) == 6 {
		n = n<<8 | 0xff
	}
	return color.NRGBA{R: uint8(n >> 24), G: uint8(n >> 16), B: uint8(n >> 8), A: uint8(n)}, nil
}
