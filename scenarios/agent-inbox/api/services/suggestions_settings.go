// Package services provides business logic orchestration.
// This file manages persisted suggestions settings in a git-tracked config file.
package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// SuggestionsAutoSuggestSettings controls auto-suggest behavior in the UI.
type SuggestionsAutoSuggestSettings struct {
	Enabled        bool `json:"enabled"`
	DebounceMS     int  `json:"debounceMs"`
	ThrottleMS     int  `json:"throttleMs"`
	MinInputLength int  `json:"minInputLength"`
	MinScore       int  `json:"minScorePercent"`
	MaxSuggestions int  `json:"maxSuggestions"`
}

// SuggestionsSettings contains server-backed suggestions configuration.
type SuggestionsSettings struct {
	AutoSuggest SuggestionsAutoSuggestSettings `json:"autoSuggest"`
}

// SuggestionsSettingsService reads/writes suggestions config with validation.
type SuggestionsSettingsService struct {
	path string
	mu   sync.RWMutex
}

// DefaultSuggestionsSettings returns the default configuration.
func DefaultSuggestionsSettings() SuggestionsSettings {
	return SuggestionsSettings{
		AutoSuggest: SuggestionsAutoSuggestSettings{
			Enabled:        true,
			DebounceMS:     900,
			ThrottleMS:     10000,
			MinInputLength: 10,
			MinScore:       35,
			MaxSuggestions: 5,
		},
	}
}

// NewSuggestionsSettingsService creates a file-backed suggestions settings service.
func NewSuggestionsSettingsService(path string) *SuggestionsSettingsService {
	return &SuggestionsSettingsService{path: path}
}

// Get returns validated settings. If file is missing, defaults are created and returned.
func (s *SuggestionsSettingsService) Get() (SuggestionsSettings, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	settings, err := s.readUnlocked()
	if err == nil {
		return settings, nil
	}
	if !os.IsNotExist(err) {
		return SuggestionsSettings{}, err
	}

	// Upgrade path: create missing file from defaults.
	defaults := DefaultSuggestionsSettings()
	if err := s.writeUnlocked(defaults); err != nil {
		return SuggestionsSettings{}, err
	}
	return defaults, nil
}

// Set validates and persists settings.
func (s *SuggestionsSettingsService) Set(settings SuggestionsSettings) (SuggestionsSettings, error) {
	if err := validateSuggestionsSettings(settings); err != nil {
		return SuggestionsSettings{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.writeUnlocked(settings); err != nil {
		return SuggestionsSettings{}, err
	}
	return settings, nil
}

func (s *SuggestionsSettingsService) readUnlocked() (SuggestionsSettings, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		return SuggestionsSettings{}, err
	}

	var settings SuggestionsSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return SuggestionsSettings{}, fmt.Errorf("failed to parse suggestions settings: %w", err)
	}
	if err := validateSuggestionsSettings(settings); err != nil {
		return SuggestionsSettings{}, err
	}
	return settings, nil
}

func (s *SuggestionsSettingsService) writeUnlocked(settings SuggestionsSettings) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode suggestions settings: %w", err)
	}
	data = append(data, '\n')

	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("failed to ensure config directory: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0o644); err != nil {
		return fmt.Errorf("failed to write suggestions settings: %w", err)
	}
	return nil
}

func validateSuggestionsSettings(settings SuggestionsSettings) error {
	cfg := settings.AutoSuggest
	if cfg.DebounceMS < 100 || cfg.DebounceMS > 10000 {
		return fmt.Errorf("autoSuggest.debounceMs must be between 100 and 10000")
	}
	if cfg.ThrottleMS < 1000 || cfg.ThrottleMS > 120000 {
		return fmt.Errorf("autoSuggest.throttleMs must be between 1000 and 120000")
	}
	if cfg.MinInputLength < 1 || cfg.MinInputLength > 200 {
		return fmt.Errorf("autoSuggest.minInputLength must be between 1 and 200")
	}
	if cfg.MinScore < 0 || cfg.MinScore > 100 {
		return fmt.Errorf("autoSuggest.minScorePercent must be between 0 and 100")
	}
	if cfg.MaxSuggestions < 1 || cfg.MaxSuggestions > 20 {
		return fmt.Errorf("autoSuggest.maxSuggestions must be between 1 and 20")
	}
	return nil
}
