package aisearch

import (
	"context"
	"testing"
)

// MockRankingConfigProvider injects fixed ranking levers in tests.
type MockRankingConfigProvider struct {
	cfg DiscoverRankingConfig
	err error
}

func (m *MockRankingConfigProvider) Get(_ context.Context) (DiscoverRankingConfig, error) {
	return m.cfg, m.err
}

func TestDefaultDiscoverRankingConfig_Valid(t *testing.T) {
	cfg := DefaultDiscoverRankingConfig()
	if err := ValidateDiscoverRankingConfig(cfg, 0.5); err != nil {
		t.Fatalf("default config should validate against the default threshold: %v", err)
	}
	if cfg.TopicGate <= 0.5 {
		t.Errorf("topicGate (%.2f) must exceed the 0.5 skill threshold", cfg.TopicGate)
	}
}

func TestValidateDiscoverRankingConfig_Bounds(t *testing.T) {
	base := DefaultDiscoverRankingConfig()

	tests := []struct {
		name      string
		mutate    func(*DiscoverRankingConfig)
		threshold float64
		wantErr   bool
	}{
		{"valid defaults", func(*DiscoverRankingConfig) {}, 0.5, false},
		{"gate equals threshold", func(c *DiscoverRankingConfig) { c.TopicGate = 0.5 }, 0.5, true},
		{"gate below threshold", func(c *DiscoverRankingConfig) { c.TopicGate = 0.45 }, 0.5, true},
		{"gate above threshold ok", func(c *DiscoverRankingConfig) { c.TopicGate = 0.6 }, 0.5, false},
		{"gate over 1", func(c *DiscoverRankingConfig) { c.TopicGate = 1.5 }, 0.5, true},
		{"bar zero", func(c *DiscoverRankingConfig) { c.HighConfidenceBar = 0 }, 0.5, true},
		{"bar over 1", func(c *DiscoverRankingConfig) { c.HighConfidenceBar = 1.1 }, 0.5, true},
		{"negative above-pack cap", func(c *DiscoverRankingConfig) { c.MaxIndividualsAbovePack = -1 }, 0.5, true},
		{"zero above-pack cap ok", func(c *DiscoverRankingConfig) { c.MaxIndividualsAbovePack = 0 }, 0.5, false},
		{"zero skill cap", func(c *DiscoverRankingConfig) { c.TopicSkillCap = 0 }, 0.5, true},
		{"gate must beat custom threshold", func(c *DiscoverRankingConfig) { c.TopicGate = 0.55 }, 0.7, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			cfg := base
			tc.mutate(&cfg)
			err := ValidateDiscoverRankingConfig(cfg, tc.threshold)
			if tc.wantErr && err == nil {
				t.Errorf("expected validation error, got nil for %+v (threshold %.2f)", cfg, tc.threshold)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestDiscoverRankingConfigStore_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	store := NewDiscoverRankingConfigStore(dir, 0.5)

	// Missing file → defaults.
	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("Get on empty store: %v", err)
	}
	if got != DefaultDiscoverRankingConfig() {
		t.Errorf("empty store should return defaults, got %+v", got)
	}

	// Persist and reload.
	custom := DiscoverRankingConfig{TopicGate: 0.6, HighConfidenceBar: 0.7, MaxIndividualsAbovePack: 2, TopicSkillCap: 8}
	if err := store.Put(context.Background(), custom); err != nil {
		t.Fatalf("Put: %v", err)
	}
	got, err = store.Get(context.Background())
	if err != nil {
		t.Fatalf("Get after Put: %v", err)
	}
	if got != custom {
		t.Errorf("round-trip mismatch: got %+v, want %+v", got, custom)
	}

	// Invalid config is rejected on Put.
	if err := store.Put(context.Background(), DiscoverRankingConfig{TopicGate: 0.4, HighConfidenceBar: 0.7, MaxIndividualsAbovePack: 1, TopicSkillCap: 8}); err == nil {
		t.Error("expected Put to reject a gate below the threshold")
	}
}
