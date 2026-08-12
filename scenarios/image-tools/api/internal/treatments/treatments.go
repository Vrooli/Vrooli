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
	"sort"
	"strconv"
	"strings"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

const (
	// defaultDisplacementWavelength is 2*pi/0.12, the vertical field's period
	// before spacing was honoured. Keeping it exact means an unparameterised
	// displacement still renders byte-for-byte what it always did.
	defaultDisplacementWavelength = 2 * math.Pi / 0.12
	// displacementAxisRatio is 0.09/0.12: the horizontal field's frequency as a
	// fraction of the vertical one, likewise preserved from the hardcoded form.
	displacementAxisRatio = 0.75
	// minInkWidth is the narrowest mark a treatment may draw, in pixels. A mark
	// thinner than one whole pixel does not render as a thin mark — it renders
	// as an aliased dotted trail, which is noise wearing the costume of detail.
	minInkWidth = 1.0
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
	Spacing          float64
	Radius           int
	BladeCount       int
	Distance         int
	Amplitude        float64
	Threshold        float64
	Curve            float64
	BlockSize        int
	Axis             string
	// Normalize stretches the source's tonal range onto the full ink ramp
	// before mapping (a p1–p99 auto-level). It makes a low-contrast source use
	// the whole ramp instead of a sliver of it. It is off by default because it
	// makes the result depend on whole-image statistics: the same crop rendered
	// alone and rendered as part of a larger frame will differ. Styles that
	// render whole frames should turn it on; tiled or incremental renders
	// should leave it off.
	Normalize bool
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

// luminance is CIE relative luminance in linear light. It is the correct
// quantity for physical light maths and for WCAG contrast ratios. It is NOT a
// perceptual scale: sRGB mid-grey (128) lands at 0.216, not 0.5, and on a real
// photograph the median pixel sits near 0.05. Do not drive an ink ramp, a dot
// radius, or a quantisation threshold with it — use lightness instead.
func luminance(c color.NRGBA) float64 {
	return 0.2126*srgbLinear(c.R) + 0.7152*srgbLinear(c.G) + 0.0722*srgbLinear(c.B)
}

// lightness is perceptual lightness (CIE L*, normalised to 0..1). Tonal
// treatments map ink with this: it distributes a natural image's tones evenly
// across the ramp, which is what makes a halftone legible and a duotone reach
// both inks. Using luminance here collapses most of an image into the darkest
// few percent of the ramp.
func lightness(c color.NRGBA) float64 {
	y := luminance(c)
	var f float64
	if y > 216.0/24389.0 {
		f = math.Cbrt(y)
	} else {
		f = (24389.0/27.0*y + 16) / 116
	}
	l := (116*f - 16) / 100
	if l < 0 {
		return 0
	}
	if l > 1 {
		return 1
	}
	return l
}

// colorOr resolves an ink for the Tier-2 operations, which are dispatched
// through a signature that cannot return an error. An unparseable value falls
// back rather than failing the render; Tier-1 operations still validate and
// report. Honouring Dark/Light here is what lets a line screen, a stipple or an
// engraving be locked to a brand palette instead of a hardcoded ink.
func colorOr(value string, fallback color.NRGBA) color.NRGBA {
	c, err := parseColor(value, fallback)
	if err != nil {
		return fallback
	}
	return c
}

// toneMapper returns the lightness function a treatment should use for the
// given source. With Normalize set it first measures the source's p1–p99
// lightness span and stretches that onto 0..1, so a low-contrast source still
// uses the whole ink ramp. The returned function is pure and allocation-free.
func toneMapper(src *image.NRGBA, normalize bool) func(color.NRGBA) float64 {
	if !normalize {
		return lightness
	}
	const buckets = 1024
	var hist [buckets]int
	total := 0
	b := src.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			idx := int(lightness(src.NRGBAAt(x, y)) * (buckets - 1))
			hist[idx]++
			total++
		}
	}
	if total == 0 {
		return lightness
	}
	pick := func(frac float64) float64 {
		want, seen := int(float64(total)*frac), 0
		for i := 0; i < buckets; i++ {
			seen += hist[i]
			if seen >= want {
				return float64(i) / (buckets - 1)
			}
		}
		return 1
	}
	lo, hi := pick(0.01), pick(0.99)
	if hi-lo < 1e-3 {
		return lightness
	}
	span := hi - lo
	return func(c color.NRGBA) float64 {
		l := (lightness(c) - lo) / span
		if l < 0 {
			return 0
		}
		if l > 1 {
			return 1
		}
		return l
	}
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

// Duotone maps perceptual lightness onto a two-ink ramp. When Mid is set, the
// third ink is used only in the declared normalized lightness band.
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
	tone := toneMapper(out, p.Normalize)
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			c := out.NRGBAAt(x, y)
			l := tone(c)
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

// Posterize quantizes perceptual lightness to a fixed number of levels and
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
	tone := toneMapper(out, p.Normalize)
	n := float64(p.Levels - 1)
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			c := out.NRGBAAt(x, y)
			ink := mix(dark, light, math.Round(tone(c)*n)/n)
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
	tone := toneMapper(in, p.Normalize)
	out := image.NewNRGBA(in.Bounds())
	draw.Draw(out, out.Bounds(), &image.Uniform{C: light}, image.Point{}, draw.Src)
	// LPI is lines across the image width, which is what makes the screen
	// resolution-independent: the same LPI yields the same visual screen
	// coarseness whether the frame is 400px or 4000px wide. Treating LPI as a
	// pixel pitch (as this did) makes the screen get finer as the image grows.
	step := float64(in.Bounds().Dx()) / float64(p.LPI)
	if step < 2 {
		step = 2
	}
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
			l := tone(c)
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
	tone := toneMapper(out, p.Normalize)
	w, h := out.Bounds().Dx(), out.Bounds().Dy()
	if diffusion {
		values := make([]float64, w*h)
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				values[y*w+x] = tone(out.NRGBAAt(x, y))
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
			if tone(c) >= threshold {
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
// owns registration while this package owns pixel semantics. The implementations
// intentionally operate from a frozen source image so a treatment is stable
// regardless of traversal order and can be used for golden-image evidence.
func Tier2(src image.Image, name string, p Params) (image.Image, error) {
	in := clone(src)
	switch name {
	case "line_screen":
		return lineScreen(in, p), nil
	case "stipple":
		return stipple(in, p), nil
	case "engraving":
		return engraving(in, p), nil
	case "aberration":
		return aberration(in, p), nil
	case "bloom":
		return bloom(in, p), nil
	case "curve":
		return curve(in, p), nil
	case "defocus":
		return defocus(in, p), nil
	case "motion_blur":
		return motionBlur(in, p), nil
	case "ascii_mosaic":
		return asciiMosaic(in, p), nil
	case "pixel_sort":
		return pixelSort(in, p), nil
	case "displacement":
		return displacement(in, p), nil
	default:
		return nil, fmt.Errorf("treatments: unknown tier-2 operation %q", name)
	}
}

func screenPoint(x, y, w, h int, angle float64) (float64, float64) {
	rad := angle * math.Pi / 180
	cs, sn := math.Cos(rad), math.Sin(rad)
	cx, cy := float64(w)/2, float64(h)/2
	dx, dy := float64(x)-cx, float64(y)-cy
	return dx*cs - dy*sn + cx, dx*sn + dy*cs + cy
}

func lineScreen(in *image.NRGBA, p Params) *image.NRGBA {
	spacing := p.Spacing
	if spacing < 3 {
		spacing = 8
	}
	angle := p.Angle
	dark := colorOr(p.Dark, color.NRGBA{R: 31, G: 35, B: 45, A: 255})
	light := colorOr(p.Light, color.NRGBA{R: 244, G: 238, B: 220, A: 255})
	tone := toneMapper(in, p.Normalize)
	out := image.NewNRGBA(in.Bounds())
	for y := 0; y < in.Bounds().Dy(); y++ {
		for x := 0; x < in.Bounds().Dx(); x++ {
			c := in.NRGBAAt(x, y)
			u, _ := screenPoint(x, y, in.Bounds().Dx(), in.Bounds().Dy(), angle)
			// Triangular phase: 0 on a line's centreline, 1 midway between
			// lines. Ink is laid where the phase falls inside the tonal width,
			// so a dark region grows its lines until they merge and a light one
			// thins them to nothing. The previous form multiplied the source
			// colour by its own luminance, which darkened every pixel twice and
			// drove the whole frame to black.
			phase := math.Abs(math.Mod(math.Mod(u, spacing)+spacing, spacing)/spacing*2 - 1)
			ink := light
			if phase < 1-tone(c) {
				ink = dark
			}
			ink.A = c.A
			out.SetNRGBA(x, y, ink)
		}
	}
	return out
}

func stipple(in *image.NRGBA, p Params) *image.NRGBA {
	spacing := p.Spacing
	if spacing < 3 {
		spacing = 7
	}
	dark := colorOr(p.Dark, color.NRGBA{R: 22, G: 25, B: 35, A: 255})
	light := colorOr(p.Light, color.NRGBA{R: 246, G: 242, B: 232, A: 255})
	tone := toneMapper(in, p.Normalize)
	out := image.NewNRGBA(in.Bounds())
	draw.Draw(out, out.Bounds(), &image.Uniform{C: light}, image.Point{}, draw.Src)
	state := uint64(p.Seed) + 0x9e3779b97f4a7c15
	next := func() float64 {
		state ^= state << 13
		state ^= state >> 7
		state ^= state << 17
		return float64(state>>11) / float64(uint64(1)<<53)
	}
	for gy := 0.0; gy < float64(in.Bounds().Dy()); gy += spacing {
		for gx := 0.0; gx < float64(in.Bounds().Dx()); gx += spacing {
			x := int(gx + (next()-.5)*spacing*.72)
			y := int(gy + (next()-.5)*spacing*.72)
			if x < 0 || y < 0 || x >= in.Bounds().Dx() || y >= in.Bounds().Dy() {
				continue
			}
			l := tone(in.NRGBAAt(x, y))
			radius := (1 - l) * spacing * .46
			for yy := -int(radius) - 1; yy <= int(radius)+1; yy++ {
				for xx := -int(radius) - 1; xx <= int(radius)+1; xx++ {
					if float64(xx*xx+yy*yy) <= radius*radius {
						px, py := x+xx, y+yy
						if px >= 0 && py >= 0 && px < in.Bounds().Dx() && py < in.Bounds().Dy() {
							ink := dark
							ink.A = in.NRGBAAt(px, py).A
							out.SetNRGBA(px, py, ink)
						}
					}
				}
			}
		}
	}
	return out
}

func engraving(in *image.NRGBA, p Params) *image.NRGBA {
	dark := colorOr(p.Dark, color.NRGBA{R: 31, G: 35, B: 45, A: 255})
	light := colorOr(p.Light, color.NRGBA{R: 244, G: 238, B: 220, A: 255})
	tone := toneMapper(in, p.Normalize)
	out := image.NewNRGBA(in.Bounds())
	draw.Draw(out, out.Bounds(), &image.Uniform{C: light}, image.Point{}, draw.Src)
	spacing := p.Spacing
	if spacing < 3 {
		spacing = 9
	}
	for y := 0; y < in.Bounds().Dy(); y++ {
		for x := 0; x < in.Bounds().Dx(); x++ {
			c := in.NRGBAAt(x, y)
			l := tone(c)
			u, v := screenPoint(x, y, in.Bounds().Dx(), in.Bounds().Dy(), 24)
			// Hatching carries tone through line *width* on a fixed period. The
			// previous form varied the period while holding width constant,
			// which changes the texture's frequency without changing how much
			// ink lands — so every tone rendered at the same density.
			//
			// A line narrower than one pixel cannot be drawn; asking for one
			// yields a dotted trail of aliased fragments instead. The previous
			// floor of 0.6px did exactly that, and because it was a floor
			// rather than a cutoff it laid those fragments over the highlights
			// too — every square pixel of paper in the frame carried a broken
			// hatch. Measured on the arcade scene at 1440x720, 31.8% of the
			// result's ink runs were one or two pixels wide, against 1.5% for
			// the same scene under a line screen. That is what made
			// `engraved-colonnade` read as diagonal moire rather than as a
			// colonnade.
			//
			// An engraver does not draw an infinitely fine line for a pale
			// tone; they leave the paper blank. Below one whole pixel this
			// does the same.
			width := (1 - l) * spacing * 0.55
			drawInk := width >= minInkWidth && math.Mod(math.Mod(u+v*.12, spacing)+spacing, spacing) < width
			if l < .42 {
				// Cross-hatch only the shadows, as an engraver would.
				cross := (.42 - l) / .42 * spacing * 0.42
				drawInk = drawInk || (cross >= minInkWidth && math.Mod(math.Mod(u-v*.12, spacing)+spacing, spacing) < cross)
			}
			if drawInk {
				ink := dark
				ink.A = c.A
				out.SetNRGBA(x, y, ink)
			}
		}
	}
	return out
}

func sample(in *image.NRGBA, x, y int) color.NRGBA {
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x >= in.Bounds().Dx() {
		x = in.Bounds().Dx() - 1
	}
	if y >= in.Bounds().Dy() {
		y = in.Bounds().Dy() - 1
	}
	return in.NRGBAAt(x, y)
}

func aberration(in *image.NRGBA, p Params) *image.NRGBA {
	amount := p.Amplitude
	if amount <= 0 {
		amount = 3
	}
	out := image.NewNRGBA(in.Bounds())
	cx, cy := float64(in.Bounds().Dx())/2, float64(in.Bounds().Dy())/2
	for y := 0; y < in.Bounds().Dy(); y++ {
		for x := 0; x < in.Bounds().Dx(); x++ {
			dx, dy := float64(x)-cx, float64(y)-cy
			d := math.Hypot(dx, dy)
			if d == 0 {
				d = 1
			}
			nx, ny := dx/d, dy/d
			r := sample(in, x+int(nx*amount), y+int(ny*amount))
			g := sample(in, x, y)
			b := sample(in, x-int(nx*amount), y-int(ny*amount))
			out.SetNRGBA(x, y, color.NRGBA{R: r.R, G: g.G, B: b.B, A: g.A})
		}
	}
	return out
}

func bloom(in *image.NRGBA, p Params) *image.NRGBA {
	radius := p.Radius
	if radius < 1 {
		radius = 4
	}
	threshold := p.Threshold
	if threshold <= 0 {
		threshold = .72
	}
	bright := image.NewNRGBA(in.Bounds())
	for y := 0; y < in.Bounds().Dy(); y++ {
		for x := 0; x < in.Bounds().Dx(); x++ {
			c := in.NRGBAAt(x, y)
			// Perceptual, so a Threshold of 0.72 means "the top quarter of the
			// tonal range" as an operator would read it. Against linear
			// luminance almost nothing cleared 0.72 and the bloom never showed.
			l := lightness(c)
			if l > threshold {
				bright.SetNRGBA(x, y, color.NRGBA{R: c.R, G: c.G, B: c.B, A: uint8((l - threshold) / (1 - threshold) * 255)})
			}
		}
	}
	blurred := boxBlur(bright, radius)
	out := clone(in)
	for y := 0; y < in.Bounds().Dy(); y++ {
		for x := 0; x < in.Bounds().Dx(); x++ {
			a := blurred.NRGBAAt(x, y)
			c := out.NRGBAAt(x, y)
			k := float64(a.A) / 255 * .65
			out.SetNRGBA(x, y, color.NRGBA{R: uint8(math.Min(255, float64(c.R)+float64(a.R)*k)), G: uint8(math.Min(255, float64(c.G)+float64(a.G)*k)), B: uint8(math.Min(255, float64(c.B)+float64(a.B)*k)), A: c.A})
		}
	}
	return out
}

func boxBlur(in *image.NRGBA, radius int) *image.NRGBA {
	out := image.NewNRGBA(in.Bounds())
	for y := 0; y < in.Bounds().Dy(); y++ {
		for x := 0; x < in.Bounds().Dx(); x++ {
			out.SetNRGBA(x, y, average(in, x, y, radius))
		}
	}
	return out
}

func curve(in *image.NRGBA, p Params) *image.NRGBA {
	exponent := p.Curve
	if exponent <= 0 {
		exponent = .72
	}
	out := clone(in)
	for y := 0; y < out.Bounds().Dy(); y++ {
		for x := 0; x < out.Bounds().Dx(); x++ {
			c := out.NRGBAAt(x, y)
			f := func(v uint8) uint8 { return uint8(math.Pow(float64(v)/255, exponent)*255 + .5) }
			c.R, c.G, c.B = f(c.R), f(c.G), f(c.B)
			out.SetNRGBA(x, y, c)
		}
	}
	return out
}

func defocus(in *image.NRGBA, p Params) *image.NRGBA {
	radius := p.Radius
	if radius < 1 {
		radius = 2
	}
	return boxBlur(in, radius)
}

func motionBlur(in *image.NRGBA, p Params) *image.NRGBA {
	distance := p.Distance
	if distance < 1 {
		distance = 6
	}
	angle := p.Angle * math.Pi / 180
	out := image.NewNRGBA(in.Bounds())
	for y := 0; y < in.Bounds().Dy(); y++ {
		for x := 0; x < in.Bounds().Dx(); x++ {
			var r, g, b, n int
			for i := -distance; i <= distance; i++ {
				c := sample(in, x+int(float64(i)*math.Cos(angle)), y+int(float64(i)*math.Sin(angle)))
				r += int(c.R)
				g += int(c.G)
				b += int(c.B)
				n++
			}
			c := in.NRGBAAt(x, y)
			out.SetNRGBA(x, y, color.NRGBA{R: uint8(r / n), G: uint8(g / n), B: uint8(b / n), A: c.A})
		}
	}
	return out
}

// asciiRamp runs sparse to dense. A cell's glyph is chosen by how dark the cell
// is, so denser glyphs carry more ink and the image survives the substitution.
var asciiRamp = []rune(" .:-=+*#%@")

// asciiGlyphMasks pre-renders the ramp once into 7x13 bitmasks. basicfont is a
// compiled-in bitmap face, so this is deterministic and touches no files.
func asciiGlyphMasks() [][]bool {
	const gw, gh = 7, 13
	out := make([][]bool, len(asciiRamp))
	for i, r := range asciiRamp {
		tile := image.NewAlpha(image.Rect(0, 0, gw, gh))
		d := font.Drawer{
			Dst:  tile,
			Src:  image.NewUniform(color.Alpha{A: 255}),
			Face: basicfont.Face7x13,
			Dot:  fixed.P(0, gh-3),
		}
		d.DrawString(string(r))
		m := make([]bool, gw*gh)
		for y := 0; y < gh; y++ {
			for x := 0; x < gw; x++ {
				m[y*gw+x] = tile.AlphaAt(x, y).A > 127
			}
		}
		out[i] = m
	}
	return out
}

// asciiMosaic rebuilds the image out of characters: each cell picks a glyph
// whose density matches the cell's tone, and the glyph is blitted scaled to the
// cell. The previous implementation averaged each block to a colour from a
// five-entry ramp and filled the block with it — a pixelate, with no glyph
// anywhere in the output, which is not what the operation is named for.
func asciiMosaic(in *image.NRGBA, p Params) *image.NRGBA {
	const gw, gh = 7, 13
	cell := p.BlockSize
	if cell < 3 {
		// Default to the face's native advance so glyphs are blitted 1:1
		// rather than resampled, which keeps the characters crisp.
		cell = gw
	}
	cw := cell
	ch := cell * gh / gw
	if ch < 1 {
		ch = 1
	}
	dark := colorOr(p.Dark, color.NRGBA{R: 22, G: 26, B: 36, A: 255})
	light := colorOr(p.Light, color.NRGBA{R: 244, G: 240, B: 226, A: 255})
	tone := toneMapper(in, p.Normalize)
	masks := asciiGlyphMasks()

	w, h := in.Bounds().Dx(), in.Bounds().Dy()
	out := image.NewNRGBA(in.Bounds())
	draw.Draw(out, out.Bounds(), &image.Uniform{C: light}, image.Point{}, draw.Src)

	for by := 0; by < h; by += ch {
		for bx := 0; bx < w; bx += cw {
			var total float64
			var n int
			for y := by; y < by+ch && y < h; y++ {
				for x := bx; x < bx+cw && x < w; x++ {
					total += tone(in.NRGBAAt(x, y))
					n++
				}
			}
			if n == 0 {
				continue
			}
			idx := int((1-total/float64(n))*float64(len(asciiRamp)-1) + 0.5)
			if idx < 0 {
				idx = 0
			}
			if idx >= len(asciiRamp) {
				idx = len(asciiRamp) - 1
			}
			mask := masks[idx]
			for cy := 0; cy < ch; cy++ {
				py := by + cy
				if py >= h {
					break
				}
				gy := cy * gh / ch
				for cx := 0; cx < cw; cx++ {
					px := bx + cx
					if px >= w {
						break
					}
					if !mask[gy*gw+cx*gw/cw] {
						continue
					}
					ink := dark
					ink.A = in.NRGBAAt(px, py).A
					out.SetNRGBA(px, py, ink)
				}
			}
		}
	}
	return out
}

func pixelSort(in *image.NRGBA, p Params) *image.NRGBA {
	out := clone(in)
	threshold := p.Threshold
	if threshold <= 0 {
		threshold = .78
	}
	axis := p.Axis
	if axis == "" {
		axis = "horizontal"
	}
	if axis == "vertical" {
		for x := 0; x < in.Bounds().Dx(); x++ {
			sortColumn(out, x, threshold)
		}
	} else {
		for y := 0; y < in.Bounds().Dy(); y++ {
			sortRow(out, y, threshold)
		}
	}
	return out
}

// sortRun reorders one contiguous run of pixels by lightness. The previous
// implementation was an in-place selection sort (O(n^2) per run), which on a
// 1600px frame with a long run meant billions of comparisons; sort.Slice makes
// it O(n log n) and keeps the result identical, since lightness is a total
// order on the run's pixels.
func sortRun(get func(int) color.NRGBA, set func(int, color.NRGBA), start, end int) {
	if end-start < 2 {
		return
	}
	run := make([]color.NRGBA, 0, end-start)
	for i := start; i < end; i++ {
		run = append(run, get(i))
	}
	sort.SliceStable(run, func(a, b int) bool { return lightness(run[a]) < lightness(run[b]) })
	for i, c := range run {
		set(start+i, c)
	}
}

// sortRuns walks one scanline, finds each contiguous span brighter than the
// threshold, and sorts it.
func sortRuns(n int, get func(int) color.NRGBA, set func(int, color.NRGBA), threshold float64) {
	start := -1
	for i := 0; i <= n; i++ {
		active := i < n && lightness(get(i)) > threshold
		if active && start < 0 {
			start = i
		}
		if !active && start >= 0 {
			sortRun(get, set, start, i)
			start = -1
		}
	}
}

func sortRow(img *image.NRGBA, y int, threshold float64) {
	n := img.Bounds().Dx()
	sortRuns(n,
		func(x int) color.NRGBA { return img.NRGBAAt(x, y) },
		func(x int, c color.NRGBA) { img.SetNRGBA(x, y, c) },
		threshold)
}

func sortColumn(img *image.NRGBA, x int, threshold float64) {
	n := img.Bounds().Dy()
	sortRuns(n,
		func(y int) color.NRGBA { return img.NRGBAAt(x, y) },
		func(y int, c color.NRGBA) { img.SetNRGBA(x, y, c) },
		threshold)
}

// displacement wobbles the frame along two out-of-phase sine fields. Spacing is
// the wavelength of the vertical field in pixels; the horizontal field runs at
// a fixed 4:3 ratio of it, which is what keeps the two from beating into a
// visible grid. The defaults reproduce the wavelengths that were hardcoded
// before spacing was honoured at all — the parameter was declared on the wire,
// mapped through the handler, and then dropped on the floor here, so every
// caller that set it got the same picture.
func displacement(in *image.NRGBA, p Params) *image.NRGBA {
	amp := p.Amplitude
	if amp <= 0 {
		amp = 5
	}
	wavelength := p.Spacing
	if wavelength < 2 {
		wavelength = defaultDisplacementWavelength
	}
	freqY := 2 * math.Pi / wavelength
	freqX := freqY * displacementAxisRatio
	out := image.NewNRGBA(in.Bounds())
	for y := 0; y < in.Bounds().Dy(); y++ {
		for x := 0; x < in.Bounds().Dx(); x++ {
			dx := int(math.Sin(float64(y)*freqY) * amp)
			dy := int(math.Cos(float64(x)*freqX) * amp * .45)
			out.SetNRGBA(x, y, sample(in, x+dx, y+dy))
		}
	}
	return out
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
