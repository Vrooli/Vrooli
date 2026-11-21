package perceptual

import (
	"image"
	"image/color"
	"math"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/require"
)

// The synthetic sources below have known answers by construction, so a metric
// that drifts is caught without needing a rendered image on disk.

// gradientScene is a source with real tonal depth and a clear composition: a
// sky that ramps, a horizon, and a bright focal disc. This is what a treatment
// is supposed to survive.
func gradientScene(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	cx, cy, r := float64(w)*0.7, float64(h)*0.3, float64(h)*0.12
	for y := 0; y < h; y++ {
		fy := float64(y) / float64(h)
		for x := 0; x < w; x++ {
			v := 0.85 - 0.6*fy
			if fy > 0.55 {
				v = 0.25 + 0.2*(fy-0.55)
			}
			dx, dy := float64(x)-cx, float64(y)-cy
			if math.Hypot(dx, dy) < r {
				v = 0.98
			}
			img.SetNRGBA(x, y, gray(v))
		}
	}
	return img
}

// screenedScene is gradientScene rendered on a line screen whose line width
// tracks the source tone — an idealised good treatment. The composition is
// still readable and the ink coverage varies with the picture.
//
// The screen period is a fraction of the width, not a pixel count, because
// that is what every seeded style now sends. A fixed-pixel screen here would
// make the resolution-independence test measure the fixture's defect rather
// than the metric's.
func screenedScene(w, h int) image.Image {
	src := gradientScene(w, h)
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	period := max(2, w/48)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			tone := lightnessAt(src, x, y)
			ink := float64(x%period) / float64(period)
			if ink < 1-tone {
				img.SetNRGBA(x, y, gray(0.08))
			} else {
				img.SetNRGBA(x, y, gray(0.95))
			}
		}
	}
	return img
}

// moireField is the `engraved-colonnade` failure in synthetic form: maximum
// local contrast, uniform across the whole frame, carrying no picture at all.
func moireField(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			if (x+y)%3 == 0 {
				img.SetNRGBA(x, y, gray(0.05))
			} else {
				img.SetNRGBA(x, y, gray(0.95))
			}
		}
	}
	return img
}

// flatField is the collapsed-dither failure: a treatment that quantised its
// whole source into one ink.
func flatField(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, gray(0.12))
		}
	}
	return img
}

func gray(v float64) color.NRGBA {
	// Invert the L* transform so a requested lightness lands where asked.
	l := math.Min(1, math.Max(0, v))
	f := (l*100 + 16) / 116
	var y float64
	if f > 6.0/29.0 {
		y = f * f * f
	} else {
		y = (116*f - 16) * 27 / 24389
	}
	var s float64
	if y <= 0.0031308 {
		s = y * 12.92
	} else {
		s = 1.055*math.Pow(y, 1/2.4) - 0.055
	}
	c := uint8(math.Round(math.Min(1, math.Max(0, s)) * 255))
	return color.NRGBA{R: c, G: c, B: c, A: 255}
}

const testW, testH = 480, 270

func TestSubjectSurvivalSeparatesAScreenFromNoise(t *testing.T) {
	src := gradientScene(testW, testH)

	good := Score(src, screenedScene(testW, testH), nil, Thresholds{})
	bad := Score(src, moireField(testW, testH), nil, Thresholds{})

	goodScore := metricValue(t, good, MetricSubjectSurvival)
	badScore := metricValue(t, bad, MetricSubjectSurvival)

	require.Greater(t, goodScore, 0.8, "a screen that tracks its source must preserve the composition")
	require.Less(t, badScore, 0.2, "uniform moire carries no composition at all")
	require.Greater(t, goodScore-badScore, 0.6, "the metric must separate the two cases by a wide margin, not a hair")
}

func TestSubjectSurvivalIsBlindToInkAndPolarity(t *testing.T) {
	// A duotone changes every pixel's colour and no pixel's structure; an
	// inverted print does the same. Both must score as survival, or every
	// legitimate treatment fails the gate.
	src := gradientScene(testW, testH)
	tinted := image.NewNRGBA(image.Rect(0, 0, testW, testH))
	inverted := image.NewNRGBA(image.Rect(0, 0, testW, testH))
	for y := 0; y < testH; y++ {
		for x := 0; x < testW; x++ {
			l := lightnessAt(src, x, y)
			tinted.SetNRGBA(x, y, color.NRGBA{R: uint8(l * 200), G: uint8(l * 60), B: uint8(30 + l*120), A: 255})
			inverted.SetNRGBA(x, y, gray(1-l))
		}
	}
	require.Greater(t, metricValue(t, Score(src, tinted, nil, Thresholds{}), MetricSubjectSurvival), 0.95)
	require.Greater(t, metricValue(t, Score(src, inverted, nil, Thresholds{}), MetricSubjectSurvival), 0.95)
}

func TestTonalOccupancyCatchesACollapsedResult(t *testing.T) {
	src := gradientScene(testW, testH)
	require.Less(t, metricValue(t, Score(src, flatField(testW, testH), nil, Thresholds{}), MetricTonalOccupancy), 0.02,
		"a single flat ink occupies none of the ramp")
	require.Greater(t, metricValue(t, Score(src, screenedScene(testW, testH), nil, Thresholds{}), MetricTonalOccupancy), 0.7,
		"a two-ink screen occupies the whole ramp between its inks")
}

func TestFrequencyModulationCatchesUniformTexture(t *testing.T) {
	src := gradientScene(testW, testH)
	uniform := metricValue(t, Score(src, moireField(testW, testH), nil, Thresholds{}), MetricFrequencyModulation)
	varying := metricValue(t, Score(src, screenedScene(testW, testH), nil, Thresholds{}), MetricFrequencyModulation)

	require.Less(t, uniform, 0.02, "moire holds the same ink coverage everywhere, so it carries no tone")
	require.Greater(t, varying, 0.15, "a screen carries tone by varying its own coverage")
}

func TestFrequencyModulationIsNotFooledByRandomNoise(t *testing.T) {
	// Random noise has high local contrast and no picture. It must fail on
	// survival even though its energy varies block to block by chance.
	rng := rand.New(rand.NewSource(7))
	noise := image.NewNRGBA(image.Rect(0, 0, testW, testH))
	for y := 0; y < testH; y++ {
		for x := 0; x < testW; x++ {
			noise.SetNRGBA(x, y, gray(rng.Float64()))
		}
	}
	v := Score(gradientScene(testW, testH), noise, nil, Thresholds{MinSubjectSurvival: 0.35})
	require.False(t, v.Passed)
	require.Contains(t, v.Error(), MetricSubjectSurvival)
}

func TestReservedRegionQuietCatchesTextureUnderTheHeadline(t *testing.T) {
	src := gradientScene(testW, testH)
	region := []Region{{X: 0.05, Y: 0.1, Width: 0.4, Height: 0.3}}

	// A frame whose texture is concentrated exactly where the headline goes.
	busy := image.NewNRGBA(image.Rect(0, 0, testW, testH))
	for y := 0; y < testH; y++ {
		for x := 0; x < testW; x++ {
			inRegion := float64(x)/testW < 0.45 && float64(y)/testH > 0.1 && float64(y)/testH < 0.4
			if inRegion && x%2 == 0 {
				busy.SetNRGBA(x, y, gray(0.05))
			} else {
				busy.SetNRGBA(x, y, gray(0.9))
			}
		}
	}
	v := Score(src, busy, region, Thresholds{MaxReservedQuiet: 1.2})
	require.False(t, v.Passed)
	require.Contains(t, v.Error(), MetricReservedQuiet)
	require.Greater(t, metricValue(t, v, MetricReservedQuiet), 2.0)

	// The same texture spread evenly over the frame is not a reserved-region
	// problem, even though it is exactly as busy inside the region.
	even := moireField(testW, testH)
	require.InDelta(t, 1.0, metricValue(t, Score(src, even, region, Thresholds{}), MetricReservedQuiet), 0.15)
}

func TestReservedMetricIsAbsentWithoutRegions(t *testing.T) {
	v := Score(gradientScene(testW, testH), screenedScene(testW, testH), nil, Thresholds{})
	for _, m := range v.Metrics {
		require.NotEqual(t, MetricReservedQuiet, m.Name,
			"a style with no reserved region must not be judged on one, nor reported a number it has no basis for")
	}
}

func TestAZeroThresholdMeasuresWithoutJudging(t *testing.T) {
	// The corpus needs every number even where a style opts out of enforcing
	// it, so a zero threshold must still produce a reported metric.
	v := Score(gradientScene(testW, testH), moireField(testW, testH), nil, Thresholds{})
	require.True(t, v.Passed, "nothing was asked of it, so nothing failed")
	require.Len(t, v.Metrics, 3)
	require.Less(t, metricValue(t, v, MetricSubjectSurvival), 0.2, "but the number is still there and still damning")
}

func TestVerdictErrorNamesTheMetricAndTheBound(t *testing.T) {
	v := Score(gradientScene(testW, testH), moireField(testW, testH), nil,
		Thresholds{MinSubjectSurvival: 0.35, MinFrequencyModulation: 0.15})
	require.False(t, v.Passed)
	require.Len(t, v.Failures(), 2)
	msg := v.Error()
	require.Contains(t, msg, MetricSubjectSurvival)
	require.Contains(t, msg, "0.350")
	require.Contains(t, msg, MetricFrequencyModulation)
	require.Contains(t, msg, "below")
}

func TestMetricsAreIndependentOfDeliveryResolution(t *testing.T) {
	// The same picture at two sizes must score the same, or a style would pass
	// at one surface and fail at another for no reason a designer could act on.
	small := Score(gradientScene(390, 220), screenedScene(390, 220), nil, Thresholds{})
	large := Score(gradientScene(1440, 810), screenedScene(1440, 810), nil, Thresholds{})
	for _, name := range []string{MetricSubjectSurvival, MetricTonalOccupancy} {
		require.InDeltaf(t, metricValue(t, small, name), metricValue(t, large, name), 0.1,
			"%s must not depend on the delivery size", name)
	}
}

func TestScoreToleratesDegenerateInput(t *testing.T) {
	empty := image.NewNRGBA(image.Rect(0, 0, 0, 0))
	v := Score(empty, empty, []Region{{X: 0, Y: 0, Width: 1, Height: 1}}, Thresholds{MinSubjectSurvival: 0.35})
	require.False(t, v.Passed, "an empty image survived nothing")
	require.NotPanics(t, func() { _ = v.Error() })
}

func metricValue(t *testing.T, v Verdict, name string) float64 {
	t.Helper()
	for _, m := range v.Metrics {
		if m.Name == name {
			return m.Value
		}
	}
	t.Fatalf("verdict carries no %q metric", name)
	return 0
}
