package inference

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"
)

// DefaultRoleTimeout applies when a role declares no timeout_ms. It is sized
// for a local model producing a small schema-constrained value, which is far
// beyond providers.DefaultCommandTimeout — that 5s default is calibrated for
// metadata commands such as `policy resolve`, not for generation.
const DefaultRoleTimeout = 90 * time.Second

// MaxRoleTimeout bounds what a catalog may declare so a single role cannot pin
// a worker indefinitely.
const MaxRoleTimeout = 10 * time.Minute

type RoleCatalog struct {
	SchemaVersion int                      `json:"schema_version"`
	Roles         map[string]InferenceRole `json:"roles"`
}

type InferenceRole struct {
	Description string      `json:"description"`
	Candidates  []Candidate `json:"candidates"`
	// TimeoutMS bounds one provider execution for this role. Generation cost
	// varies by an order of magnitude between a short enum classification and a
	// large structured extraction, so the bound is per role rather than global.
	TimeoutMS int `json:"timeout_ms,omitempty"`
}

// Timeout resolves the declared per-role bound, falling back to
// DefaultRoleTimeout when the catalog leaves it unset.
func (r InferenceRole) Timeout() time.Duration {
	if r.TimeoutMS <= 0 {
		return DefaultRoleTimeout
	}
	return time.Duration(r.TimeoutMS) * time.Millisecond
}

type Candidate struct {
	Provider            string `json:"provider"`
	ResourceRole        string `json:"resource_role"`
	Model               string `json:"model"`
	MinimumQuantization string `json:"minimum_quantization,omitempty"`
	Reason              string `json:"reason"`
}

func LoadCatalog(path string) (RoleCatalog, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return RoleCatalog{}, fmt.Errorf("read inference role catalog %s: %w", path, err)
	}
	var catalog RoleCatalog
	if err := json.Unmarshal(data, &catalog); err != nil {
		return RoleCatalog{}, fmt.Errorf("parse inference role catalog: %w", err)
	}
	if catalog.SchemaVersion != 1 {
		return RoleCatalog{}, fmt.Errorf("unsupported inference role catalog schema_version %d", catalog.SchemaVersion)
	}
	if err := catalog.Validate(); err != nil {
		return RoleCatalog{}, err
	}
	return catalog, nil
}

func (c RoleCatalog) Validate() error {
	if len(c.Roles) == 0 {
		return fmt.Errorf("inference role catalog has no roles")
	}
	for role, definition := range c.Roles {
		if strings.TrimSpace(role) == "" || len(definition.Candidates) == 0 {
			return fmt.Errorf("inference role %q has no candidates", role)
		}
		if definition.TimeoutMS < 0 {
			return fmt.Errorf("inference role %q timeout_ms must be positive", role)
		}
		if definition.Timeout() > MaxRoleTimeout {
			return fmt.Errorf("inference role %q timeout_ms exceeds the %s maximum", role, MaxRoleTimeout)
		}
		seen := map[string]struct{}{}
		for _, candidate := range definition.Candidates {
			for field, value := range map[string]string{"provider": candidate.Provider, "resource_role": candidate.ResourceRole, "reason": candidate.Reason} {
				if strings.TrimSpace(value) == "" {
					return fmt.Errorf("inference role %q candidate %s is required", role, field)
				}
			}
			provider := strings.ToLower(strings.TrimSpace(candidate.Provider))
			if provider != "ollama" && provider != "openrouter" {
				return fmt.Errorf("inference role %q candidate provider %q is unsupported", role, candidate.Provider)
			}
			key := provider + ":" + candidate.ResourceRole
			if _, exists := seen[key]; exists {
				return fmt.Errorf("inference role %q repeats candidate %q", role, key)
			}
			seen[key] = struct{}{}
		}
	}
	return nil
}

func (c RoleCatalog) RoleNames() []string {
	roles := make([]string, 0, len(c.Roles))
	for role := range c.Roles {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	return roles
}
