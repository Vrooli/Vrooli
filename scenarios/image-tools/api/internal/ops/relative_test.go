package ops

import (
	"image"
	"image/color"
	"math"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestResolveSpatialParamsConvertsAgainstTheShortEdge(t *testing.T) {
	// A landscape frame's short edge is its height, a portrait frame's is its
	// width. Both must resolve the same declared fraction to the same pixels,
	// which is the whole point of choosing the short edge as the reference.
	for _, tc := range []struct {
		name string
		w, h int
	}{
		{"landscape", 1344, 448},
		{"portrait", 448, 1344},
		{"square", 448, 448},
	} {
		t.Run(tc.name, func(t *testing.T) {
			p := &Params{SpacingRel: 0.02}
			resolved := ResolveSpatialParams(p, tc.w, tc.h)
			require.Equal(t, 9.0, p.Spacing, "0.02 * 448 = 8.96, rounded")
			require.Equal(t, map[string]float64{"spacing": 9}, resolved)
		})
	}
}

func TestResolveSpatialParamsScalesWithTheImage(t *testing.T) {
	small := &Params{SpacingRel: 0.02}
	large := &Params{SpacingRel: 0.02}
	ResolveSpatialParams(small, 768, 448)
	ResolveSpatialParams(large, 2304, 1344)
	require.Equal(t, 9.0, small.Spacing)
	require.Equal(t, 27.0, large.Spacing)
	// Three times the frame, three times the pitch: the same number of lines
	// across the picture, which is the same picture.
	require.InDelta(t, 3.0, large.Spacing/small.Spacing, 1e-9)
}

func TestRelativeWinsOverAbsolute(t *testing.T) {
	p := &Params{Spacing: 8, SpacingRel: 0.05, Radius: 3, RadiusRel: 0.01, Distance: 2, DistanceRel: 0.01}
	resolved := ResolveSpatialParams(p, 800, 400)
	require.Equal(t, 20.0, p.Spacing)
	require.Equal(t, 4, p.Radius)
	require.Equal(t, 4, p.Distance)
	require.Equal(t, map[string]float64{"spacing": 20, "radius": 4, "distance": 4}, resolved)
}

func TestAbsoluteSurvivesWhenNoRelativeIsSent(t *testing.T) {
	// A caller that means pixels must keep meaning pixels, and must not be
	// charged a resolved-params map it never asked for.
	p := &Params{Spacing: 8, Radius: 3, Distance: 2, BlockSize: 14, Amplitude: 5}
	require.Nil(t, ResolveSpatialParams(p, 800, 400))
	require.Equal(t, &Params{Spacing: 8, Radius: 3, Distance: 2, BlockSize: 14, Amplitude: 5}, p)
}

func TestResolvedSpacingClampsToTheTreatmentFloor(t *testing.T) {
	// Below the floor the treatments layer discards the value and substitutes
	// its own default, so an unclamped tiny fraction would render at a size
	// unrelated to what was asked for.
	p := &Params{SpacingRel: 0.0001}
	resolved := ResolveSpatialParams(p, 768, 448)
	require.Equal(t, float64(minSpacingPx), p.Spacing)
	require.Equal(t, float64(minSpacingPx), resolved["spacing"])
}

func TestResolvedAsciiCellSnapsToTheGlyphAdvance(t *testing.T) {
	for _, tc := range []struct {
		rel   float64
		short int
		want  int
	}{
		{rel: 0.02, short: 448, want: 7},   // 8.96 -> nearest is one advance
		{rel: 0.03, short: 448, want: 14},  // 13.44 -> two advances
		{rel: 0.03, short: 1344, want: 42}, // 40.32 -> six, holding the 3x ratio
		{rel: 0.05, short: 448, want: 21},  // 22.4 -> three advances
		{rel: 0.05, short: 1344, want: 70}, // 67.2 -> ten advances
		{rel: 0.0001, short: 448, want: 7}, // below one advance -> floored
	} {
		p := &Params{BlockSizeRel: tc.rel}
		resolved := ResolveSpatialParams(p, tc.short*3, tc.short)
		require.Equal(t, tc.want, p.BlockSize)
		require.Zero(t, p.BlockSize%asciiGlyphAdvancePx, "a cell that is not a whole multiple resamples the glyph")
		require.Equal(t, float64(tc.want), resolved["block_size"])
	}
}

func TestResolvedDistanceClearsTheLegacyAmplitudeAlias(t *testing.T) {
	// aberration folds Amplitude onto Distance when Distance is unset. An
	// explicit relative distance must not be second-guessed by a stale alias.
	p := &Params{Amplitude: 40, DistanceRel: 0.01}
	ResolveSpatialParams(p, 800, 400)
	require.Equal(t, 4, p.Distance)
	require.Zero(t, p.Amplitude)
}

func TestHalftoneRulingClampsToWhatTheGridCanDraw(t *testing.T) {
	// A hero-width frame renders the declared ruling untouched and reports
	// nothing, because nothing was overridden.
	hero := &Params{LPI: 120}
	require.Nil(t, ResolveSpatialParams(hero, 1440, 720))
	require.Equal(t, 120, hero.LPI)

	// The same style delivered to a 390px-wide phone surface would ask for a
	// 3.25px cell. It is clamped to the finest the grid can carry, and the
	// clamp is reported rather than silently rendered as mud.
	phone := &Params{LPI: 120}
	resolved := ResolveSpatialParams(phone, 390, 844)
	require.Equal(t, 48, phone.LPI, "390 / 8px = 48 cells across")
	require.Equal(t, map[string]float64{"lpi": 48}, resolved)
	require.GreaterOrEqual(t, 390.0/float64(phone.LPI), MinHalftoneCellPx)
}

func TestResolveSpatialParamsIgnoresUnusableGeometry(t *testing.T) {
	p := &Params{SpacingRel: 0.02}
	require.Nil(t, ResolveSpatialParams(p, 0, 400))
	require.Zero(t, p.Spacing, "a guessed geometry is worse than no conversion")
	require.Nil(t, ResolveSpatialParams(nil, 800, 400))
}

// tonalRamp is a source with a full continuous tonal range in both axes, so
// every screening treatment has something to modulate at every scale. A flat
// or two-tone source would let a broken density measurement pass.
func tonalRamp(w, h int) image.Image {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			fx, fy := float64(x)/float64(w), float64(y)/float64(h)
			v := 0.5 + 0.45*math.Sin(fx*math.Pi*1.5)*math.Cos(fy*math.Pi*0.9)
			l := uint8(math.Round(math.Max(0, math.Min(1, v)) * 255))
			img.SetNRGBA(x, y, color.NRGBA{R: l, G: l, B: l, A: 255})
		}
	}
	return img
}

// screenDensity counts ink transitions along the image's middle rows and
// normalises them to the short edge, which makes the measure comparable across
// resolutions: a screen with the same visual coarseness scores the same number
// at any size.
func screenDensity(t *testing.T, data []byte) float64 {
	t.Helper()
	img, _, err := Decode(data)
	require.NoError(t, err)
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	short := math.Min(float64(w), float64(h))
	lum := func(x, y int) float64 {
		r, g, bb, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
		return (0.2126*float64(r) + 0.7152*float64(g) + 0.0722*float64(bb)) / 65535
	}
	// Sample a band of rows rather than one, so a scanline that happens to fall
	// on a screen node does not decide the verdict.
	rows, transitions := 0, 0
	for y := h * 4 / 10; y < h*6/10; y += max(1, h/64) {
		prev := lum(0, y) > 0.5
		for x := 1; x < w; x++ {
			cur := lum(x, y) > 0.5
			if cur != prev {
				transitions++
			}
			prev = cur
		}
		rows++
	}
	require.Positive(t, rows)
	return float64(transitions) / float64(rows) / float64(w) * short
}

// TestSpatialTreatmentsHoldDensityAcrossResolutions is the acceptance test for
// this phase: one declared relative parameter, two delivery sizes, one picture.
//
// It runs through Execute, not through the treatments package directly, because
// the resolution happens on the operation path — a treatment tested in
// isolation would prove nothing about what a caller actually gets.
func TestSpatialTreatmentsHoldDensityAcrossResolutions(t *testing.T) {
	const (
		smallW, smallH = 768, 448
		largeW, largeH = 2304, 1344
	)
	for _, tc := range []struct {
		op string
		p  Params
		// tolerance is the permitted fractional drift in normalised density
		// between the two sizes. Screens quantise to whole pixels, so a small
		// drift is arithmetic, not mistuning.
		tolerance float64
	}{
		{op: "line_screen", p: Params{SpacingRel: 0.02}, tolerance: 0.06},
		{op: "stipple", p: Params{SpacingRel: 0.02, Seed: 7}, tolerance: 0.12},
		{op: "engraving", p: Params{SpacingRel: 0.02}, tolerance: 0.10},
		{op: "ascii_mosaic", p: Params{BlockSizeRel: 0.03}, tolerance: 0.20},
		{op: "displacement", p: Params{SpacingRel: 0.08, AmplitudeRel: 0.02}, tolerance: 0.10},
		{op: "aberration", p: Params{DistanceRel: 0.01}, tolerance: 0.10},
		{op: "bloom", p: Params{RadiusRel: 0.01}, tolerance: 0.10},
		{op: "defocus", p: Params{RadiusRel: 0.01}, tolerance: 0.10},
		{op: "motion_blur", p: Params{DistanceRel: 0.02}, tolerance: 0.10},
	} {
		t.Run(tc.op, func(t *testing.T) {
			small := tc.p
			large := tc.p
			smallRes, err := Execute(tc.op, encodePNG(t, tonalRamp(smallW, smallH)), &small)
			require.NoError(t, err)
			largeRes, err := Execute(tc.op, encodePNG(t, tonalRamp(largeW, largeH)), &large)
			require.NoError(t, err)

			require.NotEmpty(t, smallRes.ResolvedParams, "a relative request must report what it resolved to")
			require.NotEmpty(t, largeRes.ResolvedParams)
			for name, smallPx := range smallRes.ResolvedParams {
				largePx := largeRes.ResolvedParams[name]
				require.InDelta(t, 3.0, largePx/smallPx, 0.25,
					"%s: %s resolved %v px at %dx%d and %v px at %dx%d; a 3x frame needs a ~3x parameter",
					tc.op, name, smallPx, smallW, smallH, largePx, largeW, largeH)
			}

			sd, ld := screenDensity(t, smallRes.Bytes), screenDensity(t, largeRes.Bytes)
			require.Positive(t, sd, "%s produced no measurable structure at %dx%d", tc.op, smallW, smallH)
			require.InEpsilon(t, sd, ld, tc.tolerance,
				"%s: normalised density %.2f at %dx%d vs %.2f at %dx%d — the same style is a different picture at each size",
				tc.op, sd, smallW, smallH, ld, largeW, largeH)
		})
	}
}

// TestAbsoluteSpatialParametersDriftAcrossResolutions is the negative control.
// It asserts the defect this phase fixes is real and that the density measure
// above can actually observe it — a tolerance that passes both forms would
// prove nothing.
func TestAbsoluteSpatialParametersDriftAcrossResolutions(t *testing.T) {
	smallP, largeP := Params{Spacing: 8}, Params{Spacing: 8}
	small, err := Execute("line_screen", encodePNG(t, tonalRamp(768, 448)), &smallP)
	require.NoError(t, err)
	large, err := Execute("line_screen", encodePNG(t, tonalRamp(2304, 1344)), &largeP)
	require.NoError(t, err)
	require.Nil(t, small.ResolvedParams, "no relative parameter was sent")

	sd, ld := screenDensity(t, small.Bytes), screenDensity(t, large.Bytes)
	require.Greater(t, ld/sd, 2.0,
		"a fixed 8px pitch should triple the normalised density on a 3x frame (%.2f -> %.2f)", sd, ld)
}

// TestHalftoneRulingIsAlreadyResolutionIndependent records why halftone has no
// relative twin, against the plan's finding B3 which listed it as absolute. The
// implementation derives the screen cell from width/lpi, so the declared value
// is a count of lines across the frame and already scales — exactly, to the
// limit of the measurement, at every ruling whose cell clears the quantisation
// floor asserted below.
func TestHalftoneRulingIsAlreadyResolutionIndependent(t *testing.T) {
	for _, lpi := range []int{16, 32, 48, 64, 96} {
		smallP, largeP := Params{LPI: lpi}, Params{LPI: lpi}
		small, err := Execute("halftone", encodePNG(t, tonalRamp(768, 448)), &smallP)
		require.NoError(t, err)
		large, err := Execute("halftone", encodePNG(t, tonalRamp(2304, 1344)), &largeP)
		require.NoError(t, err)
		require.InEpsilon(t, screenDensity(t, small.Bytes), screenDensity(t, large.Bytes), 0.005,
			"lpi=%d: a line count across the width must not depend on the width", lpi)
	}
}

// TestHalftoneCellBelowTheQuantisationFloorLosesDensity pins a real limit found
// while proving the above, and is the reason MinHalftoneCellPx exists.
//
// A screen cell is drawn out of whole pixels. Once the cell falls to a handful
// of pixels the dot cannot carry tone: at lpi=130 on a 768px frame the cell is
// 5.9px and the rendered density lands 29.6% below the same ruling on a 2304px
// frame. The ruling is still resolution-independent by construction; the
// *rendering* is not, below the floor. Callers pick rulings that clear it —
// see backdrop-studio's seeded styles — rather than discovering it in delivery.
func TestHalftoneCellBelowTheQuantisationFloorLosesDensity(t *testing.T) {
	const lpi = 130
	smallP, largeP := Params{LPI: lpi}, Params{LPI: lpi}
	small, err := Execute("halftone", encodePNG(t, tonalRamp(768, 448)), &smallP)
	require.NoError(t, err)
	large, err := Execute("halftone", encodePNG(t, tonalRamp(2304, 1344)), &largeP)
	require.NoError(t, err)

	require.Less(t, 768.0/lpi, MinHalftoneCellPx, "this case only means something below the floor")
	drift := math.Abs(screenDensity(t, large.Bytes)/screenDensity(t, small.Bytes) - 1)
	require.Greater(t, drift, 0.2,
		"the documented floor claims a sub-8px cell diverges; if this no longer holds, the floor is wrong and the docs must move with it")
}
