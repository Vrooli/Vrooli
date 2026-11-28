package scenes

import (
	"math"
	"sort"
)

// Helpers shared by every generator. They live here rather than in whichever
// file first needed them so that no generator reaches into another: a generator
// file should read as one picture's worth of decisions, and nothing else should
// break when it is rewritten.

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return v
}

// mixRGB linearly interpolates two colours. Interpolation is in sRGB rather
// than linear light on purpose: these are art-directed palette stops chosen by
// eye against the rendered result, so blending them the way they were chosen
// keeps the midpoint where the chooser expected it.
func mixRGB(a, b [3]float64, t float64) [3]float64 {
	t = clamp01(t)
	return [3]float64{
		a[0] + (b[0]-a[0])*t,
		a[1] + (b[1]-a[1])*t,
		a[2] + (b[2]-a[2])*t,
	}
}

// percentile returns the value at the given quantile of a buffer, without
// disturbing the caller's slice.
//
// Accumulating generators need this rather than a maximum: a few crossing
// points in a flow field or a caustic cusp run far above the body of the
// distribution, and normalising against them compresses the whole picture into
// the bottom of the range. Normalising against the 99.5th percentile lets those
// few points clip — which is what a highlight is — and keeps the rest open.
func percentile(buf []float64, q float64) float64 {
	if len(buf) == 0 {
		return 0
	}
	sorted := make([]float64, len(buf))
	copy(sorted, buf)
	sort.Float64s(sorted)
	i := int(float64(len(sorted)-1) * clamp01(q))
	return sorted[i]
}

// sampleBilinear reads a coarse simulation grid at fractional coordinates.
// Nearest-neighbour sampling leaves stair-stepped edges, and a fine screen
// downstream turns those steps into aliasing — the exact failure mode this
// scenario spent a phase removing from its resampler.
func sampleBilinear(buf []float64, w, h int, x, y float64) float64 {
	if w <= 0 || h <= 0 {
		return 0
	}
	x = math.Max(0, math.Min(float64(w-1), x))
	y = math.Max(0, math.Min(float64(h-1), y))
	x0, y0 := int(x), int(y)
	x1, y1 := min(x0+1, w-1), min(y0+1, h-1)
	fx, fy := x-float64(x0), y-float64(y0)
	top := buf[y0*w+x0]*(1-fx) + buf[y0*w+x1]*fx
	bot := buf[y1*w+x0]*(1-fx) + buf[y1*w+x1]*fx
	return top*(1-fy) + bot*fy
}

// expandRange rescales three colour channels in place so the field's own
// luminance spans the target range, holding hue and saturation.
//
// Smearing and blurring are averaging operations, so they compress the tonal
// range by construction: a mesh gradient whose palette runs from near-black to
// near-white comes out of a nine-sample smear occupying the middle two thirds
// of the ramp. That is not a palette problem to be solved by choosing more
// extreme stops — the compression scales with the smear length, so any fixed
// palette is wrong at some setting. Measuring what came out and re-expanding it
// is the only correction that holds across the parameter space.
//
// Scaling all three channels by one factor is what preserves the hue: it moves
// the colour along the ray from black through itself, which is a lightness
// change and nothing else.
func expandRange(channels [][]float64, lo, hi float64) {
	if len(channels) != 3 || len(channels[0]) == 0 {
		return
	}
	n := len(channels[0])
	lum := make([]float64, n)
	for i := 0; i < n; i++ {
		lum[i] = (0.299*channels[0][i] + 0.587*channels[1][i] + 0.114*channels[2][i]) / 255
	}
	// Percentiles rather than extremes: a handful of pixels at a stop's exact
	// centre should not set the scale for the whole frame.
	p1, p99 := percentile(lum, 0.01), percentile(lum, 0.99)
	if p99-p1 < 1e-6 {
		return
	}
	for i := 0; i < n; i++ {
		if lum[i] <= 1e-6 {
			continue
		}
		want := lo + (lum[i]-p1)/(p99-p1)*(hi-lo)
		if want < 0 {
			want = 0
		}
		gain := want / lum[i]
		for ch := 0; ch < 3; ch++ {
			channels[ch][i] = math.Min(255, channels[ch][i]*gain)
		}
	}
}

// blurBuffer applies a separable box blur of the given radius. Two passes, so
// the cost is linear in the radius rather than quadratic.
func blurBuffer(buf []float64, w, h, radius int) []float64 {
	if radius < 1 || w <= 0 || h <= 0 {
		return buf
	}
	tmp := make([]float64, len(buf))
	out := make([]float64, len(buf))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum float64
			var n int
			for k := -radius; k <= radius; k++ {
				xx := x + k
				if xx < 0 || xx >= w {
					continue
				}
				sum += buf[y*w+xx]
				n++
			}
			tmp[y*w+x] = sum / float64(n)
		}
	}
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var sum float64
			var n int
			for k := -radius; k <= radius; k++ {
				yy := y + k
				if yy < 0 || yy >= h {
					continue
				}
				sum += tmp[yy*w+x]
				n++
			}
			out[y*w+x] = sum / float64(n)
		}
	}
	return out
}
