package ensure

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Config is the scenario-provided payload for `ensure`. Ollama declarations
// live under the dependency block in a scenario's service.json and are routed
// to this resource verbatim.
type Config struct {
	Models []ModelSpec `json:"models,omitempty"`
}

// ModelSpec identifies one Ollama model to ensure. Accepts either a bare
// string ("qwen3:4b") or an object ({"name":"qwen3","tag":"4b"}) so service.json
// authors can pick whichever reads best.
type ModelSpec struct {
	Name         string `json:"name"`
	Tag          string `json:"tag,omitempty"`
	Size         string `json:"size,omitempty"`
	Quantization string `json:"quantization,omitempty"`
}

// Ref returns the canonical model reference passed to Ollama (`name:tag`).
func (m ModelSpec) Ref() string {
	name := strings.TrimSpace(m.Name)
	tag := strings.TrimSpace(m.Tag)
	if name == "" {
		return ""
	}
	if tag == "" || strings.Contains(name, ":") {
		return name
	}
	return name + ":" + tag
}

func (m *ModelSpec) UnmarshalJSON(data []byte) error {
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" || trimmed == "null" {
		return nil
	}
	if trimmed[0] == '"' {
		var s string
		if err := json.Unmarshal(data, &s); err != nil {
			return err
		}
		*m = ModelSpec{Name: s}
		return nil
	}
	type alias ModelSpec
	var aux alias
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	*m = ModelSpec(aux)
	return nil
}

// ParseConfig parses the JSON object shipped from the orchestrator. Unknown
// keys are ignored — the resource owns the schema and callers may pass
// forward-compatible extensions.
func ParseConfig(data []byte) (Config, error) {
	var cfg Config
	if len(data) == 0 {
		return cfg, nil
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse ensure config: %w", err)
	}
	return cfg, nil
}
