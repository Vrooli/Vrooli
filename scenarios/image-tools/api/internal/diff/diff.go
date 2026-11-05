// Package diff is image-tools' general-purpose image diff / visual-comparison
// engine (IMG-P1-009). It compares two images and reports what changed both
// numerically (pixel + perceptual metrics) and visually (a heat-map overlay),
// pure-Go with no model — the headless-completeness floor, so it runs on any
// host. It is deliberately a first-class capability so test-genie's future
// visual-diff phase (and branding-manager / AI-UGC) can adopt it without
// re-implementing image comparison.
//
// Two metric families are always computed; the requested Mode selects which one
// drives the headline verdict:
//   - PIXEL — exact per-channel difference with a tolerance band → changed-pixel
//     fraction + MAE / RMSE / PSNR. Sensitive to any change.
//   - PERCEPTUAL — a DCT-based perceptual hash on both images + their Hamming
//     distance, plus a structural-similarity estimate. Robust to re-encode.
//
// Mismatched dimensions are handled honestly: the compare image is resized to
// the base for the pixel metrics and a warning is attached (the perceptual hash
// is resolution-independent by construction).
package diff

import (
	"bytes"
	"fmt"
	"image"

	"github.com/disintegration/imaging"
)

// Mode selects which comparison verdict drives the headline result.
type Mode string

const (
	// ModePixel — exact per-channel difference with a tolerance band.
	ModePixel Mode = "pixel"
	// ModePerceptual — perceptual-hash distance + structural similarity.
	ModePerceptual Mode = "perceptual"
)

// ModeInfo describes one comparison mode for discovery.
type ModeInfo struct {
	Name    string
	Summary string
}

// Modes returns the comparison-mode catalog in stable order (discovery).
func Modes() []ModeInfo {
	return []ModeInfo{
		{Name: string(ModePixel), Summary: "Exact per-channel pixel difference with a tolerance band (byte-level changes)"},
		{Name: string(ModePerceptual), Summary: "Perceptual-hash distance + structural similarity (is it the same picture?)"},
	}
}

// Params controls a comparison.
type Params struct {
	// Mode picks which family drives the verdict (default: pixel).
	Mode Mode
	// Tolerance is the per-channel difference threshold 0..1 (pixel mode).
	Tolerance float64
	// IncludeHeatmap controls heat-map generation.
	IncludeHeatmap bool
	// HighlightHex is the heat-map highlight colour (#rrggbb); empty = default.
	HighlightHex string
}

// Result is the full outcome of a comparison: every metric plus the heat-map
// PNG bytes (nil when not requested). Handlers store the heat-map and translate
// the rest to the proto DiffResult.
type Result struct {
	Verdict         string
	DimensionsMatch bool
	BaseWidth       int
	BaseHeight      int
	CompareWidth    int
	CompareHeight   int

	ChangedPixels   int64
	TotalPixels     int64
	ChangedFraction float64
	MAE             float64
	RMSE            float64
	PSNR            float64

	PhashDistance   int
	PhashSimilarity float64
	SSIM            float64

	HeatmapPNG []byte
	Warnings   []string
}

// psnrIdentical is the sentinel reported when the two images are byte-identical
// (true PSNR is +Inf, which doesn't survive protojson).
const psnrIdentical = 99.0

// Compare runs the full comparison of base vs compare under params, returning
// every metric (and, when requested, the heat-map PNG). It never errors on a
// dimension mismatch — it resizes and warns.
func Compare(baseData, compareData []byte, params Params) (Result, error) {
	base, err := decode(baseData)
	if err != nil {
		return Result{}, fmt.Errorf("decode base image: %w", err)
	}
	cmp, err := decode(compareData)
	if err != nil {
		return Result{}, fmt.Errorf("decode compare image: %w", err)
	}

	bb := base.Bounds()
	cb := cmp.Bounds()
	res := Result{
		BaseWidth:       bb.Dx(),
		BaseHeight:      bb.Dy(),
		CompareWidth:    cb.Dx(),
		CompareHeight:   cb.Dy(),
		DimensionsMatch: bb.Dx() == cb.Dx() && bb.Dy() == cb.Dy(),
	}

	// Pixel metrics run at the base resolution; resize the compare if needed.
	cmpAligned := cmp
	if !res.DimensionsMatch {
		cmpAligned = imaging.Resize(cmp, bb.Dx(), bb.Dy(), imaging.Lanczos)
		res.Warnings = append(res.Warnings, fmt.Sprintf(
			"images differ in size (base %dx%d, compare %dx%d) — compare resized to base for pixel metrics",
			bb.Dx(), bb.Dy(), cb.Dx(), cb.Dy()))
	}

	baseRGBA := imaging.Clone(base)
	cmpRGBA := imaging.Clone(cmpAligned)

	pm := pixelMetrics(baseRGBA, cmpRGBA, clamp01(params.Tolerance))
	res.ChangedPixels = pm.changed
	res.TotalPixels = pm.total
	res.ChangedFraction = pm.changedFraction
	res.MAE = pm.mae
	res.RMSE = pm.rmse
	res.PSNR = pm.psnr

	res.PhashDistance = hammingDistance(perceptualHash(base), perceptualHash(cmp))
	res.PhashSimilarity = 1 - float64(res.PhashDistance)/64
	res.SSIM = ssim(baseRGBA, cmpRGBA)

	res.Verdict = verdict(params.Mode, res)

	if params.IncludeHeatmap {
		hm, herr := heatmap(baseRGBA, cmpRGBA, clamp01(params.Tolerance), params.HighlightHex)
		if herr != nil {
			res.Warnings = append(res.Warnings, "heat-map generation failed: "+herr.Error())
		} else {
			res.HeatmapPNG = hm
		}
	}
	return res, nil
}

// verdict reduces the metric set to a headline string under the chosen mode.
func verdict(mode Mode, r Result) string {
	if mode == ModePerceptual {
		switch {
		case r.PhashDistance == 0:
			return "identical"
		case r.PhashDistance <= 6:
			return "similar"
		default:
			return "different"
		}
	}
	// Pixel mode (default).
	switch {
	case r.ChangedPixels == 0:
		return "identical"
	case r.ChangedFraction <= 0.02:
		return "similar"
	default:
		return "different"
	}
}

func decode(data []byte) (image.Image, error) {
	return imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
}

func clamp01(v float64) float64 {
	switch {
	case v < 0:
		return 0
	case v > 1:
		return 1
	default:
		return v
	}
}
