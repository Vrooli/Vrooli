package ops

import (
	"bytes"
	"encoding/json"
	"image"
	"image/color"
	"testing"
)

// solid builds a w×h image filled with c (PNG-encoded source bytes helper).
func solid(w, h int, c color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func encodePNG(t *testing.T, img image.Image) []byte {
	t.Helper()
	data, err := Encode(img, FormatPNG, EncodeOptions{})
	if err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return data
}

func TestRegistryCoversReq01(t *testing.T) {
	want := []string{
		"adjust", "canny", "canvas", "compress", "convert", "crop", "deskew",
		"filter", "flip", "metadata", "overlay", "resize", "rotate", "thumbnail",
	}
	for _, n := range want {
		if !Has(n) {
			t.Errorf("operation %q not registered", n)
		}
	}
	if len(Names()) != len(want) {
		t.Errorf("registry has %d ops, want %d: %v", len(Names()), len(want), Names())
	}
}

func TestExecuteResize(t *testing.T) {
	src := encodePNG(t, solid(100, 50, color.RGBA{R: 200, A: 255}))
	t.Run("fit", func(t *testing.T) {
		res, err := Execute("resize", src, &Params{Width: 50, Fit: "fit"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Width != 50 || res.Height != 25 {
			t.Fatalf("fit resize = %dx%d, want 50x25", res.Width, res.Height)
		}
	})
	t.Run("fill", func(t *testing.T) {
		res, err := Execute("resize", src, &Params{Width: 40, Height: 40, Fit: "fill"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Width != 40 || res.Height != 40 {
			t.Fatalf("fill resize = %dx%d, want 40x40", res.Width, res.Height)
		}
	})
	t.Run("stretch", func(t *testing.T) {
		res, err := Execute("resize", src, &Params{Width: 33, Height: 77, Fit: "stretch"})
		if err != nil {
			t.Fatal(err)
		}
		if res.Width != 33 || res.Height != 77 {
			t.Fatalf("stretch resize = %dx%d, want 33x77", res.Width, res.Height)
		}
	})
	t.Run("missing dims", func(t *testing.T) {
		if _, err := Execute("resize", src, &Params{}); err == nil {
			t.Fatal("expected error for resize with no dimensions")
		}
	})
}

func TestExecuteCrop(t *testing.T) {
	src := encodePNG(t, solid(100, 100, color.RGBA{G: 200, A: 255}))
	res, err := Execute("crop", src, &Params{X: 10, Y: 20, Width: 30, Height: 40})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 30 || res.Height != 40 {
		t.Fatalf("crop = %dx%d, want 30x40", res.Width, res.Height)
	}
	if _, err := Execute("crop", src, &Params{X: 90, Y: 90, Width: 50, Height: 50}); err == nil {
		t.Fatal("expected out-of-bounds crop error")
	}
}

func TestExecuteRotateFlip(t *testing.T) {
	src := encodePNG(t, solid(60, 20, color.RGBA{B: 200, A: 255}))
	res, err := Execute("rotate", src, &Params{Angle: 90})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 20 || res.Height != 60 {
		t.Fatalf("rotate90 = %dx%d, want 20x60", res.Width, res.Height)
	}
	if _, err := Execute("flip", src, &Params{Axis: "horizontal"}); err != nil {
		t.Fatalf("flip h: %v", err)
	}
	if _, err := Execute("flip", src, &Params{Axis: "diagonal"}); err == nil {
		t.Fatal("expected bad-axis error")
	}
}

func TestExecuteConvert(t *testing.T) {
	src := encodePNG(t, solid(20, 20, color.RGBA{R: 10, G: 20, B: 30, A: 255}))
	res, err := Execute("convert", src, &Params{Format: "jpeg", Quality: 80})
	if err != nil {
		t.Fatal(err)
	}
	if res.Format != FormatJPEG || res.Mime != "image/jpeg" {
		t.Fatalf("convert format = %q mime %q", res.Format, res.Mime)
	}
	if _, _, err := decodeFormat(res.Bytes); err != nil {
		t.Fatalf("converted bytes not decodable: %v", err)
	}
	if _, err := Execute("convert", src, &Params{}); err == nil {
		t.Fatal("expected error: convert needs a format")
	}
	if _, err := Execute("convert", src, &Params{Format: "heic"}); err == nil {
		t.Fatal("expected error: heic is not encodable")
	}
}

func TestExecuteCompressTargetSize(t *testing.T) {
	// A noisy image so JPEG size responds to quality.
	img := image.NewRGBA(image.Rect(0, 0, 256, 256))
	for y := 0; y < 256; y++ {
		for x := 0; x < 256; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x ^ y), G: uint8(x * y), B: uint8(x + y), A: 255})
		}
	}
	src := encodePNG(t, img)
	const target = 8000
	res, err := Execute("compress", src, &Params{Format: "jpeg", TargetBytes: target})
	if err != nil {
		t.Fatal(err)
	}
	if int64(len(res.Bytes)) > target {
		t.Fatalf("compress produced %d bytes, want <= %d", len(res.Bytes), target)
	}
	if len(res.Bytes) == 0 {
		t.Fatal("compress produced empty output")
	}
}

func TestExecuteAdjustAndFilter(t *testing.T) {
	src := encodePNG(t, solid(30, 30, color.RGBA{R: 120, G: 120, B: 120, A: 255}))
	if _, err := Execute("adjust", src, &Params{Brightness: 20, Contrast: 10, Saturation: -30, Hue: 45}); err != nil {
		t.Fatalf("adjust: %v", err)
	}
	for _, f := range []string{"grayscale", "sepia", "invert", "blur", "sharpen"} {
		if _, err := Execute("filter", src, &Params{Filter: f, Amount: 2}); err != nil {
			t.Fatalf("filter %s: %v", f, err)
		}
	}
	if _, err := Execute("filter", src, &Params{Filter: "bogus"}); err == nil {
		t.Fatal("expected unknown-filter error")
	}
}

func TestFilterGrayscaleActuallyGray(t *testing.T) {
	src := encodePNG(t, solid(8, 8, color.RGBA{R: 200, G: 50, B: 10, A: 255}))
	res, err := Execute("filter", src, &Params{Filter: "grayscale"})
	if err != nil {
		t.Fatal(err)
	}
	img, _, err := Decode(res.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := img.At(4, 4).RGBA()
	if r != g || g != b {
		t.Fatalf("grayscale pixel not gray: r=%d g=%d b=%d", r>>8, g>>8, b>>8)
	}
}

func TestExecuteThumbnailCanvas(t *testing.T) {
	src := encodePNG(t, solid(200, 100, color.RGBA{R: 5, G: 5, B: 5, A: 255}))
	res, err := Execute("thumbnail", src, &Params{Width: 64, Height: 64})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 64 || res.Height != 64 {
		t.Fatalf("thumbnail = %dx%d, want 64x64", res.Width, res.Height)
	}
	res, err = Execute("canvas", src, &Params{Width: 300, Height: 300, Background: "#ffffff", Gravity: "center"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 300 || res.Height != 300 {
		t.Fatalf("canvas = %dx%d, want 300x300", res.Width, res.Height)
	}
}

func TestExecuteOverlayText(t *testing.T) {
	src := encodePNG(t, solid(200, 80, color.RGBA{R: 0, G: 0, B: 0, A: 255}))
	res, err := Execute("overlay", src, &Params{Text: "© Vrooli", Position: "bottom-right", Color: "#ffffff", FontSize: 18})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 200 || res.Height != 80 {
		t.Fatalf("overlay changed size to %dx%d", res.Width, res.Height)
	}
	if _, err := Execute("overlay", src, &Params{}); err == nil {
		t.Fatal("expected error: overlay needs image or text")
	}
}

func TestExecuteOverlayImage(t *testing.T) {
	base := encodePNG(t, solid(100, 100, color.RGBA{A: 255}))
	wm := encodePNG(t, solid(20, 20, color.RGBA{R: 255, G: 255, B: 255, A: 255}))
	res, err := Execute("overlay", base, &Params{OverlayImage: wm, Position: "top-left", Opacity: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if res.Width != 100 || res.Height != 100 {
		t.Fatalf("image overlay changed base size to %dx%d", res.Width, res.Height)
	}
}

func TestMetadataReadStrip(t *testing.T) {
	src := encodePNG(t, solid(40, 30, color.RGBA{R: 1, A: 255}))
	// Read: a PNG with no EXIF still returns a well-formed report.
	res, err := Execute("metadata", src, &Params{})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mime != "application/json" {
		t.Fatalf("metadata read mime = %q, want application/json", res.Mime)
	}
	var report MetaReport
	if err := json.Unmarshal(res.Bytes, &report); err != nil {
		t.Fatalf("report not valid json: %v", err)
	}
	if report.Width != 40 || report.Height != 30 {
		t.Fatalf("report dims = %dx%d, want 40x30", report.Width, report.Height)
	}
	// Strip: returns an image result.
	res, err = Execute("metadata", src, &Params{StripGPS: true})
	if err != nil {
		t.Fatal(err)
	}
	if res.Mime != "image/png" {
		t.Fatalf("strip mime = %q, want image/png", res.Mime)
	}
}

func TestExecuteUnknownOp(t *testing.T) {
	if _, err := Execute("nope", []byte("x"), nil); err == nil {
		t.Fatal("expected unknown-op error")
	}
}

func TestExecuteBadImage(t *testing.T) {
	if _, err := Execute("resize", []byte("not an image"), &Params{Width: 10}); err == nil {
		t.Fatal("expected decode error for non-image bytes")
	}
}

// decodeFormat is a tiny helper around image.Decode for assertions.
func decodeFormat(data []byte) (image.Image, string, error) {
	img, meta, err := Decode(data)
	return img, meta.Format, err
}

// TestCannyDeterministicEdges asserts the canny preprocessor is deterministic and
// produces a black/white edge map that fires on a high-contrast boundary.
func TestCannyDeterministicEdges(t *testing.T) {
	// A 40x40 image split black|white down the middle → a strong vertical edge.
	img := image.NewNRGBA(image.Rect(0, 0, 40, 40))
	for y := 0; y < 40; y++ {
		for x := 0; x < 40; x++ {
			if x < 20 {
				img.SetNRGBA(x, y, color.NRGBA{A: 255})
			} else {
				img.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
			}
		}
	}
	out1, err := Canny(img, &Params{})
	if err != nil {
		t.Fatalf("canny: %v", err)
	}
	out2, _ := Canny(img, &Params{})

	n1, ok := out1.(*image.NRGBA)
	if !ok {
		t.Fatalf("canny output is %T, want *image.NRGBA", out1)
	}
	n2 := out2.(*image.NRGBA)
	if !bytes.Equal(n1.Pix, n2.Pix) {
		t.Fatal("canny is not deterministic")
	}

	// The boundary column region must contain white edge pixels; a flat corner must not.
	edges := 0
	for y := 0; y < 40; y++ {
		c := n1.NRGBAAt(19, y)
		if c.R > 200 {
			edges++
		}
	}
	if edges == 0 {
		t.Fatal("canny found no edge at the black/white boundary")
	}
	if c := n1.NRGBAAt(2, 2); c.R > 200 {
		t.Fatal("canny fired on a flat region")
	}
}
