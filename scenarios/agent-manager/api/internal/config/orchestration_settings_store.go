package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"sync"

	repocontract "github.com/vrooli/repo-contract-go"
)

// OrchestrationSettingsStore provides thread-safe access to orchestration
// settings backed by a JSON file on disk. Falls back to defaults when the
// file does not exist.
type OrchestrationSettingsStore struct {
	path     string
	mu       sync.RWMutex
	settings OrchestrationSettings
}

// NewOrchestrationSettingsStore creates a store at the given path. If the file
// exists it is loaded and validated; if missing, defaults are written to disk.
func NewOrchestrationSettingsStore(path string) (*OrchestrationSettingsStore, error) {
	s := &OrchestrationSettingsStore{path: path}

	data, err := os.ReadFile(path)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, NewInvalid("orchestrationSettings", "failed to read orchestration settings", err)
		}
		// File missing — seed with defaults.
		s.settings = DefaultOrchestrationSettings()
		if err := s.writeToDisk(s.settings); err != nil {
			return nil, err
		}
		return s, nil
	}

	var settings OrchestrationSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, NewInvalid("orchestrationSettings", "failed to parse orchestration settings", err)
	}
	if err := settings.Validate(); err != nil {
		return nil, err
	}
	s.settings = settings
	return s, nil
}

// Get returns a copy of the current settings.
func (s *OrchestrationSettingsStore) Get() OrchestrationSettings {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.settings
}

// Update validates, persists atomically, and updates the in-memory settings.
func (s *OrchestrationSettingsStore) Update(settings OrchestrationSettings) error {
	if err := settings.Validate(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.writeToDisk(settings); err != nil {
		return err
	}
	s.settings = settings
	return nil
}

// Reset restores default settings to disk and memory.
func (s *OrchestrationSettingsStore) Reset() error {
	defaults := DefaultOrchestrationSettings()
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.writeToDisk(defaults); err != nil {
		return err
	}
	s.settings = defaults
	return nil
}

// writeToDisk atomically writes settings to s.path via a temp file + rename.
// Caller must hold s.mu (or be in the constructor before the store is shared).
func (s *OrchestrationSettingsStore) writeToDisk(settings OrchestrationSettings) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return NewInvalid("orchestrationSettings", "failed to create directory", err)
	}
	payload, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return NewInvalid("orchestrationSettings", "failed to marshal settings", err)
	}
	payload = append(payload, '\n')
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, payload, 0o644); err != nil {
		return NewInvalid("orchestrationSettings", "failed to write temp file", err)
	}
	if err := os.Rename(tmp, s.path); err != nil {
		return NewInvalid("orchestrationSettings", "failed to rename temp file", err)
	}
	return nil
}

// ResolveOrchestrationSettingsPath determines the settings file path.
// Checks ORCHESTRATION_SETTINGS_PATH env var first, then falls back to
// VROOLI_ROOT/scenarios/agent-manager/config/orchestration.json.
func ResolveOrchestrationSettingsPath() string {
	path, _ := os.LookupEnv("ORCHESTRATION_SETTINGS_PATH")
	if path = strings.TrimSpace(path); path != "" {
		return path
	}
	root := resolveRepoRoot()
	if root == "" {
		root = "."
	}
	if resolved, err := repocontract.ResolveScenarioPath(root, "agent-manager"); err == nil {
		return filepath.Join(resolved, "config", "orchestration.json")
	}
	return filepath.Join(root, "scenarios", "agent-manager", "config", "orchestration.json")
}

func resolveRepoRoot() string {
	root, err := repocontract.ResolveRepoRoot()
	if err != nil {
		return ""
	}
	return root
}
