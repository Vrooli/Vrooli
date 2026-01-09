package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"sync"
)

type RulesConfig struct {
	Version      string          `json:"version"`
	EnabledRules map[string]bool `json:"enabled_rules"`
}

type ConfigStore struct {
	path string
	mu   sync.Mutex
}

func NewConfigStore(path string) *ConfigStore {
	return &ConfigStore{path: path}
}

func (s *ConfigStore) Load(ctx context.Context) (RulesConfig, error) {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()

	b, err := os.ReadFile(s.path)
	if err == nil {
		var cfg RulesConfig
		if err := json.Unmarshal(b, &cfg); err != nil {
			return RulesConfig{}, fmt.Errorf("parse rules config: %w", err)
		}
		if cfg.Version == "" {
			cfg.Version = "1.0.0"
		}
		if cfg.EnabledRules == nil {
			cfg.EnabledRules = map[string]bool{}
		}
		cfg = normalizeConfig(cfg)
		return cfg, nil
	}
	if !os.IsNotExist(err) {
		return RulesConfig{}, err
	}

	cfg := defaultRulesConfig()
	if err := s.saveLocked(cfg); err != nil {
		return RulesConfig{}, err
	}
	return cfg, nil
}

func (s *ConfigStore) Save(ctx context.Context, cfg RulesConfig) error {
	_ = ctx

	s.mu.Lock()
	defer s.mu.Unlock()
	cfg = normalizeConfig(cfg)
	return s.saveLocked(cfg)
}

func (s *ConfigStore) saveLocked(cfg RulesConfig) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}

	out, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	out = append(out, '\n')

	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, out, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func defaultRulesConfig() RulesConfig {
	enabled := map[string]bool{}
	for _, rule := range AllRuleDefinitions() {
		if rule.DefaultEnabled {
			enabled[rule.ID] = true
		}
	}
	return RulesConfig{
		Version:      "1.0.0",
		EnabledRules: enabled,
	}
}

func normalizeConfig(cfg RulesConfig) RulesConfig {
	if cfg.Version == "" {
		cfg.Version = "1.0.0"
	}
	if cfg.EnabledRules == nil {
		cfg.EnabledRules = map[string]bool{}
	}

	known := map[string]struct{}{}
	for _, rule := range AllRuleDefinitions() {
		known[rule.ID] = struct{}{}
		if _, ok := cfg.EnabledRules[rule.ID]; !ok {
			cfg.EnabledRules[rule.ID] = rule.DefaultEnabled
		}
	}

	for id := range cfg.EnabledRules {
		if _, ok := known[id]; !ok {
			delete(cfg.EnabledRules, id)
		}
	}

	// Stable ordering when written (map order is random); keep a deterministic projection around for UI/tests.
	keys := make([]string, 0, len(cfg.EnabledRules))
	for k := range cfg.EnabledRules {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	stable := make(map[string]bool, len(cfg.EnabledRules))
	for _, k := range keys {
		stable[k] = cfg.EnabledRules[k]
	}
	cfg.EnabledRules = stable
	return cfg
}
