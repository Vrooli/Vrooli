package sceneprep

import (
	"fmt"
	"image"
)

// RepairAssessment describes visual-risk signals for a proposed inpaint. It is
// advisory metadata: it does not qualify the clean plate or gate preparation.
type RepairAssessment struct {
	Width                  uint32      `json:"width"`
	Height                 uint32      `json:"height"`
	MaskAreaPixels         uint64      `json:"mask_area_pixels"`
	MaskAreaFraction       float64     `json:"mask_area_fraction"`
	MaskBounds             PixelBounds `json:"mask_bounds_xyxy"`
	MaskBoundsFraction     float64     `json:"mask_bounds_fraction"`
	ContextPixels          uint32      `json:"context_pixels"`
	ContextShortfallPixels ContextGap  `json:"context_shortfall_pixels"`
	ContextClippedSides    []string    `json:"context_clipped_sides,omitempty"`
	SelectedRegionWidth    uint32      `json:"selected_region_width"`
	SelectedRegionHeight   uint32      `json:"selected_region_height"`
	InferenceScale         float64     `json:"inference_scale,omitempty"`
	InferenceScaleKnown    bool        `json:"inference_scale_known"`
	RiskLevel              string      `json:"risk_level"`
	RiskSignals            []string    `json:"risk_signals,omitempty"`
}

// PixelBounds uses half-open coordinates relative to the mask raster.
type PixelBounds struct {
	X0 uint32 `json:"x0"`
	Y0 uint32 `json:"y0"`
	X1 uint32 `json:"x1"`
	Y1 uint32 `json:"y1"`
}

// ContextGap reports how many requested context pixels cannot be supplied on
// each side because the mask lies near the source-raster boundary.
type ContextGap struct {
	Left   uint32 `json:"left"`
	Top    uint32 `json:"top"`
	Right  uint32 `json:"right"`
	Bottom uint32 `json:"bottom"`
}

// RepairAssessmentOptions mirrors the explicit inpaint crop controls. A zero
// MaxEdgePx uses the model's native edge limit and is not inferred here.
type RepairAssessmentOptions struct {
	CropToMask    bool
	ContextPixels uint32
	MaxEdgePx     uint32
}

// AssessRepairMask reports how much source context an inpaint request will
// have. The mask must be an opaque binary raster. The result contains no
// pass/fail or inferred quality claim; the reviewer decides whether the
// coverage and available context suit the photograph.
//
// The 35% and 60% coverage markers are review heuristics only. They make broad
// reconstruction areas visible to the reviewer; they are not model limits or
// predictions of the generated pixels.
func AssessRepairMask(mask image.Image, options RepairAssessmentOptions) (RepairAssessment, error) {
	if mask == nil {
		return RepairAssessment{}, fmt.Errorf("repair mask is required")
	}
	if options.ContextPixels > 512 || (options.ContextPixels > 0 && !options.CropToMask) {
		return RepairAssessment{}, fmt.Errorf("context requires mask cropping and must be 0..512 pixels")
	}
	if !options.CropToMask && options.MaxEdgePx != 0 {
		return RepairAssessment{}, fmt.Errorf("max edge requires mask cropping")
	}
	if options.MaxEdgePx != 0 && options.MaxEdgePx != 512 && options.MaxEdgePx != 1024 {
		return RepairAssessment{}, fmt.Errorf("max edge must be 0, 512, or 1024 pixels")
	}
	b := mask.Bounds()
	w, h := b.Dx(), b.Dy()
	if w <= 0 || h <= 0 || uint64(w)*uint64(h) > 16_000_000 {
		return RepairAssessment{}, fmt.Errorf("repair mask must contain 1..16000000 pixels")
	}
	var count uint64
	x0, y0, x1, y1 := w, h, 0, 0
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			r, g, bl, a := mask.At(b.Min.X+x, b.Min.Y+y).RGBA()
			if a != 0xffff || r != g || g != bl || (r != 0 && r != 0xffff) {
				return RepairAssessment{}, fmt.Errorf("repair mask must be opaque binary black and white")
			}
			if r == 0 {
				continue
			}
			count++
			if x < x0 {
				x0 = x
			}
			if y < y0 {
				y0 = y
			}
			if x+1 > x1 {
				x1 = x + 1
			}
			if y+1 > y1 {
				y1 = y + 1
			}
		}
	}
	if count == 0 {
		return RepairAssessment{}, fmt.Errorf("repair mask is empty")
	}
	total := uint64(w) * uint64(h)
	boxArea := uint64(x1-x0) * uint64(y1-y0)
	out := RepairAssessment{
		Width: uint32(w), Height: uint32(h), MaskAreaPixels: count,
		MaskAreaFraction:   float64(count) / float64(total),
		MaskBounds:         PixelBounds{X0: uint32(x0), Y0: uint32(y0), X1: uint32(x1), Y1: uint32(y1)},
		MaskBoundsFraction: float64(boxArea) / float64(total), ContextPixels: options.ContextPixels,
		RiskLevel: "low",
	}
	if options.ContextPixels > 0 {
		c := int(options.ContextPixels)
		out.ContextShortfallPixels = ContextGap{
			Left: uint32(max(0, c-x0)), Top: uint32(max(0, c-y0)),
			Right: uint32(max(0, c-(w-x1))), Bottom: uint32(max(0, c-(h-y1))),
		}
		if out.ContextShortfallPixels.Left > 0 {
			out.ContextClippedSides = append(out.ContextClippedSides, "left")
		}
		if out.ContextShortfallPixels.Top > 0 {
			out.ContextClippedSides = append(out.ContextClippedSides, "top")
		}
		if out.ContextShortfallPixels.Right > 0 {
			out.ContextClippedSides = append(out.ContextClippedSides, "right")
		}
		if out.ContextShortfallPixels.Bottom > 0 {
			out.ContextClippedSides = append(out.ContextClippedSides, "bottom")
		}
	}
	cropMinX, cropMinY, cropMaxX, cropMaxY := 0, 0, w, h
	if options.CropToMask {
		context := int(options.ContextPixels)
		cropMinX, cropMinY = max(0, x0-context), max(0, y0-context)
		cropMaxX, cropMaxY = min(w, x1+context), min(h, y1+context)
	}
	cropW, cropH := cropMaxX-cropMinX, cropMaxY-cropMinY
	out.SelectedRegionWidth, out.SelectedRegionHeight = uint32(cropW), uint32(cropH)
	if options.MaxEdgePx > 0 {
		longEdge := max(cropW, cropH)
		if longEdge > int(options.MaxEdgePx) {
			out.InferenceScale = float64(options.MaxEdgePx) / float64(longEdge)
		} else {
			out.InferenceScale = 1
		}
		out.InferenceScaleKnown = true
	}
	if out.MaskAreaFraction >= .60 {
		out.RiskLevel = "high"
		out.RiskSignals = append(out.RiskSignals, "large_mask_area")
	} else if out.MaskAreaFraction >= .35 {
		out.RiskLevel = "review"
		out.RiskSignals = append(out.RiskSignals, "large_mask_area")
	}
	if len(out.ContextClippedSides) > 0 {
		out.RiskLevel = elevateRepairRisk(out.RiskLevel, "review")
		out.RiskSignals = append(out.RiskSignals, "requested_context_clipped_by_source_boundary")
	}
	if out.InferenceScaleKnown && out.InferenceScale < .75 {
		out.RiskLevel = elevateRepairRisk(out.RiskLevel, "review")
		out.RiskSignals = append(out.RiskSignals, "large_crop_downscaled_for_inference")
	}
	return out, nil
}

func elevateRepairRisk(current, next string) string {
	order := map[string]int{"low": 0, "review": 1, "high": 2}
	if order[next] > order[current] {
		return next
	}
	return current
}
