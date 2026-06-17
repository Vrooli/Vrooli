package analysis

import (
	internalanalysis "image-tools/internal/analysis"

	analysisv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/analysis"
)

func probeToProto(r internalanalysis.ProbeResult) *analysisv1.ProbeResult {
	out := &analysisv1.ProbeResult{
		Width:       int32(r.Width),
		Height:      int32(r.Height),
		Format:      r.Format,
		ColorModel:  r.ColorModel,
		HasAlpha:    r.HasAlpha,
		FrameCount:  int32(r.FrameCount),
		Megapixels:  r.Megapixels,
		SizeBytes:   r.SizeBytes,
		HasExif:     r.HasEXIF,
		HasGps:      r.HasGPS,
		Orientation: int32(r.Orientation),
	}
	for _, c := range r.DominantColors {
		out.DominantColors = append(out.DominantColors, &analysisv1.DominantColor{Hex: c.Hex, Fraction: c.Fraction})
	}
	return out
}

func ocrToProto(r internalanalysis.OCRResult) *analysisv1.OCRResult {
	out := &analysisv1.OCRResult{FullText: r.FullText, Language: r.Language}
	for _, b := range r.Blocks {
		out.Blocks = append(out.Blocks, &analysisv1.OCRBlock{
			Text:       b.Text,
			Confidence: b.Confidence,
			Box:        &analysisv1.BoundingBox{X: int32(b.Box.X), Y: int32(b.Box.Y), Width: int32(b.Box.Width), Height: int32(b.Box.Height)},
		})
	}
	return out
}

func nsfwToProto(r internalanalysis.NSFWResult) *analysisv1.NSFWResult {
	out := &analysisv1.NSFWResult{
		Nsfw:      r.NSFW,
		Score:     r.Score,
		Label:     r.Label,
		Threshold: r.Threshold,
	}
	for _, c := range r.Categories {
		out.Categories = append(out.Categories, &analysisv1.NSFWCategory{Label: c.Label, Score: c.Score})
	}
	return out
}
