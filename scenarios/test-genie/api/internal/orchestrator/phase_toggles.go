package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"

	"test-genie/internal/orchestrator/phases"

	"github.com/vrooli/api-core/storage"
)

// PhaseToggle represents a global toggle state for a phase, applied across all scenarios.
// When Disabled is true, the phase is skipped for presets/all-phases runs, but can still
// be explicitly requested (with warnings).
type PhaseToggle struct {
	Disabled bool      `json:"disabled"`
	Reason   string    `json:"reason,omitempty"`
	Owner    string    `json:"owner,omitempty"`
	AddedAt  time.Time `json:"addedAt,omitempty"`
}

// PhaseToggleConfig is the persisted toggle configuration.
type PhaseToggleConfig struct {
	Phases map[string]PhaseToggle `json:"phases"`
}

type phaseToggleStore struct {
	resolver *storage.Resolver
	mu       sync.Mutex
}

const (
	phaseToggleScenarioID = "test-genie"
	phaseToggleFilename   = "phase-toggles.json"
)

func newPhaseToggleStore() *phaseToggleStore {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return nil
	}
	return &phaseToggleStore{resolver: resolver}
}

func (s *phaseToggleStore) path() (string, error) {
	if s == nil || s.resolver == nil {
		return "", nil
	}
	return s.resolver.Path(
		storage.Options{ScenarioID: phaseToggleScenarioID},
		storage.ClassConfig,
		phaseToggleFilename,
	)
}

func (s *phaseToggleStore) Load() (PhaseToggleConfig, error) {
	cfg := PhaseToggleConfig{Phases: map[string]PhaseToggle{}}
	if s == nil {
		return cfg, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	path, err := s.path()
	if err != nil {
		return cfg, fmt.Errorf("resolve phase toggle path: %w", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("read phase toggle file: %w", err)
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse phase toggle file: %w", err)
	}
	cfg = normalizePhaseToggleConfig(cfg, time.Time{})
	return cfg, nil
}

func (s *phaseToggleStore) Save(cfg PhaseToggleConfig) (PhaseToggleConfig, error) {
	if s == nil {
		return cfg, nil
	}
	cfg = normalizePhaseToggleConfig(cfg, time.Now().UTC())

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return cfg, fmt.Errorf("encode phase toggle file: %w", err)
	}

	path, err := s.path()
	if err != nil {
		return cfg, fmt.Errorf("resolve phase toggle path: %w", err)
	}
	if err := storage.WriteFileAtomic(path, data, storage.DefaultFilePerm); err != nil {
		return cfg, fmt.Errorf("write phase toggle file: %w", err)
	}
	return cfg, nil
}

func normalizePhaseToggleConfig(cfg PhaseToggleConfig, now time.Time) PhaseToggleConfig {
	normalized := PhaseToggleConfig{Phases: map[string]PhaseToggle{}}
	for name, toggle := range cfg.Phases {
		key := phases.NormalizeKey(name)
		if key == "" {
			continue
		}
		// Trim strings for cleaner output
		toggle.Reason = strings.TrimSpace(toggle.Reason)
		toggle.Owner = strings.TrimSpace(toggle.Owner)

		if !toggle.Disabled {
			toggle.AddedAt = time.Time{}
		} else if toggle.AddedAt.IsZero() {
			toggle.AddedAt = now
		}

		normalized.Phases[key] = toggle
	}
	return normalized
}
