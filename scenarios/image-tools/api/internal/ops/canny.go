package ops

import (
	"image"
	"image/color"

	"github.com/disintegration/imaging"
)

// Canny produces a deterministic edge map — the ControlNet "canny" preprocessor
// (plan Phase 5). It is a pure, model-free image→image transform (grayscale →
// optional Gaussian smoothing → Sobel gradient magnitude → double-threshold with
// 8-neighbour hysteresis), emitting white edges on black so a ControlNet can
// condition a generation on the input's structure. Deterministic for a given
// input + params (unit-tested), unlike the model-backed depth/pose preprocessors.
//
// Params: Amount = Gaussian smoothing sigma (0 = none); LowThreshold / HighThreshold
// are the hysteresis bounds on the 0..255-normalized gradient magnitude (defaults
// 50 / 150). A pixel at/above HighThreshold is a strong edge; one at/above
// LowThreshold becomes an edge only if 8-connected to a strong edge.
func Canny(img image.Image, p *Params) (image.Image, error) {
	gray := imaging.Grayscale(img)
	sigma := p.Amount
	if sigma > 0 {
		gray = imaging.Blur(gray, sigma)
	}
	b := gray.Bounds()
	w, h := b.Dx(), b.Dy()
	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	if w < 3 || h < 3 {
		// Too small for a 3x3 Sobel kernel — return an all-black edge map.
		fillBlack(out)
		return out, nil
	}

	lum := func(x, y int) float64 {
		c := gray.NRGBAAt(x, y)
		return float64(c.R) // grayscale → R==G==B
	}

	low := p.LowThreshold
	if low <= 0 {
		low = 50
	}
	high := p.HighThreshold
	if high <= 0 {
		high = 150
	}
	if high < low {
		high = low
	}

	// Sobel gradient magnitude per interior pixel, normalized to 0..255. A pure
	// step edge maxes one axis at 4*255 (kernel side-weights 1+2+1), so normalize
	// by that and clamp — a vertical/horizontal boundary then reads as a full 255.
	mag := make([]float64, w*h)
	const axisMax = 4 * 255.0
	for y := 1; y < h-1; y++ {
		for x := 1; x < w-1; x++ {
			gx := (lum(x-1, y-1) + 2*lum(x-1, y) + lum(x-1, y+1)) -
				(lum(x+1, y-1) + 2*lum(x+1, y) + lum(x+1, y+1))
			gy := (lum(x-1, y-1) + 2*lum(x, y-1) + lum(x+1, y-1)) -
				(lum(x-1, y+1) + 2*lum(x, y+1) + lum(x+1, y+1))
			m := (abs(gx) + abs(gy)) / axisMax * 255 // L1 magnitude, normalized
			if m > 255 {
				m = 255
			}
			mag[y*w+x] = m
		}
	}

	// Classify: 2 = strong, 1 = weak, 0 = none.
	cls := make([]uint8, w*h)
	for i, m := range mag {
		switch {
		case m >= high:
			cls[i] = 2
		case m >= low:
			cls[i] = 1
		}
	}

	// Hysteresis: a weak pixel becomes an edge only if 8-connected to a strong one.
	isEdge := func(x, y int) bool {
		switch cls[y*w+x] {
		case 2:
			return true
		case 1:
			for dy := -1; dy <= 1; dy++ {
				for dx := -1; dx <= 1; dx++ {
					nx, ny := x+dx, y+dy
					if nx < 0 || ny < 0 || nx >= w || ny >= h {
						continue
					}
					if cls[ny*w+nx] == 2 {
						return true
					}
				}
			}
		}
		return false
	}

	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if isEdge(x, y) {
				out.SetNRGBA(x, y, white)
			} else {
				out.SetNRGBA(x, y, black)
			}
		}
	}
	return out, nil
}

var (
	white = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
	black = color.NRGBA{R: 0, G: 0, B: 0, A: 255}
)

func fillBlack(out *image.NRGBA) {
	b := out.Bounds()
	for y := 0; y < b.Dy(); y++ {
		for x := 0; x < b.Dx(); x++ {
			out.SetNRGBA(x, y, black)
		}
	}
}

func abs(v float64) float64 {
	if v < 0 {
		return -v
	}
	return v
}
