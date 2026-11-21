package ops

import (
	"image/color"
	"testing"
)

func TestParseHexColorKeepsOpaqueDefaultAndAlpha(t *testing.T) {
	if got, err := parseHexColor("  "); err != nil || got != (color.NRGBA{A: 255}) {
		t.Fatalf("empty color = %#v, %v; want opaque black", got, err)
	}
	if got, err := parseHexColor("#12345678"); err != nil || got != (color.NRGBA{R: 0x12, G: 0x34, B: 0x56, A: 0x78}) {
		t.Fatalf("RGBA color = %#v, %v", got, err)
	}
}
