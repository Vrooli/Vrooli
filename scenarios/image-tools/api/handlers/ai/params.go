package ai

import (
	"strconv"

	aiv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/ai"
)

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
	return m
}
