package rendering

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestPhysicalRGBA8PNGRequiresExactIHDRFormat(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.NRGBA{R: 5, G: 8, B: 13, A: 255})
	var output bytes.Buffer
	if err := png.Encode(&output, img); err != nil {
		t.Fatal(err)
	}
	valid := output.Bytes()
	if !isPhysicalRGBA8PNG(valid) {
		t.Fatal("Go's valid 8-bit RGBA PNG was rejected")
	}
	for _, tc := range []struct {
		name  string
		index int
		value byte
	}{{"bit depth", 24, 16}, {"color model", 25, 2}, {"interlaced", 28, 1}} {
		t.Run(tc.name, func(t *testing.T) {
			bad := bytes.Clone(valid)
			bad[tc.index] = tc.value
			if isPhysicalRGBA8PNG(bad) {
				t.Fatalf("IHDR %s mismatch was accepted", tc.name)
			}
		})
	}
}
