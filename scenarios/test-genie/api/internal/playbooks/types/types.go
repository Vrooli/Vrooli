// Package types defines shared types for the playbooks package hierarchy.
package types

import (
	"fmt"
	"time"

	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
	"test-genie/internal/shared"
)

// Registry represents the playbook registry file structure.
type Registry struct {
	Note       string  `json:"_note,omitempty"`
	Scenario   string  `json:"scenario"`
	Generated  string  `json:"generated_at"`
	Playbooks  []Entry `json:"playbooks"`
	Deprecated []Entry `json:"deprecated_playbooks,omitempty"`
}

// Entry represents a single playbook entry in the registry.
type Entry struct {
	File         string   `json:"file"`
	Description  string   `json:"description"`
	Order        string   `json:"order"`
	Requirements []string `json:"requirements"`
	Fixtures     []string `json:"fixtures"`
	Reset        string   `json:"reset"`
}

// ExecutionStatus mirrors the BAS proto Execution message for status polling.
type ExecutionStatus = basv1.Execution

// Outcome represents the result of executing a single playbook.
type Outcome struct {
	ExecutionID string
	Duration    time.Duration
	Stats       string
}

// Result represents the result of a playbook execution attempt.
type Result struct {
	Entry        Entry
	Outcome      *Outcome
	Err          error
	ArtifactPath string
}

// Re-export shared types for consistency across packages.
type (
	FailureClass    = shared.FailureClass
	ObservationType = shared.ObservationType
	Observation     = shared.Observation
)

// Re-export constants.
const (
	FailureClassNone              = shared.FailureClassNone
	FailureClassMisconfiguration  = shared.FailureClassMisconfiguration
	FailureClassMissingDependency = shared.FailureClassMissingDependency
	FailureClassSystem            = shared.FailureClassSystem
	FailureClassExecution         = shared.FailureClassExecution

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
)

// ExecutionSummary tracks playbook execution statistics.
type ExecutionSummary struct {
	WorkflowsExecuted int
	WorkflowsPassed   int
	WorkflowsFailed   int
	TotalDuration     time.Duration
}

// TotalWorkflows returns the total number of workflows in the registry.
func (s ExecutionSummary) TotalWorkflows() int {
	return s.WorkflowsExecuted
}

// PassRate returns the pass rate as a percentage (0-100).
func (s ExecutionSummary) PassRate() float64 {
	if s.WorkflowsExecuted == 0 {
		return 0
	}
	return float64(s.WorkflowsPassed) / float64(s.WorkflowsExecuted) * 100
}

// String returns a human-readable summary.
func (s ExecutionSummary) String() string {
	if s.WorkflowsExecuted == 0 {
		return "no workflows executed"
	}
	return fmt.Sprintf("%d/%d workflows passed (%.0f%%) in %s",
		s.WorkflowsPassed, s.WorkflowsExecuted, s.PassRate(), s.TotalDuration.Round(time.Millisecond))
}

// RunResult represents the overall result of the playbooks phase.
// This extends the base shared.RunResult with playbook-specific Results field.
type RunResult struct {
	Success      bool
	Error        error
	FailureClass FailureClass
	Remediation  string
	Observations []Observation
	Results      []Result // Playbook-specific: execution results for each playbook
	Summary      ExecutionSummary
}

// OK creates a successful RunResult.
func OK() *RunResult {
	return &RunResult{Success: true}
}

// OKWithResults creates a successful RunResult with execution results.
func OKWithResults(results []Result) *RunResult {
	return &RunResult{Success: true, Results: results}
}

// Fail creates a failure RunResult with the given error, classification, and remediation.
func Fail(err error, class FailureClass, remediation string) *RunResult {
	return &RunResult{
		Success:      false,
		Error:        err,
		FailureClass: class,
		Remediation:  remediation,
	}
}

// FailMisconfiguration creates a misconfiguration failure.
func FailMisconfiguration(err error, remediation string) *RunResult {
	return Fail(err, FailureClassMisconfiguration, remediation)
}

// FailMissingDependency creates a missing dependency failure.
func FailMissingDependency(err error, remediation string) *RunResult {
	return Fail(err, FailureClassMissingDependency, remediation)
}

// FailSystem creates a system-level failure.
func FailSystem(err error, remediation string) *RunResult {
	return Fail(err, FailureClassSystem, remediation)
}

// FailExecution creates an execution failure.
func FailExecution(err error, remediation string) *RunResult {
	return Fail(err, FailureClassExecution, remediation)
}

// WithObservations adds observations to the result.
func (r *RunResult) WithObservations(obs ...Observation) *RunResult {
	r.Observations = append(r.Observations, obs...)
	return r
}

// WithResults adds execution results.
func (r *RunResult) WithResults(results []Result) *RunResult {
	r.Results = results
	return r
}
