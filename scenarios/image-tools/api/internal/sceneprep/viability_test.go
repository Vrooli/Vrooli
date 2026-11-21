package sceneprep

import (
	"image"
	"image/color"
	"reflect"
	"testing"
)

func binaryMask(bounds image.Rectangle, white image.Rectangle) *image.Gray {
	m := image.NewGray(bounds)
	for y := white.Min.Y; y < white.Max.Y; y++ {
		for x := white.Min.X; x < white.Max.X; x++ {
			m.SetGray(x, y, color.Gray{Y: 255})
		}
	}
	return m
}

func TestAssessRepairMaskReportsSmallCenteredRegionWithoutQualificationClaim(t *testing.T) {
	mask := binaryMask(image.Rect(0, 0, 100, 100), image.Rect(30, 35, 70, 75))
	got, err := AssessRepairMask(mask, RepairAssessmentOptions{CropToMask: true, ContextPixels: 12, MaxEdgePx: 512})
	if err != nil {
		t.Fatal(err)
	}
	if got.MaskAreaPixels != 1600 || got.MaskAreaFraction != .16 || got.MaskBounds != (PixelBounds{X0: 30, Y0: 35, X1: 70, Y1: 75}) {
		t.Fatalf("unexpected mask coverage: %+v", got)
	}
	if got.RiskLevel != "low" || len(got.RiskSignals) != 0 || len(got.ContextClippedSides) != 0 || got.InferenceScale != 1 {
		t.Fatalf("small centered mask should have no context-risk signal: %+v", got)
	}
}

func TestAssessRepairMaskReportsLargeBoundaryCropAsReviewRisk(t *testing.T) {
	mask := binaryMask(image.Rect(0, 0, 40, 20), image.Rect(0, 0, 40, 20))
	got, err := AssessRepairMask(mask, RepairAssessmentOptions{CropToMask: true, ContextPixels: 8, MaxEdgePx: 512})
	if err != nil {
		t.Fatal(err)
	}
	if got.MaskAreaFraction != 1 || got.MaskBoundsFraction != 1 || got.RiskLevel != "high" {
		t.Fatalf("full-frame removal should be flagged for review, not rejected: %+v", got)
	}
	if !reflect.DeepEqual(got.ContextClippedSides, []string{"left", "top", "right", "bottom"}) || got.ContextShortfallPixels != (ContextGap{Left: 8, Top: 8, Right: 8, Bottom: 8}) {
		t.Fatalf("boundary context shortfall not reported: %+v", got)
	}
	if got.InferenceScale != 1 || len(got.RiskSignals) != 2 {
		t.Fatalf("unexpected review signals: %+v", got)
	}
}

func TestAssessRepairMaskUsesFullRasterWhenCropIsDisabled(t *testing.T) {
	mask := binaryMask(image.Rect(0, 0, 1200, 800), image.Rect(500, 300, 540, 340))
	got, err := AssessRepairMask(mask, RepairAssessmentOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if got.SelectedRegionWidth != 1200 || got.SelectedRegionHeight != 800 {
		t.Fatalf("full-source path assessed only mask bounds: %+v", got)
	}
	if got.InferenceScaleKnown || got.InferenceScale != 0 {
		t.Fatalf("backend-native scale must remain unknown without an explicit edge: %+v", got)
	}
}

func TestAssessRepairMaskReportsCropDownscaleAndRejectsInvalidInputs(t *testing.T) {
	mask := binaryMask(image.Rect(0, 0, 1200, 800), image.Rect(100, 100, 1100, 700))
	got, err := AssessRepairMask(mask, RepairAssessmentOptions{CropToMask: true, ContextPixels: 24, MaxEdgePx: 512})
	if err != nil {
		t.Fatal(err)
	}
	if got.MaskAreaFraction != .625 || got.InferenceScale >= .75 || got.RiskLevel != "high" {
		t.Fatalf("large downscaled mask not surfaced: %+v", got)
	}
	if len(got.RiskSignals) != 2 || got.RiskSignals[1] != "large_crop_downscaled_for_inference" {
		t.Fatalf("downscale signal missing: %+v", got)
	}
	for name, tc := range map[string]struct {
		mask image.Image
		opts RepairAssessmentOptions
	}{
		"empty":                {binaryMask(image.Rect(0, 0, 10, 10), image.Rectangle{}), RepairAssessmentOptions{}},
		"non-binary":           {grayWithValue(10, 10, 127), RepairAssessmentOptions{}},
		"context-without-crop": {mask, RepairAssessmentOptions{ContextPixels: 2}},
		"edge-without-crop":    {mask, RepairAssessmentOptions{MaxEdgePx: 512}},
		"unsupported-edge":     {mask, RepairAssessmentOptions{MaxEdgePx: 768}},
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := AssessRepairMask(tc.mask, tc.opts); err == nil {
				t.Fatal("invalid diagnostic input accepted")
			}
		})
	}
}

func grayWithValue(w, h int, value uint8) *image.Gray {
	m := image.NewGray(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			m.SetGray(x, y, color.Gray{Y: value})
		}
	}
	return m
}
