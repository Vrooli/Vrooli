// Package perceptual scores a treated image against the source it was made
// from, and answers one question: did the subject survive the treatment?
//
// It exists because nothing in this scenario could observe that an image was
// unusable. The legibility gate measures overlay text contrast, which
// high-contrast noise passes easily — `engraved-colonnade` rendered illegible
// diagonal moire with every test green, because moire has excellent contrast.
// Contrast is not legibility, and legibility is not survival.
//
// The four metrics here are chosen against observed failures, not invented:
//
//   - Subject survival catches `engraved-colonnade`: a screen that destroyed
//     the picture it was screening.
//   - Frequency modulation catches the same failure from the other side. A
//     screen carries tone by varying its own density; texture that is equally
//     busy everywhere is carrying nothing.
//   - Tonal occupancy catches the flat-field case, where a dither over a
//     source with no gradient produces a single flat colour.
//   - Reserved-region quiet catches the `ascii-field` artifact, where a hard
//     band of maximum-density glyphs landed exactly where the headline goes.
//
// What this package is not: a judge of beauty. It proves a treatment did not
// destroy its subject. An image can pass every metric here and still be ugly,
// which is why the human verdict in the catalog review remains required.
//
// It deliberately depends on nothing but the standard library. It does not call
// image-tools and does not know what a style is: it takes two images, some
// rectangles, and some numbers.
package perceptual

import (
	"fmt"
	"image"
	"math"
	"sort"
	"strings"
)

// Metric is one measurement and the bound it was judged against. Both bounds
// are carried so a report can say not just "0.12" but "0.12, needed 0.35" —
// a number without its bound tells an operator nothing.
type Metric struct {
	Name   string  `json:"name"`
	Value  float64 `json:"value"`
	Min    float64 `json:"min,omitempty"`
	Max    float64 `json:"max,omitempty"`
	Passed bool    `json:"passed"`
	// Detail explains what the metric means in the terms of the failure it
	// exists to catch, so a rejection is actionable without reading this file.
	Detail string `json:"detail,omitempty"`
}

// Verdict is the whole judgement on one candidate.
type Verdict struct {
	Passed  bool     `json:"passed"`
	Metrics []Metric `json:"metrics"`
}

// Failures returns the metrics that did not pass, in report order.
func (v Verdict) Failures() []Metric {
	out := make([]Metric, 0, len(v.Metrics))
	for _, m := range v.Metrics {
		if !m.Passed {
			out = append(out, m)
		}
	}
	return out
}

// Error renders the verdict as the message a caller sees on rejection. It names
// every failed metric, its value, and the bound it missed.
func (v Verdict) Error() string {
	failures := v.Failures()
	if len(failures) == 0 {
		return ""
	}
	parts := make([]string, 0, len(failures))
	for _, m := range failures {
		switch {
		case m.Min > 0 && m.Value < m.Min:
			parts = append(parts, fmt.Sprintf("%s %.3f is below the %.3f floor (%s)", m.Name, m.Value, m.Min, m.Detail))
		case m.Max > 0 && m.Value > m.Max:
			parts = append(parts, fmt.Sprintf("%s %.3f is above the %.3f ceiling (%s)", m.Name, m.Value, m.Max, m.Detail))
		default:
			parts = append(parts, fmt.Sprintf("%s %.3f failed (%s)", m.Name, m.Value, m.Detail))
		}
	}
	return strings.Join(parts, "; ")
}

// Metric names. They are exported because the corpus, the candidate record and
// the integration lane all key on them, and a typo in a string literal three
// packages away is not a failure anyone wants to debug.
const (
	MetricSubjectSurvival     = "subject_survival"
	MetricTonalOccupancy      = "tonal_occupancy"
	MetricFrequencyModulation = "frequency_modulation"
	MetricReservedQuiet       = "reserved_quiet"
)

// Thresholds is the bar one candidate must clear. Zero means "do not judge this
// metric", so a style can opt out of one measurement without opting out of all
// of them — a deliberately extreme style should still have to prove it did not
// erase its subject.
type Thresholds struct {
	MinSubjectSurvival     float64 `json:"min_subject_survival,omitempty"`
	MinTonalOccupancy      float64 `json:"min_tonal_occupancy,omitempty"`
	MinFrequencyModulation float64 `json:"min_frequency_modulation,omitempty"`
	MaxReservedQuiet       float64 `json:"max_reserved_quiet,omitempty"`
}

// Region is a normalised rectangle in 0..1 image coordinates — the same shape
// the catalog stores reserved regions in, expressed without importing it.
type Region struct {
	X, Y, Width, Height float64
}

// Analysis dimensions. The image is reduced to a fixed grid before anything is
// measured, which makes every metric independent of delivery resolution: the
// same picture at 390px and at 2732px reduces to the same field.
const (
	// gridW/gridH are the low-frequency field. 64x36 is coarse enough to
	// discard the screen itself and fine enough to hold a composition — a
	// horizon, a focal mass, an arcade of columns.
	gridW, gridH = 64, 36
	// blocksW/blocksH are the texture-analysis blocks. Each block must be
	// large enough to contain many screen cells or its "energy" is just one
	// cell's phase.
	blocksW, blocksH = 16, 9
)

// Score measures a treated image against its source. Regions are the style's
// reserved rectangles; pass none to skip the reserved-region metric.
//
// A metric whose threshold is zero is measured and reported but not judged, so
// the numbers are always available to the corpus even where they are not
// enforced.
func Score(source, treated image.Image, regions []Region, t Thresholds) Verdict {
	src := lightnessField(source, gridW, gridH)
	out := lightnessField(treated, gridW, gridH)

	survival := math.Abs(correlation(src, out))
	occupancy := tonalSpan(treated)
	modulation := standardDeviation(inkCoverage(treated, blocksW, blocksH))
	quiet, hasRegions := reservedActivity(treated, regions)

	metrics := []Metric{
		{
			Name: MetricSubjectSurvival, Value: survival, Min: t.MinSubjectSurvival,
			Detail: "how much of the source's composition is still readable through the treatment",
		},
		{
			Name: MetricTonalOccupancy, Value: occupancy, Min: t.MinTonalOccupancy,
			Detail: "how much of the ink ramp the result actually uses; a collapsed image scores near zero",
		},
		{
			Name: MetricFrequencyModulation, Value: modulation, Min: t.MinFrequencyModulation,
			Detail: "how much the treatment's ink coverage varies across the frame, in L* units; uniform busy-ness is noise, not a screen",
		},
	}
	if hasRegions {
		metrics = append(metrics, Metric{
			Name: MetricReservedQuiet, Value: quiet, Max: t.MaxReservedQuiet,
			Detail: "texture inside the reserved region relative to the whole frame; overlay text needs a quiet ground",
		})
	}

	verdict := Verdict{Passed: true}
	for _, m := range metrics {
		m.Passed = (m.Min <= 0 || m.Value >= m.Min) && (m.Max <= 0 || m.Value <= m.Max)
		verdict.Passed = verdict.Passed && m.Passed
		verdict.Metrics = append(verdict.Metrics, m)
	}
	return verdict
}

// lightnessField box-averages the image down to a w*h grid of perceptual
// lightness. Box averaging (rather than sampling) is what makes this a
// low-frequency representation: it integrates the screen away instead of
// landing on an arbitrary dot.
func lightnessField(img image.Image, w, h int) []float64 {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	field := make([]float64, w*h)
	if iw == 0 || ih == 0 {
		return field
	}
	counts := make([]int, w*h)
	for y := 0; y < ih; y++ {
		cy := y * h / ih
		for x := 0; x < iw; x++ {
			cx := x * w / iw
			idx := cy*w + cx
			field[idx] += lightnessAt(img, b.Min.X+x, b.Min.Y+y)
			counts[idx]++
		}
	}
	for i := range field {
		if counts[i] > 0 {
			field[i] /= float64(counts[i])
		}
	}
	return field
}

// inkCoverage reduces the image to per-block mean lightness — how much ink each
// region of the frame actually carries.
//
// This, not local gradient energy, is what "does the screen modulate?" means. A
// binary line screen crossing a tonal ramp holds its *transition count* almost
// constant (two edges per period, whatever the duty cycle) while its *coverage*
// tracks the picture exactly. A gradient-energy measure therefore reads a
// tone-carrying screen and uniform moire as nearly the same thing — measured at
// 0.065 against 0.052, which is not a gate, it is a coin toss.
func inkCoverage(img image.Image, w, h int) []float64 {
	return lightnessField(img, w, h)
}

// tonalSpan is the p2..p98 lightness range the result occupies. Percentiles
// rather than min/max, so one stray white pixel cannot certify a black frame.
func tonalSpan(img image.Image) float64 {
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw == 0 || ih == 0 {
		return 0
	}
	// Sample rather than read every pixel: the span of a 2732px frame is not
	// meaningfully different from the span of a dense sample of it, and this
	// runs inside the render path.
	const target = 200
	stepX, stepY := max(1, iw/target), max(1, ih/target)
	values := make([]float64, 0, (iw/stepX+1)*(ih/stepY+1))
	for y := 0; y < ih; y += stepY {
		for x := 0; x < iw; x += stepX {
			values = append(values, lightnessAt(img, b.Min.X+x, b.Min.Y+y))
		}
	}
	if len(values) < 2 {
		return 0
	}
	sort.Float64s(values)
	lo := values[len(values)*2/100]
	hi := values[min(len(values)-1, len(values)*98/100)]
	return hi - lo
}

// reservedActivity is the mean texture energy inside the reserved regions
// divided by the mean over the whole frame. 1.0 means the region is exactly as
// busy as its surroundings; above 1 means the treatment concentrated its
// texture precisely where the headline goes.
func reservedActivity(img image.Image, regions []Region) (float64, bool) {
	if len(regions) == 0 {
		return 0, false
	}
	b := img.Bounds()
	iw, ih := b.Dx(), b.Dy()
	if iw < 2 || ih < 2 {
		return 0, false
	}
	var inside, insideN, all, allN float64
	for y := 0; y < ih; y++ {
		fy := float64(y) / float64(ih)
		for x := 0; x < iw-1; x++ {
			e := math.Abs(lightnessAt(img, b.Min.X+x, b.Min.Y+y) - lightnessAt(img, b.Min.X+x+1, b.Min.Y+y))
			all += e
			allN++
			fx := float64(x) / float64(iw)
			for _, r := range regions {
				if fx >= r.X && fx < r.X+r.Width && fy >= r.Y && fy < r.Y+r.Height {
					inside += e
					insideN++
					break
				}
			}
		}
	}
	if insideN == 0 || allN == 0 || all == 0 {
		return 0, false
	}
	frameMean := all / allN
	if frameMean == 0 {
		return 0, false
	}
	return (inside / insideN) / frameMean, true
}

// correlation is the Pearson coefficient between two equal-length fields. It is
// the right comparison for "did the composition survive" because it is blind to
// the ink change every treatment makes: a duotone that maps the sky to brand
// primary and the sand to brand background has changed every pixel's colour and
// none of the picture's structure.
//
// The caller takes its absolute value, because a tonal inversion — dark ink on
// light paper where the source was light on dark — preserves the composition
// just as faithfully.
func correlation(a, b []float64) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}
	n := float64(len(a))
	var sumA, sumB float64
	for i := range a {
		sumA += a[i]
		sumB += b[i]
	}
	meanA, meanB := sumA/n, sumB/n
	var cov, varA, varB float64
	for i := range a {
		da, db := a[i]-meanA, b[i]-meanB
		cov += da * db
		varA += da * da
		varB += db * db
	}
	if varA == 0 || varB == 0 {
		// A constant field has no structure to correlate. Reporting 0 says
		// "nothing survived", which is the correct reading of a flat result.
		return 0
	}
	return cov / math.Sqrt(varA*varB)
}

// standardDeviation over the block coverages, in L* units. Plain deviation
// rather than a coefficient of variation: a value in the same units as the
// thing measured is one an art director can reason about (0.0 is a flat field,
// ~0.3 is a frame spanning most of the tonal range), whereas a ratio inflates
// without limit as a picture gets darker.
func standardDeviation(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	var variance float64
	for _, v := range values {
		variance += (v - mean) * (v - mean)
	}
	return math.Sqrt(variance / float64(len(values)))
}

// lightnessAt is CIE L* normalised to 0..1, matching what the treatments layer
// maps ink with. Relative luminance would collapse most of a natural image into
// the darkest few percent and make every metric here read the wrong thing.
func lightnessAt(img image.Image, x, y int) float64 {
	r, g, b, _ := img.At(x, y).RGBA()
	yy := 0.2126*srgbLinear(r) + 0.7152*srgbLinear(g) + 0.0722*srgbLinear(b)
	var f float64
	if yy > 216.0/24389.0 {
		f = math.Cbrt(yy)
	} else {
		f = (24389.0/27.0*yy + 16) / 116
	}
	return math.Min(1, math.Max(0, (116*f-16)/100))
}

func srgbLinear(v uint32) float64 {
	x := float64(v) / 65535
	if x <= 0.04045 {
		return x / 12.92
	}
	return math.Pow((x+0.055)/1.055, 2.4)
}
