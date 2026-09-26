package ops

import (
	"fmt"
	"image"
	"image/color"
	"math"

	"github.com/disintegration/imaging"
)

// Adjust applies tonal/color adjustments in a fixed, predictable order:
// brightness → contrast → gamma → saturation → hue. Each is a no-op at its
// identity value (0 for percent deltas, 0 for gamma meaning "unchanged", 0 for
// hue degrees).
func Adjust(img image.Image, p *Params) (image.Image, error) {
	out := imaging.Clone(img)
	if p.Brightness != 0 {
		out = imaging.AdjustBrightness(out, p.Brightness)
	}
	if p.Contrast != 0 {
		out = imaging.AdjustContrast(out, p.Contrast)
	}
	if p.Gamma != 0 && p.Gamma != 1 {
		out = imaging.AdjustGamma(out, p.Gamma)
	}
	if p.Saturation != 0 {
		out = imaging.AdjustSaturation(out, p.Saturation)
	}
	if p.Hue != 0 {
		out = adjustHue(out, p.Hue)
	}
	return out, nil
}

// Filter applies a single named effect: grayscale, sepia, invert, blur, or
// sharpen. blur/sharpen use p.Amount as the gaussian sigma (default 1.0).
func Filter(img image.Image, p *Params) (image.Image, error) {
	switch p.Filter {
	case "grayscale", "greyscale":
		return imaging.Grayscale(img), nil
	case "invert":
		return imaging.Invert(img), nil
	case "sepia":
		return sepia(img), nil
	case "blur":
		return imaging.Blur(img, sigmaOr(p.Amount, 1.0)), nil
	case "sharpen":
		return imaging.Sharpen(img, sigmaOr(p.Amount, 1.0)), nil
	default:
		return nil, fmt.Errorf("ops: unknown filter %q (want grayscale|sepia|invert|blur|sharpen)", p.Filter)
	}
}

func sigmaOr(v, fallback float64) float64 {
	if v <= 0 {
		return fallback
	}
	return v
}

// sepia applies the classic sepia tone matrix.
func sepia(img image.Image) *image.NRGBA {
	return imaging.AdjustFunc(img, func(c color.NRGBA) color.NRGBA {
		r, g, b := float64(c.R), float64(c.G), float64(c.B)
		nr := clamp8(0.393*r + 0.769*g + 0.189*b)
		ng := clamp8(0.349*r + 0.686*g + 0.168*b)
		nb := clamp8(0.272*r + 0.534*g + 0.131*b)
		return color.NRGBA{R: nr, G: ng, B: nb, A: c.A}
	})
}

// adjustHue rotates the hue of every pixel by deg degrees in HSL space.
func adjustHue(img image.Image, deg float64) *image.NRGBA {
	shift := math.Mod(deg, 360)
	return imaging.AdjustFunc(img, func(c color.NRGBA) color.NRGBA {
		h, s, l := rgbToHSL(c.R, c.G, c.B)
		h = math.Mod(h+shift, 360)
		if h < 0 {
			h += 360
		}
		r, g, b := hslToRGB(h, s, l)
		return color.NRGBA{R: r, G: g, B: b, A: c.A}
	})
}

func clamp8(v float64) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v + 0.5)
}

func rgbToHSL(r8, g8, b8 uint8) (h, s, l float64) {
	r, g, b := float64(r8)/255, float64(g8)/255, float64(b8)/255
	maxc := math.Max(r, math.Max(g, b))
	minc := math.Min(r, math.Min(g, b))
	l = (maxc + minc) / 2
	d := maxc - minc
	if d == 0 {
		return 0, 0, l
	}
	if l > 0.5 {
		s = d / (2 - maxc - minc)
	} else {
		s = d / (maxc + minc)
	}
	switch maxc {
	case r:
		h = math.Mod((g-b)/d, 6)
	case g:
		h = (b-r)/d + 2
	default:
		h = (r-g)/d + 4
	}
	h *= 60
	if h < 0 {
		h += 360
	}
	return h, s, l
}

func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	if s == 0 {
		v := clamp8(l * 255)
		return v, v, v
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	hk := h / 360
	r := hueToRGB(p, q, hk+1.0/3.0)
	g := hueToRGB(p, q, hk)
	b := hueToRGB(p, q, hk-1.0/3.0)
	return clamp8(r * 255), clamp8(g * 255), clamp8(b * 255)
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t++
	}
	if t > 1 {
		t--
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	default:
		return p
	}
}
