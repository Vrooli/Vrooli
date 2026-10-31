package ai

import (
	"image"
	"image/color"
	"testing"
)

func solidImage(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func discMask(w, h, cx, cy, r int) *image.NRGBA {
	m := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r {
				m.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}
	return m
}

func TestMaskFillFlatRestoresExactly(t *testing.T) {
	want := color.NRGBA{100, 150, 200, 255}
	src := solidImage(64, 64, want)
	mask := discMask(64, 64, 32, 32, 12)
	out := MaskFill(src, mask, DefaultMaskFillParams())
	got := out.NRGBAAt(32, 32)
	if diff := maxDiff(got, want); diff > 1 {
		t.Fatalf("flat fill center = %v, want %v (diff %d)", got, want, diff)
	}
}

func TestMaskFillGradientRestoresWithinTwo(t *testing.T) {
	src := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			src.SetNRGBA(x, y, color.NRGBA{uint8(x * 4), 128, 64, 255})
		}
	}
	mask := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 20; y < 44; y++ {
		for x := 20; x < 44; x++ {
			mask.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}
	out := MaskFill(src, mask, DefaultMaskFillParams())
	for y := 24; y < 40; y++ {
		for x := 24; x < 40; x++ {
			want := src.NRGBAAt(x, y)
			got := out.NRGBAAt(x, y)
			if diff := maxDiff(got, want); diff > 2 {
				t.Fatalf("gradient fill (%d,%d) = %v, want %v (diff %d)", x, y, got, want, diff)
			}
		}
	}
}

func TestMaskFillAbsorbsGlowHalo(t *testing.T) {
	bg := color.NRGBA{50, 50, 50, 255}
	src := solidImage(64, 64, bg)
	// A cyan blob whose edge extends past the (undersized) mask.
	for y := 18; y < 30; y++ {
		for x := 18; x < 30; x++ {
			src.SetNRGBA(x, y, color.NRGBA{34, 211, 238, 255})
		}
	}
	mask := image.NewNRGBA(image.Rect(0, 0, 64, 64))
	for y := 20; y < 26; y++ {
		for x := 20; x < 26; x++ {
			mask.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
		}
	}
	out := MaskFill(src, mask, MaskFillParams{GrowPx: 2, HaloThreshold: 25, MaxIterations: 4000})
	// A pixel in the halo band (inside the grown mask, was cyan) must come back
	// to the flat background.
	got := out.NRGBAAt(27, 22)
	if diff := maxDiff(got, bg); diff > 3 {
		t.Fatalf("halo pixel = %v, want ~%v (diff %d)", got, bg, diff)
	}
}

func maxDiff(a, b color.NRGBA) int {
	d := absDiff(int(a.R), int(b.R))
	if v := absDiff(int(a.G), int(b.G)); v > d {
		d = v
	}
	if v := absDiff(int(a.B), int(b.B)); v > d {
		d = v
	}
	return d
}

func absDiff(a, b int) int {
	if a > b {
		return a - b
	}
	return b - a
}
