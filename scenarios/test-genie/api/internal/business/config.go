package business

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
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

// configDocument represents the structure of .vrooli/testing.json.
type configDocument struct {
	Business businessConfigSection `json:"business"`
}

type businessConfigSection struct {
	RequireModules       *bool `json:"require_modules"`
	RequireIndex         *bool `json:"require_index"`
	MinCoveragePercent   *int  `json:"min_coverage_percent"`
	ErrorCoveragePercent *int  `json:"error_coverage_percent"`
}

// LoadExpectations reads business validation expectations from .vrooli/testing.json.
// If the file doesn't exist or has no business section, default expectations are returned.
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

	if doc.Business.RequireModules != nil {
		exp.RequireModules = *doc.Business.RequireModules
	}
	if doc.Business.RequireIndex != nil {
		exp.RequireIndex = *doc.Business.RequireIndex
	}
	if doc.Business.MinCoveragePercent != nil {
		exp.MinCoveragePercent = *doc.Business.MinCoveragePercent
	}
	if doc.Business.ErrorCoveragePercent != nil {
		exp.ErrorCoveragePercent = *doc.Business.ErrorCoveragePercent
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
