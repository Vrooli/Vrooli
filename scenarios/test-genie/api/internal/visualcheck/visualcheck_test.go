package visualcheck

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// solidPNG renders a w×h image filled with one color.
func solidPNG(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return encode(t, img)
}

// gradientPNG renders a horizontal black→white gradient: lots of luminance
// variance, a healthy non-blank render.
func gradientPNG(t *testing.T, w, h int) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			v := uint8(x * 255 / w)
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	return encode(t, img)
}

// gradientWithBlock renders the gradient with one quadrant overpainted a flat
// color, simulating a localized UI change of a known magnitude.
func gradientWithBlock(t *testing.T, w, h int, c color.Color) []byte {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if x >= w/2 && y >= h/2 {
				img.Set(x, y, c)
				continue
			}
			v := uint8(x * 255 / w)
			img.Set(x, y, color.RGBA{v, v, v, 255})
		}
	}
	return encode(t, img)
}

func encode(t *testing.T, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("encode png: %v", err)
	}
	return buf.Bytes()
}

func TestRenderHealthSolidIsBroken(t *testing.T) {
	th := DefaultThresholds()
	for _, c := range []color.RGBA{
		{255, 255, 255, 255}, // white
		{0, 0, 0, 255},       // black
		{40, 120, 200, 255},  // a solid mid color
	} {
		got, err := RenderHealth(solidPNG(t, 200, 150, c), th)
		if err != nil {
			t.Fatalf("RenderHealth(%v): %v", c, err)
		}
		if !got.Broken {
			t.Errorf("solid %v: Broken=false, want true (variance=%.6f dominant=%.3f)", c, got.Variance, got.DominantFraction)
		}
	}
}

func TestRenderHealthGradientIsHealthy(t *testing.T) {
	got, err := RenderHealth(gradientPNG(t, 320, 240), DefaultThresholds())
	if err != nil {
		t.Fatalf("RenderHealth: %v", err)
	}
	if got.Broken {
		t.Errorf("gradient: Broken=true, want false (variance=%.6f dominant=%.3f reason=%q)", got.Variance, got.DominantFraction, got.Reason)
	}
}

func TestRenderHealthNearSolidIsBroken(t *testing.T) {
	// A white page with a single thin off-white line: still effectively blank.
	img := image.NewRGBA(image.Rect(0, 0, 300, 300))
	for y := 0; y < 300; y++ {
		for x := 0; x < 300; x++ {
			img.Set(x, y, color.RGBA{255, 255, 255, 255})
		}
	}
	for x := 0; x < 300; x++ {
		img.Set(x, 150, color.RGBA{250, 250, 250, 255})
	}
	got, err := RenderHealth(encode(t, img), DefaultThresholds())
	if err != nil {
		t.Fatalf("RenderHealth: %v", err)
	}
	if !got.Broken {
		t.Errorf("near-solid: Broken=false, want true (dominant=%.3f)", got.DominantFraction)
	}
}

func TestCompareIdenticalBytes(t *testing.T) {
	g := gradientPNG(t, 320, 240)
	got, err := Compare(g, g, DefaultThresholds())
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if !got.Identical || got.ChangedFraction != 0 {
		t.Errorf("identical bytes: %+v, want Identical=true ChangedFraction=0", got)
	}
}

func TestCompareIdenticalContentDifferentSize(t *testing.T) {
	// Same gradient content rendered at two sizes must downscale to the same
	// grid and read as identical.
	got, err := Compare(gradientPNG(t, 320, 240), gradientPNG(t, 640, 480), DefaultThresholds())
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if !got.Identical {
		t.Errorf("same content different size: ChangedFraction=%.4f, want identical", got.ChangedFraction)
	}
}

func TestCompareChangedQuadrant(t *testing.T) {
	base := gradientPNG(t, 320, 240)
	cur := gradientWithBlock(t, 320, 240, color.RGBA{255, 0, 0, 255})
	got, err := Compare(base, cur, DefaultThresholds())
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if got.Identical {
		t.Fatalf("changed quadrant read as identical (ChangedFraction=%.4f)", got.ChangedFraction)
	}
	// One quadrant changed: expect roughly a quarter of the grid to move, but
	// only cells whose luminance actually shifted past PixelDelta count, so keep
	// the band generous.
	if got.ChangedFraction < 0.05 || got.ChangedFraction > 0.30 {
		t.Errorf("ChangedFraction=%.4f, want within [0.05,0.30] for a single changed quadrant", got.ChangedFraction)
	}
}

func TestCompareSubToleranceNoise(t *testing.T) {
	base := gradientPNG(t, 320, 240)
	// Shift a single pixel by a hair — below PixelDelta, below ChangedTolerance.
	img, _, err := image.Decode(bytes.NewReader(base))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	rgba := image.NewRGBA(img.Bounds())
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			rgba.Set(x, y, img.At(x, y))
		}
	}
	rgba.Set(0, 0, color.RGBA{1, 1, 1, 255})
	got, err := Compare(base, encode(t, rgba), DefaultThresholds())
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if !got.Identical {
		t.Errorf("sub-tolerance noise: ChangedFraction=%.4f, want identical", got.ChangedFraction)
	}
}

func TestCompareLeverTighteningFlipsVerdict(t *testing.T) {
	base := gradientPNG(t, 320, 240)
	cur := gradientWithBlock(t, 320, 240, color.RGBA{255, 0, 0, 255})

	loose := DefaultThresholds()
	loose.ChangedTolerance = 0.95 // tolerate almost anything → identical
	got, err := Compare(base, cur, loose)
	if err != nil {
		t.Fatalf("Compare loose: %v", err)
	}
	if !got.Identical {
		t.Errorf("loose tolerance: want identical, got ChangedFraction=%.4f", got.ChangedFraction)
	}

	strict := DefaultThresholds()
	strict.ChangedTolerance = 0.0 // any change → not identical
	got, err = Compare(base, cur, strict)
	if err != nil {
		t.Fatalf("Compare strict: %v", err)
	}
	if got.Identical {
		t.Error("strict tolerance: want not identical")
	}
}

func TestRenderHealthRejectsGarbage(t *testing.T) {
	if _, err := RenderHealth([]byte("not a png"), DefaultThresholds()); err == nil {
		t.Error("expected decode error for garbage bytes")
	}
}

func TestThresholdsFromEnvOverridesAndGuards(t *testing.T) {
	t.Setenv("TEST_GENIE_VISUAL_GRID_SIZE", "16")
	t.Setenv("TEST_GENIE_VISUAL_CHANGED_TOLERANCE", "0.25")
	t.Setenv("TEST_GENIE_VISUAL_BLANK_FRACTION", "bogus") // ignored, keeps default
	t.Setenv("TEST_GENIE_VISUAL_PIXEL_DELTA", "5")        // out of range, keeps default

	got := ThresholdsFromEnv()
	def := DefaultThresholds()
	if got.GridSize != 16 {
		t.Errorf("GridSize = %d, want 16", got.GridSize)
	}
	if got.ChangedTolerance != 0.25 {
		t.Errorf("ChangedTolerance = %.3f, want 0.25", got.ChangedTolerance)
	}
	if got.BlankFraction != def.BlankFraction {
		t.Errorf("BlankFraction = %.3f, want default %.3f (bad value must be ignored)", got.BlankFraction, def.BlankFraction)
	}
	if got.PixelDelta != def.PixelDelta {
		t.Errorf("PixelDelta = %.3f, want default %.3f (out-of-range must be ignored)", got.PixelDelta, def.PixelDelta)
	}
}
