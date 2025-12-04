package performance

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
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

// configDocument represents the structure of .vrooli/testing.json.
type configDocument struct {
	Performance performanceConfigSection `json:"performance"`
}

type performanceConfigSection struct {
	GoBuildMaxSeconds *int  `json:"go_build_max_seconds"`
	UIBuildMaxSeconds *int  `json:"ui_build_max_seconds"`
	RequireGoBuild    *bool `json:"require_go_build"`
	RequireUIBuild    *bool `json:"require_ui_build"`
}

// LoadExpectations reads performance validation expectations from .vrooli/testing.json.
// If the file doesn't exist or has no performance section, default expectations are returned.
func LoadExpectations(scenarioDir string) (*Expectations, error) {
	exp := DefaultExpectations()
	configPath := filepath.Join(scenarioDir, ".vrooli", "testing.json")

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return exp, nil
		}
		return nil, fmt.Errorf("failed to read %s: %w", configPath, err)
	}

	var doc configDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("failed to parse %s: %w", configPath, err)
	}

	if doc.Performance.GoBuildMaxSeconds != nil {
		exp.GoBuildMaxDuration = time.Duration(*doc.Performance.GoBuildMaxSeconds) * time.Second
	}
	if doc.Performance.UIBuildMaxSeconds != nil {
		exp.UIBuildMaxDuration = time.Duration(*doc.Performance.UIBuildMaxSeconds) * time.Second
	}
	if doc.Performance.RequireGoBuild != nil {
		exp.RequireGoBuild = *doc.Performance.RequireGoBuild
	}
	if doc.Performance.RequireUIBuild != nil {
		exp.RequireUIBuild = *doc.Performance.RequireUIBuild
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
