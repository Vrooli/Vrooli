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
)

// Args holds parsed CLI inputs for the execute command.
type Args struct {
	Scenario    string
	Preset      string
	PhasesCSV   string
	SkipCSV     string
	Phases      []string
	Skip        []string
	RequestID   string
	FailFast    bool
	Stream      bool
	NoStream    bool // Explicitly disable streaming (use spinner instead)
	JSON        bool
	ExtraPhases []string

	// Runtime URLs for Lighthouse and integration testing
	UIURL          string
	BrowserlessURL string
}
