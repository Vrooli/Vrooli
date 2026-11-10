package colorutil

import (
	"image/color"
	"testing"
)

func TestParseHexForms(t *testing.T) {
	for _, tc := range []struct {
		value string
		want  color.NRGBA
	}{
		{"#abc", color.NRGBA{R: 0xaa, G: 0xbb, B: 0xcc, A: 255}},
		{" 112233 ", color.NRGBA{R: 0x11, G: 0x22, B: 0x33, A: 255}},
		{"#12345678", color.NRGBA{R: 0x12, G: 0x34, B: 0x56, A: 0x78}},
	} {
		got, err := ParseHex(tc.value)
		if err != nil || got != tc.want {
			t.Errorf("ParseHex(%q) = %#v, %v; want %#v", tc.value, got, err, tc.want)
		}
	}
}

func TestParseHexRejectsMalformedLiterals(t *testing.T) {
	for _, value := range []string{"", "#12", "#abcd", "+1ffff", "gg0000", "#123456789"} {
		if got, err := ParseHex(value); err == nil {
			t.Errorf("ParseHex(%q) = %#v, want error", value, got)
		}
	}
}
