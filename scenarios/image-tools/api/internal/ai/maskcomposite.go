package ai

import (
	"image"
	"image/color"

	xdraw "golang.org/x/image/draw"
)

// MaskCompositeParams tunes how a reference is laid into an image through a mask.
type MaskCompositeParams struct {
	// GrowPx widens the selected region, so the seam falls outside the thing that
	// changed (a new hairstyle's edge) rather than on it.
	GrowPx float64
	// FeatherPx softens the seam so the reference fades into the image.
	FeatherPx float64
}

// DefaultMaskCompositeParams grows the region a few pixels and feathers it about
// as much, which hides the seam of a hairstyle change against a photo background.
func DefaultMaskCompositeParams() MaskCompositeParams {
	return MaskCompositeParams{GrowPx: 6, FeatherPx: 5}
}

// MaskComposite returns src with the pixels selected by mask replaced by the same
// place in reference. The reference is resampled to the size of src when they
// differ (an editor that returns a smaller render of the same photo), and the
// mask is read at src's geometry. Everything outside the grown, feathered mask is
// src unchanged.
func MaskComposite(src, mask, reference image.Image, p MaskCompositeParams) *image.NRGBA {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	ref := image.NewNRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(ref, ref.Bounds(), reference, reference.Bounds(), xdraw.Src, nil)

	weight := sampledWeights(mask, w, h)
	if p.GrowPx >= 1 {
		weight = growWeights(weight, w, h, int(p.GrowPx+0.5))
	}
	if p.FeatherPx > 0 {
		weight = blurWeights(weight, w, h, p.FeatherPx)
	}
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			orig := color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			t := weight[y*w+x]
			if t <= 0 {
				out.SetNRGBA(x, y, orig)
				continue
			}
			r := ref.NRGBAAt(x, y)
			mix := func(a, c uint8) uint8 { return uint8(float64(a)*(1-t) + float64(c)*t + 0.5) }
			out.SetNRGBA(x, y, color.NRGBA{R: mix(orig.R, r.R), G: mix(orig.G, r.G), B: mix(orig.B, r.B), A: orig.A})
		}
	}
	return out
}

// growWeights dilates a 0/1 weight map by radius pixels (a square structuring
// element, done as two separable max passes).
func growWeights(in []float64, w, h, radius int) []float64 {
	pass := func(src []float64, horizontal bool) []float64 {
		dst := make([]float64, len(src))
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				var best float64
				for k := -radius; k <= radius; k++ {
					xx, yy := x, y
					if horizontal {
						xx = min(max(x+k, 0), w-1)
					} else {
						yy = min(max(y+k, 0), h-1)
					}
					if v := src[yy*w+xx]; v > best {
						best = v
					}
				}
				dst[y*w+x] = best
			}
		}
		return dst
	}
	return pass(pass(in, true), false)
}
