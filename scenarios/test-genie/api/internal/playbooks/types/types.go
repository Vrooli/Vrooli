// Package types defines shared types for the playbooks package hierarchy.
package types

import (
	"fmt"
	"time"
)

// Registry represents the playbook registry file structure.
type Registry struct {
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

// ExecutionStatus represents the status response from BAS API.
type ExecutionStatus struct {
	Status        string  `json:"status"`
	Progress      float64 `json:"progress"`
	CurrentStep   string  `json:"current_step"`
	FailureReason string  `json:"failure_reason"`
	Error         string  `json:"error"`
}

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

// Observation represents a single observation during playbook execution.
type Observation struct {
	Type    ObservationType
	Icon    string
	Message string
}

// ObservationType categorizes observations.
type ObservationType int

const (
	ObservationInfo ObservationType = iota
	ObservationSuccess
	ObservationWarning
	ObservationError
	ObservationSkip
	ObservationSection
)

// NewSuccessObservation creates a success observation.
func NewSuccessObservation(message string) Observation {
	return Observation{Type: ObservationSuccess, Icon: "✅", Message: message}
}

// NewWarningObservation creates a warning observation.
func NewWarningObservation(message string) Observation {
	return Observation{Type: ObservationWarning, Icon: "⚠️", Message: message}
}

// NewErrorObservation creates an error observation.
func NewErrorObservation(message string) Observation {
	return Observation{Type: ObservationError, Icon: "❌", Message: message}
}

// NewInfoObservation creates an info observation.
func NewInfoObservation(message string) Observation {
	return Observation{Type: ObservationInfo, Icon: "ℹ️", Message: message}
}

// NewSkipObservation creates a skip observation.
func NewSkipObservation(message string) Observation {
	return Observation{Type: ObservationSkip, Icon: "⏭️", Message: message}
}

// NewSectionObservation creates a section header observation.
func NewSectionObservation(icon, message string) Observation {
	return Observation{Type: ObservationSection, Icon: icon, Message: message}
}

// FailureClass categorizes playbook failures.
type FailureClass string

const (
	FailureClassMisconfiguration  FailureClass = "misconfiguration"
	FailureClassMissingDependency FailureClass = "missing_dependency"
	FailureClassSystem            FailureClass = "system"
	FailureClassExecution         FailureClass = "execution"
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
type RunResult struct {
	Success      bool
	Error        error
	FailureClass FailureClass
	Remediation  string
	Observations []Observation
	Results      []Result
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
