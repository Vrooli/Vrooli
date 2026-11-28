package ops

import (
	"image"
	"image/color"
	"testing"
)

// framed builds a 100x80 image with a `band`-pixel frame of frameCol around a
// 60x40 content block of contentCol.
func framed(frameCol, contentCol color.NRGBA, band int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, 100, 80))
	for y := 0; y < 80; y++ {
		for x := 0; x < 100; x++ {
			c := frameCol
			if x >= band && x < 100-band && y >= band && y < 80-band {
				c = contentCol
			}
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func TestScreenshotCleanupTrimsUniformFrame(t *testing.T) {
	img := framed(color.NRGBA{240, 240, 240, 255}, color.NRGBA{10, 120, 200, 255}, 10)
	out, err := ScreenshotCleanup(img, &Params{})
	if err != nil {
		t.Fatalf("ScreenshotCleanup: %v", err)
	}
	if got := out.Bounds(); got.Dx() != 80 || got.Dy() != 60 {
		t.Fatalf("bounds = %v, want 80x60", got)
	}
	if c := color.NRGBAModel.Convert(out.At(0, 0)).(color.NRGBA); c != (color.NRGBA{10, 120, 200, 255}) {
		t.Errorf("top-left after trim = %v, want content colour", c)
	}
}

func TestScreenshotCleanupTrimsAsymmetricChromeOnly(t *testing.T) {
	// A top toolbar of one colour and a left rail of another, with content below.
	img := image.NewNRGBA(image.Rect(0, 0, 60, 60))
	for y := 0; y < 60; y++ {
		for x := 0; x < 60; x++ {
			// Content varies per pixel: real content is not a flat colour, and a
			// flat block is indistinguishable from a band by design.
			c := color.NRGBA{uint8(20 + (x*7+y*13)%40), 120, 60, 255}
			if y < 8 {
				c = color.NRGBA{250, 250, 250, 255}
			} else if x < 6 {
				c = color.NRGBA{30, 30, 30, 255}
			}
			img.SetNRGBA(x, y, c)
		}
	}
	out, err := ScreenshotCleanup(img, &Params{})
	if err != nil {
		t.Fatalf("ScreenshotCleanup: %v", err)
	}
	if got := out.Bounds(); got.Dx() != 54 || got.Dy() != 52 {
		t.Fatalf("bounds = %v, want 54x52", got)
	}
}

func TestScreenshotCleanupKeepsUnframedImage(t *testing.T) {
	// A noisy image has no uniform edge band, so nothing is trimmed.
	img := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			img.SetNRGBA(x, y, color.NRGBA{uint8(x * 6), uint8(y * 6), uint8((x + y) * 3), 255})
		}
	}
	out, err := ScreenshotCleanup(img, &Params{})
	if err != nil {
		t.Fatalf("ScreenshotCleanup: %v", err)
	}
	if out.Bounds() != img.Bounds() {
		t.Fatalf("bounds = %v, want unchanged %v", out.Bounds(), img.Bounds())
	}
}

func TestScreenshotCleanupHonorsSafetyCeiling(t *testing.T) {
	// A frame wider than the default ceiling stops at the ceiling rather than
	// eating the content, and a fully uniform image is returned unchanged.
	img := framed(color.NRGBA{255, 255, 255, 255}, color.NRGBA{0, 0, 0, 255}, 30)
	out, err := ScreenshotCleanup(img, &Params{})
	if err != nil {
		t.Fatalf("ScreenshotCleanup: %v", err)
	}
	if out.Bounds() != img.Bounds() {
		t.Fatalf("bounds = %v, want unchanged when the band hits the ceiling", out.Bounds())
	}
}

func TestScreenshotCleanupPaddingRestoresMargin(t *testing.T) {
	img := framed(color.NRGBA{0, 0, 0, 255}, color.NRGBA{200, 40, 40, 255}, 10)
	out, err := ScreenshotCleanup(img, &Params{PaddingPx: 4})
	if err != nil {
		t.Fatalf("ScreenshotCleanup: %v", err)
	}
	if got := out.Bounds(); got.Dx() != 88 || got.Dy() != 68 {
		t.Fatalf("bounds = %v, want 88x68 (content + 4px each side)", got)
	}
}
