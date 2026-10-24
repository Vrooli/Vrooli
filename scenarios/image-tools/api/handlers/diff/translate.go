package diff

import (
	internaldiff "image-tools/internal/diff"

	diffv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/diff"
)

// paramsFromProto converts a DiffParams message to internal Params.
func paramsFromProto(p *diffv1.DiffParams) internaldiff.Params {
	if p == nil {
		return internaldiff.Params{Mode: internaldiff.ModePixel, IncludeHeatmap: true}
	}
	return internaldiff.Params{
		Mode:           modeFromProto(p.GetMode()),
		Tolerance:      p.GetTolerance(),
		IncludeHeatmap: p.GetIncludeHeatmap(),
		HighlightHex:   p.GetHighlightHex(),
	}
}

// modeFromProto maps the proto DiffMode to the internal Mode, defaulting an
// unspecified mode to pixel.
func modeFromProto(m diffv1.DiffMode) internaldiff.Mode {
	switch m {
	case diffv1.DiffMode_DIFF_MODE_PERCEPTUAL:
		return internaldiff.ModePerceptual
	case diffv1.DiffMode_DIFF_MODE_PIXEL:
		return internaldiff.ModePixel
	default:
		return internaldiff.ModePixel
	}
}

// resultToProto maps the internal comparison Result to the proto DiffResult.
func resultToProto(jobID, heatmapRef string, r internaldiff.Result) *diffv1.DiffResult {
	return &diffv1.DiffResult{
		JobId:           jobID,
		Verdict:         r.Verdict,
		DimensionsMatch: r.DimensionsMatch,
		BaseWidth:       int32(r.BaseWidth),
		BaseHeight:      int32(r.BaseHeight),
		CompareWidth:    int32(r.CompareWidth),
		CompareHeight:   int32(r.CompareHeight),
		ChangedPixels:   r.ChangedPixels,
		TotalPixels:     r.TotalPixels,
		ChangedFraction: r.ChangedFraction,
		Mae:             r.MAE,
		Rmse:            r.RMSE,
		Psnr:            r.PSNR,
		PhashDistance:   int32(r.PhashDistance),
		PhashSimilarity: r.PhashSimilarity,
		Ssim:            r.SSIM,
		HeatmapRef:      heatmapRef,
		Warnings:        r.Warnings,
	}
}
