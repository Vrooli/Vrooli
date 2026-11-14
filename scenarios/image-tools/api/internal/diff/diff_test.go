package diff

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"
)

// solidPNG renders a w×h image filled with c, encoded PNG.
func solidPNG(t *testing.T, w, h int, c color.NRGBA) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

// gradientPNG renders a horizontal gradient so the perceptual hash has signal.
func gradientPNG(t *testing.T, w, h int, shift int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8((x*255/w + shift) % 256)
			img.SetNRGBA(x, y, color.NRGBA{R: v, G: uint8((y * 255 / h)), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestCompare_Identical(t *testing.T) {
	img := gradientPNG(t, 64, 48, 0)
	res, err := Compare(img, img, Params{Mode: ModePixel, IncludeHeatmap: true})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if res.Verdict != "identical" {
		t.Errorf("verdict = %q, want identical", res.Verdict)
	}
	if res.ChangedPixels != 0 {
		t.Errorf("changed pixels = %d, want 0", res.ChangedPixels)
	}
	if res.ChangedFraction != 0 {
		t.Errorf("changed fraction = %v, want 0", res.ChangedFraction)
	}
	if res.PSNR != psnrIdentical {
		t.Errorf("psnr = %v, want %v (identical sentinel)", res.PSNR, psnrIdentical)
	}
	if res.PhashDistance != 0 {
		t.Errorf("phash distance = %d, want 0", res.PhashDistance)
	}
	if res.PhashSimilarity != 1 {
		t.Errorf("phash similarity = %v, want 1", res.PhashSimilarity)
	}
	if res.SSIM != 1 {
		t.Errorf("ssim = %v, want 1", res.SSIM)
	}
	if !res.DimensionsMatch {
		t.Error("dimensions should match")
	}
	if len(res.HeatmapPNG) == 0 {
		t.Error("expected a heat-map")
	}
	if res.TotalPixels != 64*48 {
		t.Errorf("total pixels = %d, want %d", res.TotalPixels, 64*48)
	}
}

func TestCompare_Different(t *testing.T) {
	base := solidPNG(t, 32, 32, color.NRGBA{R: 10, G: 10, B: 10, A: 255})
	other := solidPNG(t, 32, 32, color.NRGBA{R: 240, G: 240, B: 240, A: 255})
	res, err := Compare(base, other, Params{Mode: ModePixel, IncludeHeatmap: true})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if res.Verdict != "different" {
		t.Errorf("verdict = %q, want different", res.Verdict)
	}
	if res.ChangedPixels != res.TotalPixels {
		t.Errorf("changed = %d, want all %d", res.ChangedPixels, res.TotalPixels)
	}
	if res.ChangedFraction != 1 {
		t.Errorf("changed fraction = %v, want 1", res.ChangedFraction)
	}
	if res.MAE <= 0 {
		t.Errorf("mae = %v, want > 0", res.MAE)
	}
	if res.PSNR >= psnrIdentical {
		t.Errorf("psnr = %v, want < %v", res.PSNR, psnrIdentical)
	}
	if res.PhashSimilarity == 1 {
		t.Error("phash similarity should be < 1 for different solids? (solids may collide)")
	}
}

func TestCompare_Tolerance(t *testing.T) {
	base := solidPNG(t, 16, 16, color.NRGBA{R: 100, G: 100, B: 100, A: 255})
	// A small delta below a 0.1 (≈25/255) tolerance should count as unchanged.
	near := solidPNG(t, 16, 16, color.NRGBA{R: 110, G: 110, B: 110, A: 255})
	res, err := Compare(base, near, Params{Mode: ModePixel, Tolerance: 0.1})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if res.ChangedPixels != 0 {
		t.Errorf("changed = %d, want 0 (within tolerance)", res.ChangedPixels)
	}
	if res.Verdict != "identical" {
		t.Errorf("verdict = %q, want identical (within tolerance)", res.Verdict)
	}
	// With zero tolerance the same delta IS a change.
	res2, _ := Compare(base, near, Params{Mode: ModePixel})
	if res2.ChangedPixels != res2.TotalPixels {
		t.Errorf("zero-tolerance changed = %d, want all %d", res2.ChangedPixels, res2.TotalPixels)
	}
}

func TestCompare_DimensionMismatch(t *testing.T) {
	base := gradientPNG(t, 64, 64, 0)
	small := gradientPNG(t, 32, 32, 0)
	res, err := Compare(base, small, Params{Mode: ModePixel})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if res.DimensionsMatch {
		t.Error("dimensions should not match")
	}
	if len(res.Warnings) == 0 {
		t.Error("expected a size-mismatch warning")
	}
	if res.BaseWidth != 64 || res.CompareWidth != 32 {
		t.Errorf("dims base=%d compare=%d, want 64/32", res.BaseWidth, res.CompareWidth)
	}
	if res.TotalPixels != 64*64 {
		t.Errorf("metrics should run at base resolution: total=%d", res.TotalPixels)
	}
}

// richPNG renders an image with strong low-frequency structure (four luminance
// quadrants + a diagonal band) — representative of a real photo and the regime
// where a perceptual hash is meant to operate (it is, by design, unstable on
// near-featureless images).
func richPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			base := 40
			if x > w/2 {
				base += 80
			}
			if y > h/2 {
				base += 60
			}
			if (x+y)%w < w/4 {
				base += 30
			}
			v := uint8(base % 256)
			img.SetNRGBA(x, y, color.NRGBA{R: v, G: v + 40, B: v + 90, A: 255})
		}
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode: %v", err)
	}
	return buf.Bytes()
}

func TestCompare_PerceptualReencode(t *testing.T) {
	base := richPNG(t, 96, 96)
	// Re-encode the SAME picture as JPEG — pixels shift but it's the same image.
	img, err := png.Decode(bytes.NewReader(base))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	var jbuf bytes.Buffer
	if err := jpeg.Encode(&jbuf, img, &jpeg.Options{Quality: 80}); err != nil {
		t.Fatalf("jpeg: %v", err)
	}
	res, err := Compare(base, jbuf.Bytes(), Params{Mode: ModePerceptual})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if res.PhashDistance > 6 {
		t.Errorf("phash distance = %d, want small for a re-encode", res.PhashDistance)
	}
	if res.Verdict == "different" {
		t.Errorf("perceptual verdict = %q, want identical/similar for a re-encode", res.Verdict)
	}
}

func TestCompare_HeatmapOptOut(t *testing.T) {
	img := gradientPNG(t, 16, 16, 0)
	res, err := Compare(img, img, Params{Mode: ModePixel, IncludeHeatmap: false})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if len(res.HeatmapPNG) != 0 {
		t.Error("heat-map should be omitted when IncludeHeatmap is false")
	}
}

func TestCompare_DecodeError(t *testing.T) {
	good := gradientPNG(t, 8, 8, 0)
	if _, err := Compare([]byte("not an image"), good, Params{}); err == nil {
		t.Error("expected a decode error for the base image")
	}
	if _, err := Compare(good, []byte("not an image"), Params{}); err == nil {
		t.Error("expected a decode error for the compare image")
	}
}

func TestModes(t *testing.T) {
	modes := Modes()
	if len(modes) != 2 {
		t.Fatalf("want 2 modes, got %d", len(modes))
	}
	if modes[0].Name != "pixel" || modes[1].Name != "perceptual" {
		t.Errorf("mode names = %q/%q", modes[0].Name, modes[1].Name)
	}
}

func TestParseHex(t *testing.T) {
	cases := []struct {
		in string
		ok bool
		r  uint8
	}{
		{"#ff00c8", true, 255},
		{"00ff00", true, 0},
		{"", false, 0},
		{"#fff", false, 0},
		{"#gggggg", false, 0},
	}
	for _, c := range cases {
		got, ok := parseHex(c.in)
		if ok != c.ok {
			t.Errorf("parseHex(%q) ok = %v, want %v", c.in, ok, c.ok)
			continue
		}
		if ok && got.R != c.r {
			t.Errorf("parseHex(%q).R = %d, want %d", c.in, got.R, c.r)
		}
	}
}

func TestHeatmapHighlight(t *testing.T) {
	base := solidPNG(t, 8, 8, color.NRGBA{R: 0, G: 0, B: 0, A: 255})
	other := solidPNG(t, 8, 8, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	res, err := Compare(base, other, Params{Mode: ModePixel, IncludeHeatmap: true, HighlightHex: "#00ff00"})
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	hm, err := png.Decode(bytes.NewReader(res.HeatmapPNG))
	if err != nil {
		t.Fatalf("decode heatmap: %v", err)
	}
	// Fully-changed pixels should be tinted toward green.
	r, g, b, _ := hm.At(4, 4).RGBA()
	if g>>8 < 200 || r>>8 > 100 || b>>8 > 100 {
		t.Errorf("heat-map pixel = (%d,%d,%d), want green-dominant", r>>8, g>>8, b>>8)
	}
}
