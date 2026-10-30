package inference

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"ai-gateway/internal/providers"
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

// RoleKindDecision marks a role served by non-generative decision models
// through Service.Decide. Every other role is a typed-value role served by Run.
const RoleKindDecision = "decision"

// ProviderGateway names a candidate that is one of AI Gateway's own typed
// roles rather than a resource. A decision role uses it as its last-resort
// fallback: the typed classify path answering choice questions as an enum.
const ProviderGateway = "ai-gateway"

type InferenceRole struct {
	Description string `json:"description"`
	// Kind is empty for typed-value roles and RoleKindDecision for roles whose
	// candidates are decision models.
	Kind       string      `json:"kind,omitempty"`
	Candidates []Candidate `json:"candidates"`
	// TimeoutMS bounds one provider execution for this role. Generation cost
	// varies by an order of magnitude between a short enum classification and a
	// large structured extraction, so the bound is per role rather than global.
	TimeoutMS int `json:"timeout_ms,omitempty"`
	// Sampling declares this role's default sampling stance and whether a
	// caller may override it. A role is deterministic unless its author
	// deliberately opened it, so Overridable is false by default.
	Sampling *RoleSampling `json:"sampling,omitempty"`
}

// RoleSampling is the role's own sampling stance. It replaces the global
// determinism pin the repository used to carry: determinism is now a property a
// role declares, which is what lets one catalog serve both a judge that must
// never drift and a writing role that must.
type RoleSampling struct {
	// Temperature is the value the gateway sends when the caller sends none. A
	// candidate whose provider cannot honour it omits the control and continues
	// rather than failing: the role author expressed a preference, not a promise.
	Temperature *float64 `json:"temperature,omitempty"`
	// Overridable admits a caller-supplied temperature. When false, a caller
	// that sends one gets INVALID_REQUEST — the role forbids it, which is a
	// request defect and not provider incapacity.
	Overridable bool `json:"overridable,omitempty"`
}

// Sampling bounds — the envelope both resource policies clamp into. A catalog
// declaring outside it would be silently reshaped downstream, so it fails load.
const (
	MinRoleTemperature = 0.0
	MaxRoleTemperature = 2.0
)

// IsDecision reports whether the role is served by Service.Decide.
func (r InferenceRole) IsDecision() bool { return r.Kind == RoleKindDecision }

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
		if err := validateRole(role, definition); err != nil {
			return err
		}
		// A gateway candidate delegates to another catalog role, so it can
		// only be checked against the whole catalog.
		for _, candidate := range definition.Candidates {
			if candidate.Provider != ProviderGateway {
				continue
			}
			target, ok := c.Roles[candidate.ResourceRole]
			if !ok || target.IsDecision() {
				return fmt.Errorf("inference role %q fallback %q must name a declared typed role", role, candidate.ResourceRole)
			}
		}
	}
	return nil
}

func validateRole(role string, definition InferenceRole) error {
	if strings.TrimSpace(role) == "" || len(definition.Candidates) == 0 {
		return fmt.Errorf("inference role %q has no candidates", role)
	}
	if definition.TimeoutMS < 0 {
		return fmt.Errorf("inference role %q timeout_ms must be positive", role)
	}
	if definition.Timeout() > MaxRoleTimeout {
		return fmt.Errorf("inference role %q timeout_ms exceeds the %s maximum", role, MaxRoleTimeout)
	}
	if definition.Kind != "" && !definition.IsDecision() {
		return fmt.Errorf("inference role %q kind %q is unsupported", role, definition.Kind)
	}
	if definition.IsDecision() && definition.Sampling != nil {
		return fmt.Errorf("inference role %q is a decision role; decision models take no sampling controls", role)
	}
	if sampling := definition.Sampling; sampling != nil && sampling.Temperature != nil {
		if *sampling.Temperature < MinRoleTemperature || *sampling.Temperature > MaxRoleTemperature {
			return fmt.Errorf("inference role %q sampling.temperature %.3f is outside [%.1f,%.1f]",
				role, *sampling.Temperature, MinRoleTemperature, MaxRoleTemperature)
		}
	}
	return validateCandidates(role, definition.IsDecision(), definition.Candidates)
}

// candidateProviders is the closed set of providers each role kind may name. A
// decision model cannot answer a typed-value request and a generative model
// is reached by a decision role only through the gateway's own classify path.
var candidateProviders = map[bool]map[string]struct{}{
	false: {providers.ProviderOllama: {}, providers.ProviderOpenRouter: {}, providers.ProviderMetered: {}, providers.ProviderPiiDetector: {}},
	true:  {providers.ProviderOllaya: {}, ProviderGateway: {}},
}

func validateCandidates(role string, decision bool, candidates []Candidate) error {
	seen := map[string]struct{}{}
	for _, candidate := range candidates {
		for field, value := range map[string]string{"provider": candidate.Provider, "resource_role": candidate.ResourceRole, "reason": candidate.Reason} {
			if strings.TrimSpace(value) == "" {
				return fmt.Errorf("inference role %q candidate %s is required", role, field)
			}
		}
		provider := strings.ToLower(strings.TrimSpace(candidate.Provider))
		if _, ok := candidateProviders[decision][provider]; !ok {
			return fmt.Errorf("inference role %q candidate provider %q is unsupported", role, candidate.Provider)
		}
		key := provider + ":" + candidate.ResourceRole
		if _, exists := seen[key]; exists {
			return fmt.Errorf("inference role %q repeats candidate %q", role, key)
		}
		seen[key] = struct{}{}
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
