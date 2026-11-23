package ai

import (
	"image"
	"image/color"
	"testing"
)

func flat(w, h int, c color.NRGBA) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	return img
}

func squareMask(w, h, x0, y0, x1, y1 int) *image.NRGBA {
	m := flat(w, h, color.NRGBA{A: 255})
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			m.SetNRGBA(x, y, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
		}
	}
	return m
}

func TestMaskComposite_TakesTheReferenceInsideTheMaskOnly(t *testing.T) {
	src, ref := flat(40, 40, color.NRGBA{R: 200, A: 255}), flat(40, 40, color.NRGBA{B: 200, A: 255})
	out := MaskComposite(src, squareMask(40, 40, 10, 10, 30, 30), ref, MaskCompositeParams{})
	if got := out.NRGBAAt(20, 20); got.B != 200 || got.R != 0 {
		t.Fatalf("inside the mask = %+v, want the reference", got)
	}
	if got := out.NRGBAAt(2, 2); got.R != 200 || got.B != 0 {
		t.Fatalf("outside the mask = %+v, want the source", got)
	}
}

func TestMaskComposite_GrowAndFeatherPutTheSeamOutsideTheRegion(t *testing.T) {
	src, ref := flat(60, 60, color.NRGBA{R: 200, A: 255}), flat(60, 60, color.NRGBA{B: 200, A: 255})
	out := MaskComposite(src, squareMask(60, 60, 20, 20, 40, 40), ref, MaskCompositeParams{GrowPx: 6, FeatherPx: 4})
	if got := out.NRGBAAt(17, 30); got.B <= got.R {
		t.Fatalf("a pixel 3px outside the mask should already be mostly reference after growing, got %+v", got)
	}
	edge := out.NRGBAAt(12, 30)
	if edge.R == 0 || edge.B == 0 {
		t.Fatalf("the seam should be a blend of both images, got %+v", edge)
	}
	if got := out.NRGBAAt(2, 2); got.R != 200 || got.B != 0 {
		t.Fatalf("far from the mask the source must be untouched, got %+v", got)
	}
}

func TestMaskComposite_ResamplesASmallerReference(t *testing.T) {
	src, ref := flat(80, 80, color.NRGBA{R: 200, A: 255}), flat(20, 20, color.NRGBA{G: 200, A: 255})
	out := MaskComposite(src, squareMask(80, 80, 0, 0, 80, 80), ref, MaskCompositeParams{})
	if out.Bounds() != src.Bounds() || out.NRGBAAt(40, 40).G < 190 {
		t.Fatalf("a smaller reference must be scaled to the image, got bounds %v pixel %+v", out.Bounds(), out.NRGBAAt(40, 40))
	}
}
