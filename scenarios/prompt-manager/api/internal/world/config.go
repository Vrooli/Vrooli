// Package world persists the 3D world's operator configuration and per-scene
// layout overrides, and fans swarm signals out to feed subscribers.
// DOC: docs/reference/api-endpoints.md#world
// DOC: docs/concepts/WORLD-ARCHITECTURE.md
package world

import (
	"fmt"
	"path/filepath"
	"time"

	"prompt-manager/internal/store"
)

// Config is the operator's world preferences, shared across browsers.
type Config struct {
	Scene           string  `json:"scene"`
	QualityProfile  string  `json:"qualityProfile"`
	QualityAuto     bool    `json:"qualityAuto"`
	PeriodMode      string  `json:"periodMode"`
	TwoDMode        bool    `json:"twoDMode"`
	ShowDiagnostics bool    `json:"showDiagnostics"`
	Scale           float64 `json:"scale"`
	UpdatedAt       string  `json:"updatedAt,omitempty"`
}

var (
	validScenes   = map[string]bool{"park": true, "office": true}
	validProfiles = map[string]bool{"low": true, "medium": true, "high": true, "ultra": true}
	validPeriods  = map[string]bool{"clock": true, "dawn": true, "day": true, "dusk": true, "night": true}
)

const (
	minScale = 0.25
	maxScale = 4.0
)

// DefaultConfig is what a fresh install sees.
func DefaultConfig() Config {
	return Config{Scene: "park", QualityProfile: "high", QualityAuto: true, PeriodMode: "clock", Scale: 1}
}

// Validate checks every field against its allowed set.
func (c Config) Validate() error {
	if !validScenes[c.Scene] {
		return fmt.Errorf("scene must be park or office, got %q", c.Scene)
	}
	if !validProfiles[c.QualityProfile] {
		return fmt.Errorf("qualityProfile must be low, medium, high or ultra, got %q", c.QualityProfile)
	}
	if !validPeriods[c.PeriodMode] {
		return fmt.Errorf("periodMode must be clock, dawn, day, dusk or night, got %q", c.PeriodMode)
	}
	if c.Scale < minScale || c.Scale > maxScale {
		return fmt.Errorf("scale must be between %v and %v, got %v", minScale, maxScale, c.Scale)
	}
	return nil
}

// Store owns the files under <configDir>/world/.
type Store struct {
	dir string
	now func() time.Time
}

// NewStore creates a store rooted at configDir/world.
func NewStore(configDir string) *Store {
	return &Store{dir: filepath.Join(configDir, "world"), now: func() time.Time { return time.Now().UTC() }}
}

func (s *Store) configPath() string {
	return filepath.Join(s.dir, "config.json")
}

// LoadConfig returns the saved config, or the default when none was saved.
// A malformed file is an error, never silently replaced.
func (s *Store) LoadConfig() (Config, error) {
	path := s.configPath()
	if !store.FileExists(path) {
		return DefaultConfig(), nil
	}
	loaded, err := store.LoadJSON[Config](path)
	if err != nil {
		return Config{}, err
	}
	if err := loaded.Validate(); err != nil {
		return Config{}, fmt.Errorf("%s: %w", path, err)
	}
	return *loaded, nil
}

// SaveConfig validates and writes the config, stamping updatedAt.
func (s *Store) SaveConfig(cfg Config) (Config, error) {
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	cfg.UpdatedAt = s.now().Format(time.RFC3339)
	if err := store.SaveJSON(s.configPath(), &cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}
