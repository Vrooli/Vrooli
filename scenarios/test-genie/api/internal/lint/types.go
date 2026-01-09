package lint

import (
	"fmt"
	"strings"

	"test-genie/internal/shared"
)

// Re-export shared types for consistency across packages.
type (
	FailureClass    = shared.FailureClass
	ObservationType = shared.ObservationType
	Observation     = shared.Observation
	Result          = shared.Result
)

// Re-export constants.
const (
	FailureClassNone              = shared.FailureClassNone
	FailureClassMisconfiguration  = shared.FailureClassMisconfiguration
	FailureClassSystem            = shared.FailureClassSystem
	FailureClassMissingDependency = shared.FailureClassMissingDependency

	ObservationSection = shared.ObservationSection
	ObservationSuccess = shared.ObservationSuccess
	ObservationWarning = shared.ObservationWarning
	ObservationError   = shared.ObservationError
	ObservationInfo    = shared.ObservationInfo
	ObservationSkip    = shared.ObservationSkip
)

// Re-export constructor functions.
var (
	NewSectionObservation = shared.NewSectionObservation
	NewSuccessObservation = shared.NewSuccessObservation
	NewWarningObservation = shared.NewWarningObservation
	NewErrorObservation   = shared.NewErrorObservation
	NewInfoObservation    = shared.NewInfoObservation
	NewSkipObservation    = shared.NewSkipObservation

	OK                    = shared.OK
	OKWithCount           = shared.OKWithCount
	Fail                  = shared.Fail
	FailMisconfiguration  = shared.FailMisconfiguration
	FailMissingDependency = shared.FailMissingDependency
	FailSystem            = shared.FailSystem
)

// Severity indicates the severity of a lint issue.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// Issue represents a single lint or type error finding.
type Issue struct {
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Column   int      `json:"column,omitempty"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	Rule     string   `json:"rule,omitempty"`
	Source   string   `json:"source"` // e.g., "golangci-lint", "tsc", "eslint"
}

// RunResult is an alias for the generic shared.RunResult with LintSummary.
type RunResult = shared.RunResult[LintSummary]

// LintSummary tracks lint validation counts by language.
type LintSummary struct {
	GoChecked     bool `json:"goChecked"`
	NodeChecked   bool `json:"nodeChecked"`
	PythonChecked bool `json:"pythonChecked"`

	GoIssues     int `json:"goIssues"`
	NodeIssues   int `json:"nodeIssues"`
	PythonIssues int `json:"pythonIssues"`

	TypeErrors int `json:"typeErrors"` // Across all languages
	LintErrors int `json:"lintErrors"` // Across all languages (warnings)
}

// TotalChecks returns the number of languages checked.
func (s LintSummary) TotalChecks() int {
	count := 0
	if s.GoChecked {
		count++
	}
	if s.NodeChecked {
		count++
	}
	if s.PythonChecked {
		count++
	}
	return count
}

// TotalIssues returns the total number of issues found.
func (s LintSummary) TotalIssues() int {
	return s.GoIssues + s.NodeIssues + s.PythonIssues
}

// HasTypeErrors returns true if any type errors were found.
func (s LintSummary) HasTypeErrors() bool {
	return s.TypeErrors > 0
}

// String returns a human-readable summary.
func (s LintSummary) String() string {
	var parts []string
	if s.GoChecked {
		parts = append(parts, fmt.Sprintf("Go: %d issues", s.GoIssues))
	}
	if s.NodeChecked {
		parts = append(parts, fmt.Sprintf("Node: %d issues", s.NodeIssues))
	}
	if s.PythonChecked {
		parts = append(parts, fmt.Sprintf("Python: %d issues", s.PythonIssues))
	}
	if len(parts) == 0 {
		return "no languages checked"
	}
	return strings.Join(parts, ", ")
}

// LanguageResult holds the result of linting a single language.
type LanguageResult struct {
	Language     string        `json:"language"`
	Success      bool          `json:"success"`
	Issues       []Issue       `json:"issues,omitempty"`
	TypeErrors   int           `json:"typeErrors"`
	LintWarnings int           `json:"lintWarnings"`
	ToolsUsed    []string      `json:"toolsUsed"`
	Skipped      bool          `json:"skipped"`
	SkipReason   string        `json:"skipReason,omitempty"`
	Observations []Observation `json:"observations,omitempty"`
}

// LookupFunc is a function that looks up a command by name.
// Re-exported from shared for convenience.
type LookupFunc = shared.LookupFunc
