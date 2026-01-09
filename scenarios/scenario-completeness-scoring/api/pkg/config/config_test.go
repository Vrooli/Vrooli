package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Weights.TotalWeight() != 100 {
		t.Errorf("expected weights to sum to 100, got %d", cfg.Weights.TotalWeight())
	}

	if !cfg.Components.Quality.Enabled || !cfg.Components.Coverage.Enabled || !cfg.Components.Quantity.Enabled || !cfg.Components.UI.Enabled {
		t.Fatal("expected all dimensions enabled by default")
	}

	if !cfg.Penalties.Enabled {
		t.Error("expected penalties to be enabled by default")
	}
	if !cfg.CircuitBreaker.Enabled {
		t.Error("expected circuit breaker to be enabled by default")
	}
}

func TestValidateConfig(t *testing.T) {
	tests := []struct {
		name    string
		cfg     *ScoringConfig
		wantErr bool
	}{
		{name: "nil", cfg: nil, wantErr: true},
		{name: "default valid", cfg: ptr(DefaultConfig()), wantErr: false},
		{
			name: "weights must sum",
			cfg: ptr(func() ScoringConfig {
				c := DefaultConfig()
				c.Weights = WeightConfig{Quality: 50, Coverage: 20, Quantity: 20, UI: 20}
				return c
			}()),
			wantErr: true,
		},
		{
			name: "no enabled dimensions",
			cfg: ptr(func() ScoringConfig {
				c := DefaultConfig()
				c.Components.Quality.Enabled = false
				c.Components.Coverage.Enabled = false
				c.Components.Quantity.Enabled = false
				c.Components.UI.Enabled = false
				return c
			}()),
			wantErr: true,
		},
		{
			name: "enabled dimension needs sub-metrics",
			cfg: ptr(func() ScoringConfig {
				c := DefaultConfig()
				c.Components.Quality.Enabled = true
				c.Components.Quality.RequirementPassRate = false
				c.Components.Quality.TargetPassRate = false
				c.Components.Quality.TestPassRate = false
				return c
			}()),
			wantErr: true,
		},
		{
			name: "penalties enabled needs at least one",
			cfg: ptr(func() ScoringConfig {
				c := DefaultConfig()
				c.Penalties.Enabled = true
				c.Penalties.InsufficientTestCoverage = false
				c.Penalties.InvalidTestLocation = false
				c.Penalties.MonolithicTestFiles = false
				c.Penalties.SingleLayerValidation = false
				c.Penalties.TargetMappingRatio = false
				c.Penalties.SuperficialTestImplementation = false
				c.Penalties.ManualValidations = false
				return c
			}()),
			wantErr: true,
		},
		{
			name: "circuit breaker threshold must be >=1",
			cfg: ptr(func() ScoringConfig {
				c := DefaultConfig()
				c.CircuitBreaker.Enabled = true
				c.CircuitBreaker.FailThreshold = 0
				return c
			}()),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(tt.cfg)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoaderSaveAndLoadGlobal(t *testing.T) {
	tempDir := t.TempDir()

	originalHome := os.Getenv("HOME")
	t.Cleanup(func() { _ = os.Setenv("HOME", originalHome) })
	_ = os.Setenv("HOME", tempDir)

	loader := NewLoader(tempDir)
	cfg := DefaultConfig()
	cfg.Version = "test-1.0.0"

	if err := loader.SaveGlobal(&cfg); err != nil {
		t.Fatalf("failed to save global config: %v", err)
	}
	loader.ClearCache()

	loaded, err := loader.LoadGlobal()
	if err != nil {
		t.Fatalf("failed to load global config: %v", err)
	}
	if loaded == nil || loaded.Version != "test-1.0.0" {
		t.Fatalf("expected loaded version test-1.0.0, got %#v", loaded)
	}

	expectedPath := filepath.Join(tempDir, VrooliConfigDir, GlobalConfigFile)
	if _, err := os.Stat(expectedPath); err != nil {
		t.Fatalf("expected config file at %s: %v", expectedPath, err)
	}
}

func ptr[T any](v T) *T { return &v }

