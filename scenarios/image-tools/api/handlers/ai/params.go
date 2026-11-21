package ai

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"image-tools/internal/adapters"
	"image-tools/internal/capability"

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
			LicenseTermsAccepted: r.GetLicenseTermsAccepted(),
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
	// Operation-specific parameters declared by the capability manifest. Shared
	// typed fields above win on name collision, so op_params can only add knobs
	// the operation's schema declares.
	for k, v := range opParamsMap(p) {
		if _, exists := m[k]; !exists {
			m[k] = v
		}
	}
	return m
}

// opParamsMap flattens AIParams.op_params (a proto Struct) into the string map
// the engine threads to providers. Values are schema-validated before this runs.
func opParamsMap(p *aiv1.AIParams) map[string]string {
	out := map[string]string{}
	if p == nil || p.GetOpParams() == nil {
		return out
	}
	for k, v := range p.GetOpParams().AsMap() {
		switch tv := v.(type) {
		case string:
			out[k] = tv
		case bool:
			out[k] = strconv.FormatBool(tv)
		case float64:
			out[k] = strconv.FormatFloat(tv, 'f', -1, 64)
		case nil:
			// Explicit null means "unset"; let the provider default apply.
		default:
			if raw, err := json.Marshal(tv); err == nil {
				out[k] = string(raw)
			}
		}
	}
	return out
}

// validateOpParams enforces the operation's declared params_schema. An operation
// with no schema rejects any op_params, so typos and silently-ignored input fail
// loudly instead of being dropped.
func validateOpParams(op string, p *aiv1.AIParams) error {
	values := map[string]any{}
	if p != nil && p.GetOpParams() != nil {
		values = p.GetOpParams().AsMap()
	}
	return capability.ValidateOpParams(op, values)
}

func hasInpaintControls(p *aiv1.AIParams) bool {
	for k := range opParamsMap(p) {
		if strings.HasPrefix(k, "inpaint_") {
			return true
		}
	}
	return false
}

// validateSegmentationSemantics covers the part of the segment contract that is
// genuinely relational and therefore cannot be a JSON Schema constraint: the
// prompt must contain a positive point or a box, and segmentation accepts no
// variations or conditioning adapters. Types and coordinate ranges are enforced
// by the operation's params_schema.
func validateSegmentationSemantics(op string, p *aiv1.AIParams) error {
	if op != "segment" {
		return nil
	}
	raw, ok := p.GetOpParams().AsMap()["segmentation"]
	if !ok {
		return fmt.Errorf("segment requires a segmentation prompt (points or box)")
	}
	prompt, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("segment prompt must be an object")
	}
	if _, ok := prompt["box"].(map[string]any); ok {
		// Geometry (positive area inside the frame) is the operation's schema rule;
		// this check is only "a prompt exists to segment".
		return nil
	}
	points, _ := prompt["points"].([]any)
	for _, pt := range points {
		if m, ok := pt.(map[string]any); ok {
			if neg, _ := m["negative"].(bool); !neg {
				return nil
			}
		}
	}
	return fmt.Errorf("segment requires a positive point or an object box; auto-subject is not supported")
}

// hasBackgroundRepair reports whether the request carries background-repair
// controls (now op_params). Only the qualified big-lama model implements them.
func hasBackgroundRepair(p *aiv1.AIParams) bool {
	if p == nil || p.GetOpParams() == nil {
		return false
	}
	_, ok := p.GetOpParams().AsMap()["background_repair"]
	return ok
}
