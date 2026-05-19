package performance

import (
	"time"

	"test-genie/internal/shared"
)

// Config holds configuration for performance validation.
type Config struct {
	// ScenarioDir is the absolute path to the scenario directory.
	ScenarioDir string

	// ScenarioName is the name of the scenario (typically the directory name).
	ScenarioName string

	// Expectations holds custom validation expectations from .vrooli/testing.json.
	Expectations *Expectations

	// UIURL is the base URL for the scenario UI (for Lighthouse audits).
	// If empty, Lighthouse audits are skipped.
	UIURL string
}

// Expectations holds the configuration for performance validation, loaded from
// .vrooli/testing.json. It specifies thresholds and validation behavior.
type Expectations struct {
	// GoBuildMaxDuration is the maximum allowed Go build duration.
	// Defaults to 90 seconds.
	GoBuildMaxDuration time.Duration

	// UIBuildMaxDuration is the maximum allowed UI build duration.
	// Defaults to 180 seconds.
	UIBuildMaxDuration time.Duration

	// RequireGoBuild specifies whether Go build must succeed.
	// Defaults to true.
	RequireGoBuild bool

	// RequireUIBuild specifies whether UI build must succeed.
	// Defaults to false (only runs if ui/ directory exists).
	RequireUIBuild bool
}

// configSection represents the performance section of .vrooli/testing.json.
type configSection struct {
	GoBuildMaxSeconds *int  `json:"go_build_max_seconds"`
	UIBuildMaxSeconds *int  `json:"ui_build_max_seconds"`
	RequireGoBuild    *bool `json:"require_go_build"`
	RequireUIBuild    *bool `json:"require_ui_build"`
}

// LoadExpectations reads performance validation expectations from .vrooli/testing.json.
// If the file doesn't exist or has no performance section, default expectations are returned.
func LoadExpectations(scenarioDir string) (*Expectations, error) {
	exp := DefaultExpectations()

	section, err := shared.LoadPhaseConfig(scenarioDir, "performance", configSection{})
	if err != nil {
		return nil, err
	}

	if section.GoBuildMaxSeconds != nil {
		exp.GoBuildMaxDuration = time.Duration(*section.GoBuildMaxSeconds) * time.Second
	}
	if section.UIBuildMaxSeconds != nil {
		exp.UIBuildMaxDuration = time.Duration(*section.UIBuildMaxSeconds) * time.Second
	}
	if section.RequireGoBuild != nil {
		exp.RequireGoBuild = *section.RequireGoBuild
	}
	if section.RequireUIBuild != nil {
		exp.RequireUIBuild = *section.RequireUIBuild
	}

	return exp, nil
}

// DefaultExpectations returns the default performance validation expectations.
func DefaultExpectations() *Expectations {
	return &Expectations{
		GoBuildMaxDuration: 90 * time.Second,
		UIBuildMaxDuration: 180 * time.Second,
		RequireGoBuild:     true,
		RequireUIBuild:     false,
	}
}
