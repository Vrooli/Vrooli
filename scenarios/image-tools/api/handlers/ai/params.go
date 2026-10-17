package ai

import (
	"strconv"
	"strings"

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
