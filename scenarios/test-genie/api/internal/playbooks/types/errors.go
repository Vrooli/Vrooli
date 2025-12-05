// Package types provides shared types for the playbooks package.
package types

import (
	"fmt"
	"strings"
)

// ExecutionPhase indicates which phase of playbook execution failed.
type ExecutionPhase string

const (
	// PhaseResolve indicates failure during workflow resolution (loading, fixture expansion).
	PhaseResolve ExecutionPhase = "resolve"
	// PhaseExecute indicates failure during workflow submission to BAS.
	PhaseExecute ExecutionPhase = "execute"
	// PhaseWait indicates failure while waiting for workflow completion.
	PhaseWait ExecutionPhase = "wait"
	// PhaseArtifact indicates failure during artifact collection.
	PhaseArtifact ExecutionPhase = "artifact"
)

// PlaybookExecutionError provides rich context for playbook execution failures.
// It captures the workflow location, execution state, and any collected artifacts
// to aid debugging.
type PlaybookExecutionError struct {
	// WorkflowFile is the path to the workflow JSON file.
	WorkflowFile string

	// ExecutionID is the BAS execution ID (if available).
	ExecutionID string

	// Phase indicates which execution phase failed.
	Phase ExecutionPhase

	// NodeID is the ID of the node that was executing when failure occurred (if known).
	NodeID string

	// StepIndex is the 0-based index of the current step (if known).
	StepIndex int

	// CurrentStepDescription is a human-readable description of the current step.
	CurrentStepDescription string

	// Cause is the underlying error.
	Cause error

	// Artifacts holds paths to collected debug artifacts.
	Artifacts ExecutionArtifacts

	// BASResponse contains the raw response from BAS (if available).
	BASResponse string
}

// ExecutionArtifacts holds paths to debug artifacts collected during execution.
type ExecutionArtifacts struct {
	// Timeline is the path to the execution timeline JSON.
	Timeline string
	// Trace is the path to the execution trace JSONL file.
	Trace string
	// RawTimeline is the path to raw timeline data (when parsing failed).
	RawTimeline string
}

// Error implements the error interface.
func (e *PlaybookExecutionError) Error() string {
	var b strings.Builder

	b.WriteString("playbook execution failed")

	if e.WorkflowFile != "" {
		b.WriteString(fmt.Sprintf(" [workflow=%s]", e.WorkflowFile))
	}

	if e.Phase != "" {
		b.WriteString(fmt.Sprintf(" [phase=%s]", e.Phase))
	}

	if e.ExecutionID != "" {
		b.WriteString(fmt.Sprintf(" [execution_id=%s]", e.ExecutionID))
	}

	if e.NodeID != "" {
		b.WriteString(fmt.Sprintf(" [node=%s]", e.NodeID))
	}

	if e.StepIndex > 0 {
		b.WriteString(fmt.Sprintf(" [step=%d]", e.StepIndex))
	}

	if e.CurrentStepDescription != "" {
		b.WriteString(fmt.Sprintf(" [step_desc=%s]", e.CurrentStepDescription))
	}

	if e.Cause != nil {
		b.WriteString(": ")
		b.WriteString(e.Cause.Error())
	}

	return b.String()
}

// Unwrap returns the underlying cause for errors.Is/As support.
func (e *PlaybookExecutionError) Unwrap() error {
	return e.Cause
}

// WithArtifacts returns a copy with artifact paths set.
func (e *PlaybookExecutionError) WithArtifacts(artifacts ExecutionArtifacts) *PlaybookExecutionError {
	e.Artifacts = artifacts
	return e
}

// WithExecutionID returns a copy with execution ID set.
func (e *PlaybookExecutionError) WithExecutionID(id string) *PlaybookExecutionError {
	e.ExecutionID = id
	return e
}

// WithBASResponse attaches raw BAS response for debugging.
func (e *PlaybookExecutionError) WithBASResponse(response string) *PlaybookExecutionError {
	e.BASResponse = response
	return e
}

// DiagnosticString returns a detailed multi-line diagnostic message.
func (e *PlaybookExecutionError) DiagnosticString() string {
	var b strings.Builder

	b.WriteString("=== Playbook Execution Error ===\n")
	b.WriteString(fmt.Sprintf("Workflow:     %s\n", e.WorkflowFile))
	b.WriteString(fmt.Sprintf("Phase:        %s\n", e.Phase))

	if e.ExecutionID != "" {
		b.WriteString(fmt.Sprintf("Execution ID: %s\n", e.ExecutionID))
	}

	if e.NodeID != "" {
		b.WriteString(fmt.Sprintf("Node ID:      %s\n", e.NodeID))
	}

	if e.StepIndex > 0 {
		b.WriteString(fmt.Sprintf("Step Index:   %d\n", e.StepIndex))
	}

	if e.CurrentStepDescription != "" {
		b.WriteString(fmt.Sprintf("Step:         %s\n", e.CurrentStepDescription))
	}

	if e.Cause != nil {
		b.WriteString(fmt.Sprintf("Error:        %v\n", e.Cause))
	}

	if e.Artifacts.Timeline != "" {
		b.WriteString(fmt.Sprintf("Timeline:     %s\n", e.Artifacts.Timeline))
	}

	if e.Artifacts.RawTimeline != "" {
		b.WriteString(fmt.Sprintf("Raw Timeline: %s\n", e.Artifacts.RawTimeline))
	}

	if e.Artifacts.Trace != "" {
		b.WriteString(fmt.Sprintf("Trace:        %s\n", e.Artifacts.Trace))
	}

	if e.BASResponse != "" {
		b.WriteString(fmt.Sprintf("BAS Response: %s\n", truncate(e.BASResponse, 500)))
	}

	b.WriteString("================================\n")

	return b.String()
}

// NewResolveError creates an error for the resolve phase.
func NewResolveError(workflowFile string, cause error) *PlaybookExecutionError {
	return &PlaybookExecutionError{
		WorkflowFile: workflowFile,
		Phase:        PhaseResolve,
		Cause:        cause,
	}
}

// NewExecuteError creates an error for the execute phase.
func NewExecuteError(workflowFile string, cause error) *PlaybookExecutionError {
	return &PlaybookExecutionError{
		WorkflowFile: workflowFile,
		Phase:        PhaseExecute,
		Cause:        cause,
	}
}

// NewWaitError creates an error for the wait phase.
func NewWaitError(workflowFile, executionID string, cause error) *PlaybookExecutionError {
	return &PlaybookExecutionError{
		WorkflowFile: workflowFile,
		ExecutionID:  executionID,
		Phase:        PhaseWait,
		Cause:        cause,
	}
}

// NewArtifactError creates an error for the artifact phase.
func NewArtifactError(workflowFile string, cause error) *PlaybookExecutionError {
	return &PlaybookExecutionError{
		WorkflowFile: workflowFile,
		Phase:        PhaseArtifact,
		Cause:        cause,
	}
}

// truncate shortens a string to maxLen, adding "..." if truncated.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
