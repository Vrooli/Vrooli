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

// PolicySeverity is the severity used for lint policy findings.
type PolicySeverity string

const (
	PolicySeverityIgnore  PolicySeverity = "ignore"
	PolicySeverityInfo    PolicySeverity = "info"
	PolicySeverityWarning PolicySeverity = "warning"
	PolicySeverityError   PolicySeverity = "error"
)

// Issue represents a single lint or type error finding.
type Issue struct {
	File     string   `json:"file"`
	Line     int      `json:"line"`
	Column   int      `json:"column,omitempty"`
	Message  string   `json:"message"`
	Severity Severity `json:"severity"`
	Rule     string   `json:"rule,omitempty"`
	Source   string   `json:"source"`
}

// Component describes one top-level lint candidate.
type Component struct {
	Name            string   `json:"name"`
	RelativePath    string   `json:"relativePath"`
	AbsolutePath    string   `json:"absolutePath"`
	IsRoot          bool     `json:"isRoot"`
	CodeBearing     bool     `json:"codeBearing"`
	CodeEvidence    []string `json:"codeEvidence,omitempty"`
	DetectionReason []string `json:"detectionReason,omitempty"`
}

// PolicyFinding reports a component-level policy issue outside a handler's own tool findings.
type PolicyFinding struct {
	Component string         `json:"component"`
	Path      string         `json:"path"`
	Severity  PolicySeverity `json:"severity"`
	Message   string         `json:"message"`
}

// ComponentResult captures lint execution for one component.
type ComponentResult struct {
	Component      Component       `json:"component"`
	HandlerID      string          `json:"handlerId,omitempty"`
	Matched        bool            `json:"matched"`
	Success        bool            `json:"success"`
	Issues         []Issue         `json:"issues,omitempty"`
	TypeErrors     int             `json:"typeErrors"`
	LintWarnings   int             `json:"lintWarnings"`
	ToolsUsed      []string        `json:"toolsUsed,omitempty"`
	Skipped        bool            `json:"skipped"`
	SkipReason     string          `json:"skipReason,omitempty"`
	Strict         bool            `json:"strict"`
	Observations   []Observation   `json:"observations,omitempty"`
	PolicyFindings []PolicyFinding `json:"policyFindings,omitempty"`
}

// RunResult is the lint phase result.
type RunResult struct {
	Success        bool
	Error          error
	FailureClass   FailureClass
	Remediation    string
	Observations   []Observation
	Summary        LintSummary
	Components     []ComponentResult
	PolicyFindings []PolicyFinding
}

// LintSummary tracks lint validation counts by component and policy findings.
type LintSummary struct {
	ComponentsDiscovered int `json:"componentsDiscovered"`
	ComponentsLinted     int `json:"componentsLinted"`
	ComponentsSkipped    int `json:"componentsSkipped"`
	ComponentsUnmatched  int `json:"componentsUnmatched"`
	TypeErrors           int `json:"typeErrors"`
	LintWarnings         int `json:"lintWarnings"`
	PolicyWarnings       int `json:"policyWarnings"`
	PolicyErrors         int `json:"policyErrors"`
}

// TotalChecks returns the number of matched components linted.
func (s LintSummary) TotalChecks() int {
	return s.ComponentsLinted
}

// TotalIssues returns the total number of lint, type, and policy issues.
func (s LintSummary) TotalIssues() int {
	return s.TypeErrors + s.LintWarnings + s.PolicyWarnings + s.PolicyErrors
}

// HasTypeErrors returns true if any type errors were found.
func (s LintSummary) HasTypeErrors() bool {
	return s.TypeErrors > 0
}

// String returns a human-readable summary.
func (s LintSummary) String() string {
	parts := []string{
		fmt.Sprintf("%d components linted", s.ComponentsLinted),
	}
	if s.ComponentsUnmatched > 0 {
		parts = append(parts, fmt.Sprintf("%d unmatched", s.ComponentsUnmatched))
	}
	if s.TypeErrors > 0 {
		parts = append(parts, fmt.Sprintf("%d type errors", s.TypeErrors))
	}
	if s.LintWarnings > 0 {
		parts = append(parts, fmt.Sprintf("%d lint warnings", s.LintWarnings))
	}
	if s.PolicyWarnings > 0 {
		parts = append(parts, fmt.Sprintf("%d policy warnings", s.PolicyWarnings))
	}
	if s.PolicyErrors > 0 {
		parts = append(parts, fmt.Sprintf("%d policy errors", s.PolicyErrors))
	}
	return strings.Join(parts, ", ")
}

// LookupFunc is a function that looks up a command by name.
type LookupFunc = shared.LookupFunc
