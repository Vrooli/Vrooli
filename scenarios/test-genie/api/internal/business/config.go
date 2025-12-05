package business

import (
	"test-genie/internal/shared"
)

// Config holds configuration for business validation.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario (typically the directory name).
	ScenarioName string

	// Expectations holds custom validation expectations from .vrooli/testing.json.
	Expectations *Expectations
}

// Expectations holds the configuration for business validation, loaded from
// .vrooli/testing.json. It specifies thresholds and validation behavior.
type Expectations struct {
	// RequireModules specifies whether at least one module must exist.
	// Defaults to true.
	RequireModules bool

	// RequireIndex specifies whether requirements/index.json must exist.
	// Defaults to true.
	RequireIndex bool

	// MinCoveragePercent is the minimum coverage percentage for warnings.
	// Defaults to 80.
	MinCoveragePercent int

	// ErrorCoveragePercent is the coverage percentage below which errors are raised.
	// Defaults to 60.
	ErrorCoveragePercent int
}

// configSection represents the business section of .vrooli/testing.json.
type configSection struct {
	RequireModules       *bool `json:"require_modules"`
	RequireIndex         *bool `json:"require_index"`
	MinCoveragePercent   *int  `json:"min_coverage_percent"`
	ErrorCoveragePercent *int  `json:"error_coverage_percent"`
}

// LoadExpectations reads business validation expectations from .vrooli/testing.json.
// If the file doesn't exist or has no business section, default expectations are returned.
func LoadExpectations(scenarioDir string) (*Expectations, error) {
	exp := DefaultExpectations()

	section, err := shared.LoadPhaseConfig(scenarioDir, "business", configSection{})
	if err != nil {
		return nil, err
	}

	if section.RequireModules != nil {
		exp.RequireModules = *section.RequireModules
	}
	if section.RequireIndex != nil {
		exp.RequireIndex = *section.RequireIndex
	}
	if section.MinCoveragePercent != nil {
		exp.MinCoveragePercent = *section.MinCoveragePercent
	}
	if section.ErrorCoveragePercent != nil {
		exp.ErrorCoveragePercent = *section.ErrorCoveragePercent
	}

	return exp, nil
}

// DefaultExpectations returns the default business validation expectations.
func DefaultExpectations() *Expectations {
	return &Expectations{
		RequireModules:       true,
		RequireIndex:         true,
		MinCoveragePercent:   80,
		ErrorCoveragePercent: 60,
	}
}
