package opparams

import (
	"encoding/json"
	"testing"
)

// The decoders are generated, so these tests assert the contract the generator
// promises rather than re-listing every field: declared keys become typed
// values, absent keys stay nil (so operation defaults apply), nested JSON keeps
// its shape, and a malformed persisted value fails closed instead of being
// silently dropped.

func TestDecodeInpaintParams(t *testing.T) {
	got, err := DecodeInpaintParams(map[string]string{
		"inpaint_crop_to_mask":             "true",
		"inpaint_context_pixels":           "96",
		"inpaint_neutralize_masked_pixels": "false",
		"prompt":                           "ignored: shared field, not in this schema",
	})
	if err != nil {
		t.Fatalf("DecodeInpaintParams: %v", err)
	}
	if got.CropToMask == nil || !*got.CropToMask {
		t.Errorf("inpaint_crop_to_mask = %v, want true", got.CropToMask)
	}
	if got.ContextPixels == nil || *got.ContextPixels != 96 {
		t.Errorf("inpaint_context_pixels = %v, want 96", got.ContextPixels)
	}
	if got.NeutralizeMaskedPixels == nil || *got.NeutralizeMaskedPixels {
		t.Errorf("inpaint_neutralize_masked_pixels = %v, want false", got.NeutralizeMaskedPixels)
	}
	if got.MaxEdgePx != nil {
		t.Errorf("absent inpaint_max_edge_px = %v, want nil", *got.MaxEdgePx)
	}
}

func TestDecodeInpaintParamsRejectsMalformedValue(t *testing.T) {
	if _, err := DecodeInpaintParams(map[string]string{"inpaint_context_pixels": "wide"}); err == nil {
		t.Fatal("expected a malformed persisted value to fail closed")
	}
}

func TestDecodeObjectRemovalParamsPreservesNestedJSON(t *testing.T) {
	raw := `{"maskExpansionPixels":8,"removalRegions":[{"x":0.1,"y":0.2,"width":0.3,"height":0.4}]}`
	got, err := DecodeObjectRemovalParams(map[string]string{
		"grow_px":           "4",
		"halo_threshold":    "12",
		"background_repair": raw,
	})
	if err != nil {
		t.Fatalf("DecodeObjectRemovalParams: %v", err)
	}
	if got.BackgroundRepair == nil {
		t.Fatal("background_repair did not decode")
	}
	if got.BackgroundRepair.MaskExpansionPixels == nil || *got.BackgroundRepair.MaskExpansionPixels != 8 {
		t.Errorf("maskExpansionPixels = %v, want 8", got.BackgroundRepair.MaskExpansionPixels)
	}
	if len(got.BackgroundRepair.RemovalRegions) != 1 {
		t.Fatalf("removalRegions length = %d, want 1", len(got.BackgroundRepair.RemovalRegions))
	}
	region := got.BackgroundRepair.RemovalRegions[0]
	if region.X != 0.1 || region.Y != 0.2 || region.Width != 0.3 || region.Height != 0.4 {
		t.Errorf("region = %+v, want the declared rectangle", region)
	}
	if got.GrowPx == nil || *got.GrowPx != 4 {
		t.Errorf("grow_px = %v, want 4", got.GrowPx)
	}
	if got.HaloThreshold == nil || *got.HaloThreshold != 12 {
		t.Errorf("halo_threshold = %v, want 12", got.HaloThreshold)
	}
}

func TestDecodeSegmentParamsRoundTripsPrompts(t *testing.T) {
	got, err := DecodeSegmentParams(map[string]string{
		"segmentation": `{"points":[{"x":0.4,"y":0.6},{"x":0.1,"y":0.2,"negative":true}],"box":{"x":0.1,"y":0.1,"width":0.5,"height":0.5}}`,
	})
	if err != nil {
		t.Fatalf("DecodeSegmentParams: %v", err)
	}
	if got.Segmentation == nil || len(got.Segmentation.Points) != 2 {
		t.Fatalf("points did not decode: %+v", got.Segmentation)
	}
	if got.Segmentation.Points[1].Negative == nil || !*got.Segmentation.Points[1].Negative {
		t.Errorf("second point negative = %v, want true", got.Segmentation.Points[1].Negative)
	}
	if got.Segmentation.Box == nil || got.Segmentation.Box.Width != 0.5 {
		t.Errorf("box did not decode: %+v", got.Segmentation.Box)
	}
	roundTrip, err := json.Marshal(got.Segmentation)
	if err != nil {
		t.Fatalf("marshal decoded prompts: %v", err)
	}
	if !json.Valid(roundTrip) {
		t.Fatalf("re-encoded prompts are not valid JSON: %s", roundTrip)
	}
}

func TestDecodeImageToObjectParamsDefault(t *testing.T) {
	got, err := DecodeImageToObjectParams(nil)
	if err != nil {
		t.Fatalf("DecodeImageToObjectParams(nil): %v", err)
	}
	if got.MeshResolution != nil {
		t.Errorf("absent mesh_resolution = %v, want nil so the caller default applies", *got.MeshResolution)
	}
	got, err = DecodeImageToObjectParams(map[string]string{"mesh_resolution": "128"})
	if err != nil || got.MeshResolution == nil || *got.MeshResolution != 128 {
		t.Fatalf("mesh_resolution = %+v, err = %v, want 128", got.MeshResolution, err)
	}
}
