package aisearch

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"prompt-manager/store"
)

// DiscoverRankingConfig holds the tunable levers that shape how discover composes
// its skill results in curated (plan-authoring) mode: how strong a topic must be
// to force-include its pack, how strong a lone skill must be to outrank a pack,
// how many such skills may sit above the pack block, and how many skills all
// selected packs may contribute in total. See I1–I7 in
// docs/reference/discovery-pipeline.md for the invariants these govern.
//
// Persisted at store/config/discover-ranking.json; hot-loadable like budgets.json.
type DiscoverRankingConfig struct {
	// TopicGate is the minimum topic match score for a topic's pack to be
	// force-included. Higher = fewer, more-relevant packs. Must exceed the skill
	// similarity threshold (a pack is a larger commitment than one skill).
	TopicGate float64 `json:"topicGate"`
	// HighConfidenceBar is the minimum individual match score for a non-pack skill
	// to rank ABOVE the pack block. Higher = packs dominate the top; lower = strong
	// direct matches surface first.
	HighConfidenceBar float64 `json:"highConfidenceBar"`
	// MaxIndividualsAbovePack caps how many high-confidence non-pack skills sit
	// above the pack block. Bounds how far a flood of strong individuals can push
	// curated packs down. Ignored when no pack is selected (pure-score fallback).
	MaxIndividualsAbovePack int `json:"maxIndividualsAbovePack"`
	// TopicSkillCap bounds the total number of skills all selected packs may
	// contribute. Packs are added whole in descending topic relevance; one that
	// would overflow the cap is skipped so a smaller, more-relevant pack can fit.
	TopicSkillCap int `json:"topicSkillCap"`
}

// DiscoverRankingConfigProvider reads discover ranking configuration.
type DiscoverRankingConfigProvider interface {
	Get(ctx context.Context) (DiscoverRankingConfig, error)
}

// DiscoverRankingConfigStore persists discover ranking config under the scenario store.
type DiscoverRankingConfigStore struct {
	configDir string
	// skillThreshold is the skill similarity floor this config sits above; the
	// topic gate must exceed it (a pack clears a higher bar than a single skill).
	skillThreshold float64
	mu             sync.RWMutex
}

const discoverRankingConfigRelativePath = "config/discover-ranking.json"

// NewDiscoverRankingConfigStore creates a file-backed ranking config store. The
// skillThreshold is the skill similarity floor (AI_SEARCH_THRESHOLD) the topic
// gate must exceed; it is used only for validation, never persisted.
func NewDiscoverRankingConfigStore(configDir string, skillThreshold float64) *DiscoverRankingConfigStore {
	if skillThreshold <= 0 {
		skillThreshold = 0.5
	}
	return &DiscoverRankingConfigStore{configDir: configDir, skillThreshold: skillThreshold}
}

func (s *DiscoverRankingConfigStore) path() string {
	return filepath.Join(s.configDir, discoverRankingConfigRelativePath)
}

// Get loads config from disk or returns defaults if missing.
func (s *DiscoverRankingConfigStore) Get(_ context.Context) (DiscoverRankingConfig, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	path := s.path()
	if !store.FileExists(path) {
		return DefaultDiscoverRankingConfig(), nil
	}

	loaded, err := store.LoadJSON[DiscoverRankingConfig](path)
	if err != nil {
		return DiscoverRankingConfig{}, err
	}
	cfg := *loaded
	if err := ValidateDiscoverRankingConfig(cfg, s.skillThreshold); err != nil {
		return DiscoverRankingConfig{}, err
	}
	return cfg, nil
}

// Put validates and saves config to disk.
func (s *DiscoverRankingConfigStore) Put(_ context.Context, cfg DiscoverRankingConfig) error {
	if err := ValidateDiscoverRankingConfig(cfg, s.skillThreshold); err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	return store.SaveJSON(s.path(), &cfg)
}

// DefaultDiscoverRankingConfig returns the canonical ranking-lever defaults.
// These are the documented starting points; tune from telemetry, not hunches.
func DefaultDiscoverRankingConfig() DiscoverRankingConfig {
	return DiscoverRankingConfig{
		TopicGate:               0.55,
		HighConfidenceBar:       0.65,
		MaxIndividualsAbovePack: 3,
		TopicSkillCap:           12,
	}
}

// ValidateDiscoverRankingConfig checks that levers are within bounds and that the
// topic gate exceeds the skill similarity threshold it sits above.
func ValidateDiscoverRankingConfig(cfg DiscoverRankingConfig, skillThreshold float64) error {
	if skillThreshold <= 0 {
		skillThreshold = 0.5
	}
	if cfg.TopicGate <= skillThreshold || cfg.TopicGate > 1 {
		return fmt.Errorf("topicGate must be in (%.2f, 1], got %.2f", skillThreshold, cfg.TopicGate)
	}
	if cfg.HighConfidenceBar <= 0 || cfg.HighConfidenceBar > 1 {
		return fmt.Errorf("highConfidenceBar must be in (0, 1], got %.2f", cfg.HighConfidenceBar)
	}
	if cfg.MaxIndividualsAbovePack < 0 {
		return fmt.Errorf("maxIndividualsAbovePack must be >= 0, got %d", cfg.MaxIndividualsAbovePack)
	}
	if cfg.TopicSkillCap <= 0 {
		return fmt.Errorf("topicSkillCap must be > 0, got %d", cfg.TopicSkillCap)
	}
	return nil
}
