package ai

import (
	"errors"
	"image"
	"image/color"
	"math"
	"sort"
)

// ColorTransferParams tunes how far a masked region of one image is pulled
// toward the colour of a reference region.
type ColorTransferParams struct {
	// Strength is the blend of the transferred colour into the region, 0..1.
	Strength float64
	// Luminance is the share of the reference's lightness offset that is applied,
	// 0..1. Zero keeps the region's own shading exactly; one moves its mean
	// lightness to the reference's (a dark head of hair onto a fair one).
	Luminance float64
	// Flatten is how far the region's chroma spread may shrink toward the
	// reference's, 0..1. Chroma is only ever flattened, never amplified, so
	// noise in a small region cannot saturate it.
	Flatten float64
	// FeatherPx softens the region edge so the change fades into its surroundings.
	FeatherPx float64
}

// DefaultColorTransferParams is a skin-tone match: full colour, half the
// lightness offset, mild flattening, a few pixels of feather.
func DefaultColorTransferParams() ColorTransferParams {
	return ColorTransferParams{Strength: 1, Luminance: 0.5, Flatten: 0.4, FeatherPx: 4}
}

// regionLo and regionHi bound the lightness percentile band that feeds the
// statistics, so a specular highlight or a deep shadow inside the region does
// not move its mean.
const regionLo, regionHi = 0.05, 0.95

// ColorTransfer recolours the pixels of src selected by mask so their mean
// colour (CIELAB) and spread match those of the reference region, and keeps
// everything else untouched. The reference's alpha channel selects its region:
// transparent pixels are ignored, so a fully opaque reference contributes every
// pixel. The shading and texture inside the region survive because each pixel is
// shifted, not replaced.
func ColorTransfer(src, mask, reference image.Image, p ColorTransferParams) (*image.NRGBA, error) {
	b := src.Bounds()
	w, h := b.Dx(), b.Dy()
	target := make([]float64, w*h*3)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			c := color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			l, a, bb := srgbToLab(c.R, c.G, c.B)
			i := (y*w + x) * 3
			target[i], target[i+1], target[i+2] = l, a, bb
		}
	}
	weight := sampledWeights(mask, w, h)
	region := make([]bool, w*h)
	for i, v := range weight {
		region[i] = v > 0.5
	}
	tMean, tStd, ok := labStats(target, region)
	if !ok {
		return nil, errors.New("color transfer: the mask selects no pixels of the image")
	}

	rb := reference.Bounds()
	rw, rh := rb.Dx(), rb.Dy()
	ref := make([]float64, rw*rh*3)
	refRegion := make([]bool, rw*rh)
	for y := 0; y < rh; y++ {
		for x := 0; x < rw; x++ {
			c := color.NRGBAModel.Convert(reference.At(rb.Min.X+x, rb.Min.Y+y)).(color.NRGBA)
			i := y*rw + x
			refRegion[i] = c.A >= 128
			ref[i*3], ref[i*3+1], ref[i*3+2] = srgbToLab(c.R, c.G, c.B)
		}
	}
	rMean, rStd, ok := labStats(ref, refRegion)
	if !ok {
		return nil, errors.New("color transfer: the reference has no opaque pixels to take colour from")
	}

	strength := clamp01(p.Strength)
	lum := clamp01(p.Luminance)
	floor := 1 - clamp01(p.Flatten)
	var gain [3]float64
	for c := 1; c < 3; c++ {
		gain[c] = math.Min(1, math.Max(floor, rStd[c]/tStd[c]))
	}
	if p.FeatherPx > 0 {
		weight = blurWeights(weight, w, h, p.FeatherPx)
	}

	out := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			i := y*w + x
			orig := color.NRGBAModel.Convert(src.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
			wt := weight[i] * strength
			if wt <= 0 {
				out.SetNRGBA(x, y, orig)
				continue
			}
			l, a, bb := target[i*3], target[i*3+1], target[i*3+2]
			nl := l + lum*(rMean[0]-tMean[0])
			na := (a-tMean[1])*gain[1] + rMean[1]
			nb := (bb-tMean[2])*gain[2] + rMean[2]
			r, g, bl := labToSRGB(l+(nl-l)*wt, a+(na-a)*wt, bb+(nb-bb)*wt)
			out.SetNRGBA(x, y, color.NRGBA{R: r, G: g, B: bl, A: orig.A})
		}
	}
	return out, nil
}

func clamp01(v float64) float64 { return math.Min(1, math.Max(0, v)) }

// sampledWeights reads the mask at the source's geometry (they need not share a
// size) as 0/1 weights: selected where it is opaque and brighter than mid-grey.
func sampledWeights(mask image.Image, w, h int) []float64 {
	mb := mask.Bounds()
	mw, mh := mb.Dx(), mb.Dy()
	out := make([]float64, w*h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, a := mask.At(mb.Min.X+x*mw/w, mb.Min.Y+y*mh/h).RGBA()
			if a > 0 && (int(r>>8)+int(g>>8)+int(bl>>8))/3 > 127 {
				out[y*w+x] = 1
			}
		}
	}
	return out
}

// labStats is the mean and standard deviation of each Lab channel over the
// region, with the lightest and darkest 5% of its pixels left out.
func labStats(lab []float64, region []bool) (mean, std [3]float64, ok bool) {
	var idx []int
	for i, in := range region {
		if in {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		return mean, std, false
	}
	sort.Slice(idx, func(a, b int) bool { return lab[idx[a]*3] < lab[idx[b]*3] })
	lo, hi := int(float64(len(idx))*regionLo), int(float64(len(idx))*regionHi)
	if hi-lo < 1 {
		lo, hi = 0, len(idx)
	}
	kept := idx[lo:hi]
	for _, i := range kept {
		for c := 0; c < 3; c++ {
			mean[c] += lab[i*3+c]
		}
	}
	n := float64(len(kept))
	for c := range mean {
		mean[c] /= n
	}
	for _, i := range kept {
		for c := 0; c < 3; c++ {
			d := lab[i*3+c] - mean[c]
			std[c] += d * d
		}
	}
	for c := range std {
		std[c] = math.Sqrt(std[c]/n) + 1e-6
	}
	return mean, std, true
}

// blurWeights feathers a 0/1 weight map with three box passes (a close Gaussian
// at radius sigma), separable so it stays linear in the pixel count.
func blurWeights(in []float64, w, h int, sigma float64) []float64 {
	radius := int(math.Max(1, math.Round(sigma*1.2)))
	cur := append([]float64(nil), in...)
	tmp := make([]float64, len(in))
	for pass := 0; pass < 3; pass++ {
		boxBlurAxis(cur, tmp, w, h, radius, true)
		boxBlurAxis(tmp, cur, w, h, radius, false)
	}
	return cur
}

func boxBlurAxis(src, dst []float64, w, h, radius int, horizontal bool) {
	lines, length := h, w
	if !horizontal {
		lines, length = w, h
	}
	at := func(line, pos int) int {
		if horizontal {
			return line*w + pos
		}
		return pos*w + line
	}
	span := float64(2*radius + 1)
	for line := 0; line < lines; line++ {
		var sum float64
		for k := -radius; k <= radius; k++ {
			sum += src[at(line, min(max(k, 0), length-1))]
		}
		for pos := 0; pos < length; pos++ {
			dst[at(line, pos)] = sum / span
			sum += src[at(line, min(pos+radius+1, length-1))] - src[at(line, max(pos-radius, 0))]
		}
	}
}

var (
	labWhite = [3]float64{0.95047, 1, 1.08883}
	rgbToXYZ = [3][3]float64{{0.4124564, 0.3575761, 0.1804375}, {0.2126729, 0.7151522, 0.0721750}, {0.0193339, 0.1191920, 0.9503041}}
	xyzToRGB = [3][3]float64{{3.2404542, -1.5371385, -0.4985314}, {-0.9692660, 1.8760108, 0.0415560}, {0.0556434, -0.2040259, 1.0572252}}
)

const labEpsilon, labKappa = 216.0 / 24389, 24389.0 / 27

func srgbToLab(r, g, b uint8) (float64, float64, float64) {
	lin := [3]float64{srgbToLinear(r), srgbToLinear(g), srgbToLinear(b)}
	var f [3]float64
	for i := range f {
		xyz := (rgbToXYZ[i][0]*lin[0] + rgbToXYZ[i][1]*lin[1] + rgbToXYZ[i][2]*lin[2]) / labWhite[i]
		if xyz > labEpsilon {
			f[i] = math.Cbrt(xyz)
		} else {
			f[i] = (labKappa*xyz + 16) / 116
		}
	}
	return 116*f[1] - 16, 500 * (f[0] - f[1]), 200 * (f[1] - f[2])
}

func labToSRGB(l, a, b float64) (uint8, uint8, uint8) {
	fy := (l + 16) / 116
	f := [3]float64{fy + a/500, fy, fy - b/200}
	var xyz [3]float64
	for i, v := range f {
		if cube := v * v * v; cube > labEpsilon {
			xyz[i] = cube * labWhite[i]
		} else {
			xyz[i] = (116*v - 16) / labKappa * labWhite[i]
		}
	}
	var out [3]uint8
	for i := range out {
		lin := xyzToRGB[i][0]*xyz[0] + xyzToRGB[i][1]*xyz[1] + xyzToRGB[i][2]*xyz[2]
		out[i] = linearToSRGB(lin)
	}
	return out[0], out[1], out[2]
}

func srgbToLinear(v uint8) float64 {
	c := float64(v) / 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func linearToSRGB(lin float64) uint8 {
	lin = clamp01(lin)
	c := 12.92 * lin
	if lin > 0.0031308 {
		c = 1.055*math.Pow(lin, 1/2.4) - 0.055
	}
	return uint8(math.Round(clamp01(c) * 255))
}
