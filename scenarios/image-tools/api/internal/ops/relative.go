package ops

import "math"

// Relative spatial parameters.
//
// A spatial parameter expressed in pixels ties a style to exactly one output
// size. The same `spacing: 8` line screen carries ~96 lines across a 768px-wide
// frame and ~288 across a 2304px-wide one: the same declared style, a different
// picture at every delivery surface.
//
// The fix is a second form for each such parameter, expressed as a fraction of
// the image's SHORT edge. The short edge (rather than the width, the diagonal,
// or the area) is the reference because it is the dimension a viewer's eye
// scales against on both landscape and portrait crops of one master, so a
// derived variant keeps the density of the master it came from.
//
//	px = round(rel * min(width, height))
//
// clamped up to the operation's minimum legal value. Nothing here removes the
// absolute form: a caller who means pixels keeps meaning pixels, and only sees
// this machinery if it sends a `_rel` parameter.

// Per-operation minimums. Below these the treatments layer discards the value
// and substitutes its own default, so a relative parameter that resolved under
// the floor would silently render at an unrelated size — worse than the pixel
// behaviour it replaces. Clamping up keeps the resolved value honest, and the
// caller sees the clamp in the reported result.
const (
	minSpacingPx = 3
	minRadiusPx  = 1
	minExtentPx  = 1
	// asciiGlyphAdvancePx is the width of one cell of the 7x13 bitmap face
	// ascii_mosaic blits. A cell that is not a whole multiple of the advance
	// resamples the glyph and smears the characters the operation exists to
	// draw, so a resolved ASCII cell snaps to a multiple of it.
	asciiGlyphAdvancePx = 7
)

// MinHalftoneCellPx is the smallest halftone screen cell that still carries
// tone reliably. Halftone's `lpi` is a count of lines across the image width
// and needs no relative form, but a cell drawn out of fewer than this many
// pixels has too few pixels to modulate: measured against a tonal ramp, lpi=130
// on a 768px frame (5.9px cell) renders 29.6% less dense than the identical
// ruling on a 2304px frame, while every ruling at or above this floor matches
// to within 0.5%. A caller choosing a ruling for a delivery surface should keep
// `shortestDeliveredWidth / lpi` at or above this value.
const MinHalftoneCellPx = 8.0

// ResolveSpatialParams settles every spatial parameter on p against the actual
// image geometry: it converts each relative value into pixels, and clamps any
// absolute value the pixel grid cannot render. It returns what it resolved,
// keyed by the absolute parameter's name, for reporting back to the caller. The
// map is nil when nothing needed resolving — the common case of a request that
// sent only pixel values the image can honour.
//
// A zero or negative relative value is "not set" and leaves its absolute twin
// alone. Geometry with a non-positive edge cannot resolve anything and is left
// untouched rather than guessed at.
func ResolveSpatialParams(p *Params, width, height int) map[string]float64 {
	if p == nil || width <= 0 || height <= 0 {
		return nil
	}
	short := float64(min(width, height))
	var resolved map[string]float64
	record := func(name string, value float64) {
		if resolved == nil {
			resolved = make(map[string]float64, 4)
		}
		resolved[name] = value
	}

	if p.SpacingRel > 0 {
		p.Spacing = math.Max(minSpacingPx, math.Round(p.SpacingRel*short))
		record("spacing", p.Spacing)
	}
	if p.RadiusRel > 0 {
		p.Radius = max(minRadiusPx, int(math.Round(p.RadiusRel*short)))
		record("radius", float64(p.Radius))
	}
	if p.DistanceRel > 0 {
		p.Distance = max(minExtentPx, int(math.Round(p.DistanceRel*short)))
		record("distance", float64(p.Distance))
		// aberration accepts Amplitude as the older wire name for the same
		// knob and folds it onto Distance when Distance is unset. A resolved
		// relative distance is the caller's explicit intent, so it must not be
		// second-guessed by a stale amplitude left on the same request.
		p.Amplitude = 0
	}
	if p.AmplitudeRel > 0 {
		p.Amplitude = math.Max(minExtentPx, math.Round(p.AmplitudeRel*short))
		record("amplitude", p.Amplitude)
	}
	if p.BlockSizeRel > 0 {
		p.BlockSize = snapToGlyphCell(p.BlockSizeRel * short)
		record("block_size", float64(p.BlockSize))
	}
	// Halftone's ruling needs no relative twin — it is already a count across
	// the width — but it can still ask for a cell the pixel grid cannot draw.
	// A ruling finer than the frame supports renders as mud, not as a fine
	// screen, so it is clamped to the finest the grid can carry and the clamp
	// is reported. Silently rendering the mud is how a style that looks right
	// on a hero ships broken to a phone.
	if maxLPI := int(math.Floor(float64(width) / MinHalftoneCellPx)); p.LPI > maxLPI && maxLPI >= 1 {
		p.LPI = maxLPI
		record("lpi", float64(maxLPI))
	}
	return resolved
}

// snapToGlyphCell rounds a desired ASCII cell width to the NEAREST whole
// multiple of the bitmap face's advance, with one advance as the floor.
//
// Nearest, not down: at small cells the advance is a coarse quantum, so
// rounding down turns a 13.4px request into 7px — a 48% miss that survives into
// the delivered picture. Rounding to nearest halves the worst-case error and,
// more importantly, keeps it symmetric, so the same relative value resolves to
// a proportional cell at both a 448px and a 1344px short edge instead of
// collapsing to the floor at the small one. The residual quantisation is why
// this treatment carries a wider density tolerance than the screens do.
func snapToGlyphCell(desired float64) int {
	multiples := int(math.Round(desired / asciiGlyphAdvancePx))
	if multiples < 1 {
		multiples = 1
	}
	return multiples * asciiGlyphAdvancePx
}
