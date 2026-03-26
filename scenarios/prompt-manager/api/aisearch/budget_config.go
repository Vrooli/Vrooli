package aisearch

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"prompt-manager/store"
)

// BudgetConfig maps complexity tiers to character budgets for skill discovery.
type BudgetConfig struct {
	Minor         int `json:"minor"`
	Moderate      int `json:"moderate"`
	Major         int `json:"major"`
	Architectural int `json:"architectural"`
}

// BudgetConfigProvider reads budget configuration.
type BudgetConfigProvider interface {
	Get(ctx context.Context) (BudgetConfig, error)
}

// BudgetConfigStore persists budget config under the scenario store.
type BudgetConfigStore struct {
	storeDir string
	mu       sync.RWMutex
}

const budgetConfigRelativePath = "config/budgets.json"

// NewBudgetConfigStore creates a file-backed budget config store.
func NewBudgetConfigStore(storeDir string) *BudgetConfigStore {
	return &BudgetConfigStore{storeDir: storeDir}
}

func (s *BudgetConfigStore) path() string {
	return filepath.Join(s.storeDir, budgetConfigRelativePath)
}

// Get loads config from disk or returns defaults if missing.
func (s *BudgetConfigStore) Get(_ context.Context) (BudgetConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.path()
	if !store.FileExists(path) {
		return DefaultBudgetConfig(), nil
	}

	loaded, err := store.LoadJSON[BudgetConfig](path)
	if err != nil {
		return BudgetConfig{}, err
	}
	cfg := *loaded
	if err := ValidateBudgetConfig(cfg); err != nil {
		return BudgetConfig{}, err
	}
	return cfg, nil
}

// Put validates and saves config to disk.
func (s *BudgetConfigStore) Put(_ context.Context, cfg BudgetConfig) error {
	if err := ValidateBudgetConfig(cfg); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return store.SaveJSON(s.path(), &cfg)
}

// DefaultBudgetConfig returns the canonical budget defaults.
func DefaultBudgetConfig() BudgetConfig {
	return BudgetConfig{
		Minor:         4000,
		Moderate:      8000,
		Major:         12000,
		Architectural: 18000,
	}
}

// ValidateBudgetConfig checks that all tiers are positive, within bounds, and ascending.
func ValidateBudgetConfig(cfg BudgetConfig) error {
	tiers := []struct {
		name  string
		value int
	}{
		{"minor", cfg.Minor},
		{"moderate", cfg.Moderate},
		{"major", cfg.Major},
		{"architectural", cfg.Architectural},
	}

	prev := 0
	for _, t := range tiers {
		if t.value <= 0 {
			return fmt.Errorf("%s must be > 0, got %d", t.name, t.value)
		}
		if t.value > 200000 {
			return fmt.Errorf("%s must be <= 200000, got %d", t.name, t.value)
		}
		if t.value <= prev {
			return fmt.Errorf("%s (%d) must be greater than the previous tier (%d)", t.name, t.value, prev)
		}
		prev = t.value
	}
	return nil
}

// ForTier returns the budget for a given complexity tier.
func (cfg BudgetConfig) ForTier(tier string) (int, bool) {
	switch tier {
	case "minor":
		return cfg.Minor, true
	case "moderate":
		return cfg.Moderate, true
	case "major":
		return cfg.Major, true
	case "architectural":
		return cfg.Architectural, true
	default:
		return 0, false
	}
}

// ToMap converts the config to a map for compatibility with existing logic.
func (cfg BudgetConfig) ToMap() map[string]int {
	return map[string]int{
		"minor":         cfg.Minor,
		"moderate":      cfg.Moderate,
		"major":         cfg.Major,
		"architectural": cfg.Architectural,
	}
}
