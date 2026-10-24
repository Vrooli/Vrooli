package selection

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
)

// solidImage builds a uniform-color image.
func solidImage(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

// rectOn paints a filled rectangle of fg onto a bg-filled canvas.
func rectOn(w, h int, bg color.NRGBA, rx, ry, rw, rh int, fg color.NRGBA) *image.NRGBA {
	img := solidImage(w, h, bg)
	for y := ry; y < ry+rh; y++ {
		for x := rx; x < rx+rw; x++ {
			img.SetNRGBA(x, y, fg)
		}
	}
	return img
}

func decodeMask(t *testing.T, pngBytes []byte) *image.Gray {
	t.Helper()
	img, err := png.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		t.Fatalf("decode mask png: %v", err)
	}
	b := img.Bounds()
	g := image.NewGray(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, _, _, _ := img.At(x, y).RGBA()
			if r > 0x7fff {
				g.SetGray(x, y, color.Gray{Y: 255})
			}
		}
	}
	return g
}

var (
	red  = color.NRGBA{R: 220, G: 20, B: 20, A: 255}
	blue = color.NRGBA{R: 30, G: 60, B: 220, A: 255}
)

func TestSegmentPointSelectsRegion(t *testing.T) {
	// A 40x40 red square at (30,30) on a 100x100 blue canvas.
	img := rectOn(100, 100, blue, 30, 30, 40, 40, red)
	// Click the center of the square (0.5, 0.5).
	maskPNG, box, area, err := Segment(img, Params{Mode: ModePoint, Points: []Point{{X: 0.5, Y: 0.5}}})
	if err != nil {
		t.Fatalf("Segment: %v", err)
	}
	// The square is 1600/10000 = 0.16 of the image; allow margin.
	if area < 0.10 || area > 0.25 {
		t.Errorf("area fraction = %.3f, want ~0.16", area)
	}
	// Bounding box should cover roughly (0.30,0.30)-(0.70,0.70).
	if box.X < 0.20 || box.X > 0.40 || box.W < 0.30 || box.W > 0.55 {
		t.Errorf("box = %+v, want ~{X:0.30 W:0.40}", box)
	}
	// The mask center pixel must be selected; a far-corner background pixel not.
	g := decodeMask(t, maskPNG)
	if g.GrayAt(50, 50).Y != 255 {
		t.Error("center pixel not selected")
	}
	if g.GrayAt(2, 2).Y != 0 {
		t.Error("background corner pixel selected (region bled into background)")
	}
}

func TestSegmentPointMissingSeedErrors(t *testing.T) {
	img := solidImage(20, 20, blue)
	if _, _, _, err := Segment(img, Params{Mode: ModePoint}); err == nil {
		t.Fatal("expected error for point mode without a seed")
	}
}

func TestSegmentAutoExtractsForeground(t *testing.T) {
	// Centered red subject on a blue border-background.
	img := rectOn(80, 80, blue, 25, 25, 30, 30, red)
	maskPNG, _, area, err := Segment(img, Params{Mode: ModeAuto})
	if err != nil {
		t.Fatalf("Segment auto: %v", err)
	}
	// Foreground ~ 900/6400 = 0.14.
	if area < 0.08 || area > 0.25 {
		t.Errorf("auto area = %.3f, want ~0.14", area)
	}
	g := decodeMask(t, maskPNG)
	if g.GrayAt(40, 40).Y != 255 {
		t.Error("subject center not selected by auto")
	}
	if g.GrayAt(2, 2).Y != 0 {
		t.Error("border background selected by auto")
	}
}

func TestSegmentBoxSelectsForegroundInBox(t *testing.T) {
	img := rectOn(100, 100, blue, 40, 40, 20, 20, red)
	// Box loosely around the square: (0.30,0.30) w/h 0.40.
	box := Box{X: 0.30, Y: 0.30, W: 0.40, H: 0.40}
	maskPNG, _, area, err := Segment(img, Params{Mode: ModeBox, Box: &box})
	if err != nil {
		t.Fatalf("Segment box: %v", err)
	}
	if area <= 0 {
		t.Fatal("box mode selected nothing")
	}
	g := decodeMask(t, maskPNG)
	if g.GrayAt(50, 50).Y != 255 {
		t.Error("box foreground center not selected")
	}
}

func TestSegmentBoxRequiresBox(t *testing.T) {
	img := solidImage(20, 20, blue)
	if _, _, _, err := Segment(img, Params{Mode: ModeBox}); err == nil {
		t.Fatal("expected error for box mode without a box")
	}
}

func TestSegmentDeterministic(t *testing.T) {
	img := rectOn(60, 60, blue, 20, 20, 20, 20, red)
	p := Params{Mode: ModePoint, Points: []Point{{X: 0.5, Y: 0.5}}}
	a, _, _, err := Segment(img, p)
	if err != nil {
		t.Fatal(err)
	}
	b, _, _, err := Segment(img, p)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a, b) {
		t.Error("segmentation is not deterministic")
	}
}

func TestSegmentToleranceWidensRegion(t *testing.T) {
	// A gradient-ish two-tone square: tighter tolerance selects less.
	img := rectOn(100, 100, blue, 30, 30, 40, 40, red)
	// Add a slightly-different-red band so tolerance matters.
	for y := 30; y < 70; y++ {
		for x := 50; x < 70; x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 180, G: 70, B: 60, A: 255})
		}
	}
	_, _, tight, err := Segment(img, Params{Mode: ModePoint, Points: []Point{{X: 0.35, Y: 0.5}}, Tolerance: 0.05})
	if err != nil {
		t.Fatal(err)
	}
	_, _, wide, err := Segment(img, Params{Mode: ModePoint, Points: []Point{{X: 0.35, Y: 0.5}}, Tolerance: 0.5})
	if err != nil {
		t.Fatal(err)
	}
	if wide < tight {
		t.Errorf("higher tolerance should not shrink the region: tight=%.3f wide=%.3f", tight, wide)
	}
}

func TestServiceSegmentClassifiesAndSuggests(t *testing.T) {
	img := rectOn(100, 100, blue, 30, 30, 40, 40, red)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	svc := NewService()
	res, err := svc.Segment(context.Background(), buf.Bytes(), Params{Mode: ModePoint, Points: []Point{{X: 0.5, Y: 0.5}}})
	if err != nil {
		t.Fatalf("service segment: %v", err)
	}
	if len(res.MaskPNG) == 0 {
		t.Error("no mask produced")
	}
	if res.Tier != TierBuiltinCPU {
		t.Errorf("tier = %q, want %q", res.Tier, TierBuiltinCPU)
	}
	if res.RegionClass == "" || len(res.Edits) == 0 {
		t.Errorf("expected a class + contextual edits, got class=%q edits=%d", res.RegionClass, len(res.Edits))
	}
}

func TestServiceModelOverrideFallsBackWithWarning(t *testing.T) {
	img := rectOn(60, 60, blue, 20, 20, 20, 20, red)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	svc := NewService()
	res, err := svc.Segment(context.Background(), buf.Bytes(), Params{
		Mode: ModePoint, Points: []Point{{X: 0.5, Y: 0.5}}, ModelOverride: "mobilesam",
	})
	if err != nil {
		t.Fatalf("service segment: %v", err)
	}
	if res.Tier != TierBuiltinCPU {
		t.Errorf("override should fall back to builtin tier, got %q", res.Tier)
	}
	if len(res.Warnings) == 0 {
		t.Error("expected a warning that the SAM model is not wired")
	}
}
