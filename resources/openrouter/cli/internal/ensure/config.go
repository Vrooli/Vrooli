package ensure

import (
	"encoding/json"
	"fmt"
	"resource-openrouter/cli/internal/policy"
)

// Config is the scenario-provided payload for `ensure`, routed verbatim from a
// scenario's service.json `dependencies.resources.openrouter` block. The
// greenfield contract is role-only: concrete `model`/`models` fields are
// deprecated and rejected. They are captured here (as raw) solely so ensure can
// emit a precise failure rather than silently ignoring them.
type Config struct {
	ModelRoles []policy.RoleRequest `json:"model_roles,omitempty"`

	DeprecatedModel  json.RawMessage `json:"model,omitempty"`
	DeprecatedModels json.RawMessage `json:"models,omitempty"`
}

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

// DeprecatedFields reports concrete-model fields that violate the role-only
// contract. A non-empty result is a hard failure in Run.
func (c Config) DeprecatedFields() []string {
	var found []string
	if len(c.DeprecatedModel) > 0 && string(c.DeprecatedModel) != "null" {
		found = append(found, "model")
	}
	if len(c.DeprecatedModels) > 0 && string(c.DeprecatedModels) != "null" {
		found = append(found, "models")
	}
	return found
}

func (c Config) ResolveRequest() policy.ResolveRequest {
	return policy.ResolveRequest{ModelRoles: c.ModelRoles}
}
