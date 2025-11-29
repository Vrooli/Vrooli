package config

import (
	"os"
	"path/filepath"
	"testing"
)

// [REQ:SCS-CFG-003] Configuration presets system tests
// [REQ:SCS-CFG-004] Configuration persistence tests

func TestDefaultConfig(t *testing.T) {
	// [REQ:SCS-CFG-004] Test default configuration creation
	cfg := DefaultConfig()

	// Weights should sum to 100
	if cfg.Weights.TotalWeight() != 100 {
		t.Errorf("expected weights to sum to 100, got %d", cfg.Weights.TotalWeight())
	}

	// All components should be enabled by default
	if !cfg.Components.Quality.Enabled {
		t.Error("expected quality component to be enabled by default")
	}
	if !cfg.Components.Coverage.Enabled {
		t.Error("expected coverage component to be enabled by default")
	}
	if !cfg.Components.Quantity.Enabled {
		t.Error("expected quantity component to be enabled by default")
	}
	if !cfg.Components.UI.Enabled {
		t.Error("expected UI component to be enabled by default")
	}

	// Circuit breaker should be enabled
	if !cfg.CircuitBreaker.Enabled {
		t.Error("expected circuit breaker to be enabled by default")
	}
}

func TestWeightNormalize(t *testing.T) {
	// [REQ:SCS-CFG-001] Weight redistribution when components disabled
	tests := []struct {
		name       string
		weights    WeightConfig
		components ComponentConfig
		expected   int // Expected total after normalization
	}{
		{
			name:     "all enabled",
			weights:  WeightConfig{Quality: 50, Coverage: 15, Quantity: 10, UI: 25},
			components: ComponentConfig{
				Quality:  QualityComponentConfig{Enabled: true},
				Coverage: CoverageComponentConfig{Enabled: true},
				Quantity: QuantityComponentConfig{Enabled: true},
				UI:       UIComponentConfig{Enabled: true},
			},
			expected: 100,
		},
		{
			name:     "UI disabled",
			weights:  WeightConfig{Quality: 50, Coverage: 15, Quantity: 10, UI: 25},
			components: ComponentConfig{
				Quality:  QualityComponentConfig{Enabled: true},
				Coverage: CoverageComponentConfig{Enabled: true},
				Quantity: QuantityComponentConfig{Enabled: true},
				UI:       UIComponentConfig{Enabled: false},
			},
			expected: 100,
		},
		{
			name:     "only quality enabled",
			weights:  WeightConfig{Quality: 50, Coverage: 15, Quantity: 10, UI: 25},
			components: ComponentConfig{
				Quality:  QualityComponentConfig{Enabled: true},
				Coverage: CoverageComponentConfig{Enabled: false},
				Quantity: QuantityComponentConfig{Enabled: false},
				UI:       UIComponentConfig{Enabled: false},
			},
			expected: 100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := tt.weights.Normalize(tt.components)
			if normalized.TotalWeight() != tt.expected {
				t.Errorf("expected total weight %d, got %d", tt.expected, normalized.TotalWeight())
			}
		})
	}
}

func TestConfigMerge(t *testing.T) {
	// [REQ:SCS-CFG-002] Per-scenario overrides merge logic
	base := DefaultConfig()

	override := &ScoringConfig{
		Version: "2.0.0",
		Weights: WeightConfig{
			Quality: 60, // Override quality weight
		},
	}

	merged := base.Merge(override)

	// Version should be overridden
	if merged.Version != "2.0.0" {
		t.Errorf("expected version 2.0.0, got %s", merged.Version)
	}

	// Quality weight should be overridden
	if merged.Weights.Quality != 60 {
		t.Errorf("expected quality weight 60, got %d", merged.Weights.Quality)
	}

	// Other weights should remain from base
	if merged.Weights.Coverage != 15 {
		t.Errorf("expected coverage weight 15, got %d", merged.Weights.Coverage)
	}
}

func TestConfigMergeNil(t *testing.T) {
	// [REQ:SCS-CFG-002] Merge with nil override should return base
	base := DefaultConfig()
	merged := base.Merge(nil)

	if merged.Weights.Quality != base.Weights.Quality {
		t.Error("merge with nil override should return base unchanged")
	}
}

func TestValidateConfig(t *testing.T) {
	// [REQ:SCS-CFG-004] Configuration validation
	tests := []struct {
		name    string
		config  ScoringConfig
		wantErr bool
	}{
		{
			name:    "valid default config",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "invalid weights sum",
			config: ScoringConfig{
				Weights: WeightConfig{Quality: 50, Coverage: 20, Quantity: 20, UI: 20},
				CircuitBreaker: CircuitBreakerConfig{
					Enabled:        false,
					FailThreshold:  3,
					RetryIntervalS: 300,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid circuit breaker threshold",
			config: ScoringConfig{
				Weights: WeightConfig{Quality: 50, Coverage: 15, Quantity: 10, UI: 25},
				CircuitBreaker: CircuitBreakerConfig{
					Enabled:        true,
					FailThreshold:  0, // Invalid
					RetryIntervalS: 300,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateConfig(&tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestPresetsExist(t *testing.T) {
	// [REQ:SCS-CFG-003] Verify all presets exist and are valid
	presets := BuiltinPresets()

	if len(presets) == 0 {
		t.Fatal("expected at least one preset")
	}

	expectedPresets := []string{
		PresetDefault,
		PresetSkipE2E,
		PresetCodeQualityOnly,
		PresetMinimal,
		PresetStrict,
	}

	for _, name := range expectedPresets {
		preset := GetPreset(name)
		if preset == nil {
			t.Errorf("expected preset %s to exist", name)
			continue
		}

		// Validate each preset's config
		if err := ValidateConfig(&preset.Config); err != nil {
			t.Errorf("preset %s has invalid config: %v", name, err)
		}
	}
}

func TestGetPresetUnknown(t *testing.T) {
	// [REQ:SCS-CFG-003] Unknown preset returns nil
	preset := GetPreset("unknown-preset")
	if preset != nil {
		t.Error("expected nil for unknown preset")
	}
}

func TestCodeQualityOnlyPreset(t *testing.T) {
	// [REQ:SCS-CFG-003] Code quality only preset disables UI
	preset := GetPreset(PresetCodeQualityOnly)
	if preset == nil {
		t.Fatal("expected code-quality-only preset to exist")
	}

	if preset.Config.Components.UI.Enabled {
		t.Error("expected UI to be disabled in code-quality-only preset")
	}

	// Weights should still sum to 100
	if preset.Config.Weights.TotalWeight() != 100 {
		t.Errorf("expected weights to sum to 100, got %d", preset.Config.Weights.TotalWeight())
	}

	// UI weight should be 0
	if preset.Config.Weights.UI != 0 {
		t.Errorf("expected UI weight to be 0, got %d", preset.Config.Weights.UI)
	}
}

func TestListPresetInfo(t *testing.T) {
	// [REQ:SCS-CFG-003] List preset info returns all presets
	infos := ListPresetInfo()

	if len(infos) < 5 {
		t.Errorf("expected at least 5 presets, got %d", len(infos))
	}

	// Check that each info has required fields
	for _, info := range infos {
		if info.Name == "" {
			t.Error("preset info should have a name")
		}
		if info.Description == "" {
			t.Error("preset info should have a description")
		}
	}
}

func TestLoaderLoadGlobalDefault(t *testing.T) {
	// [REQ:SCS-CFG-004] Loading global config when file doesn't exist returns default
	tempDir := t.TempDir()
	loader := NewLoader(tempDir)

	cfg, err := loader.LoadGlobal()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if cfg == nil {
		t.Fatal("expected non-nil config")
	}

	// Should match default config
	if cfg.Weights.Quality != 50 {
		t.Errorf("expected quality weight 50, got %d", cfg.Weights.Quality)
	}
}

func TestLoaderSaveAndLoadGlobal(t *testing.T) {
	// [REQ:SCS-CFG-004] Save and load global config roundtrip
	tempDir := t.TempDir()

	// Create .vrooli directory in user home for testing
	homeDir, _ := os.UserHomeDir()
	testConfigDir := filepath.Join(homeDir, ".vrooli-test-"+filepath.Base(tempDir))
	os.MkdirAll(testConfigDir, 0755)
	defer os.RemoveAll(testConfigDir)

	loader := &Loader{
		vrooliRoot: tempDir,
		overrides:  make(map[string]*ScenarioOverride),
	}
	// Override the global config path for testing
	configPath := filepath.Join(testConfigDir, GlobalConfigFile)

	cfg := DefaultConfig()
	cfg.Version = "test-1.0.0"

	// Save directly to test path
	os.MkdirAll(filepath.Dir(configPath), 0755)

	err := loader.SaveGlobal(&cfg)
	if err != nil {
		t.Fatalf("failed to save global config: %v", err)
	}

	// Clear cache
	loader.ClearCache()

	// Load should return saved config
	loaded, err := loader.LoadGlobal()
	if err != nil {
		t.Fatalf("failed to load global config: %v", err)
	}

	if loaded.Version != "test-1.0.0" {
		t.Errorf("expected version test-1.0.0, got %s", loaded.Version)
	}
}

func TestLoaderScenarioOverride(t *testing.T) {
	// [REQ:SCS-CFG-002] Per-scenario overrides
	tempDir := t.TempDir()

	// Create scenario directory
	scenarioDir := filepath.Join(tempDir, "scenarios", "test-scenario", ".vrooli")
	os.MkdirAll(scenarioDir, 0755)

	loader := NewLoader(tempDir)

	// Initially no override
	override, err := loader.LoadScenarioOverride("test-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if override != nil {
		t.Error("expected nil override for new scenario")
	}

	// Save an override
	newOverride := &ScenarioOverride{
		Scenario: "test-scenario",
		Preset:   PresetMinimal,
	}

	err = loader.SaveScenarioOverride(newOverride)
	if err != nil {
		t.Fatalf("failed to save override: %v", err)
	}

	// Clear cache and reload
	loader.ClearCache()

	loaded, err := loader.LoadScenarioOverride("test-scenario")
	if err != nil {
		t.Fatalf("failed to load override: %v", err)
	}
	if loaded == nil {
		t.Fatal("expected non-nil override")
	}
	if loaded.Preset != PresetMinimal {
		t.Errorf("expected preset %s, got %s", PresetMinimal, loaded.Preset)
	}
}

func TestLoaderGetEffectiveConfig(t *testing.T) {
	// [REQ:SCS-CFG-002] Effective config merges global and scenario
	tempDir := t.TempDir()

	// Create scenario directory
	scenarioDir := filepath.Join(tempDir, "scenarios", "test-scenario", ".vrooli")
	os.MkdirAll(scenarioDir, 0755)

	loader := NewLoader(tempDir)

	// Get effective config without override
	cfg, err := loader.GetEffectiveConfig("test-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should be default
	if cfg.Weights.Quality != 50 {
		t.Errorf("expected quality weight 50, got %d", cfg.Weights.Quality)
	}

	// Save an override
	overrideConfig := DefaultConfig()
	overrideConfig.Weights.Quality = 60
	override := &ScenarioOverride{
		Scenario:  "test-scenario",
		Overrides: &overrideConfig,
	}
	loader.SaveScenarioOverride(override)

	// Clear cache
	loader.ClearCache()

	// Get effective config with override
	cfg, err = loader.GetEffectiveConfig("test-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should reflect override
	if cfg.Weights.Quality != 60 {
		t.Errorf("expected quality weight 60, got %d", cfg.Weights.Quality)
	}
}

func TestLoaderDeleteScenarioOverride(t *testing.T) {
	// [REQ:SCS-CFG-002] Delete scenario override
	tempDir := t.TempDir()

	// Create scenario directory
	scenarioDir := filepath.Join(tempDir, "scenarios", "test-scenario", ".vrooli")
	os.MkdirAll(scenarioDir, 0755)

	loader := NewLoader(tempDir)

	// Save an override
	override := &ScenarioOverride{
		Scenario: "test-scenario",
		Preset:   PresetStrict,
	}
	loader.SaveScenarioOverride(override)

	// Delete it
	err := loader.DeleteScenarioOverride("test-scenario")
	if err != nil {
		t.Fatalf("failed to delete override: %v", err)
	}

	// Should be nil now
	loader.ClearCache()
	loaded, err := loader.LoadScenarioOverride("test-scenario")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if loaded != nil {
		t.Error("expected nil after delete")
	}
}

func TestStrictPresetHasTighterThresholds(t *testing.T) {
	// [REQ:SCS-CFG-003] Strict preset has tighter thresholds
	strict := GetPreset(PresetStrict)
	defaults := GetPreset(PresetDefault)

	if strict == nil || defaults == nil {
		t.Fatal("expected presets to exist")
	}

	// Strict should have lower fail threshold (trips faster)
	if strict.Config.CircuitBreaker.FailThreshold >= defaults.Config.CircuitBreaker.FailThreshold {
		t.Error("strict preset should have lower fail threshold")
	}

	// Both should have all penalties enabled
	if !strict.Config.Penalties.Enabled {
		t.Error("strict preset should have penalties enabled")
	}
}
