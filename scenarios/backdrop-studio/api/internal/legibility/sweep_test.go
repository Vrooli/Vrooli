package legibility

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"

	"github.com/stretchr/testify/require"
)

// band builds a plate that is opaque between two vertical fractions and
// transparent elsewhere — a headland, a bank, any mass with an edge.
func band(w, h int, top, bottom float64, c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := int(top * float64(h)); y < int(bottom*float64(h)) && y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

func flat(w, h int, c color.NRGBA) []byte {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.SetNRGBA(x, y, c)
		}
	}
	var buf bytes.Buffer
	_ = png.Encode(&buf, img)
	return buf.Bytes()
}

// The case this gate exists for: a style that passes at rest and fails in
// motion.
//
// White type sits in the upper third over a bright sky. A dark mass starts
// below the reserved region — clearing it completely at rest — and rises into it
// as the page scrolls. The rest-only measurement cannot see that, and neither
// can a reviewer until they scroll.
func TestAStyleThatPassesAtRestAndFailsInMotionIsRefusedWithItsOffset(t *testing.T) {
	const w, h = 400, 400
	regions := []Region{{X: 0.05, Y: 0.05, Width: 0.5, Height: 0.25, Kind: "overlay", TextColor: "#111111"}}

	// The composite at rest: bright everywhere the type sits, so dark type is
	// comfortably legible.
	composite := flat(w, h, color.NRGBA{R: 250, G: 248, B: 240, A: 255})
	layers := []Layer{
		{Name: "sky", PNG: flat(w, h, color.NRGBA{A: 0}), Parallax: 0.02, Opacity: 1},
		// A dark mass occupying the lower half, travelling far enough to reach
		// the reserved region.
		{Name: "mass", PNG: band(w, h, 0.55, 1.0, color.NRGBA{R: 12, G: 16, B: 20, A: 255}), Parallax: 0.60, Opacity: 1},
	}

	rest, err := Measure(composite, regions, 4.5, "full_bleed")
	require.NoError(t, err)
	require.Truef(t, rest.Passes, "the fixture must pass at rest, or it proves nothing about motion (rest ratio %.2f)", rest.MinimumRatio)

	verdict, err := Sweep(composite, layers, regions, 4.5, "full_bleed")
	require.NoError(t, err)
	require.False(t, verdict.Passes, "a dark mass rising into the reserved region must be refused")
	require.Positive(t, verdict.Worst.Offset, "the failure is in motion, so the worst offset cannot be rest")
	require.Contains(t, verdict.Error(), "scroll offset")
	require.Contains(t, verdict.Error(), "passes at rest")

	// The reduced-motion viewer sees only rest, and it passes for them.
	require.True(t, verdict.ReducedMotion.Passes,
		"the reduced-motion composite is measured on its own; this fixture is legible at rest")

	// The scrim amendment is sized for the worst offset, not for rest.
	require.NotEmpty(t, verdict.Worst.Verdict.Amendments)
	require.Contains(t, verdict.Worst.Verdict.Amendments[0].Description, "least legible point in the sweep")
}

// The sample count, measured rather than asserted.
//
// The plan requires recording why the chosen count catches the failures a finer
// sampling finds. This runs the same fixtures at 9 and at 17 offsets and
// requires the verdicts to agree — if a finer sampling ever finds a failure the
// coarser one misses, this fails and the constant needs raising.
func TestAFinerSamplingFindsNoFailureTheChosenCountMisses(t *testing.T) {
	const w, h = 400, 400
	regions := []Region{{X: 0.05, Y: 0.05, Width: 0.5, Height: 0.25, Kind: "overlay", TextColor: "#111111"}}
	composite := flat(w, h, color.NRGBA{R: 250, G: 248, B: 240, A: 255})

	// A family of masses at different depths and travels, including thin ones
	// that cross the region quickly — the case a coarse sampling would miss.
	for _, tc := range []struct {
		name        string
		top, bottom float64
		parallax    float64
	}{
		{"thick mass, slow", 0.55, 1.00, 0.30},
		{"thick mass, fast", 0.55, 1.00, 0.60},
		{"thin band, slow", 0.62, 0.70, 0.30},
		{"thin band, fast", 0.62, 0.70, 0.60},
		{"very thin band, very fast", 0.66, 0.69, 0.90},
		{"band that never reaches", 0.90, 0.95, 0.05},
	} {
		t.Run(tc.name, func(t *testing.T) {
			layers := []Layer{
				{Name: "ground", PNG: flat(w, h, color.NRGBA{A: 0}), Parallax: 0.01, Opacity: 1},
				{Name: "mass", PNG: band(w, h, tc.top, tc.bottom, color.NRGBA{R: 12, G: 16, B: 20, A: 255}), Parallax: tc.parallax, Opacity: 1},
			}
			coarse, err := sweepAt(composite, layers, regions, 4.5, "full_bleed", SweepSamples)
			require.NoError(t, err)
			fine, err := sweepAt(composite, layers, regions, 4.5, "full_bleed", 17)
			require.NoError(t, err)

			require.Equalf(t, fine.Passes, coarse.Passes,
				"sampling at %d disagrees with sampling at 17 (coarse worst %.2f at %.3f; fine worst %.2f at %.3f); raise SweepSamples and re-record the measurement",
				SweepSamples, coarse.Worst.Verdict.MinimumRatio, coarse.Worst.Offset,
				fine.Worst.Verdict.MinimumRatio, fine.Worst.Offset)
		})
	}
}

// A stack whose plates all move together has nothing to sweep, so its verdict is
// the rest verdict — the same answer as before, by the same path.
func TestAStackWithNoDepthSweepsToItsRestVerdict(t *testing.T) {
	const w, h = 200, 200
	regions := []Region{{X: 0.05, Y: 0.05, Width: 0.5, Height: 0.25, Kind: "overlay", TextColor: "#111111"}}
	composite := flat(w, h, color.NRGBA{R: 250, G: 248, B: 240, A: 255})
	layers := []Layer{
		{Name: "a", PNG: flat(w, h, color.NRGBA{A: 0}), Parallax: 0.2, Opacity: 1},
		{Name: "b", PNG: flat(w, h, color.NRGBA{A: 0}), Parallax: 0.2, Opacity: 1},
	}
	verdict, err := Sweep(composite, layers, regions, 4.5, "full_bleed")
	require.NoError(t, err)
	require.Len(t, verdict.Samples, 1, "nothing moves, so there is one offset to measure")
	require.Zero(t, verdict.Worst.Offset)
	require.True(t, verdict.Passes)
}

// A candidate that fails at rest fails the sweep, and says so without inventing
// a scroll offset it did not fail at.
func TestAFailureAtRestIsReportedAsRest(t *testing.T) {
	const w, h = 200, 200
	regions := []Region{{X: 0.05, Y: 0.05, Width: 0.5, Height: 0.25, Kind: "overlay", TextColor: "#ffffff"}}
	composite := flat(w, h, color.NRGBA{R: 250, G: 250, B: 250, A: 255})
	// White type on white paper.
	regions[0].TextColor = "#fbfbfb"
	verdict, err := Sweep(composite, nil, regions, 4.5, "full_bleed")
	require.NoError(t, err)
	require.False(t, verdict.Passes)
	require.Zero(t, verdict.Worst.Offset)
	require.Contains(t, verdict.Error(), "at rest")
	require.NotContains(t, verdict.Error(), "scroll offset")
	require.False(t, verdict.ReducedMotion.Passes,
		"a picture illegible at rest fails a reduced-motion viewer too, and must say so independently")
}
