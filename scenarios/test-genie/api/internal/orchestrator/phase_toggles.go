package orchestrator

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	path string
	mu   sync.Mutex
}

func newPhaseToggleStore(projectRoot string) *phaseToggleStore {
	return &phaseToggleStore{
		path: filepath.Join(projectRoot, ".vrooli", "test-genie-phase-toggles.json"),
	}
}

func (s *phaseToggleStore) Load() (PhaseToggleConfig, error) {
	cfg := PhaseToggleConfig{Phases: map[string]PhaseToggle{}}
	if s == nil {
		return cfg, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := os.ReadFile(s.path)
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

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return cfg, fmt.Errorf("prepare phase toggle directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return cfg, fmt.Errorf("encode phase toggle file: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return cfg, fmt.Errorf("write phase toggle file: %w", err)
	}
	return cfg, nil
}

func normalizePhaseToggleConfig(cfg PhaseToggleConfig, now time.Time) PhaseToggleConfig {
	normalized := PhaseToggleConfig{Phases: map[string]PhaseToggle{}}
	for name, toggle := range cfg.Phases {
		key := normalizePhaseName(name)
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
