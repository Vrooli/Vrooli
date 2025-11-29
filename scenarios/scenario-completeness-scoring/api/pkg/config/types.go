// Package config provides configuration management for scenario completeness scoring.
// [REQ:SCS-CFG-001] Component toggle API
// [REQ:SCS-CFG-002] Per-scenario overrides
// [REQ:SCS-CFG-003] Configuration presets system
// [REQ:SCS-CFG-004] Configuration persistence
package config

// ScoringConfig represents the complete scoring configuration
type ScoringConfig struct {
	Version        string                `json:"version"`
	Components     ComponentConfig       `json:"components"`
	Penalties      PenaltyConfig         `json:"penalties"`
	CircuitBreaker CircuitBreakerConfig  `json:"circuit_breaker"`
	Weights        WeightConfig          `json:"weights"`
}

// ComponentConfig controls which scoring components are enabled
type ComponentConfig struct {
	Quality  QualityComponentConfig  `json:"quality"`
	Coverage CoverageComponentConfig `json:"coverage"`
	Quantity QuantityComponentConfig `json:"quantity"`
	UI       UIComponentConfig       `json:"ui"`
}

// QualityComponentConfig controls quality scoring components
type QualityComponentConfig struct {
	Enabled             bool `json:"enabled"`
	RequirementPassRate bool `json:"requirement_pass_rate"`
	TargetPassRate      bool `json:"target_pass_rate"`
	TestPassRate        bool `json:"test_pass_rate"`
}

// CoverageComponentConfig controls coverage scoring components
type CoverageComponentConfig struct {
	Enabled           bool `json:"enabled"`
	TestCoverageRatio bool `json:"test_coverage_ratio"`
	RequirementDepth  bool `json:"requirement_depth"`
}

// QuantityComponentConfig controls quantity scoring components
type QuantityComponentConfig struct {
	Enabled      bool `json:"enabled"`
	Requirements bool `json:"requirements"`
	Targets      bool `json:"targets"`
	Tests        bool `json:"tests"`
}

// UIComponentConfig controls UI scoring components
type UIComponentConfig struct {
	Enabled             bool `json:"enabled"`
	TemplateDetection   bool `json:"template_detection"`
	ComponentComplexity bool `json:"component_complexity"`
	APIIntegration      bool `json:"api_integration"`
	Routing             bool `json:"routing"`
	CodeVolume          bool `json:"code_volume"`
}

// PenaltyConfig controls which penalties are applied
type PenaltyConfig struct {
	Enabled               bool `json:"enabled"`
	InvalidTestLocation   bool `json:"invalid_test_location"`
	MonolithicTestFiles   bool `json:"monolithic_test_files"`
	SingleLayerValidation bool `json:"single_layer_validation"`
	TargetMappingRatio    bool `json:"target_mapping_ratio"`
	ManualValidations     bool `json:"manual_validations"`
}

// CircuitBreakerConfig controls circuit breaker behavior
type CircuitBreakerConfig struct {
	Enabled        bool `json:"enabled"`
	FailThreshold  int  `json:"fail_threshold"`
	RetryIntervalS int  `json:"retry_interval_seconds"`
	AutoDisable    bool `json:"auto_disable"`
}

// WeightConfig controls the weight distribution for each dimension
type WeightConfig struct {
	Quality  int `json:"quality"`
	Coverage int `json:"coverage"`
	Quantity int `json:"quantity"`
	UI       int `json:"ui"`
}

// ScenarioOverride represents scenario-specific configuration overrides
type ScenarioOverride struct {
	Scenario  string         `json:"scenario"`
	Preset    string         `json:"preset,omitempty"`
	Overrides *ScoringConfig `json:"overrides,omitempty"`
}

// Preset represents a pre-defined configuration
type Preset struct {
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Config      ScoringConfig `json:"config"`
}

// DefaultConfig returns the default scoring configuration with all components enabled
func DefaultConfig() ScoringConfig {
	return ScoringConfig{
		Version: "1.0.0",
		Components: ComponentConfig{
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
		},
		Penalties: PenaltyConfig{
			Enabled:               true,
			InvalidTestLocation:   true,
			MonolithicTestFiles:   true,
			SingleLayerValidation: true,
			TargetMappingRatio:    true,
			ManualValidations:     true,
		},
		CircuitBreaker: CircuitBreakerConfig{
			Enabled:        true,
			FailThreshold:  3,
			RetryIntervalS: 300,
			AutoDisable:    true,
		},
		Weights: WeightConfig{
			Quality:  50,
			Coverage: 15,
			Quantity: 10,
			UI:       25,
		},
	}
}

// Merge merges an override config into a base config
// Non-zero values in override replace values in base
func (c *ScoringConfig) Merge(override *ScoringConfig) ScoringConfig {
	if override == nil {
		return *c
	}

	result := *c

	// Version
	if override.Version != "" {
		result.Version = override.Version
	}

	// Weights (only override if non-zero)
	if override.Weights.Quality > 0 {
		result.Weights.Quality = override.Weights.Quality
	}
	if override.Weights.Coverage > 0 {
		result.Weights.Coverage = override.Weights.Coverage
	}
	if override.Weights.Quantity > 0 {
		result.Weights.Quantity = override.Weights.Quantity
	}
	if override.Weights.UI > 0 {
		result.Weights.UI = override.Weights.UI
	}

	// Circuit breaker
	if override.CircuitBreaker.FailThreshold > 0 {
		result.CircuitBreaker.FailThreshold = override.CircuitBreaker.FailThreshold
	}
	if override.CircuitBreaker.RetryIntervalS > 0 {
		result.CircuitBreaker.RetryIntervalS = override.CircuitBreaker.RetryIntervalS
	}

	return result
}

// TotalWeight returns the sum of all dimension weights
func (w WeightConfig) TotalWeight() int {
	return w.Quality + w.Coverage + w.Quantity + w.UI
}

// Normalize redistributes weights proportionally when some components are disabled
func (w WeightConfig) Normalize(components ComponentConfig) WeightConfig {
	// Count enabled dimensions and their original weights
	var enabledTotal int
	if components.Quality.Enabled {
		enabledTotal += w.Quality
	}
	if components.Coverage.Enabled {
		enabledTotal += w.Coverage
	}
	if components.Quantity.Enabled {
		enabledTotal += w.Quantity
	}
	if components.UI.Enabled {
		enabledTotal += w.UI
	}

	if enabledTotal == 0 || enabledTotal == 100 {
		return w
	}

	// Redistribute proportionally
	multiplier := 100.0 / float64(enabledTotal)
	result := WeightConfig{}

	if components.Quality.Enabled {
		result.Quality = int(float64(w.Quality) * multiplier)
	}
	if components.Coverage.Enabled {
		result.Coverage = int(float64(w.Coverage) * multiplier)
	}
	if components.Quantity.Enabled {
		result.Quantity = int(float64(w.Quantity) * multiplier)
	}
	if components.UI.Enabled {
		result.UI = int(float64(w.UI) * multiplier)
	}

	// Adjust for rounding to ensure total is exactly 100
	diff := 100 - result.TotalWeight()
	if diff != 0 {
		// Add difference to the largest enabled component
		if components.Quality.Enabled && result.Quality >= result.Coverage && result.Quality >= result.Quantity && result.Quality >= result.UI {
			result.Quality += diff
		} else if components.UI.Enabled && result.UI >= result.Coverage && result.UI >= result.Quantity {
			result.UI += diff
		} else if components.Coverage.Enabled && result.Coverage >= result.Quantity {
			result.Coverage += diff
		} else if components.Quantity.Enabled {
			result.Quantity += diff
		}
	}

	return result
}
