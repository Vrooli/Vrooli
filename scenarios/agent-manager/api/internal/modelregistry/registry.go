package modelregistry

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"agent-manager/internal/domain"
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

// RunnerModelRegistry holds model choices and preset mappings for a runner.
type RunnerModelRegistry struct {
	Models  []ModelOption     `json:"models"`
	Presets map[string]string `json:"presets,omitempty"`
}

// Registry contains model catalog data for all runners.
type Registry struct {
	Version int                            `json:"version"`
	FallbackRunnerTypes []string           `json:"fallbackRunnerTypes,omitempty"`
	Runners map[string]RunnerModelRegistry `json:"runners"`
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
		presets := make(map[string]string, len(runner.Presets))
		for presetKey, modelID := range runner.Presets {
			presets[presetKey] = modelID
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

		seen := make(map[string]struct{}, len(runner.Models))
		for _, model := range runner.Models {
			id := strings.TrimSpace(model.ID)
			if id == "" {
				return domain.NewValidationError("modelRegistry.runners."+runnerKey+".models", "model id cannot be empty")
			}
			if _, exists := seen[id]; exists {
				return domain.NewValidationError("modelRegistry.runners."+runnerKey+".models", fmt.Sprintf("duplicate model id %s", id))
			}
			seen[id] = struct{}{}
		}

		for presetKey, modelID := range runner.Presets {
			key := strings.ToUpper(strings.TrimSpace(presetKey))
			if key == "" {
				return domain.NewValidationError("modelRegistry.runners."+runnerKey+".presets", "preset key cannot be empty")
			}
			if !isKnownPreset(key) {
				return domain.NewValidationError("modelRegistry.runners."+runnerKey+".presets", fmt.Sprintf("invalid preset key %s", key))
			}
			modelID = strings.TrimSpace(modelID)
			if modelID == "" {
				return domain.NewValidationError("modelRegistry.runners."+runnerKey+".presets", "preset model id cannot be empty")
			}
			if _, exists := seen[modelID]; !exists {
				return domain.NewValidationError("modelRegistry.runners."+runnerKey+".presets", fmt.Sprintf("unknown model id %s", modelID))
			}
		}
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

func isKnownPreset(preset string) bool {
	switch preset {
	case "FAST", "CHEAP", "SMART":
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

func (s *Store) ResolvePreset(runner string, preset string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.registry == nil {
		return "", false
	}
	runnerConfig, ok := s.registry.Runners[runner]
	if !ok {
		return "", false
	}
	modelID, ok := runnerConfig.Presets[strings.ToUpper(preset)]
	return modelID, ok
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
	root := strings.TrimSpace(os.Getenv("VROOLI_ROOT"))
	if root == "" {
		home, _ := os.UserHomeDir()
		if home == "" {
			home = "."
		}
		root = filepath.Join(home, "Vrooli")
	}
	return filepath.Join(root, "scenarios", "agent-manager", "config", "model-registry.json")
}
