// Package types provides shared types for the playbooks package.
package types

import (
	"fmt"
	"strings"

	basv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1"
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

	// Timeline contains the parsed proto timeline for rich diagnostics.
	// This provides structured access to execution frames, assertions, and errors.
	Timeline *basv1.ExecutionTimeline
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

// WithTimeline attaches the parsed proto timeline for rich diagnostics.
func (e *PlaybookExecutionError) WithTimeline(timeline *basv1.ExecutionTimeline) *PlaybookExecutionError {
	e.Timeline = timeline
	// Also extract node/step info from the failed frame if not already set
	if timeline != nil && e.NodeID == "" {
		if failed := findFailedFrame(timeline); failed != nil {
			e.NodeID = failed.GetNodeId()
			e.StepIndex = int(failed.GetStepIndex())
			if e.CurrentStepDescription == "" {
				e.CurrentStepDescription = StepTypeToString(failed.GetStepType())
			}
		}
	}
	return e
}

// findFailedFrame locates the first failed frame in the timeline.
func findFailedFrame(timeline *basv1.ExecutionTimeline) *basv1.TimelineFrame {
	for _, frame := range timeline.GetFrames() {
		if !frame.GetSuccess() || frame.GetError() != "" {
			return frame
		}
	}
	return nil
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

	// Include rich timeline diagnostics if available
	if e.Timeline != nil {
		b.WriteString("\n--- Timeline Details ---\n")
		b.WriteString(fmt.Sprintf("Status:       %s\n", e.Timeline.GetStatus()))
		b.WriteString(fmt.Sprintf("Progress:     %d%%\n", e.Timeline.GetProgress()))

		frames := e.Timeline.GetFrames()
		b.WriteString(fmt.Sprintf("Total Steps:  %d\n", len(frames)))

		// Find and display failed frame details
		if failed := findFailedFrame(e.Timeline); failed != nil {
			b.WriteString("\n--- Failed Step ---\n")
			b.WriteString(fmt.Sprintf("  Index:      %d\n", failed.GetStepIndex()))
			b.WriteString(fmt.Sprintf("  Node ID:    %s\n", failed.GetNodeId()))
			b.WriteString(fmt.Sprintf("  Type:       %s\n", failed.GetStepType()))
			b.WriteString(fmt.Sprintf("  Status:     %s\n", failed.GetStatus()))
			if failed.GetError() != "" {
				b.WriteString(fmt.Sprintf("  Error:      %s\n", failed.GetError()))
			}
			if failed.GetFinalUrl() != "" {
				b.WriteString(fmt.Sprintf("  URL:        %s\n", failed.GetFinalUrl()))
			}
			if failed.GetDurationMs() > 0 {
				b.WriteString(fmt.Sprintf("  Duration:   %dms\n", failed.GetDurationMs()))
			}

			// Display assertion details if this was an assert step
			if assertion := failed.GetAssertion(); assertion != nil {
				b.WriteString("\n--- Assertion Details ---\n")
				b.WriteString(fmt.Sprintf("  Mode:       %s\n", assertion.GetMode()))
				b.WriteString(fmt.Sprintf("  Selector:   %s\n", assertion.GetSelector()))
				if expected := assertion.GetExpected(); expected != nil {
					b.WriteString(fmt.Sprintf("  Expected:   %v\n", expected.AsInterface()))
				}
				if actual := assertion.GetActual(); actual != nil {
					b.WriteString(fmt.Sprintf("  Actual:     %v\n", actual.AsInterface()))
				}
				if assertion.GetMessage() != "" {
					b.WriteString(fmt.Sprintf("  Message:    %s\n", assertion.GetMessage()))
				}
			}

			// Display retry info if retries were attempted
			if failed.GetRetryAttempt() > 0 {
				b.WriteString(fmt.Sprintf("\n  Retry:      attempt %d of %d\n",
					failed.GetRetryAttempt(), failed.GetRetryMaxAttempts()))
			}
		}

		// Summarize all failed assertions
		var failedAssertions []*basv1.TimelineFrame
		for _, frame := range frames {
			if frame.GetStepType() == basv1.StepType_STEP_TYPE_ASSERT && !frame.GetSuccess() {
				failedAssertions = append(failedAssertions, frame)
			}
		}
		if len(failedAssertions) > 0 {
			b.WriteString(fmt.Sprintf("\n--- Failed Assertions (%d) ---\n", len(failedAssertions)))
			for _, frame := range failedAssertions {
				assertion := frame.GetAssertion()
				if assertion != nil {
					b.WriteString(fmt.Sprintf("  [%d] %s on '%s'\n",
						frame.GetStepIndex(), assertion.GetMode(), assertion.GetSelector()))
				}
			}
		}
	}

	// Artifact paths for further investigation
	if e.Artifacts.Timeline != "" || e.Artifacts.RawTimeline != "" || e.Artifacts.Trace != "" {
		b.WriteString("\n--- Artifact Paths ---\n")
		if e.Artifacts.Timeline != "" {
			b.WriteString(fmt.Sprintf("Timeline:     %s\n", e.Artifacts.Timeline))
		}
		if e.Artifacts.RawTimeline != "" {
			b.WriteString(fmt.Sprintf("Raw Timeline: %s\n", e.Artifacts.RawTimeline))
		}
		if e.Artifacts.Trace != "" {
			b.WriteString(fmt.Sprintf("Trace:        %s\n", e.Artifacts.Trace))
		}
	}

	if e.BASResponse != "" {
		b.WriteString(fmt.Sprintf("\nBAS Response: %s\n", truncate(e.BASResponse, 500)))
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

// StepTypeToString converts a proto StepType enum to its string representation.
func StepTypeToString(st basv1.StepType) string {
	switch st {
	case basv1.StepType_STEP_TYPE_NAVIGATE:
		return "navigate"
	case basv1.StepType_STEP_TYPE_CLICK:
		return "click"
	case basv1.StepType_STEP_TYPE_ASSERT:
		return "assert"
	case basv1.StepType_STEP_TYPE_SUBFLOW:
		return "subflow"
	case basv1.StepType_STEP_TYPE_INPUT:
		return "input"
	case basv1.StepType_STEP_TYPE_CUSTOM:
		return "custom"
	default:
		return "unknown"
	}
}

// StepStatusToString converts a proto StepStatus enum to its string representation.
func StepStatusToString(ss basv1.StepStatus) string {
	switch ss {
	case basv1.StepStatus_STEP_STATUS_PENDING:
		return "pending"
	case basv1.StepStatus_STEP_STATUS_RUNNING:
		return "running"
	case basv1.StepStatus_STEP_STATUS_COMPLETED:
		return "completed"
	case basv1.StepStatus_STEP_STATUS_FAILED:
		return "failed"
	case basv1.StepStatus_STEP_STATUS_CANCELLED:
		return "cancelled"
	case basv1.StepStatus_STEP_STATUS_SKIPPED:
		return "skipped"
	case basv1.StepStatus_STEP_STATUS_RETRYING:
		return "retrying"
	default:
		return "unknown"
	}
}

// ExecutionStatusToString converts a proto ExecutionStatus enum to its string representation.
func ExecutionStatusToString(es basv1.ExecutionStatus) string {
	switch es {
	case basv1.ExecutionStatus_EXECUTION_STATUS_PENDING:
		return "pending"
	case basv1.ExecutionStatus_EXECUTION_STATUS_RUNNING:
		return "running"
	case basv1.ExecutionStatus_EXECUTION_STATUS_COMPLETED:
		return "completed"
	case basv1.ExecutionStatus_EXECUTION_STATUS_FAILED:
		return "failed"
	case basv1.ExecutionStatus_EXECUTION_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unknown"
	}
}

// LogLevelToString converts a proto LogLevel enum to its string representation.
func LogLevelToString(ll basv1.LogLevel) string {
	switch ll {
	case basv1.LogLevel_LOG_LEVEL_DEBUG:
		return "debug"
	case basv1.LogLevel_LOG_LEVEL_INFO:
		return "info"
	case basv1.LogLevel_LOG_LEVEL_WARN:
		return "warn"
	case basv1.LogLevel_LOG_LEVEL_ERROR:
		return "error"
	default:
		return "unknown"
	}
}

// StringToStepType converts a string to a proto StepType enum.
func StringToStepType(s string) basv1.StepType {
	switch s {
	case "navigate":
		return basv1.StepType_STEP_TYPE_NAVIGATE
	case "click":
		return basv1.StepType_STEP_TYPE_CLICK
	case "assert":
		return basv1.StepType_STEP_TYPE_ASSERT
	case "subflow":
		return basv1.StepType_STEP_TYPE_SUBFLOW
	case "input":
		return basv1.StepType_STEP_TYPE_INPUT
	case "custom":
		return basv1.StepType_STEP_TYPE_CUSTOM
	default:
		return basv1.StepType_STEP_TYPE_UNSPECIFIED
	}
}
