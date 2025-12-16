package config

// [REQ:SCS-CFG-003] Configuration presets system

// PresetName constants for pre-defined configurations
const (
	PresetDefault         = "default"
	PresetSkipE2E         = "skip-e2e"
	PresetCodeQualityOnly = "code-quality-only"
	PresetMinimal         = "minimal"
	PresetStrict          = "strict"
)

// BuiltinPresets returns all built-in configuration presets
func BuiltinPresets() []Preset {
	return []Preset{
		DefaultPreset(),
		SkipE2EPreset(),
		CodeQualityOnlyPreset(),
		MinimalPreset(),
		StrictPreset(),
	}
}

// GetPreset returns a preset by name, or nil if not found
func GetPreset(name string) *Preset {
	presets := BuiltinPresets()
	for _, p := range presets {
		if p.Name == name {
			return &p
		}
	}
	return nil
}

// DefaultPreset returns the default preset with all components enabled
func DefaultPreset() Preset {
	return Preset{
		Name:        PresetDefault,
		Description: "Enable everything: all scoring components, sub-metrics, penalties, and circuit breaker",
		Config:      DefaultConfig(),
	}
}

// SkipE2EPreset returns a preset that disables E2E/Lighthouse metrics
// Useful when E2E infrastructure is not available
func SkipE2EPreset() Preset {
	cfg := DefaultConfig()
	cfg.Components.UI.Enabled = true
	// UI scoring still enabled, but we adjust weights to compensate
	// The circuit breaker should auto-disable failing collectors
	cfg.CircuitBreaker.AutoDisable = true

	return Preset{
		Name:        PresetSkipE2E,
		Description: "Full scoring but gracefully handles missing E2E/Lighthouse infrastructure",
		Config:      cfg,
	}
}

// CodeQualityOnlyPreset returns a preset focused on code quality metrics
// Disables UI scoring entirely
func CodeQualityOnlyPreset() Preset {
	cfg := DefaultConfig()
	cfg.Components.UI = UIComponentConfig{
		Enabled:             false,
		TemplateDetection:   false,
		ComponentComplexity: false,
		APIIntegration:      false,
		Routing:             false,
		CodeVolume:          false,
	}

	// Redistribute weights: 50 + 15 + 10 + 25 = 100
	// With UI disabled: Quality gets more weight
	cfg.Weights = WeightConfig{
		Quality:  67, // 50 + (25 * 50/75) ≈ 67
		Coverage: 20, // 15 + (25 * 15/75) = 20
		Quantity: 13, // 10 + (25 * 10/75) ≈ 13
		UI:       0,
	}

	return Preset{
		Name:        PresetCodeQualityOnly,
		Description: "Focus on code quality metrics only, ignoring UI scoring",
		Config:      cfg,
	}
}

// MinimalPreset returns a minimal scoring preset with fewer components
// Good for early-stage scenarios that haven't implemented all features
func MinimalPreset() Preset {
	cfg := DefaultConfig()

	// Disable depth requirements
	cfg.Components.Coverage.RequirementDepth = false

	// Disable stricter penalties
	cfg.Penalties.SingleLayerValidation = false
	cfg.Penalties.TargetMappingRatio = false

	// Keep UI but with reduced strictness
	cfg.Components.UI.Routing = false
	cfg.Components.UI.CodeVolume = false

	return Preset{
		Name:        PresetMinimal,
		Description: "Minimal scoring for early-stage scenarios with reduced strictness",
		Config:      cfg,
	}
}

// StrictPreset returns a strict scoring preset with all penalties enabled
// and tighter thresholds
func StrictPreset() Preset {
	cfg := DefaultConfig()

	// All components enabled
	cfg.Components = ComponentConfig{
		Quality: QualityComponentConfig{
			Enabled:             true,
			RequirementPassRate: true,
			TargetPassRate:      true,
			TestPassRate:        true,
		},
		Coverage: CoverageComponentConfig{
			Enabled:           true,
			TestCoverageRatio: true,
			RequirementDepth:  true,
		},
		Quantity: QuantityComponentConfig{
			Enabled:      true,
			Requirements: true,
			Targets:      true,
			Tests:        true,
		},
		UI: UIComponentConfig{
			Enabled:             true,
			TemplateDetection:   true,
			ComponentComplexity: true,
			APIIntegration:      true,
			Routing:             true,
			CodeVolume:          true,
		},
	}

	// All penalties enabled
	cfg.Penalties = PenaltyConfig{
		Enabled:               true,
		InvalidTestLocation:   true,
		MonolithicTestFiles:   true,
		SingleLayerValidation: true,
		TargetMappingRatio:    true,
		ManualValidations:     true,
	}

	// More sensitive circuit breaker
	cfg.CircuitBreaker = CircuitBreakerConfig{
		Enabled:        true,
		FailThreshold:  2, // Trip faster
		RetryIntervalS: 600,
		AutoDisable:    true,
	}

	return Preset{
		Name:        PresetStrict,
		Description: "Strict scoring with all penalties and tighter thresholds",
		Config:      cfg,
	}
}

// PresetInfo returns summary information about a preset
type PresetInfo struct {
	Name          string   `json:"name"`
	Description   string   `json:"description"`
	EnabledDims   []string `json:"enabled_dimensions"`
	PenaltiesOn   bool     `json:"penalties_enabled"`
	CircuitBreaker bool    `json:"circuit_breaker_enabled"`
}

// GetPresetInfo returns summary info for a preset
func GetPresetInfo(preset Preset) PresetInfo {
	var dims []string
	if preset.Config.Components.Quality.Enabled {
		dims = append(dims, "quality")
	}
	if preset.Config.Components.Coverage.Enabled {
		dims = append(dims, "coverage")
	}
	if preset.Config.Components.Quantity.Enabled {
		dims = append(dims, "quantity")
	}
	if preset.Config.Components.UI.Enabled {
		dims = append(dims, "ui")
	}

	return PresetInfo{
		Name:          preset.Name,
		Description:   preset.Description,
		EnabledDims:   dims,
		PenaltiesOn:   preset.Config.Penalties.Enabled,
		CircuitBreaker: preset.Config.CircuitBreaker.Enabled,
	}
}

// ListPresetInfo returns info for all built-in presets
func ListPresetInfo() []PresetInfo {
	presets := BuiltinPresets()
	infos := make([]PresetInfo, len(presets))
	for i, p := range presets {
		infos[i] = GetPresetInfo(p)
	}
	return infos
}
