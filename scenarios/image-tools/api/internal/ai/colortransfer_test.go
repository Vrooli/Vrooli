package ai

import (
	"image"
	"image/color"
	"math"
	"testing"
)

// flat returns a w*h image of one colour with a left-to-right brightness ramp of
// +-ramp so there is shading for the transfer to preserve.
func shaded(w, h int, base color.NRGBA, ramp int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			d := (x*2*ramp)/w - ramp
			img.SetNRGBA(x, y, color.NRGBA{R: uint8(int(base.R) + d), G: uint8(int(base.G) + d), B: uint8(int(base.B) + d), A: 255})
		}
	}
	return img
}

func rect(w, h int, in image.Rectangle) *image.Gray {
	m := image.NewGray(image.Rect(0, 0, w, h))
	for y := in.Min.Y; y < in.Max.Y; y++ {
		for x := in.Min.X; x < in.Max.X; x++ {
			m.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	return m
}

func meanLab(img image.Image, in image.Rectangle) [3]float64 {
	var sum [3]float64
	n := 0
	for y := in.Min.Y; y < in.Max.Y; y++ {
		for x := in.Min.X; x < in.Max.X; x++ {
			c := color.NRGBAModel.Convert(img.At(x, y)).(color.NRGBA)
			l, a, b := srgbToLab(c.R, c.G, c.B)
			sum[0], sum[1], sum[2] = sum[0]+l, sum[1]+a, sum[2]+b
			n++
		}
	}
	return [3]float64{sum[0] / float64(n), sum[1] / float64(n), sum[2] / float64(n)}
}

func deltaE(a, b [3]float64) float64 {
	return math.Sqrt((a[0]-b[0])*(a[0]-b[0]) + (a[1]-b[1])*(a[1]-b[1]) + (a[2]-b[2])*(a[2]-b[2]))
}

func TestColorTransferMovesTheMaskedRegionOntoTheReferenceAndLeavesTheRestAlone(t *testing.T) {
	src := shaded(64, 64, color.NRGBA{R: 200, G: 150, B: 130}, 20)
	ref := shaded(32, 32, color.NRGBA{R: 120, G: 80, B: 60}, 10)
	region := image.Rect(16, 16, 48, 48)
	p := DefaultColorTransferParams()
	p.Luminance, p.FeatherPx = 1, 0

	out, err := ColorTransfer(src, rect(64, 64, region), ref, p)
	if err != nil {
		t.Fatal(err)
	}
	want := meanLab(ref, ref.Bounds())
	before, after := deltaE(meanLab(src, region), want), deltaE(meanLab(out, region), want)
	if after > 2 || after >= before/4 {
		t.Fatalf("region did not reach the reference colour: dE %.1f before, %.1f after", before, after)
	}
	for _, at := range []image.Point{{2, 2}, {60, 60}, {10, 40}} {
		if out.NRGBAAt(at.X, at.Y) != src.NRGBAAt(at.X, at.Y) {
			t.Fatalf("pixel %v outside the mask changed", at)
		}
	}
}

func TestColorTransferKeepsTheShadingInsideTheRegion(t *testing.T) {
	src := shaded(64, 64, color.NRGBA{R: 200, G: 150, B: 130}, 30)
	ref := shaded(32, 32, color.NRGBA{R: 90, G: 70, B: 60}, 5)
	whole := image.Rect(0, 0, 64, 64)
	p := DefaultColorTransferParams()
	p.FeatherPx = 0
	out, err := ColorTransfer(src, rect(64, 64, whole), ref, p)
	if err != nil {
		t.Fatal(err)
	}
	l := func(img *image.NRGBA, x int) float64 {
		c := img.NRGBAAt(x, 32)
		v, _, _ := srgbToLab(c.R, c.G, c.B)
		return v
	}
	if !(l(out, 8) < l(out, 32) && l(out, 32) < l(out, 56)) {
		t.Fatalf("the left-to-right shading ramp was flattened: %.1f %.1f %.1f", l(out, 8), l(out, 32), l(out, 56))
	}
}

func TestColorTransferStrengthZeroIsIdentity(t *testing.T) {
	src := shaded(32, 32, color.NRGBA{R: 200, G: 150, B: 130}, 10)
	ref := shaded(8, 8, color.NRGBA{R: 40, G: 90, B: 160}, 0)
	p := DefaultColorTransferParams()
	p.Strength = 0
	out, err := ColorTransfer(src, rect(32, 32, src.Bounds()), ref, p)
	if err != nil {
		t.Fatal(err)
	}
	for i := range src.Pix {
		if d := int(src.Pix[i]) - int(out.Pix[i]); d < -1 || d > 1 {
			t.Fatalf("strength 0 changed pixel byte %d by %d", i, d)
		}
	}
}

func TestColorTransferReferenceAlphaSelectsItsRegion(t *testing.T) {
	src := shaded(32, 32, color.NRGBA{R: 200, G: 150, B: 130}, 0)
	// Left half of the reference is the colour to take, right half is excluded.
	ref := image.NewNRGBA(image.Rect(0, 0, 20, 10))
	for y := 0; y < 10; y++ {
		for x := 0; x < 20; x++ {
			if x < 10 {
				ref.SetNRGBA(x, y, color.NRGBA{R: 30, G: 30, B: 30, A: 255})
			} else {
				ref.SetNRGBA(x, y, color.NRGBA{R: 250, G: 250, B: 250, A: 0})
			}
		}
	}
	p := DefaultColorTransferParams()
	p.Luminance, p.FeatherPx = 1, 0
	out, err := ColorTransfer(src, rect(32, 32, src.Bounds()), ref, p)
	if err != nil {
		t.Fatal(err)
	}
	if l := meanLab(out, out.Bounds())[0]; l > 20 {
		t.Fatalf("the transparent half of the reference leaked into the statistics: L=%.1f", l)
	}
}

func TestColorTransferRefusesAnEmptyMaskAndAnEmptyReference(t *testing.T) {
	src := shaded(16, 16, color.NRGBA{R: 100, G: 100, B: 100}, 0)
	ref := shaded(8, 8, color.NRGBA{R: 50, G: 60, B: 70}, 0)
	if _, err := ColorTransfer(src, image.NewGray(image.Rect(0, 0, 16, 16)), ref, DefaultColorTransferParams()); err == nil {
		t.Fatal("an empty mask must fail loud")
	}
	clear := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	if _, err := ColorTransfer(src, rect(16, 16, src.Bounds()), clear, DefaultColorTransferParams()); err == nil {
		t.Fatal("a fully transparent reference must fail loud")
	}
}

func TestLabRoundTripIsLossless(t *testing.T) {
	for _, c := range []color.NRGBA{{0, 0, 0, 255}, {255, 255, 255, 255}, {200, 120, 80, 255}, {12, 200, 90, 255}, {128, 128, 128, 255}} {
		l, a, b := srgbToLab(c.R, c.G, c.B)
		r, g, bl := labToSRGB(l, a, b)
		if r != c.R || g != c.G || bl != c.B {
			t.Fatalf("round trip %v -> %d,%d,%d", c, r, g, bl)
		}
	}
}
