package ai

import (
	"encoding/json"
	"strings"
	"testing"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

// The operation-specific parameter contracts now live in the capability
// manifests and are enforced by internal/paramschema. These tests exercise the
// schema path end to end: validation at the edge and threading into the backend
// request map (whose keys the Python sidecars consume).

func TestImageToObjectResolutionSchema(t *testing.T) {
	for _, resolution := range []float64{128, 256} {
		params := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{"mesh_resolution": resolution})}
		if err := validateOpParams("image_to_object", params); err != nil {
			t.Errorf("resolution %v rejected: %v", resolution, err)
		}
	}
	bad := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{"mesh_resolution": float64(512)})}
	if err := validateOpParams("image_to_object", bad); err == nil {
		t.Fatal("unbounded mesh resolution accepted")
	}
	if err := validateOpParams("depth_map", bad); err == nil {
		t.Fatal("image_to_object params accepted for an unrelated operation")
	}
}

func TestInpaintCropSchema(t *testing.T) {
	valid := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"inpaint_crop_to_mask":             true,
		"inpaint_context_pixels":           float64(96),
		"inpaint_max_edge_px":              float64(1024),
		"inpaint_neutralize_masked_pixels": true,
	})}
	if err := validateOpParams("inpaint", valid); err != nil {
		t.Fatalf("valid crop controls rejected: %v", err)
	}
	got := paramsMap(valid)
	want := map[string]string{
		"inpaint_crop_to_mask":             "true",
		"inpaint_context_pixels":           "96",
		"inpaint_max_edge_px":              "1024",
		"inpaint_neutralize_masked_pixels": "true",
	}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("%s = %q, want %q; map=%#v", key, got[key], value, got)
		}
	}
	missing := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{"inpaint_context_pixels": float64(96)})}
	if err := validateOpParams("inpaint", missing); err == nil {
		t.Fatal("crop control without crop_to_mask accepted")
	}
	tooMuch := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"inpaint_crop_to_mask":   true,
		"inpaint_context_pixels": float64(600),
	})}
	if err := validateOpParams("inpaint", tooMuch); err == nil {
		t.Fatal("out-of-range context pixels accepted")
	}
}

func TestSegmentationTextPhrase(t *testing.T) {
	text := func(prompt map[string]any) *aiv1.AIParams {
		return &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{"segmentation": prompt})}
	}
	phrase := text(map[string]any{"text": "the tan suede jacket"})
	if err := validateOpParams("segment", phrase); err != nil {
		t.Fatalf("text phrase rejected by schema: %v", err)
	}
	if err := validateSegmentationSemantics("segment", phrase); err != nil {
		t.Fatalf("text phrase rejected by semantics: %v", err)
	}
	if raw := paramsMap(phrase)["segmentation"]; !strings.Contains(raw, "tan suede jacket") {
		t.Fatalf("phrase not threaded to the backend params: %q", raw)
	}
	mixed := text(map[string]any{"text": "the jacket", "box": map[string]any{"x": 0.1, "y": 0.1, "width": 0.5, "height": 0.5}})
	if err := validateSegmentationSemantics("segment", mixed); err == nil {
		t.Fatal("a phrase combined with a box was accepted; the grounded model chooses its own box")
	}
	for name, bad := range map[string]map[string]any{
		"empty":    {"text": ""},
		"too-long": {"text": strings.Repeat("a", 201)},
	} {
		if err := validateOpParams("segment", text(bad)); err == nil {
			t.Fatalf("%s phrase accepted", name)
		}
	}
}

func TestSegmentationSchemaAndSemantics(t *testing.T) {
	valid := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"segmentation": map[string]any{
			"points": []any{
				map[string]any{"x": 0.4, "y": 0.4},
				map[string]any{"x": 0.7, "y": 0.6, "negative": true},
			},
			"box": map[string]any{"x": 0.1, "y": 0.1, "width": 0.6, "height": 0.5},
		},
	})}
	if err := validateOpParams("segment", valid); err != nil {
		t.Fatalf("valid segmentation rejected: %v", err)
	}
	if err := validateSegmentationSemantics("segment", valid); err != nil {
		t.Fatalf("valid segmentation failed semantics: %v", err)
	}
	raw := paramsMap(valid)["segmentation"]
	if raw == "" {
		t.Fatal("segmentation not threaded to params")
	}
	var parsed map[string]any
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatalf("threaded segmentation is not JSON: %v", err)
	}
	if _, ok := parsed["box"]; !ok {
		t.Fatalf("threaded segmentation lost box: %s", raw)
	}

	// Relational rule: only negative points, no box → rejected.
	negativeOnly := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"segmentation": map[string]any{
			"points": []any{map[string]any{"x": 0.4, "y": 0.4, "negative": true}},
		},
	})}
	if err := validateSegmentationSemantics("segment", negativeOnly); err == nil {
		t.Fatal("all-negative prompt accepted")
	}

	// Range rule (schema).
	outOfRange := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"segmentation": map[string]any{"points": []any{map[string]any{"x": 1.4, "y": 0.4}}},
	})}
	if err := validateOpParams("segment", outOfRange); err == nil {
		t.Fatal("out-of-range point accepted")
	}

	// Unknown key (schema: additionalProperties=false).
	unknown := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"segmentation": map[string]any{"mask": "nope"},
	})}
	if err := validateOpParams("segment", unknown); err == nil {
		t.Fatal("unknown segmentation key accepted")
	}
}

func TestSceneGeometrySupportROISchema(t *testing.T) {
	valid := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"support_roi": map[string]any{"x": 0.0, "y": 0.85, "width": 1.0, "height": 0.15},
	})}
	if err := validateOpParams("scene_geometry", valid); err != nil {
		t.Fatalf("valid support ROI rejected: %v", err)
	}
	var roi map[string]any
	if err := json.Unmarshal([]byte(paramsMap(valid)["support_roi"]), &roi); err != nil {
		t.Fatalf("support ROI not threaded as JSON: %v", err)
	}
	if roi["y"] != 0.85 || roi["height"] != 0.15 {
		t.Fatalf("threaded support ROI = %#v", roi)
	}
	outOfRange := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"support_roi": map[string]any{"x": 0.5, "y": 0.5, "width": 0.6, "height": 0.5},
	})}
	// Each field is in 0..1, so only the relational rule can reject this; it is
	// the schema's `withinFrame`, not a Go function.
	if err := validateOpParams("scene_geometry", outOfRange); err == nil {
		t.Fatal("support ROI extending outside the frame accepted")
	}
	missing := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"support_roi": map[string]any{"x": 0.0, "y": 0.0, "width": 1.0},
	})}
	if err := validateOpParams("scene_geometry", missing); err == nil {
		t.Fatal("support ROI missing height accepted")
	}
}

func TestBackgroundRepairSchemaAndSemantics(t *testing.T) {
	valid := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"background_repair": map[string]any{
			"maskExpansionPixels": float64(12),
			"removalRegions": []any{
				map[string]any{"x": 0.1, "y": 0.8, "width": 0.2, "height": 0.1},
			},
		},
	})}
	if err := validateOpParams("object_removal", valid); err != nil {
		t.Fatalf("valid repair controls rejected: %v", err)
	}
	var repair map[string]any
	if err := json.Unmarshal([]byte(paramsMap(valid)["background_repair"]), &repair); err != nil {
		t.Fatalf("repair controls not threaded as JSON: %v", err)
	}
	if repair["maskExpansionPixels"] != 12.0 {
		t.Fatalf("threaded repair = %#v", repair)
	}

	tooLarge := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"background_repair": map[string]any{"maskExpansionPixels": float64(65)},
	})}
	if err := validateOpParams("object_removal", tooLarge); err == nil {
		t.Fatal("unbounded mask expansion accepted")
	}

	outOfFrame := &aiv1.AIParams{OpParams: opParamsStruct(t, map[string]any{
		"background_repair": map[string]any{
			"removalRegions": []any{
				map[string]any{"x": 0.8, "y": 0.8, "width": 0.3, "height": 0.3},
			},
		},
	})}
	if err := validateOpParams("object_removal", outOfFrame); err == nil {
		t.Fatal("out-of-frame removal region accepted")
	}
}
