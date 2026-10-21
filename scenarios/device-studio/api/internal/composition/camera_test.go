package composition

import (
	"context"
	"image"
	"image/color"
	"testing"
)

func TestProjectRasterRectUsesCompositionContainAndHomography(t *testing.T) {
	quad := [4]Point{{10, 20}, {190, 10}, {180, 180}, {20, 170}}
	got, err := ProjectRasterRect(image.Pt(200, 100), image.Pt(200, 180), quad, image.Rect(13, 17, 213, 117), image.Rect(13, 17, 213, 117))
	if err != nil {
		t.Fatal(err)
	}
	for i := range quad {
		if !nearPoint(got[i], quad[i]) {
			t.Fatalf("full raster projection corner %d = %+v, want %+v", i, got[i], quad[i])
		}
	}
	// A 200x100 raster is contained into a square with 25px top and bottom
	// bars. The centered 100x100 source rectangle maps to x=[25,75], y=[25,75].
	inner, err := ProjectRasterRect(image.Pt(100, 100), image.Pt(100, 100), [4]Point{{0, 0}, {100, 0}, {100, 100}, {0, 100}}, image.Rect(13, 17, 213, 117), image.Rect(63, 17, 163, 117))
	if err != nil {
		t.Fatal(err)
	}
	want := [4]Point{{25, 25}, {75, 25}, {75, 75}, {25, 75}}
	for i := range want {
		if !nearPoint(inner[i], want[i]) {
			t.Fatalf("contain projection corner %d = %+v, want %+v", i, inner[i], want[i])
		}
	}
}

func TestInterpolateQuadPreservesCornersAndRejectsFold(t *testing.T) {
	from := [4]Point{{0, 0}, {20, 0}, {20, 20}, {0, 20}}
	to := [4]Point{{5, 5}, {15, 5}, {15, 15}, {5, 15}}
	got, err := InterpolateQuad(from, to, 0.5)
	if err != nil {
		t.Fatal(err)
	}
	want := [4]Point{{2.5, 2.5}, {17.5, 2.5}, {17.5, 17.5}, {2.5, 17.5}}
	if got != want {
		t.Fatalf("interpolated quad = %+v, want %+v", got, want)
	}
	fold := [4]Point{{0, 0}, {5, 5}, {20, 0}, {0, 20}}
	if _, err := InterpolateQuad(from, fold, 1); err == nil {
		t.Fatal("folded endpoint was accepted")
	}
	if _, err := InterpolateQuad(from, to, 1.1); err == nil {
		t.Fatal("out-of-range progress was accepted")
	}
}

func TestRenderScreenEntryFrameZeroAndDirectMatteEndpoint(t *testing.T) {
	scene := image.NewRGBA64(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			scene.SetRGBA64(x, y, color.RGBA64{R: 50000, G: 1000, B: 2000, A: 65535})
		}
	}
	direct := image.NewNRGBA(image.Rect(7, 11, 27, 51))
	for y := direct.Bounds().Min.Y; y < direct.Bounds().Max.Y; y++ {
		for x := direct.Bounds().Min.X; x < direct.Bounds().Max.X; x++ {
			direct.SetNRGBA(x, y, color.NRGBA{R: 5, G: 20, B: 240, A: 255})
		}
	}
	from := [4]Point{{16, 12}, {44, 12}, {44, 52}, {16, 52}}
	to := [4]Point{{22, 12}, {42, 12}, {42, 52}, {22, 52}}
	matte := color.NRGBA{R: 17, G: 191, B: 73, A: 255}
	zero, err := RenderScreenEntry(context.Background(), scene, from, from, direct, matte, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if !samePixel(zero.At(2, 2), scene.At(2, 2)) || !samePixel(zero.At(30, 30), scene.At(30, 30)) {
		t.Fatal("frame zero changed scene pixels")
	}
	end, err := RenderScreenEntry(context.Background(), scene, from, to, direct, matte, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	if got := end.RGBA64At(30, 30); got.R != 5*257 || got.G != 20*257 || got.B != 240*257 || got.A != 65535 {
		t.Fatalf("endpoint active app pixel = %+v", got)
	}
	if got := end.RGBA64At(2, 2); got.R != 17*257 || got.G != 191*257 || got.B != 73*257 || got.A != 65535 {
		t.Fatalf("endpoint outside matte = %+v", got)
	}
	cancelled, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RenderScreenEntry(cancelled, scene, from, to, direct, matte, 1, 1); err == nil {
		t.Fatal("cancelled camera frame rendered")
	}
}

func TestRenderScreenEntryUsesMatteWhenInverseWarpLeavesPlate(t *testing.T) {
	scene := image.NewRGBA64(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			scene.SetRGBA64(x, y, color.RGBA64{R: 50000, A: 65535})
		}
	}
	app := image.NewNRGBA(image.Rect(0, 0, 24, 24))
	from := [4]Point{{2, 20}, {22, 20}, {22, 44}, {2, 44}}
	to := [4]Point{{20, 20}, {44, 20}, {44, 44}, {20, 44}}
	matte := color.NRGBA{R: 11, G: 22, B: 33, A: 255}
	got, err := RenderScreenEntry(context.Background(), scene, from, to, app, matte, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	pixel := got.RGBA64At(0, 32)
	if pixel.R != 11*257 || pixel.G != 22*257 || pixel.B != 33*257 || pixel.A != 65535 {
		t.Fatalf("out-of-plate inverse sample did not use pinned matte: %+v", pixel)
	}
}

func TestRenderScreenEntryEndpointMatchesFullWarpEndpoint(t *testing.T) {
	scene := image.NewRGBA64(image.Rect(0, 0, 48, 32))
	for y := 0; y < 32; y++ {
		for x := 0; x < 48; x++ {
			scene.SetRGBA64(x, y, color.RGBA64{R: uint16(x * 900), G: uint16(y * 1700), B: 19000, A: 65535})
		}
	}
	app := image.NewNRGBA(image.Rect(5, 9, 25, 29))
	for y := app.Bounds().Min.Y; y < app.Bounds().Max.Y; y++ {
		for x := app.Bounds().Min.X; x < app.Bounds().Max.X; x++ {
			app.SetNRGBA(x, y, color.NRGBA{R: uint8(x * 9), G: uint8(y * 7), B: 211, A: 255})
		}
	}
	from := [4]Point{{5, 3}, {35, 3}, {35, 29}, {5, 29}}
	target := [4]Point{{16, 0}, {32, 0}, {32, 32}, {16, 32}}
	matte := color.NRGBA{R: 17, G: 29, B: 43, A: 255}
	full, err := RenderScreenEntry(context.Background(), scene, from, target, app, matte, 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	direct, err := RenderScreenEntryEndpoint(context.Background(), image.Pt(48, 32), app, target, matte)
	if err != nil {
		t.Fatal(err)
	}
	for y := 0; y < 32; y++ {
		for x := 0; x < 48; x++ {
			if got, want := direct.RGBA64At(x, y), full.RGBA64At(x, y); got != want {
				t.Fatalf("endpoint pixel (%d,%d) = %+v, full path gives %+v", x, y, got, want)
			}
		}
	}
}

func samePixel(a, b color.Color) bool {
	ar, ag, ab, aa := a.RGBA()
	br, bg, bb, ba := b.RGBA()
	return ar == br && ag == bg && ab == bb && aa == ba
}

func nearPoint(a, b Point) bool {
	return abs(a.X-b.X) < 1e-9 && abs(a.Y-b.Y) < 1e-9
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
