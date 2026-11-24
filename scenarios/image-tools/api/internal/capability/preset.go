package capability

import (
	"encoding/json"
	"fmt"
	"strings"
)

// PresetSpec declares an operation that is a named way of running another one: a
// request for it becomes a request for Runs, with a default model and a standing
// conditioning stack whose images come from the request's own uploads. The preset
// is the product-level door ("put this character in that pose"); the operation it
// runs stays the one that plans, admits, runs and reviews, so a preset never has
// a path of its own around consent, capacity or the safety review.
type PresetSpec struct {
	// Runs is the operation the request becomes.
	Runs string `json:"runs"`
	// Model is the model used when the caller names none.
	Model string `json:"model,omitempty"`
	// Adapters are the conditioning adapters the preset adds, in the order given.
	Adapters []PresetAdapter `json:"adapters"`
	// Prompt wraps the caller's prompt: {prompt} is replaced by it. Empty uses the
	// caller's prompt as written.
	Prompt string `json:"prompt,omitempty"`
	// Negative, Steps and CfgScale are used when the caller gave none.
	Negative string  `json:"negative,omitempty"`
	Steps    int     `json:"steps,omitempty"`
	CfgScale float64 `json:"cfg_scale,omitempty"`
}

// PromptPlaceholder marks where the caller's prompt goes in a preset's Prompt.
const PromptPlaceholder = "{prompt}"

// PresetAdapter is one standing adapter of a preset.
type PresetAdapter struct {
	// ID is the adapter catalog id.
	ID string `json:"id"`
	// Input names the port whose upload the adapter reads (its control or
	// reference image). An adapter whose port was not supplied is left out, so an
	// optional port makes its adapter optional.
	Input string `json:"input"`
	// Scale is the strength used when ScaleParam is absent from the request; 0
	// takes the adapter's own default.
	Scale float64 `json:"scale,omitempty"`
	// ScaleParam is the op parameter that sets this adapter's strength.
	ScaleParam string `json:"scale_param,omitempty"`
}

// ImagePort is the input name of an operation's primary image port.
const ImagePort = "image"

func validatePreset(r *Registry, m Manifest) error {
	p := m.Preset
	if p == nil {
		return nil
	}
	target, ok := r.byID[p.Runs]
	switch {
	case p.Runs == "" || p.Runs == m.ID:
		return fmt.Errorf("capability: preset %q must run another operation", m.ID)
	case !ok:
		return fmt.Errorf("capability: preset %q runs unknown operation %q", m.ID, p.Runs)
	case target.Preset != nil:
		return fmt.Errorf("capability: preset %q runs %q, which is itself a preset", m.ID, p.Runs)
	case target.Engine != EngineAI || m.Engine != EngineAI:
		return fmt.Errorf("capability: preset %q and the operation it runs must both be engine ai", m.ID)
	case p.Prompt != "" && !strings.Contains(p.Prompt, PromptPlaceholder):
		return fmt.Errorf("capability: preset %q prompt must contain %s", m.ID, PromptPlaceholder)
	case p.Steps < 0 || p.CfgScale < 0:
		return fmt.Errorf("capability: preset %q has a negative steps or cfg_scale", m.ID)
	case len(p.Adapters) == 0:
		return fmt.Errorf("capability: preset %q declares no adapters; use the operation directly", m.ID)
	}
	ports := map[string]bool{}
	for _, in := range m.IO.Inputs {
		ports[in.Name] = true
	}
	for _, a := range p.Adapters {
		if a.ID == "" || strings.TrimSpace(a.ID) != a.ID {
			return fmt.Errorf("capability: preset %q has an adapter with no id", m.ID)
		}
		if !ports[a.Input] {
			return fmt.Errorf("capability: preset %q adapter %q reads port %q, which the operation does not declare", m.ID, a.ID, a.Input)
		}
		if a.Scale < 0 {
			return fmt.Errorf("capability: preset %q adapter %q has a negative scale", m.ID, a.ID)
		}
		if a.ScaleParam != "" && !schemaDeclares(m.ParamsSchema, a.ScaleParam) {
			return fmt.Errorf("capability: preset %q adapter %q scale_param %q is not in the params_schema", m.ID, a.ID, a.ScaleParam)
		}
	}
	return nil
}

// PresetTarget reports what a preset operation runs: the target operation, the
// default model (empty when the preset names none) and its standing adapters.
// The submit edge uses it to expand the request; the read-only explain surface
// uses it for the same expansion so `--explain` describes the job that would run.
func PresetTarget(op string) (runs string, model string, adapters []PresetAdapter, ok bool) {
	m, found := Get(op)
	if !found || m.Preset == nil {
		return "", "", nil, false
	}
	return m.Preset.Runs, m.Preset.Model, m.Preset.Adapters, true
}

// schemaDeclares reports whether a params_schema has a property of this name.
func schemaDeclares(schema json.RawMessage, name string) bool {
	var doc struct {
		Properties map[string]json.RawMessage `json:"properties"`
	}
	if json.Unmarshal(schema, &doc) != nil {
		return false
	}
	_, ok := doc.Properties[name]
	return ok
}
