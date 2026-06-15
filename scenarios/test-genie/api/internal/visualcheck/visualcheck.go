// Package visualcheck provides engine-agnostic, dependency-free analysis of
// rendered UI screenshots. It answers two questions over raw PNG bytes:
//
//   - RenderHealth: is this single capture clearly broken (blank / solid color)?
//     A genuinely broken render is a hard failure with no baseline required.
//   - Compare: how different are two captures of the same surface? The result is
//     a neutral magnitude (changed fraction), never a verdict — the caller
//     decides whether a difference is a pass, a "review before/after" signal, or
//     a failure.
//
// Both operate on a coarse downscaled luminance grid. Downscaling to a fixed
// grid makes the analysis robust to anti-aliasing/font jitter and to size
// differences between two captures, and keeps the work O(pixels) with no
// external image library — only the standard library's image/png decoder.
package visualcheck

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png" // register the PNG decoder for image.Decode
	"math"
	"os"
	"strconv"
)

// Thresholds is the tunable control surface for visual analysis. Every field is
// bounded and monotonic; see DefaultThresholds for the documented defaults and
// docs/reference/configuration.md for operator guidance.
type Thresholds struct {
	// GridSize is the edge length of the square luminance grid each image is
	// downscaled to before analysis. Higher = finer detail (more sensitive,
	// slower); lower = coarser (more jitter-tolerant). Clamped to the image
	// dimensions for tiny images.
	GridSize int

	// BlankFraction is the share of grid cells that must fall in a single
	// luminance band for a capture to be judged a blank/solid-color broken
	// render. Higher = stricter (only near-perfectly-uniform captures are
	// broken). Range (0,1].
	BlankFraction float64

	// MinVariance is the lower bound on grid-luminance variance below which a
	// capture is judged flat/broken regardless of banding. Higher = stricter
	// (more captures flagged broken). Luminance is normalized to [0,1], so this
	// is a small number.
	MinVariance float64

	// PixelDelta is the per-cell normalized luminance difference above which a
	// cell counts as changed in Compare. Higher = looser (small shifts ignored);
	// lower = stricter (tiny shifts count). Range [0,1].
	PixelDelta float64

	// ChangedTolerance is the share of changed cells at or below which two
	// captures are considered identical. Higher = looser (more drift tolerated
	// as "identical"); lower = stricter. Range [0,1].
	ChangedTolerance float64
}

// DefaultThresholds returns the tuned defaults. They are intentionally
// conservative: a capture must be almost perfectly uniform to be "broken", and
// only a non-trivial share of the grid must move for two captures to count as
// "changed", so anti-aliasing/font jitter does not produce false signals.
func DefaultThresholds() Thresholds {
	return Thresholds{
		GridSize:         32,
		BlankFraction:    0.98,
		MinVariance:      0.0005,
		PixelDelta:       0.06,
		ChangedTolerance: 0.01,
	}
}

// ThresholdsFromEnv returns DefaultThresholds with any TEST_GENIE_VISUAL_*
// overrides applied. Unset or unparseable variables leave the default in place,
// so a malformed lever degrades to the safe default rather than failing. The
// recognized levers:
//
//	TEST_GENIE_VISUAL_GRID_SIZE         (int    > 0)
//	TEST_GENIE_VISUAL_BLANK_FRACTION    (float (0,1])
//	TEST_GENIE_VISUAL_MIN_VARIANCE      (float >= 0)
//	TEST_GENIE_VISUAL_PIXEL_DELTA       (float [0,1])
//	TEST_GENIE_VISUAL_CHANGED_TOLERANCE (float [0,1])
func ThresholdsFromEnv() Thresholds {
	t := DefaultThresholds()
	if v, ok := envInt("TEST_GENIE_VISUAL_GRID_SIZE"); ok && v > 0 {
		t.GridSize = v
	}
	if v, ok := envFloat("TEST_GENIE_VISUAL_BLANK_FRACTION"); ok && v > 0 && v <= 1 {
		t.BlankFraction = v
	}
	if v, ok := envFloat("TEST_GENIE_VISUAL_MIN_VARIANCE"); ok && v >= 0 {
		t.MinVariance = v
	}
	if v, ok := envFloat("TEST_GENIE_VISUAL_PIXEL_DELTA"); ok && v >= 0 && v <= 1 {
		t.PixelDelta = v
	}
	if v, ok := envFloat("TEST_GENIE_VISUAL_CHANGED_TOLERANCE"); ok && v >= 0 && v <= 1 {
		t.ChangedTolerance = v
	}
	return t
}

func envInt(key string) (int, bool) {
	raw := os.Getenv(key)
	if raw == "" {
		return 0, false
	}
	n, err := strconv.Atoi(raw)
	if err != nil {
		return 0, false
	}
	return n, true
}

func envFloat(key string) (float64, bool) {
	raw := os.Getenv(key)
	if raw == "" {
		return 0, false
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

// luminanceBands is the number of equal-width luminance buckets used for
// dominant-band (blank/solid) detection. Twelve bands give ~0.083 width, fine
// enough to separate a real UI's tones while still collapsing a flat fill into
// a single band.
const luminanceBands = 12

// Health is the outcome of a single-capture render-health check.
type Health struct {
	// Broken is true when the capture is a blank or solid-color render.
	Broken bool
	// Reason is a short human explanation when Broken; empty otherwise.
	Reason string
	// DominantFraction is the share of grid cells in the most-populated
	// luminance band (1.0 for a perfectly uniform image).
	DominantFraction float64
	// Variance is the population variance of grid-cell luminance in [0,1].
	Variance float64
}

// Delta is the outcome of comparing two captures of the same surface.
type Delta struct {
	// ChangedFraction is the share of grid cells whose luminance moved by more
	// than Thresholds.PixelDelta, in [0,1].
	ChangedFraction float64
	// Identical is true when ChangedFraction is within Thresholds.ChangedTolerance.
	Identical bool
}

// RenderHealth decodes a PNG and reports whether it is a clearly broken
// (blank / solid-color) render. It needs no baseline: a flat fill is broken on
// its own terms.
func RenderHealth(png []byte, t Thresholds) (Health, error) {
	grid, err := luminanceGrid(png, t.GridSize)
	if err != nil {
		return Health{}, err
	}

	variance := populationVariance(grid)
	dominant := dominantBandFraction(grid)

	switch {
	case variance <= t.MinVariance:
		return Health{
			Broken:           true,
			Reason:           fmt.Sprintf("luminance variance %.5f ≤ %.5f", variance, t.MinVariance),
			DominantFraction: dominant,
			Variance:         variance,
		}, nil
	case dominant >= t.BlankFraction:
		return Health{
			Broken:           true,
			Reason:           fmt.Sprintf("%.0f%% of the frame is a single tone", dominant*100),
			DominantFraction: dominant,
			Variance:         variance,
		}, nil
	default:
		return Health{DominantFraction: dominant, Variance: variance}, nil
	}
}

// Compare decodes two PNGs and reports the magnitude of visual change between
// them. Both are downscaled to the same grid, so captures of differing pixel
// dimensions compare cleanly. The result is a neutral magnitude — Compare never
// decides pass/fail.
func Compare(base, cur []byte, t Thresholds) (Delta, error) {
	baseGrid, err := luminanceGrid(base, t.GridSize)
	if err != nil {
		return Delta{}, fmt.Errorf("baseline: %w", err)
	}
	curGrid, err := luminanceGrid(cur, t.GridSize)
	if err != nil {
		return Delta{}, fmt.Errorf("current: %w", err)
	}
	if len(baseGrid) != len(curGrid) {
		// Different effective grids (one image was smaller than GridSize).
		// Re-derive both at the smaller common edge so they align.
		edge := minInt(gridEdge(len(baseGrid)), gridEdge(len(curGrid)))
		if baseGrid, err = luminanceGrid(base, edge); err != nil {
			return Delta{}, fmt.Errorf("baseline realign: %w", err)
		}
		if curGrid, err = luminanceGrid(cur, edge); err != nil {
			return Delta{}, fmt.Errorf("current realign: %w", err)
		}
	}

	changed := 0
	for i := range baseGrid {
		if math.Abs(baseGrid[i]-curGrid[i]) > t.PixelDelta {
			changed++
		}
	}
	frac := 0.0
	if len(baseGrid) > 0 {
		frac = float64(changed) / float64(len(baseGrid))
	}
	return Delta{ChangedFraction: frac, Identical: frac <= t.ChangedTolerance}, nil
}

// luminanceGrid decodes a PNG and downscales it to a gridSize×gridSize grid of
// mean normalized luminance, returned row-major. gridSize is clamped to the
// image's smaller dimension so tiny images still produce a (smaller) square
// grid.
func luminanceGrid(png []byte, gridSize int) ([]float64, error) {
	if gridSize <= 0 {
		gridSize = DefaultThresholds().GridSize
	}
	img, _, err := image.Decode(bytes.NewReader(png))
	if err != nil {
		return nil, fmt.Errorf("decode png: %w", err)
	}
	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 {
		return nil, fmt.Errorf("empty image bounds %dx%d", w, h)
	}
	edge := gridSize
	if w < edge {
		edge = w
	}
	if h < edge {
		edge = h
	}

	sum := make([]float64, edge*edge)
	count := make([]int, edge*edge)
	for y := 0; y < h; y++ {
		cy := y * edge / h
		for x := 0; x < w; x++ {
			cx := x * edge / w
			r, g, bl, _ := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			// Rec. 601 luma, normalized to [0,1] (RGBA returns 16-bit channels).
			lum := (0.299*float64(r) + 0.587*float64(g) + 0.114*float64(bl)) / 65535.0
			idx := cy*edge + cx
			sum[idx] += lum
			count[idx]++
		}
	}
	grid := make([]float64, edge*edge)
	for i := range grid {
		if count[i] > 0 {
			grid[i] = sum[i] / float64(count[i])
		}
	}
	return grid, nil
}

// populationVariance returns the population variance of the grid luminances.
func populationVariance(grid []float64) float64 {
	if len(grid) == 0 {
		return 0
	}
	var mean float64
	for _, v := range grid {
		mean += v
	}
	mean /= float64(len(grid))
	var sq float64
	for _, v := range grid {
		d := v - mean
		sq += d * d
	}
	return sq / float64(len(grid))
}

// dominantBandFraction buckets grid luminance into luminanceBands equal-width
// bands and returns the share held by the most-populated band.
func dominantBandFraction(grid []float64) float64 {
	if len(grid) == 0 {
		return 0
	}
	var bands [luminanceBands]int
	for _, v := range grid {
		band := int(v * luminanceBands)
		if band >= luminanceBands {
			band = luminanceBands - 1
		}
		if band < 0 {
			band = 0
		}
		bands[band]++
	}
	maxBand := 0
	for _, n := range bands {
		if n > maxBand {
			maxBand = n
		}
	}
	return float64(maxBand) / float64(len(grid))
}

// gridEdge returns the edge length of a square grid given its cell count.
func gridEdge(cells int) int { return int(math.Round(math.Sqrt(float64(cells)))) }

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
