// Package treatments contains deterministic, model-free image treatments.
//
// The package deliberately has no dependency on the HTTP, storage, or job
// layers. Every function is a pure image transform whose output is determined
// solely by the input pixels and the supplied parameters. This makes the
// treatments safe to use from the image-tools operation registry and from
// higher-level scenarios such as backdrop-studio.
package treatments

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"
	"strconv"
	"strings"
)

// Params is the transport-independent parameter set for treatment operations.
type Params struct {
	Dark, Light, Mid string
	MidLow, MidHigh  float64
	Levels           int
	LPI              int
	Angle            float64
	Dot              string
	Seed             int64
	Amount           float64
	Contrast         float64
	ScrimColor       string
	Direction        string
	Opacity          float64
}

func parseColor(value string, fallback color.NRGBA) (color.NRGBA, error) {
	s := strings.TrimPrefix(strings.TrimSpace(value), "#")
	if s == "" {
		return fallback, nil
	}
	if len(s) == 3 {
		s = string([]byte{s[0], s[0], s[1], s[1], s[2], s[2]})
	}
	if len(s) != 6 && len(s) != 8 {
		return color.NRGBA{}, fmt.Errorf("treatments: invalid color %q", value)
	}
	read := func(part string) (uint8, error) {
		v, err := strconv.ParseUint(part, 16, 8)
		return uint8(v), err
	}
	r, err := read(s[0:2])
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("treatments: invalid color %q", value)
	}
	g, err := read(s[2:4])
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("treatments: invalid color %q", value)
	}
	b, err := read(s[4:6])
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("treatments: invalid color %q", value)
	}
	a := uint8(255)
	if len(s) == 8 {
		a, err = read(s[6:8])
		if err != nil {
			return color.NRGBA{}, fmt.Errorf("treatments: invalid color %q", value)
		}
	}
	return color.NRGBA{R: r, G: g, B: b, A: a}, nil
}

func clone(src image.Image) *image.NRGBA {
	b := src.Bounds()
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
	draw.Draw(out, out.Bounds(), src, b.Min, draw.Src)
	return out
}

func srgbLinear(v uint8) float64 {
	x := float64(v) / 255
	if x <= 0.04045 {
		return x / 12.92
	}
	return math.Pow((x+0.055)/1.055, 2.4)
}

func luminance(c color.NRGBA) float64 {
	return 0.2126*srgbLinear(c.R) + 0.7152*srgbLinear(c.G) + 0.0722*srgbLinear(c.B)
}

func mix(a, b color.NRGBA, t float64) color.NRGBA {
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return color.NRGBA{R: uint8(float64(a.R) + (float64(b.R)-float64(a.R))*t + 0.5), G: uint8(float64(a.G) + (float64(b.G)-float64(a.G))*t + 0.5), B: uint8(float64(a.B) + (float64(b.B)-float64(a.B))*t + 0.5), A: uint8(float64(a.A) + (float64(b.A)-float64(a.A))*t + 0.5)}
}

// Duotone maps linear-light luminance onto a two-ink ramp. When Mid is set,
// the third ink is used only in the declared normalized luminance band.
func Duotone(src image.Image, p Params) (image.Image, error) {
	dark, err := parseColor(p.Dark, color.NRGBA{R: 20, G: 24, B: 40, A: 255})
	if err != nil {
		return nil, err
	}
	light, err := parseColor(p.Light, color.NRGBA{R: 240, G: 220, B: 170, A: 255})
	if err != nil {
		return nil, err
	}
	mid, err := parseColor(p.Mid, color.NRGBA{})
	if err != nil {
		return nil, err
	}
	low, high := p.MidLow, p.MidHigh
	if high <= low {
		low, high = 0.42, 0.58
	}
	out := clone(src)
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			c := out.NRGBAAt(x, y)
			l := luminance(c)
			ink := mix(dark, light, l)
			if strings.TrimSpace(p.Mid) != "" && l >= low && l <= high {
				ink = mid
			}
			ink.A = c.A
			out.SetNRGBA(x, y, ink)
		}
	}
	return out, nil
}

// Posterize quantizes linear-light luminance to a fixed number of levels and
// remaps the result through a deterministic ink ramp.
func Posterize(src image.Image, p Params) (image.Image, error) {
	if p.Levels < 2 || p.Levels > 256 {
		return nil, fmt.Errorf("treatments: posterize levels must be 2..256")
	}
	dark, err := parseColor(p.Dark, color.NRGBA{A: 255})
	if err != nil {
		return nil, err
	}
	light, err := parseColor(p.Light, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	if err != nil {
		return nil, err
	}
	out := clone(src)
	n := float64(p.Levels - 1)
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			c := out.NRGBAAt(x, y)
			ink := mix(dark, light, math.Round(luminance(c)*n)/n)
			ink.A = c.A
			out.SetNRGBA(x, y, ink)
		}
	}
	return out, nil
}

// Halftone renders luminance as a rotated, resolution-independent dot screen.
func Halftone(src image.Image, p Params) (image.Image, error) {
	if p.LPI < 2 || p.LPI > 512 {
		return nil, fmt.Errorf("treatments: halftone lpi must be 2..512")
	}
	dark, err := parseColor(p.Dark, color.NRGBA{A: 255})
	if err != nil {
		return nil, err
	}
	light, err := parseColor(p.Light, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	if err != nil {
		return nil, err
	}
	in := clone(src)
	out := image.NewNRGBA(in.Bounds())
	draw.Draw(out, out.Bounds(), &image.Uniform{C: light}, image.Point{}, draw.Src)
	step := float64(p.LPI)
	rad := p.Angle * math.Pi / 180
	cs, sn := math.Cos(rad), math.Sin(rad)
	cx, cy := float64(in.Bounds().Dx())/2, float64(in.Bounds().Dy())/2
	for gy := -in.Bounds().Dy(); gy < in.Bounds().Dy()*2; gy += int(math.Max(1, step)) {
		for gx := -in.Bounds().Dx(); gx < in.Bounds().Dx()*2; gx += int(math.Max(1, step)) {
			rx, ry := float64(gx)-cx, float64(gy)-cy
			x := int(rx*cs - ry*sn + cx)
			y := int(rx*sn + ry*cs + cy)
			if x < 0 || y < 0 || x >= in.Bounds().Dx() || y >= in.Bounds().Dy() {
				continue
			}
			c := in.NRGBAAt(x, y)
			l := luminance(c)
			radius := (step * 0.48) * (1 - l)
			if radius <= 0.5 {
				continue
			}
			for yy := int(-radius); yy <= int(radius); yy++ {
				for xx := int(-radius); xx <= int(radius); xx++ {
					if p.Dot == "square" || float64(xx*xx+yy*yy) <= radius*radius {
						px, py := x+xx, y+yy
						if px >= 0 && py >= 0 && px < in.Bounds().Dx() && py < in.Bounds().Dy() {
							out.SetNRGBA(px, py, dark)
						}
					}
				}
			}
		}
	}
	return out, nil
}

var bayer4 = [4][4]int{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}

func ditherInk(src image.Image, p Params, diffusion bool) (image.Image, error) {
	dark, err := parseColor(p.Dark, color.NRGBA{A: 255})
	if err != nil {
		return nil, err
	}
	light, err := parseColor(p.Light, color.NRGBA{R: 255, G: 255, B: 255, A: 255})
	if err != nil {
		return nil, err
	}
	out := clone(src)
	w, h := out.Bounds().Dx(), out.Bounds().Dy()
	if diffusion {
		values := make([]float64, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				values[y*w+x] = luminance(out.NRGBAAt(x, y))
			}
		}
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				i := y*w + x
				old := values[i]
				next := 0.0
				if old >= 0.5 {
					next = 1
				}
				values[i] = next
				c := dark
				if next == 1 {
					c = light
				}
				c.A = out.NRGBAAt(x, y).A
				out.SetNRGBA(x, y, c)
				e := old - next
				if x+1 < w {
					values[i+1] += e * 7 / 16
				}
				if y+1 < h {
					if x > 0 {
						values[i+w-1] += e * 3 / 16
					}
					values[i+w] += e * 5 / 16
					if x+1 < w {
						values[i+w+1] += e / 16
					}
				}
			}
		}
		return out, nil
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := out.NRGBAAt(x, y)
			threshold := (float64(bayer4[y%4][x%4]) + 0.5) / 16
			ink := dark
			if luminance(c) >= threshold {
				ink = light
			}
			ink.A = c.A
			out.SetNRGBA(x, y, ink)
		}
	}
	return out, nil
}

func DitherOrdered(src image.Image, p Params) (image.Image, error)   { return ditherInk(src, p, false) }
func DitherDiffusion(src image.Image, p Params) (image.Image, error) { return ditherInk(src, p, true) }

// Grain adds deterministic zero-mean noise. xorshift is used instead of the
// package-global PRNG so the output is stable across processes and platforms.
func Grain(src image.Image, p Params) (image.Image, error) {
	amount := p.Amount
	if amount <= 0 {
		amount = 0.08
	}
	contrast := p.Contrast
	if contrast == 0 {
		contrast = 1
	}
	state := uint64(p.Seed)
	if state == 0 {
		state = 0x9e3779b97f4a7c15
	}
	next := func() float64 {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		return float64(state>>11) / float64(uint64(1)<<53)
	}
	out := clone(src)
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			c := out.NRGBAAt(x, y)
			n := (next()*2 - 1) * amount * 255
			adjust := func(v uint8) uint8 {
				q := (float64(v)-127.5)*contrast + 127.5 + n
				if q < 0 {
					q = 0
				}
				if q > 255 {
					q = 255
				}
				return uint8(q + 0.5)
			}
			out.SetNRGBA(x, y, color.NRGBA{R: adjust(c.R), G: adjust(c.G), B: adjust(c.B), A: c.A})
		}
	}
	return out, nil
}

// Scrim composites a deterministic directional wash, useful for preserving
// foreground text contrast without changing the source image's geometry.
func Scrim(src image.Image, p Params) (image.Image, error) {
	c, err := parseColor(p.ScrimColor, color.NRGBA{A: 255})
	if err != nil {
		return nil, err
	}
	opacity := p.Opacity
	if opacity <= 0 {
		opacity = 0.55
	}
	if opacity > 1 {
		opacity = 1
	}
	out := clone(src)
	w, h := out.Bounds().Dx(), out.Bounds().Dy()
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			t := float64(y) / math.Max(1, float64(h-1))
			if strings.EqualFold(p.Direction, "left") {
				t = float64(x) / math.Max(1, float64(w-1))
			}
			if strings.EqualFold(p.Direction, "right") {
				t = 1 - float64(x)/math.Max(1, float64(w-1))
			}
			if strings.EqualFold(p.Direction, "top") {
				t = 1 - t
			}
			if t < 0 {
				t = 0
			}
			if t > 1 {
				t = 1
			}
			a := opacity * t
			old := out.NRGBAAt(x, y)
			out.SetNRGBA(x, y, color.NRGBA{R: uint8(float64(old.R)*(1-a) + float64(c.R)*a + 0.5), G: uint8(float64(old.G)*(1-a) + float64(c.G)*a + 0.5), B: uint8(float64(old.B)*(1-a) + float64(c.B)*a + 0.5), A: old.A})
		}
	}
	return out, nil
}

// Tier2 applies the reusable breadth operations. Each operation is seeded (or
// entirely coordinate-derived) and remains a pure transform; the dispatcher
// owns registration while this package owns pixel semantics.
func Tier2(src image.Image, name string, p Params) (image.Image, error) {
	out := clone(src)
	w, h := out.Bounds().Dx(), out.Bounds().Dy()
	seed := uint64(p.Seed) + 0x9e3779b97f4a7c15
	next := func() uint8 { seed ^= seed << 7; seed ^= seed >> 9; return uint8(seed >> 56) }
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := out.NRGBAAt(x, y)
			v := uint8(luminance(c) * 255)
			switch name {
			case "line_screen":
				if (x+y/3)%8 < 2 {
					c.R, c.G, c.B = v/3, v/3, v/3
				}
			case "stipple":
				if int(next()) > int(v) {
					c.R, c.G, c.B = 20, 20, 20
				} else {
					c.R, c.G, c.B = 245, 245, 245
				}
			case "engraving":
				if (x+y)%int(math.Max(2, 12-float64(v)/24)) < 2 {
					c.R, c.G, c.B = 20, 20, 20
				}
			case "aberration":
				if x%11 == 0 {
					c.R = uint8(math.Min(255, float64(c.R)+45))
				}
			case "bloom":
				if v > 190 {
					c.R = uint8(math.Min(255, float64(c.R)+25))
					c.G = uint8(math.Min(255, float64(c.G)+25))
					c.B = uint8(math.Min(255, float64(c.B)+25))
				}
			case "curve":
				c.R, c.G, c.B = uint8(math.Sqrt(float64(c.R)/255)*255), uint8(math.Sqrt(float64(c.G)/255)*255), uint8(math.Sqrt(float64(c.B)/255)*255)
			case "defocus":
				c = average(out, x, y, 1)
			case "motion_blur":
				c = averageLine(out, x, y, 3)
			case "ascii_mosaic":
				if x%4 != 0 || y%4 != 0 {
					continue
				}
				c.R, c.G, c.B = v, v, v
			case "pixel_sort":
				if x%8 == 0 {
					c.R, c.G, c.B = v, v, v
				}
			case "displacement":
				if x > 2 {
					c = out.NRGBAAt(x-2, y)
				}
			}
			out.SetNRGBA(x, y, c)
		}
	}
	return out, nil
}

func average(img *image.NRGBA, x, y, radius int) color.NRGBA {
	var r, g, b, n int
	for yy := y - radius; yy <= y+radius; yy++ {
		for xx := x - radius; xx <= x+radius; xx++ {
			if xx >= 0 && yy >= 0 && xx < img.Bounds().Dx() && yy < img.Bounds().Dy() {
				c := img.NRGBAAt(xx, yy)
				r += int(c.R)
				g += int(c.G)
				b += int(c.B)
				n++
			}
		}
	}
	c := img.NRGBAAt(x, y)
	if n > 0 {
		c.R, c.G, c.B = uint8(r/n), uint8(g/n), uint8(b/n)
	}
	return c
}
func averageLine(img *image.NRGBA, x, y, radius int) color.NRGBA {
	var r, g, b, n int
	for i := -radius; i <= radius; i++ {
		xx := x + i
		if xx >= 0 && xx < img.Bounds().Dx() {
			c := img.NRGBAAt(xx, y)
			r += int(c.R)
			g += int(c.G)
			b += int(c.B)
			n++
		}
	}
	c := img.NRGBAAt(x, y)
	if n > 0 {
		c.R, c.G, c.B = uint8(r/n), uint8(g/n), uint8(b/n)
	}
	return c
}
