package ai

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"google.golang.org/protobuf/encoding/protojson"

	"image-tools/internal/adapters"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

// adapterRequests converts the typed AdapterRef list on AIParams into the
// resolver's conditioning request shape (decision C2: typed, never the params
// bag). Empty when no adapters were requested.
func adapterRequests(p *aiv1.AIParams) []adapters.AdapterRequest {
	refs := p.GetAdapters()
	if len(refs) == 0 {
		return nil
	}
	out := make([]adapters.AdapterRequest, 0, len(refs))
	for _, r := range refs {
		out = append(out, adapters.AdapterRequest{
			ID:                   r.GetAdapterId(),
			Scale:                r.GetScale(),
			ConditioningImageKey: r.GetConditioningImageKey(),
			PreprocessorOverride: adapters.Preprocessor(r.GetPreprocessorOverride()),
		})
	}
	return out
}

func qualityPolicyForParams(p *aiv1.AIParams) string {
	switch strings.ToLower(strings.TrimSpace(p.GetQualityPolicy())) {
	case "fast", "balanced", "quality":
		return strings.ToLower(strings.TrimSpace(p.GetQualityPolicy()))
	default:
		return ""
	}
}

func fallbackPolicyForParams(p *aiv1.AIParams) string {
	switch strings.ToLower(strings.TrimSpace(p.GetFallbackPolicy())) {
	case "local_only", "cloud_allowed", "any":
		return strings.ToLower(strings.TrimSpace(p.GetFallbackPolicy()))
	default:
		return ""
	}
}

func allowBYOKForParams(p *aiv1.AIParams) bool {
	if fallbackPolicyForParams(p) == "local_only" {
		return false
	}
	return p.GetAllowByok()
}

func priorityForParams(p *aiv1.AIParams) string {
	switch strings.ToLower(strings.TrimSpace(p.GetPriority())) {
	case "batch", "service", "interactive":
		return strings.ToLower(strings.TrimSpace(p.GetPriority()))
	default:
		return "service"
	}
}

func allowReclaimForParams(p *aiv1.AIParams) bool {
	if p.AllowReclaim == nil {
		return true
	}
	return p.GetAllowReclaim()
}

// paramsMap flattens AIParams into the string map the engine threads to the
// backend providers. Only meaningful (non-zero / non-empty) fields are included
// so a provider's per-op defaults apply to the rest.
func paramsMap(p *aiv1.AIParams) map[string]string {
	m := map[string]string{}
	if v := p.GetSegmentation(); v != nil {
		// Admission validates finite, bounded coordinates before this conversion.
		if raw, err := protojson.Marshal(v); err == nil {
			m["segmentation"] = string(raw)
		}
	}
	if v := p.GetSceneGeometry(); v != nil && v.GetSupportRoi() != nil {
		if raw, err := (protojson.MarshalOptions{EmitUnpopulated: true}).Marshal(v.GetSupportRoi()); err == nil {
			m["support_roi"] = string(raw)
		}
	}
	if v := p.GetImageToObject(); v != nil && v.GetMeshResolution() != 0 {
		m["mesh_resolution"] = strconv.FormatUint(uint64(v.GetMeshResolution()), 10)
	}
	if v := p.GetBackgroundRepair(); v != nil {
		if raw, err := protojson.Marshal(v); err == nil {
			m["background_repair"] = string(raw)
		}
	}
	if v := p.GetInpaint(); v != nil {
		m["inpaint_crop_to_mask"] = strconv.FormatBool(v.GetCropToMask())
		m["inpaint_context_pixels"] = strconv.FormatUint(uint64(v.GetContextPixels()), 10)
		m["inpaint_max_edge_px"] = strconv.FormatUint(uint64(v.GetMaxEdgePx()), 10)
		m["inpaint_neutralize_masked_pixels"] = strconv.FormatBool(v.GetNeutralizeMaskedPixels())
	}
	if v := p.GetPrompt(); v != "" {
		m["prompt"] = v
	}
	if v := p.GetNegativePrompt(); v != "" {
		m["negative_prompt"] = v
	}
	if v := p.GetSeed(); v != 0 {
		m["seed"] = strconv.FormatInt(v, 10)
	}
	if v := p.GetWidth(); v != 0 {
		m["width"] = strconv.Itoa(int(v))
	}
	if v := p.GetHeight(); v != 0 {
		m["height"] = strconv.Itoa(int(v))
	}
	if v := p.GetSteps(); v != 0 {
		m["steps"] = strconv.Itoa(int(v))
	}
	if v := p.GetCfgScale(); v != 0 {
		m["cfg_scale"] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	if v := p.GetStrength(); v != 0 {
		m["strength"] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	if v := p.GetScale(); v != 0 {
		m["scale"] = strconv.Itoa(int(v))
	}
	if v := p.GetRealism(); v != 0 {
		m["realism"] = strconv.FormatFloat(v, 'f', -1, 64)
	}
	if p.GetFaceAware() {
		m["face_aware"] = "true"
	}
	if v := p.GetOpenrouterRole(); v != "" {
		m["openrouter_role"] = v
	}
	return m
}

func validateBackgroundRepair(op string, p *aiv1.AIParams) error {
	params := p.GetBackgroundRepair()
	if op != "object_removal" {
		if params != nil {
			return fmt.Errorf("background_repair parameters apply only to object_removal")
		}
		return nil
	}
	if params == nil {
		return nil
	}
	if params.GetMaskExpansionPixels() > 64 {
		return fmt.Errorf("background_repair mask_expansion_pixels must be at most 64")
	}
	if len(params.GetRemovalRegions()) > 32 {
		return fmt.Errorf("background_repair accepts at most 32 removal_regions")
	}
	unit := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1 }
	for _, b := range params.GetRemovalRegions() {
		if b == nil || !unit(b.GetX()) || !unit(b.GetY()) || !unit(b.GetWidth()) || !unit(b.GetHeight()) || b.GetWidth() <= 0 || b.GetHeight() <= 0 || b.GetX()+b.GetWidth() > 1 || b.GetY()+b.GetHeight() > 1 {
			return fmt.Errorf("background_repair removal_regions must have positive area within normalized 0..1 coordinates")
		}
	}
	return nil
}

func validateImageToObject(op string, p *aiv1.AIParams) error {
	params := p.GetImageToObject()
	if op != "image_to_object" {
		if params != nil {
			return fmt.Errorf("image_to_object parameters apply only to image_to_object")
		}
		return nil
	}
	if params == nil || params.GetMeshResolution() == 0 || params.GetMeshResolution() == 128 || params.GetMeshResolution() == 256 {
		return nil
	}
	return fmt.Errorf("image_to_object mesh_resolution must be 128 or 256")
}

func validateSegmentation(op string, p *aiv1.AIParams) error {
	s := p.GetSegmentation()
	if op != "segment" {
		if s != nil {
			return fmt.Errorf("segmentation prompts apply only to segment")
		}
		return nil
	}
	if s == nil || len(s.GetPoints()) > 64 {
		return fmt.Errorf("segment requires a box or at most 64 explicit points")
	}
	unit := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1 }
	positive := false
	for _, point := range s.GetPoints() {
		if point == nil || !unit(point.GetX()) || !unit(point.GetY()) {
			return fmt.Errorf("segment points must have finite coordinates in 0..1")
		}
		positive = positive || !point.GetNegative()
	}
	if b := s.GetBox(); b != nil {
		if !unit(b.X) || !unit(b.Y) || !unit(b.Width) || !unit(b.Height) || b.Width <= 0 || b.Height <= 0 || b.X+b.Width > 1 || b.Y+b.Height > 1 {
			return fmt.Errorf("segment box must have positive area within 0..1")
		}
		positive = true
	}
	if !positive {
		return fmt.Errorf("segment requires a positive point or object box; auto-subject is not supported")
	}
	if p.GetVariations() > 1 || len(p.GetAdapters()) > 0 {
		return fmt.Errorf("segment does not accept variations or conditioning adapters")
	}
	return nil
}

func validateSceneGeometry(op string, p *aiv1.AIParams) error {
	g := p.GetSceneGeometry()
	if op != "scene_geometry" {
		if g != nil {
			return fmt.Errorf("scene_geometry parameters apply only to scene_geometry")
		}
		return nil
	}
	if g == nil || g.GetSupportRoi() == nil {
		return nil
	}
	b := g.GetSupportRoi()
	unit := func(v float64) bool { return !math.IsNaN(v) && !math.IsInf(v, 0) && v >= 0 && v <= 1 }
	if !unit(b.GetX()) || !unit(b.GetY()) || !unit(b.GetWidth()) || !unit(b.GetHeight()) || b.GetWidth() <= 0 || b.GetHeight() <= 0 || b.GetX()+b.GetWidth() > 1 || b.GetY()+b.GetHeight() > 1 {
		return fmt.Errorf("scene_geometry support_roi must have positive area within normalized 0..1 coordinates")
	}
	return nil
}

// validateInpaint prevents crop controls from silently disappearing in other ops.
func validateInpaint(op string, p *aiv1.AIParams) error {
	crop := p.GetInpaint()
	if crop == nil {
		return nil
	}
	if op != "inpaint" {
		return fmt.Errorf("inpaint controls are supported only by inpaint")
	}
	if !crop.CropToMask {
		return fmt.Errorf("inpaint crop controls require crop_to_mask=true; omit inpaint to use the full source")
	}
	if crop.ContextPixels > 512 {
		return fmt.Errorf("inpaint context_pixels must be 0..512")
	}
	if crop.MaxEdgePx != 0 && crop.MaxEdgePx != 512 && crop.MaxEdgePx != 1024 {
		return fmt.Errorf("inpaint max_edge_px must be 512 or 1024, or unset")
	}
	return nil
}
