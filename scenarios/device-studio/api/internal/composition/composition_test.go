package composition

import (
	"context"
	"errors"
	"image"
	"image/color"
	"math"
	"testing"
	"time"
)

func TestValidatePlateQuadUsesCompositionLimits(t *testing.T) {
	valid := rectangularQuad(0, 0, 12, 8)
	tests := []struct {
		name     string
		plateW   int
		plateH   int
		surfaceW int
		surfaceH int
		quad     [4]Point
		wantErr  bool
	}{
		{name: "valid", plateW: 12, plateH: 8, surfaceW: 12, surfaceH: 8, quad: valid},
		{name: "plate-over-budget", plateW: maxDimension + 1, plateH: 8, surfaceW: 12, surfaceH: 8, quad: valid, wantErr: true},
		{name: "surface-over-budget", plateW: 12, plateH: 8, surfaceW: maxDimension + 1, surfaceH: 8, quad: valid, wantErr: true},
		{name: "out-of-bounds", plateW: 12, plateH: 8, surfaceW: 12, surfaceH: 8, quad: rectangularQuad(0, 0, 13, 8), wantErr: true},
		{name: "self-crossing", plateW: 12, plateH: 8, surfaceW: 12, surfaceH: 8, quad: [4]Point{{0, 0}, {12, 8}, {12, 0}, {0, 8}}, wantErr: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidatePlateQuad(test.plateW, test.plateH, test.surfaceW, test.surfaceH, test.quad)
			if (err != nil) != test.wantErr {
				t.Fatalf("ValidatePlateQuad() error = %v, wantErr %v", err, test.wantErr)
			}
		})
	}
}

func TestComposeIdentityRetainsScreenPixels(t *testing.T) {
	plate := solidRGBA(image.Rect(0, 0, 4, 4), color.RGBA{R: 20, G: 30, B: 40, A: 255})
	screen := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			screen.SetNRGBA(x, y, color.NRGBA{R: uint8(10 + 30*x), G: uint8(20 + 20*y), B: 80, A: 255})
		}
	}
	result, err := Compose(context.Background(), Inputs{
		Plate: plate, Screen: screen, SurfaceSize: image.Pt(4, 4),
		Quad: rectangularQuad(0, 0, 4, 4), ActiveMask: solidMask(4, 4, 255), Samples: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			assertRGBAEqual(t, result.Image.At(x, y), screen.At(x, y))
		}
	}
	if result.Receipt.ProtectedChangedPixelCount != 0 || result.Receipt.ProtectedPixelCount != 0 {
		t.Fatalf("unexpected protected pixels in a full-screen identity: %+v", result.Receipt)
	}
}

func TestComposePerspectiveQuadMapsCornersAndGrid(t *testing.T) { // [REQ:DS-PHOTO-CALIBRATION]
	quad := [4]Point{{2, 1}, {18, 3}, {16, 18}, {4, 16}}
	forward, inverse, err := homographyFromRect(4, 4, quad)
	if err != nil {
		t.Fatal(err)
	}
	sourceCorners := [4]Point{{0, 0}, {4, 0}, {4, 4}, {0, 4}}
	for i, src := range sourceCorners {
		x, y, ok := project(forward, src.X, src.Y)
		if !ok || math.Hypot(x-quad[i].X, y-quad[i].Y) > 1e-8 {
			t.Fatalf("corner %d mapped to (%f,%f), want %+v", i, x, y, quad[i])
		}
		u, v, ok := project(inverse, x, y)
		if !ok || math.Hypot(u-src.X, v-src.Y) > 1e-8 {
			t.Fatalf("corner %d inverse mapped to (%f,%f), want %+v", i, u, v, src)
		}
	}

	plate := solidRGBA(image.Rect(0, 0, 20, 20), color.RGBA{A: 255})
	screen := image.NewRGBA(image.Rect(0, 0, 4, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 4; x++ {
			c := color.RGBA{R: 220, A: 255}
			if x >= 2 {
				c = color.RGBA{G: 220, A: 255}
			}
			if y >= 2 {
				c = color.RGBA{B: 220, A: 255}
			}
			if x >= 2 && y >= 2 {
				c = color.RGBA{R: 220, G: 220, B: 220, A: 255}
			}
			screen.SetRGBA(x, y, c)
		}
	}
	result, err := Compose(context.Background(), Inputs{
		Plate: plate, Screen: screen, SurfaceSize: image.Pt(4, 4), Quad: quad,
		ActiveMask: solidMask(20, 20, 255), Samples: 2,
	})
	if err != nil {
		t.Fatal(err)
	}
	colors := map[string]bool{}
	for y := 3; y < 17; y++ {
		for x := 3; x < 17; x++ {
			r, g, b, _ := result.Image.At(x, y).RGBA()
			colors[quadrantName(r, g, b)] = true
		}
	}
	for _, want := range []string{"red", "green", "blue", "white"} {
		if !colors[want] {
			t.Errorf("perspective composition did not preserve %s grid region: %v", want, colors)
		}
	}
}

func TestComposeActiveAndHardwareMasksProtectPlate(t *testing.T) { // [REQ:DS-SCREEN-MASKS]
	plate := image.NewRGBA(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 8; x++ {
			plate.SetRGBA(x, y, color.RGBA{R: 30, G: 60, B: uint8(80 + x), A: 255})
		}
	}
	screen := solidRGBA(image.Rect(0, 0, 8, 4), color.RGBA{G: 230, A: 255})
	active := solidMask(8, 4, 255)
	hardware := image.NewAlpha(image.Rect(0, 0, 8, 4))
	for y := 0; y < 4; y++ {
		for x := 0; x < 2; x++ {
			hardware.SetAlpha(x, y, color.Alpha{A: 255})
		}
	}
	result, err := Compose(context.Background(), Inputs{Plate: plate, Screen: screen, SurfaceSize: image.Pt(8, 4), Quad: rectangularQuad(0, 0, 8, 4), ActiveMask: active, HardwareMask: hardware, Samples: 1})
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 4; y++ {
		for x := 0; x < 2; x++ {
			assertRGBAEqual(t, result.Image.At(x, y), plate.At(x, y))
		}
		for x := 2; x < 8; x++ {
			_, g, _, _ := result.Image.At(x, y).RGBA()
			if g < 220*257 {
				t.Fatalf("active display pixel (%d,%d) was not replaced: green=%d", x, y, g)
			}
		}
	}
	if result.Receipt.ProtectedChangedPixelCount != 0 || result.Receipt.ProtectedPixelCount != 8 {
		t.Fatalf("hardware pixels were not reported protected: %+v", result.Receipt)
	}
}

func TestComposeProtectsLiteralZeroActiveMaskAcrossAntialiasBoundary(t *testing.T) { // [REQ:DS-SCREEN-MASKS]
	plate := image.NewRGBA(image.Rect(0, 0, 8, 3))
	for y := 0; y < 3; y++ {
		for x := 0; x < 8; x++ {
			plate.SetRGBA(x, y, color.RGBA{R: uint8(20 + x*10), G: uint8(40 + y*10), B: 70, A: 255})
		}
	}
	screen := solidRGBA(image.Rect(0, 0, 8, 3), color.RGBA{G: 240, A: 255})
	active := image.NewGray(image.Rect(0, 0, 8, 3))
	for y := 0; y < 3; y++ {
		active.SetGray(1, y, color.Gray{Y: 255})
	}
	result, err := Compose(context.Background(), Inputs{Plate: plate, Screen: screen, SurfaceSize: image.Pt(8, 3), Quad: rectangularQuad(0, 0, 8, 3), ActiveMask: active, Samples: 2})
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 3; y++ {
		for _, x := range []int{0, 2, 3, 7} {
			assertRGBAEqual(t, result.Image.At(x, y), plate.At(x, y))
		}
	}
	if result.Receipt.ProtectedPixelCount != 21 || result.Receipt.ProtectedChangedPixelCount != 0 {
		t.Fatalf("literal zero-mask protection receipt = %+v", result.Receipt)
	}
}

func TestComposeProtectsFullyOccludedHardwarePixelAtAntialiasBoundary(t *testing.T) { // [REQ:DS-SCREEN-MASKS]
	plate := solidRGBA(image.Rect(0, 0, 8, 3), color.RGBA{R: 90, B: 40, A: 255})
	screen := solidRGBA(image.Rect(0, 0, 8, 3), color.RGBA{G: 240, A: 255})
	active := solidMask(8, 3, 255)
	hardware := image.NewGray(image.Rect(0, 0, 8, 3))
	for y := 0; y < 3; y++ {
		hardware.SetGray(1, y, color.Gray{Y: 255})
	}
	result, err := Compose(context.Background(), Inputs{Plate: plate, Screen: screen, SurfaceSize: image.Pt(8, 3), Quad: rectangularQuad(0, 0, 8, 3), ActiveMask: active, HardwareMask: hardware, Samples: 2})
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 3; y++ {
		assertRGBAEqual(t, result.Image.At(1, y), plate.At(1, y))
	}
	if result.Receipt.ProtectedChangedPixelCount != 0 {
		t.Fatalf("hardware pixel changed: %+v", result.Receipt)
	}
}

func TestComposeMixedAlphaEdgeHasNoTransparentColorFringe(t *testing.T) {
	plate := solidRGBA(image.Rect(0, 0, 8, 2), color.RGBA{A: 255})
	screen := image.NewNRGBA(image.Rect(0, 0, 2, 2))
	for y := 0; y < 2; y++ {
		screen.SetNRGBA(0, y, color.NRGBA{R: 255, A: 0}) // hidden red must not leak into interpolation.
		screen.SetNRGBA(1, y, color.NRGBA{B: 255, A: 255})
	}
	result, err := Compose(context.Background(), Inputs{Plate: plate, Screen: screen, SurfaceSize: image.Pt(8, 2), Quad: rectangularQuad(0, 0, 8, 2), ActiveMask: solidMask(8, 2, 255), Samples: 2})
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 2; y++ {
		for x := 0; x < 8; x++ {
			r, _, _, _ := result.Image.At(x, y).RGBA()
			if r != 0 {
				t.Fatalf("transparent red fringe at (%d,%d): red=%d", x, y, r)
			}
		}
	}
}

func TestComposeContainsSourceWithoutStretchOrCrop(t *testing.T) { // [REQ:DS-SCREEN-FIT]
	plate := solidRGBA(image.Rect(0, 0, 4, 4), color.RGBA{R: 10, A: 255})
	screen := solidRGBA(image.Rect(0, 0, 4, 2), color.RGBA{B: 240, A: 255})
	result, err := Compose(context.Background(), Inputs{Plate: plate, Screen: screen, SurfaceSize: image.Pt(4, 4), Quad: rectangularQuad(0, 0, 4, 4), ActiveMask: solidMask(4, 4, 255), Samples: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Receipt.ContainScale != 1 || result.Receipt.ContainOffsetX != 0 || result.Receipt.ContainOffsetY != 1 {
		t.Fatalf("contain fit = scale %v offset (%v,%v)", result.Receipt.ContainScale, result.Receipt.ContainOffsetX, result.Receipt.ContainOffsetY)
	}
	for _, y := range []int{0, 3} {
		for x := 0; x < 4; x++ {
			assertRGBAEqual(t, result.Image.At(x, y), plate.At(x, y))
		}
	}
	_, _, b, _ := result.Image.At(1, 1).RGBA()
	if b < 230*257 {
		t.Fatalf("contained source content missing: blue=%d", b)
	}
}

func TestComposePreservesNonzeroImageBounds(t *testing.T) {
	bounds := image.Rect(11, 17, 13, 19)
	plate := solidRGBA(bounds, color.RGBA{R: 10, A: 255})
	screen := image.NewNRGBA(image.Rect(-4, 9, -2, 11))
	screen.SetNRGBA(-4, 9, color.NRGBA{G: 220, A: 255})
	screen.SetNRGBA(-3, 9, color.NRGBA{B: 220, A: 255})
	screen.SetNRGBA(-4, 10, color.NRGBA{R: 220, A: 255})
	screen.SetNRGBA(-3, 10, color.NRGBA{R: 220, G: 220, A: 255})
	mask := image.NewGray(image.Rect(23, 31, 25, 33))
	for y := 31; y < 33; y++ {
		for x := 23; x < 25; x++ {
			mask.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	result, err := Compose(context.Background(), Inputs{Plate: plate, Screen: screen, SurfaceSize: image.Pt(2, 2), Quad: rectangularQuad(0, 0, 2, 2), ActiveMask: mask, Samples: 1})
	if err != nil {
		t.Fatal(err)
	}
	if result.Image.Bounds() != bounds {
		t.Fatalf("output bounds = %v, want %v", result.Image.Bounds(), bounds)
	}
	assertRGBAEqual(t, result.Image.At(11, 17), screen.At(-4, 9))
}

func TestComposeRejectsMalformedGeometryMasksAndComplexity(t *testing.T) {
	plate := solidRGBA(image.Rect(0, 0, 8, 8), color.RGBA{A: 255})
	screen := solidRGBA(image.Rect(0, 0, 4, 4), color.RGBA{B: 255, A: 255})
	base := Inputs{Plate: plate, Screen: screen, SurfaceSize: image.Pt(4, 4), Quad: rectangularQuad(0, 0, 8, 8), ActiveMask: solidMask(8, 8, 255)}
	cases := []struct {
		name   string
		mutate func(*Inputs)
	}{
		{"bowtie", func(i *Inputs) { i.Quad = [4]Point{{0, 0}, {8, 8}, {8, 0}, {0, 8}} }},
		{"concave", func(i *Inputs) { i.Quad = [4]Point{{0, 0}, {8, 0}, {3, 3}, {0, 8}} }},
		{"degenerate", func(i *Inputs) { i.Quad = [4]Point{{0, 0}, {8, 0}, {8, 0}, {0, 8}} }},
		{"nonfinite", func(i *Inputs) { i.Quad[1].X = math.NaN() }},
		{"out-of-bounds", func(i *Inputs) { i.Quad[1].X = 9 }},
		{"mask-size", func(i *Inputs) { i.ActiveMask = solidMask(7, 8, 255) }},
		{"ambiguous-rgba-mask", func(i *Inputs) {
			rgba := image.NewRGBA(image.Rect(0, 0, 8, 8))
			for y := 0; y < 8; y++ {
				for x := 0; x < 8; x++ {
					rgba.SetRGBA(x, y, color.RGBA{A: 255})
				}
			}
			i.ActiveMask = rgba
		}},
		{"empty-effective-mask", func(i *Inputs) { i.ActiveMask = solidMask(8, 8, 0) }},
		{"samples", func(i *Inputs) { i.Samples = 5 }},
		{"surface-too-large", func(i *Inputs) { i.SurfaceSize = image.Pt(maxDimension+1, 1) }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			in := base
			tc.mutate(&in)
			if _, err := Compose(context.Background(), in); err == nil {
				t.Fatal("expected validation error")
			}
		})
	}
	tooLarge := boundsOnlyImage{image.Rect(0, 0, maxDimension+1, 1)}
	in := base
	in.Plate = tooLarge
	if _, err := Compose(context.Background(), in); err == nil {
		t.Fatal("expected oversized plate to be rejected before allocation")
	}
	maxInt := int(^uint(0) >> 1)
	wrappedBounds := boundsOnlyImage{image.Rect(-maxInt-1, 0, maxInt, 1)}
	in.Plate = wrappedBounds
	if _, err := Compose(context.Background(), in); err == nil {
		t.Fatal("expected overflowing image bounds to be rejected")
	}
}

func TestComposeHonorsCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	in := Inputs{Plate: solidRGBA(image.Rect(0, 0, 4, 4), color.RGBA{A: 255}), Screen: solidRGBA(image.Rect(0, 0, 4, 4), color.RGBA{A: 255}), SurfaceSize: image.Pt(4, 4), Quad: rectangularQuad(0, 0, 4, 4), ActiveMask: solidMask(4, 4, 255)}
	if _, err := Compose(ctx, in); !errors.Is(err, context.Canceled) {
		t.Fatalf("Compose error = %v, want context.Canceled", err)
	}
}

func TestComposeCanCancelDuringRows(t *testing.T) {
	in := Inputs{Plate: solidRGBA(image.Rect(0, 0, 64, 64), color.RGBA{A: 255}), Screen: solidRGBA(image.Rect(0, 0, 64, 64), color.RGBA{G: 255, A: 255}), SurfaceSize: image.Pt(64, 64), Quad: rectangularQuad(0, 0, 64, 64), ActiveMask: solidMask(64, 64, 255), Samples: 1}
	ctx := &cancelAfterChecks{cancelAt: 3}
	if _, err := Compose(ctx, in); !errors.Is(err, context.Canceled) {
		t.Fatalf("Compose error = %v, want cancellation during row processing", err)
	}
}

type cancelAfterChecks struct{ checks, cancelAt int }

func (c *cancelAfterChecks) Deadline() (time.Time, bool) { return time.Time{}, false }
func (c *cancelAfterChecks) Done() <-chan struct{}       { return nil }
func (c *cancelAfterChecks) Err() error {
	c.checks++
	if c.checks >= c.cancelAt {
		return context.Canceled
	}
	return nil
}
func (c *cancelAfterChecks) Value(any) any { return nil }

type boundsOnlyImage struct{ bounds image.Rectangle }

func (i boundsOnlyImage) ColorModel() color.Model { return color.RGBAModel }
func (i boundsOnlyImage) Bounds() image.Rectangle { return i.bounds }
func (i boundsOnlyImage) At(x, y int) color.Color {
	panic("oversized image should be rejected before sampling")
}

func solidRGBA(bounds image.Rectangle, c color.RGBA) *image.RGBA {
	i := image.NewRGBA(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			i.SetRGBA(x, y, c)
		}
	}
	return i
}

func solidMask(width, height int, value uint8) *image.Alpha {
	i := image.NewAlpha(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			i.SetAlpha(x, y, color.Alpha{A: value})
		}
	}
	return i
}

func rectangularQuad(x0, y0, x1, y1 float64) [4]Point {
	return [4]Point{{x0, y0}, {x1, y0}, {x1, y1}, {x0, y1}}
}

func assertRGBAEqual(t *testing.T, got, want color.Color) {
	t.Helper()
	gr, gg, gb, ga := got.RGBA()
	wr, wg, wb, wa := want.RGBA()
	if gr != wr || gg != wg || gb != wb || ga != wa {
		t.Fatalf("RGBA = (%d,%d,%d,%d), want (%d,%d,%d,%d)", gr, gg, gb, ga, wr, wg, wb, wa)
	}
}

func quadrantName(r, g, b uint32) string {
	switch {
	case r > g && r > b:
		return "red"
	case g > r && g > b:
		return "green"
	case b > r && b > g:
		return "blue"
	case r > 30000 && g > 30000 && b > 30000:
		return "white"
	default:
		return "other"
	}
}
