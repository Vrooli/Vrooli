package modelregistry

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"agent-manager/internal/domain"

	repocontract "github.com/vrooli/repo-contract-go"
)

// ModelOption represents a selectable model with an optional description.
// It marshals as a string when no description is present.
type ModelOption struct {
	ID          string `json:"id"`
	Description string `json:"description,omitempty"`
}

func (m *ModelOption) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return domain.NewValidationErrorWithCode("modelOption", "payload is empty", domain.ErrCodeValidationRequired)
	}
	if data[0] == '"' {
		var id string
		if err := json.Unmarshal(data, &id); err != nil {
			return domain.NewValidationErrorWithCode("modelOption", "invalid string payload", domain.ErrCodeValidationFormat)
		}
		m.ID = strings.TrimSpace(id)
		m.Description = ""
		return nil
	}

	var aux struct {
		ID          string `json:"id"`
		Description string `json:"description,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return domain.NewValidationErrorWithCode("modelOption", "invalid object payload", domain.ErrCodeValidationFormat)
	}
	m.ID = strings.TrimSpace(aux.ID)
	m.Description = strings.TrimSpace(aux.Description)
	return nil
}

func (m ModelOption) MarshalJSON() ([]byte, error) {
	id := strings.TrimSpace(m.ID)
	if m.Description == "" {
		return json.Marshal(id)
	}
	return json.Marshal(struct {
		ID          string `json:"id"`
		Description string `json:"description,omitempty"`
	}{
		ID:          id,
		Description: m.Description,
	})
}

// PresetChain is the ordered list of model IDs that a preset expands into.
// The first non-empty entry is the primary choice. Subsequent entries are fallbacks
// tried in order when the runner rejects a model at execution time. A single empty-string
// entry at the final position signals "invoke the runner without a model flag, letting
// the CLI use its built-in default."
type PresetChain []string

// Primary returns the first non-empty model ID in the chain, or the empty string if
// the chain contains only the runner-default sentinel.
func (c PresetChain) Primary() string {
	for _, entry := range c {
		if strings.TrimSpace(entry) != "" {
			return entry
		}
	}
	return ""
}

// At returns the model ID at the given position and whether it is within bounds.
// An empty return value with ok=true is the runner-default sentinel.
func (c PresetChain) At(index int) (string, bool) {
	if index < 0 || index >= len(c) {
		return "", false
	}
	return c[index], true
}

// AllowRunnerDefault reports whether the final entry is the empty-string sentinel.
func (c PresetChain) AllowRunnerDefault() bool {
	if len(c) == 0 {
		return false
	}
	return c[len(c)-1] == ""
}

// ConcreteModels returns only the non-empty entries in order.
func (c PresetChain) ConcreteModels() []string {
	out := make([]string, 0, len(c))
	for _, entry := range c {
		if strings.TrimSpace(entry) != "" {
			out = append(out, entry)
		}
	}
	return out
}

// UnmarshalJSON accepts only an array of strings. Rejects the legacy scalar-string
// shape with an actionable error message.
func (c *PresetChain) UnmarshalJSON(data []byte) error {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return domain.NewValidationErrorWithCode("modelRegistry.presets", "preset value is empty", domain.ErrCodeValidationRequired)
	}
	if trimmed[0] == '"' {
		return domain.NewValidationError(
			"modelRegistry.presets",
			"preset values must be arrays of model IDs (e.g. [\"gpt-5.2-codex\", \"gpt-5.1-codex-max\"]); the legacy scalar-string shape is no longer supported",
		)
	}
	if trimmed[0] != '[' {
		return domain.NewValidationError("modelRegistry.presets", "preset value must be a JSON array of strings")
	}
	var list []string
	if err := json.Unmarshal(data, &list); err != nil {
		return domain.NewValidationErrorWithCode("modelRegistry.presets", "invalid preset chain payload", domain.ErrCodeValidationFormat)
	}
	*c = list
	return nil
}

// RunnerModelRegistry holds model choices and preset mappings for a runner.
type RunnerModelRegistry struct {
	Models  []ModelOption          `json:"models"`
	Presets map[string]PresetChain `json:"presets,omitempty"`
}

// Registry contains model catalog data for all runners.
type Registry struct {
	Version             int                            `json:"version"`
	FallbackRunnerTypes []string                       `json:"fallbackRunnerTypes,omitempty"`
	Runners             map[string]RunnerModelRegistry `json:"runners"`
}

func (r *Registry) Clone() *Registry {
	if r == nil {
		return nil
	}
	clone := &Registry{
		Version: r.Version,
		Runners: make(map[string]RunnerModelRegistry, len(r.Runners)),
	}
	if len(r.FallbackRunnerTypes) > 0 {
		clone.FallbackRunnerTypes = append([]string(nil), r.FallbackRunnerTypes...)
	}
	for key, runner := range r.Runners {
		models := make([]ModelOption, len(runner.Models))
		copy(models, runner.Models)
		presets := make(map[string]PresetChain, len(runner.Presets))
		for presetKey, chain := range runner.Presets {
			presets[presetKey] = append(PresetChain(nil), chain...)
		}
		clone.Runners[key] = RunnerModelRegistry{
			Models:  models,
			Presets: presets,
		}
	}
	return clone
}

func (r *Registry) Validate() error {
	if r == nil {
		return domain.NewValidationErrorWithCode("modelRegistry", "field is required", domain.ErrCodeValidationRequired)
	}
	if r.Version <= 0 {
		return domain.NewValidationError("modelRegistry.version", "must be greater than zero")
	}
	if len(r.Runners) == 0 {
		return domain.NewValidationError("modelRegistry.runners", "must define at least one runner")
	}

	if err := validateFallbackRunnerTypes(r.FallbackRunnerTypes); err != nil {
		return err
	}

	for runnerKey, runner := range r.Runners {
		if strings.TrimSpace(runnerKey) == "" {
			return domain.NewValidationError("modelRegistry.runners", "runner key cannot be empty")
		}

		known := make(map[string]struct{}, len(runner.Models))
		for _, model := range runner.Models {
			id := strings.TrimSpace(model.ID)
			if id == "" {
				return domain.NewValidationError("modelRegistry.runners."+runnerKey+".models", "model id cannot be empty")
			}
			if _, exists := known[id]; exists {
				return domain.NewValidationError("modelRegistry.runners."+runnerKey+".models", fmt.Sprintf("duplicate model id %s", id))
			}
			known[id] = struct{}{}
		}

		for presetKey, chain := range runner.Presets {
			if err := validatePresetChain(runnerKey, presetKey, chain, known); err != nil {
				return err
			}
		}
	}

	return nil
}

// validatePresetChain enforces the preset-chain contract documented on PresetChain.
func validatePresetChain(runnerKey, presetKey string, chain PresetChain, knownModels map[string]struct{}) error {
	fieldPrefix := "modelRegistry.runners." + runnerKey + ".presets"
	normalizedPreset := strings.ToUpper(strings.TrimSpace(presetKey))
	if normalizedPreset == "" {
		return domain.NewValidationError(fieldPrefix, "preset key cannot be empty")
	}
	if !isKnownPreset(normalizedPreset) {
		return domain.NewValidationError(fieldPrefix, fmt.Sprintf("invalid preset key %s", normalizedPreset))
	}

	if len(chain) == 0 {
		return domain.NewValidationError(fieldPrefix+"."+normalizedPreset, "preset chain must contain at least one entry")
	}

	seen := make(map[string]struct{}, len(chain))
	hasConcrete := false
	for index, entry := range chain {
		if entry == "" {
			// Empty-string sentinel: runner-default parachute.
			if index != len(chain)-1 {
				return domain.NewValidationError(
					fieldPrefix+"."+normalizedPreset,
					"the runner-default sentinel (empty string) may only appear as the final entry",
				)
			}
			if normalizedPreset == presetKeyCheap {
				return domain.NewValidationError(
					fieldPrefix+"."+normalizedPreset,
					"the CHEAP preset cannot fall back to the runner default (typically the flagship model, which would silently invert cost expectations); pick an explicit low-cost model for every entry",
				)
			}
			continue
		}

		trimmed := strings.TrimSpace(entry)
		if trimmed != entry {
			return domain.NewValidationError(
				fieldPrefix+"."+normalizedPreset,
				fmt.Sprintf("entry %q has leading or trailing whitespace", entry),
			)
		}
		if _, exists := seen[trimmed]; exists {
			return domain.NewValidationError(
				fieldPrefix+"."+normalizedPreset,
				fmt.Sprintf("duplicate entry %s", trimmed),
			)
		}
		seen[trimmed] = struct{}{}
		if _, exists := knownModels[trimmed]; !exists {
			return domain.NewValidationError(
				fieldPrefix+"."+normalizedPreset,
				fmt.Sprintf("unknown model id %s", trimmed),
			)
		}
		hasConcrete = true
	}

	if !hasConcrete {
		return domain.NewValidationError(
			fieldPrefix+"."+normalizedPreset,
			"preset chain must contain at least one concrete model id",
		)
	}
	return nil
}

func validateFallbackRunnerTypes(values []string) error {
	if len(values) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return domain.NewValidationError("fallbackRunnerTypes", "contains empty runner type")
		}
		if !domain.RunnerType(trimmed).IsValid() {
			return domain.NewValidationError("fallbackRunnerTypes", fmt.Sprintf("contains invalid runner type %s", trimmed))
		}
		if _, exists := seen[trimmed]; exists {
			return domain.NewValidationError("fallbackRunnerTypes", fmt.Sprintf("contains duplicate runner type %s", trimmed))
		}
		seen[trimmed] = struct{}{}
	}
	return nil
}

const (
	presetKeyFast  = "FAST"
	presetKeyCheap = "CHEAP"
	presetKeySmart = "SMART"
)

func isKnownPreset(preset string) bool {
	switch preset {
	case presetKeyFast, presetKeyCheap, presetKeySmart:
		return true
	default:
		return false
	}
}

// Store manages registry state and persistence.
type Store struct {
	path     string
	mu       sync.RWMutex
	registry *Registry
}

func NewStore(path string) (*Store, error) {
	reg, err := Load(path)
	if err != nil {
		return nil, err
	}
	return &Store{path: path, registry: reg}, nil
}

func (s *Store) Get() *Registry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.registry.Clone()
}

func (s *Store) Update(registry *Registry) (*Registry, error) {
	if registry == nil {
		return nil, domain.NewValidationErrorWithCode("modelRegistry", "field is required", domain.ErrCodeValidationRequired)
	}
	if err := registry.Validate(); err != nil {
		return nil, err
	}
	if err := Save(s.path, registry); err != nil {
		return nil, err
	}
	s.mu.Lock()
	s.registry = registry.Clone()
	s.mu.Unlock()
	return s.Get(), nil
}

// ResolvePreset returns the ordered chain of model IDs for a preset on a runner.
// The caller walks the chain in order, treating an empty entry as "invoke the runner
// without a model flag." A nil or empty chain is reported via ok=false.
func (s *Store) ResolvePreset(runner string, preset string) (PresetChain, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.registry == nil {
		return nil, false
	}
	runnerConfig, ok := s.registry.Runners[runner]
	if !ok {
		return nil, false
	}
	chain, ok := runnerConfig.Presets[strings.ToUpper(preset)]
	if !ok || len(chain) == 0 {
		return nil, false
	}
	// Defensive copy so callers cannot mutate registry state.
	return append(PresetChain(nil), chain...), true
}

// Load reads the model registry from disk.
func Load(path string) (*Registry, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return nil, domain.NewConfigInvalidError("modelRegistry", "failed to read model registry", err)
	}

	var registry Registry
	if err := json.Unmarshal(bytes, &registry); err != nil {
		return nil, domain.NewConfigInvalidError("modelRegistry", "failed to parse model registry", err)
	}
	if err := registry.Validate(); err != nil {
		return nil, err
	}
	return registry.Clone(), nil
}

// Save writes the model registry to disk.
func Save(path string, registry *Registry) error {
	if registry == nil {
		return domain.NewValidationErrorWithCode("modelRegistry", "field is required", domain.ErrCodeValidationRequired)
	}
	if err := registry.Validate(); err != nil {
		return err
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return domain.NewConfigInvalidError("modelRegistry", "failed to create model registry directory", err)
	}
	payload, err := json.MarshalIndent(registry, "", "  ")
	if err != nil {
		return domain.NewConfigInvalidError("modelRegistry", "failed to marshal model registry", err)
	}
	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return domain.NewConfigInvalidError("modelRegistry", "failed to write model registry", err)
	}
	return nil
}

// ResolvePath determines the default registry path.
func ResolvePath() string {
	if path := strings.TrimSpace(os.Getenv("AGENT_MANAGER_MODEL_REGISTRY_PATH")); path != "" {
		return path
	}
	root := resolveRepoRoot()
	if root == "" {
		root = "."
	}
	if resolved, err := repocontract.ResolveScenarioPath(root, "agent-manager"); err == nil {
		return filepath.Join(resolved, "config", "model-registry.json")
	}
	return filepath.Join(root, "scenarios", "agent-manager", "config", "model-registry.json")
}

func resolveRepoRoot() string {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return ""
	}
	return root
}
