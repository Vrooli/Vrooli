package ensure

import (
	"encoding/json"
	"fmt"

	"github.com/vrooli/vrooli/resources/ollama/cli/internal/policy"
)

// Config is the scenario-provided payload for `ensure`. Ollama declarations
// live under the dependency block in a scenario's service.json and are routed
// to this resource verbatim.
type Config struct {
	Model      string                      `json:"model,omitempty"`
	ModelRoles []policy.RoleRequest        `json:"model_roles,omitempty"`
	Models     []policy.DirectModelRequest `json:"models,omitempty"`
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

func (c Config) ResolveRequest() policy.ResolveRequest {
	return policy.ResolveRequest{
		ModelRoles:      c.ModelRoles,
		Models:          c.Models,
		DeprecatedModel: c.Model,
	}
}
