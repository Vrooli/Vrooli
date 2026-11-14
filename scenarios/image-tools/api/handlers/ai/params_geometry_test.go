package ai

import (
	"testing"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
	selectionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/selection"
	"google.golang.org/protobuf/encoding/protojson"
)

func TestValidateImageToObjectResolution(t *testing.T) {
	for _, resolution := range []uint32{0, 128, 256} {
		if err := validateImageToObject("image_to_object", &aiv1.AIParams{ImageToObject: &aiv1.ImageToObjectParams{MeshResolution: resolution}}); err != nil {
			t.Errorf("resolution %d rejected: %v", resolution, err)
		}
	}
	if err := validateImageToObject("image_to_object", &aiv1.AIParams{ImageToObject: &aiv1.ImageToObjectParams{MeshResolution: 512}}); err == nil {
		t.Fatal("unbounded resolution accepted")
	}
	if err := validateImageToObject("depth_map", &aiv1.AIParams{ImageToObject: &aiv1.ImageToObjectParams{MeshResolution: 128}}); err == nil {
		t.Fatal("image-to-object params accepted for unrelated operation")
	}
}

func TestParamsMapPreservesExplicitImageToObjectResolution(t *testing.T) {
	got := paramsMap(&aiv1.AIParams{ImageToObject: &aiv1.ImageToObjectParams{MeshResolution: 128}})
	if got["mesh_resolution"] != "128" {
		t.Fatalf("mesh resolution not forwarded: %#v", got)
	}
}

func TestValidateSceneGeometrySupportROI(t *testing.T) {
	valid := &aiv1.AIParams{SceneGeometry: &aiv1.SceneGeometryParams{SupportRoi: &selectionv1.Box{X: 0, Y: 0.85, Width: 1, Height: 0.15}}}
	if err := validateSceneGeometry("scene_geometry", valid); err != nil {
		t.Fatalf("valid support ROI rejected: %v", err)
	}
	if err := validateSceneGeometry("depth_map", valid); err == nil {
		t.Fatal("scene geometry parameters accepted by depth_map")
	}
	invalid := &aiv1.AIParams{SceneGeometry: &aiv1.SceneGeometryParams{SupportRoi: &selectionv1.Box{X: 0.5, Y: 0.5, Width: 0.6, Height: 0.5}}}
	if err := validateSceneGeometry("scene_geometry", invalid); err == nil {
		t.Fatal("support ROI extending outside normalized image coordinates was accepted")
	}
}

func TestParamsMapPreservesTypedSupportROI(t *testing.T) {
	p := &aiv1.AIParams{SceneGeometry: &aiv1.SceneGeometryParams{SupportRoi: &selectionv1.Box{X: 0.1, Y: 0.2, Width: 0.3, Height: 0.4}}}
	got := paramsMap(p)["support_roi"]
	var roi selectionv1.Box
	if got == "" || protojson.Unmarshal([]byte(got), &roi) != nil || roi.GetX() != 0.1 || roi.GetY() != 0.2 || roi.GetWidth() != 0.3 || roi.GetHeight() != 0.4 {
		t.Fatalf("support ROI provider parameter = %q", got)
	}
}

func TestValidateBackgroundRepairAdditions(t *testing.T) {
	valid := &aiv1.AIParams{BackgroundRepair: &aiv1.BackgroundRepairParams{
		MaskExpansionPixels: 12,
		RemovalRegions:      []*selectionv1.Box{{X: .1, Y: .8, Width: .2, Height: .1}},
	}}
	if err := validateBackgroundRepair("object_removal", valid); err != nil {
		t.Fatalf("valid explicit additions rejected: %v", err)
	}
	if err := validateBackgroundRepair("denoise", valid); err == nil {
		t.Fatal("background repair params accepted for unrelated operation")
	}
	if err := validateBackgroundRepair("object_removal", &aiv1.AIParams{BackgroundRepair: &aiv1.BackgroundRepairParams{MaskExpansionPixels: 65}}); err == nil {
		t.Fatal("unbounded mask expansion accepted")
	}
	if err := validateBackgroundRepair("object_removal", &aiv1.AIParams{BackgroundRepair: &aiv1.BackgroundRepairParams{RemovalRegions: []*selectionv1.Box{{X: .8, Y: .8, Width: .3, Height: .3}}}}); err == nil {
		t.Fatal("out-of-frame removal region accepted")
	}
}

func TestParamsMapPreservesTypedBackgroundRepair(t *testing.T) {
	p := &aiv1.AIParams{BackgroundRepair: &aiv1.BackgroundRepairParams{MaskExpansionPixels: 6, RemovalRegions: []*selectionv1.Box{{X: .2, Y: .7, Width: .3, Height: .1}}}}
	raw := paramsMap(p)["background_repair"]
	var parsed aiv1.BackgroundRepairParams
	if raw == "" || protojson.Unmarshal([]byte(raw), &parsed) != nil || parsed.GetMaskExpansionPixels() != 6 || len(parsed.GetRemovalRegions()) != 1 {
		t.Fatalf("typed repair controls not preserved: %q", raw)
	}
}

func TestParamsMapPreservesTypedInpaintCrop(t *testing.T) {
	got := paramsMap(&aiv1.AIParams{Inpaint: &aiv1.InpaintParams{CropToMask: true, ContextPixels: 96, MaxEdgePx: 1024, NeutralizeMaskedPixels: true}})
	want := map[string]string{"inpaint_crop_to_mask": "true", "inpaint_context_pixels": "96", "inpaint_max_edge_px": "1024", "inpaint_neutralize_masked_pixels": "true"}
	for key, value := range want {
		if got[key] != value {
			t.Fatalf("%s = %q, want %q; map=%#v", key, got[key], value, got)
		}
	}
}
