// Package execute provides test suite execution capabilities.
package execute

import (
	execTypes "test-genie/cli/internal/execute"
)

// Re-export types from internal package for external use
type (
	Request      = execTypes.Request
	Response     = execTypes.Response
	Phase        = execTypes.Phase
	PhaseSummary = execTypes.PhaseSummary
	RunHandle    = execTypes.RunHandle
)

// Args holds parsed CLI inputs for the execute command.
type Args struct {
	Scenario               string
	Preset                 string
	PhasesCSV              string
	SkipCSV                string
	Phases                 []string
	Skip                   []string
	PhaseWarnings          []string
	DiagnosticsPreset      string
	CaptureProfile         string
	RetainForEvidence      bool
	RetentionReason        string
	FailFast               bool
	Wait                   bool // Force block-to-completion inline (CI / lifecycle); never auto-background
	JSON                   bool
	JSONL                  bool // Stream canonical newline-delimited phase events
	ExtraPhases            []string
	ScenarioPath           string
	LogicalRepoRoot        string
	LogicalScenarioRelPath string

	// Runtime URLs for Lighthouse and integration testing
	UIURL  string
	APIURL string
}
