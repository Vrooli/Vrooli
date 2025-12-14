// Package types provides shared types for the playbooks package.
package types

import (
	"fmt"
	"strings"

	basactions "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/actions"
	basbase "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/base"
	bastimeline "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/timeline"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
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
	// This provides structured access to execution entries, assertions, and errors.
	Timeline *bastimeline.ExecutionTimeline
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
func (e *PlaybookExecutionError) WithTimeline(timeline *bastimeline.ExecutionTimeline) *PlaybookExecutionError {
	e.Timeline = timeline
	// Also extract node/step info from the failed entry if not already set
	if timeline != nil && e.NodeID == "" {
		if failed := findFailedEntry(timeline); failed != nil {
			e.NodeID = failed.GetNodeId()
			e.StepIndex = int(failed.GetStepIndex())
			if e.CurrentStepDescription == "" && failed.GetAction() != nil {
				e.CurrentStepDescription = ActionTypeToString(failed.GetAction().GetType())
			}
		}
	}
	return e
}

// findFailedEntry locates the first failed entry in the timeline.
func findFailedEntry(timeline *bastimeline.ExecutionTimeline) *bastimeline.TimelineEntry {
	for _, entry := range timeline.GetEntries() {
		ctx := entry.GetContext()
		if ctx != nil && (!ctx.GetSuccess() || ctx.GetError() != "") {
			return entry
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

		entries := e.Timeline.GetEntries()
		b.WriteString(fmt.Sprintf("Total Steps:  %d\n", len(entries)))

		// Find and display failed entry details
		if failed := findFailedEntry(e.Timeline); failed != nil {
			b.WriteString("\n--- Failed Step ---\n")
			b.WriteString(fmt.Sprintf("  Index:      %d\n", failed.GetStepIndex()))
			b.WriteString(fmt.Sprintf("  Node ID:    %s\n", failed.GetNodeId()))
			if action := failed.GetAction(); action != nil {
				b.WriteString(fmt.Sprintf("  Type:       %s\n", action.GetType()))
			}
			if agg := failed.GetAggregates(); agg != nil {
				b.WriteString(fmt.Sprintf("  Status:     %s\n", agg.GetStatus()))
				if agg.GetFinalUrl() != "" {
					b.WriteString(fmt.Sprintf("  URL:        %s\n", agg.GetFinalUrl()))
				}
			}
			if ctx := failed.GetContext(); ctx != nil && ctx.GetError() != "" {
				b.WriteString(fmt.Sprintf("  Error:      %s\n", ctx.GetError()))
			}
			if failed.GetDurationMs() > 0 {
				b.WriteString(fmt.Sprintf("  Duration:   %dms\n", failed.GetDurationMs()))
			}

			// Display assertion details if this was an assert step
			if ctx := failed.GetContext(); ctx != nil {
				if assertion := ctx.GetAssertion(); assertion != nil {
					b.WriteString("\n--- Assertion Details ---\n")
					b.WriteString(fmt.Sprintf("  Mode:       %s\n", assertion.GetMode()))
					b.WriteString(fmt.Sprintf("  Selector:   %s\n", assertion.GetSelector()))
					if expected := assertion.GetExpected(); expected != nil {
						b.WriteString(fmt.Sprintf("  Expected:   %v\n", jsonValueToString(expected)))
					}
					if actual := assertion.GetActual(); actual != nil {
						b.WriteString(fmt.Sprintf("  Actual:     %v\n", jsonValueToString(actual)))
					}
					if assertion.GetMessage() != "" {
						b.WriteString(fmt.Sprintf("  Message:    %s\n", assertion.GetMessage()))
					}
				}

				// Display retry info if retries were attempted
				if retry := ctx.GetRetryStatus(); retry != nil && retry.GetCurrentAttempt() > 0 {
					b.WriteString(fmt.Sprintf("\n  Retry:      attempt %d of %d\n",
						retry.GetCurrentAttempt(), retry.GetMaxAttempts()))
				}
			}
		}

		// Summarize all failed assertions
		var failedAssertions []*bastimeline.TimelineEntry
		for _, entry := range entries {
			action := entry.GetAction()
			ctx := entry.GetContext()
			if action != nil && action.GetType() == basactions.ActionType_ACTION_TYPE_ASSERT && ctx != nil && !ctx.GetSuccess() {
				failedAssertions = append(failedAssertions, entry)
			}
		}
		if len(failedAssertions) > 0 {
			b.WriteString(fmt.Sprintf("\n--- Failed Assertions (%d) ---\n", len(failedAssertions)))
			for _, entry := range failedAssertions {
				ctx := entry.GetContext()
				if ctx != nil {
					if assertion := ctx.GetAssertion(); assertion != nil {
						b.WriteString(fmt.Sprintf("  [%d] %s on '%s'\n",
							entry.GetStepIndex(), assertion.GetMode(), assertion.GetSelector()))
					}
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

// ActionTypeToString converts a proto ActionType enum to its string representation.
func ActionTypeToString(at basactions.ActionType) string {
	switch at {
	case basactions.ActionType_ACTION_TYPE_NAVIGATE:
		return "navigate"
	case basactions.ActionType_ACTION_TYPE_CLICK:
		return "click"
	case basactions.ActionType_ACTION_TYPE_ASSERT:
		return "assert"
	case basactions.ActionType_ACTION_TYPE_SUBFLOW:
		return "subflow"
	case basactions.ActionType_ACTION_TYPE_INPUT:
		return "input"
	case basactions.ActionType_ACTION_TYPE_WAIT:
		return "wait"
	case basactions.ActionType_ACTION_TYPE_SCROLL:
		return "scroll"
	case basactions.ActionType_ACTION_TYPE_SELECT:
		return "select"
	case basactions.ActionType_ACTION_TYPE_EVALUATE:
		return "evaluate"
	case basactions.ActionType_ACTION_TYPE_KEYBOARD:
		return "keyboard"
	case basactions.ActionType_ACTION_TYPE_HOVER:
		return "hover"
	case basactions.ActionType_ACTION_TYPE_SCREENSHOT:
		return "screenshot"
	case basactions.ActionType_ACTION_TYPE_FOCUS:
		return "focus"
	case basactions.ActionType_ACTION_TYPE_BLUR:
		return "blur"
	default:
		return "unknown"
	}
}


// StepStatusToString converts a proto StepStatus enum to its string representation.
func StepStatusToString(ss basbase.StepStatus) string {
	switch ss {
	case basbase.StepStatus_STEP_STATUS_PENDING:
		return "pending"
	case basbase.StepStatus_STEP_STATUS_RUNNING:
		return "running"
	case basbase.StepStatus_STEP_STATUS_COMPLETED:
		return "completed"
	case basbase.StepStatus_STEP_STATUS_FAILED:
		return "failed"
	case basbase.StepStatus_STEP_STATUS_CANCELLED:
		return "cancelled"
	case basbase.StepStatus_STEP_STATUS_SKIPPED:
		return "skipped"
	case basbase.StepStatus_STEP_STATUS_RETRYING:
		return "retrying"
	default:
		return "unknown"
	}
}

// ExecutionStatusToString converts a proto ExecutionStatus enum to its string representation.
func ExecutionStatusToString(es basbase.ExecutionStatus) string {
	switch es {
	case basbase.ExecutionStatus_EXECUTION_STATUS_PENDING:
		return "pending"
	case basbase.ExecutionStatus_EXECUTION_STATUS_RUNNING:
		return "running"
	case basbase.ExecutionStatus_EXECUTION_STATUS_COMPLETED:
		return "completed"
	case basbase.ExecutionStatus_EXECUTION_STATUS_FAILED:
		return "failed"
	case basbase.ExecutionStatus_EXECUTION_STATUS_CANCELLED:
		return "cancelled"
	default:
		return "unknown"
	}
}

// LogLevelToString converts a proto LogLevel enum to its string representation.
func LogLevelToString(ll basbase.LogLevel) string {
	switch ll {
	case basbase.LogLevel_LOG_LEVEL_DEBUG:
		return "debug"
	case basbase.LogLevel_LOG_LEVEL_INFO:
		return "info"
	case basbase.LogLevel_LOG_LEVEL_WARN:
		return "warn"
	case basbase.LogLevel_LOG_LEVEL_ERROR:
		return "error"
	default:
		return "unknown"
	}
}

// StringToActionType converts a string to a proto ActionType enum.
func StringToActionType(s string) basactions.ActionType {
	switch s {
	case "navigate":
		return basactions.ActionType_ACTION_TYPE_NAVIGATE
	case "click":
		return basactions.ActionType_ACTION_TYPE_CLICK
	case "assert":
		return basactions.ActionType_ACTION_TYPE_ASSERT
	case "subflow":
		return basactions.ActionType_ACTION_TYPE_SUBFLOW
	case "input":
		return basactions.ActionType_ACTION_TYPE_INPUT
	case "wait":
		return basactions.ActionType_ACTION_TYPE_WAIT
	case "scroll":
		return basactions.ActionType_ACTION_TYPE_SCROLL
	case "select", "select_option":
		return basactions.ActionType_ACTION_TYPE_SELECT
	case "evaluate":
		return basactions.ActionType_ACTION_TYPE_EVALUATE
	case "keyboard":
		return basactions.ActionType_ACTION_TYPE_KEYBOARD
	case "hover":
		return basactions.ActionType_ACTION_TYPE_HOVER
	case "screenshot":
		return basactions.ActionType_ACTION_TYPE_SCREENSHOT
	case "focus":
		return basactions.ActionType_ACTION_TYPE_FOCUS
	case "blur":
		return basactions.ActionType_ACTION_TYPE_BLUR
	default:
		return basactions.ActionType_ACTION_TYPE_UNSPECIFIED
	}
}

// jsonValueToString converts a commonv1.JsonValue to a string representation.
func jsonValueToString(v *commonv1.JsonValue) string {
	if v == nil {
		return ""
	}
	switch kind := v.GetKind().(type) {
	case *commonv1.JsonValue_StringValue:
		return kind.StringValue
	case *commonv1.JsonValue_IntValue:
		return fmt.Sprintf("%d", kind.IntValue)
	case *commonv1.JsonValue_DoubleValue:
		return fmt.Sprintf("%v", kind.DoubleValue)
	case *commonv1.JsonValue_BoolValue:
		return fmt.Sprintf("%v", kind.BoolValue)
	case *commonv1.JsonValue_NullValue:
		return "null"
	default:
		return fmt.Sprintf("%v", v)
	}
}
