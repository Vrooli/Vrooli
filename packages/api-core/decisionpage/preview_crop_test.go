package decisionpage

import (
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"testing"
)

// A dark page often ends in a shading, dithered background rather than one
// exact colour; the shot is still cut just below the last content row.
func TestAShotIsCutBelowItsContentEvenOverAShadingBackground(t *testing.T) {
	const w, h, contentEnd = 40, 600, 100
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			shade := uint8(20 + y/60)   // slow vertical gradient
			shade += uint8((x + y) % 3) // dithering noise
			c := color.NRGBA{shade, shade, shade + 4, 255}
			if y < contentEnd && x > 5 && x < 30 && y%10 < 6 {
				c = color.NRGBA{230, 230, 230, 255} // text
			}
			img.Set(x, y, c)
		}
	}
	path := filepath.Join(t.TempDir(), "shot.png")
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	f.Close()
	if err := cropBottom(path); err != nil {
		t.Fatal(err)
	}
	f, err = os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	cfg, err := png.DecodeConfig(f)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Height > contentEnd+40 || cfg.Height < contentEnd {
		t.Fatalf("height %d; want the shot cut just below row %d", cfg.Height, contentEnd)
	}
}
