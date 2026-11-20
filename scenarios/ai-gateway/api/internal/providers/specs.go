package providers

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

const (
	ProviderOllama       = "ollama"
	ProviderOpenRouter   = "openrouter"
	ProviderOrcaRouter   = "orcarouter"
	ProviderAbliteration = "abliteration"
	ProviderMetered      = "lpbs"
	ProviderPiiDetector  = "pii-detector"
	// ProviderOllaya is the decision-model resource: the local Ollaya daemon
	// plus the credential-gated hosted System One upstreams it owns.
	ProviderOllaya = "ollaya"
)

// Provider families describe the wire contract a resource implements. The
// gateway switches on the family, not on the provider id, so adding another
// OpenAI-compatible resource is one registry entry rather than another branch.
const (
	FamilyOllama           = "ollama"
	FamilyOpenAICompatible = "openai-compatible"
	FamilyPII              = "pii-detector"
	FamilyDecision         = "decision"
	FamilyMetered          = "metered"
)

// ProviderSpec is the static registration record for one provider resource.
type ProviderSpec struct {
	ID           string `json:"id"`
	CommandName  string `json:"command_name"`
	Locality     string `json:"locality"`
	Family       string `json:"family"`
	Text         bool   `json:"text"`
	SpendCapable bool   `json:"spend_capable"`
}

//go:embed specs.seed.json
var providerSpecSeed []byte

var (
	providerSpecsOnce sync.Once
	providerSpecs     []ProviderSpec
)

// SupportedFamilies returns the wire families the adapter can execute. It is
// the closed vocabulary a seed row may name; FamilyMetered is intentionally
// absent because the metered door is not a resource-backed adapter entry.
func SupportedFamilies() map[string]struct{} {
	return map[string]struct{}{
		FamilyOllama:           {},
		FamilyOpenAICompatible: {},
		FamilyPII:              {},
		FamilyDecision:         {},
	}
}

// LoadProviderSpecs decodes and validates a provider registry. The embedded
// seed is the production input; tests call this directly with fixtures.
func LoadProviderSpecs(data []byte) ([]ProviderSpec, error) {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var specs []ProviderSpec
	if err := decoder.Decode(&specs); err != nil {
		return nil, fmt.Errorf("decode provider specs: %w", err)
	}
	if err := ensureJSONEOF(decoder); err != nil {
		return nil, err
	}
	if len(specs) == 0 {
		return nil, fmt.Errorf("provider specs seed declares no providers")
	}
	families := SupportedFamilies()
	seenIDs := make(map[string]struct{}, len(specs))
	for index, spec := range specs {
		if err := validateProviderSpec(spec, index+1, families, seenIDs); err != nil {
			return nil, err
		}
	}
	return specs, nil
}

func validateProviderSpec(spec ProviderSpec, position int, families, seenIDs map[string]struct{}) error {
	id, err := validateProviderIdentity(spec, position, seenIDs)
	if err != nil {
		return err
	}
	return validateProviderExecutionFields(spec, id, families)
}

func validateProviderIdentity(spec ProviderSpec, position int, seenIDs map[string]struct{}) (string, error) {
	id := strings.TrimSpace(spec.ID)
	if id == "" {
		return "", fmt.Errorf("provider spec %d: id is required", position)
	}
	if id != spec.ID {
		return "", fmt.Errorf("provider spec %d: id %q must not have surrounding whitespace", position, spec.ID)
	}
	if _, duplicate := seenIDs[id]; duplicate {
		return "", fmt.Errorf("provider spec %d: duplicate id %q", position, id)
	}
	seenIDs[id] = struct{}{}
	return id, nil
}

func validateProviderExecutionFields(spec ProviderSpec, id string, families map[string]struct{}) error {
	if strings.TrimSpace(spec.CommandName) == "" {
		return fmt.Errorf("provider spec %q: command_name is required", id)
	}
	if spec.CommandName != strings.TrimSpace(spec.CommandName) {
		return fmt.Errorf("provider spec %q: command_name %q must not have surrounding whitespace", id, spec.CommandName)
	}
	if spec.Locality != "local" && spec.Locality != "remote" {
		return fmt.Errorf("provider spec %q: locality %q must be local or remote", id, spec.Locality)
	}
	if strings.TrimSpace(spec.Family) == "" {
		return fmt.Errorf("provider spec %q: family is required", id)
	}
	if _, supported := families[spec.Family]; !supported {
		return fmt.Errorf("provider spec %q: family %q is not executable by the adapter", id, spec.Family)
	}
	return nil
}

func ensureJSONEOF(decoder *json.Decoder) error {
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return fmt.Errorf("provider specs seed has trailing JSON content")
		}
		return fmt.Errorf("decode provider specs: %w", err)
	}
	return nil
}

func loadProviderSpecs() []ProviderSpec {
	providerSpecsOnce.Do(func() {
		specs, err := LoadProviderSpecs(providerSpecSeed)
		if err != nil {
			panic(fmt.Sprintf("providers: invalid embedded provider spec seed: %v", err))
		}
		providerSpecs = specs
	})
	return providerSpecs
}

// DefaultProviderSpecs returns the resource-backed providers the gateway knows
// how to execute. Adding a provider is an entry in specs.seed.json plus its
// resource. The returned slice is a copy so callers cannot mutate the registry.
func DefaultProviderSpecs() []ProviderSpec {
	specs := loadProviderSpecs()
	out := make([]ProviderSpec, len(specs))
	copy(out, specs)
	return out
}

// SpecFor returns the registered spec for a provider id.
func SpecFor(provider string) (ProviderSpec, bool) {
	provider = strings.TrimSpace(provider)
	for _, spec := range DefaultProviderSpecs() {
		if spec.ID == provider {
			return spec, true
		}
	}
	return ProviderSpec{}, false
}

// TextProviderIDs returns the provider ids admissible for typed-value roles.
func TextProviderIDs() map[string]struct{} {
	out := map[string]struct{}{}
	for _, spec := range DefaultProviderSpecs() {
		if spec.Text {
			out[spec.ID] = struct{}{}
		}
	}
	return out
}

// AllowedCommands returns the resource commands the runner may execute.
func AllowedCommands() map[string]struct{} {
	out := map[string]struct{}{}
	for _, spec := range DefaultProviderSpecs() {
		out[spec.CommandName] = struct{}{}
	}
	return out
}
