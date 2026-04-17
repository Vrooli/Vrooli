package aisearch

import (
	"context"
	"fmt"
	"path/filepath"
	"prompt-manager/store"
	"sync"
)

// DiscoverFilterConfig controls which skills are excluded from discovery results.
// Persisted at store/config/discover-filters.json.
type DiscoverFilterConfig struct {
	IncludeDrafts bool     `json:"includeDrafts"`          // false = exclude drafts (default)
	ExcludeModes  []string `json:"excludeModes,omitempty"` // skills with ANY of these modes are excluded
	ExcludeIDs    []string `json:"excludeIds,omitempty"`   // specific skill IDs to exclude
	ExcludeTags   []string `json:"excludeTags,omitempty"`  // skills with ANY of these tags are excluded
}

// DiscoverFilterConfigProvider reads discover filter configuration.
type DiscoverFilterConfigProvider interface {
	Get(ctx context.Context) (DiscoverFilterConfig, error)
}

// DiscoverFilterConfigStore persists discover filter config under the scenario store.
type DiscoverFilterConfigStore struct {
	storeDir string
	mu       sync.RWMutex
}

const discoverFilterConfigRelativePath = "config/discover-filters.json"

// NewDiscoverFilterConfigStore creates a file-backed discover filter config store.
func NewDiscoverFilterConfigStore(storeDir string) *DiscoverFilterConfigStore {
	return &DiscoverFilterConfigStore{storeDir: storeDir}
}

func (s *DiscoverFilterConfigStore) path() string {
	return filepath.Join(s.storeDir, discoverFilterConfigRelativePath)
}

// Get loads config from disk or returns defaults if missing.
func (s *DiscoverFilterConfigStore) Get(_ context.Context) (DiscoverFilterConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.path()
	if !store.FileExists(path) {
		return DefaultDiscoverFilterConfig(), nil
	}

	loaded, err := store.LoadJSON[DiscoverFilterConfig](path)
	if err != nil {
		return DiscoverFilterConfig{}, err
	}
	cfg := *loaded
	if err := ValidateDiscoverFilterConfig(cfg); err != nil {
		return DiscoverFilterConfig{}, err
	}
	return cfg, nil
}

// Put validates and saves config to disk.
func (s *DiscoverFilterConfigStore) Put(_ context.Context, cfg DiscoverFilterConfig) error {
	if err := ValidateDiscoverFilterConfig(cfg); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return store.SaveJSON(s.path(), &cfg)
}

// DefaultDiscoverFilterConfig returns the canonical filter defaults.
func DefaultDiscoverFilterConfig() DiscoverFilterConfig {
	return DiscoverFilterConfig{
		IncludeDrafts: false,
		ExcludeModes:  []string{"scope"},
	}
}

const maxExcludeEntries = 500

// ValidateDiscoverFilterConfig checks that filter config entries are within reasonable bounds.
func ValidateDiscoverFilterConfig(cfg DiscoverFilterConfig) error {
	if len(cfg.ExcludeModes) > maxExcludeEntries {
		return fmt.Errorf("excludeModes has %d entries, max is %d", len(cfg.ExcludeModes), maxExcludeEntries)
	}
	if len(cfg.ExcludeIDs) > maxExcludeEntries {
		return fmt.Errorf("excludeIds has %d entries, max is %d", len(cfg.ExcludeIDs), maxExcludeEntries)
	}
	if len(cfg.ExcludeTags) > maxExcludeEntries {
		return fmt.Errorf("excludeTags has %d entries, max is %d", len(cfg.ExcludeTags), maxExcludeEntries)
	}
	return nil
}
