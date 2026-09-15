package pixel

import (
	"bytes"
	"fmt"
	"image"
	_ "image/png"
	"math"
	"os"
	"strconv"
)

type Thresholds struct {
	GridSize         int
	BlankFraction    float64
	MinVariance      float64
	PixelDelta       float64
	ChangedTolerance float64
}

func DefaultThresholds() Thresholds {
	return Thresholds{
		GridSize:         32,
		BlankFraction:    0.98,
		MinVariance:      0.0005,
		PixelDelta:       0.06,
		ChangedTolerance: 0.01,
	}
}

func ThresholdsFromEnv() Thresholds {
	t := DefaultThresholds()
	if v, ok := envInt("UI_HEALTH_VISUAL_GRID_SIZE"); ok && v > 0 {
		t.GridSize = v
	}
	if v, ok := envFloat("UI_HEALTH_VISUAL_BLANK_FRACTION"); ok && v > 0 && v <= 1 {
		t.BlankFraction = v
	}
	if v, ok := envFloat("UI_HEALTH_VISUAL_MIN_VARIANCE"); ok && v >= 0 {
		t.MinVariance = v
	}
	if v, ok := envFloat("UI_HEALTH_VISUAL_PIXEL_DELTA"); ok && v >= 0 && v <= 1 {
		t.PixelDelta = v
	}
	if v, ok := envFloat("UI_HEALTH_VISUAL_CHANGED_TOLERANCE"); ok && v >= 0 && v <= 1 {
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

const luminanceBands = 12

type Health struct {
	Broken           bool
	Reason           string
	DominantFraction float64
	Variance         float64
}

type Delta struct {
	ChangedFraction float64
	Identical       bool
}

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
			Reason:           fmt.Sprintf("luminance variance %.5f <= %.5f", variance, t.MinVariance),
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

func gridEdge(cells int) int { return int(math.Round(math.Sqrt(float64(cells)))) }

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
