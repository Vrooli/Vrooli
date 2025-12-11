// Package agents provides agent lifecycle management.
// This file defines structured failure types and classifications for the agent system.
// These types enable:
//   - Observable failure patterns without exposing sensitive details
//   - Graceful degradation decisions
//   - Consistent error handling across the agent lifecycle
//
// CHANGE_AXIS: Failure Types and Classification
// When adding new failure modes:
//  1. Add a FailureCategory constant
//  2. Add the corresponding FailureCode constant
//  3. Update ClassifyError to handle the new error type
//  4. Consider if new failure needs graceful degradation handling
package agents

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

// FailureCategory groups related failures for decision-making.
// Higher categories are more severe.
type FailureCategory string

const (
	// FailureCategoryNone indicates no failure occurred.
	FailureCategoryNone FailureCategory = "none"

	// FailureCategoryTransient indicates a temporary failure that may resolve on retry.
	// Examples: network timeout, database connection lost, rate limit hit
	FailureCategoryTransient FailureCategory = "transient"

	// FailureCategoryValidation indicates invalid input from the client.
	// Examples: missing required field, invalid scope path, blocked command
	FailureCategoryValidation FailureCategory = "validation"

	// FailureCategoryConflict indicates a resource conflict with existing operations.
	// Examples: scope lock conflict, duplicate agent, session conflict
	FailureCategoryConflict FailureCategory = "conflict"

	// FailureCategoryResource indicates a required resource is unavailable.
	// Examples: Docker not running, CLI tool not installed, no disk space
	FailureCategoryResource FailureCategory = "resource"

	// FailureCategoryInternal indicates an unexpected internal error.
	// Examples: nil pointer, unhandled case, programming error
	FailureCategoryInternal FailureCategory = "internal"
)

// FailureCode provides machine-readable failure identification.
// Codes are stable across versions for client handling.
type FailureCode string

const (
	// Transient failures
	FailureCodeDatabaseTimeout     FailureCode = "database_timeout"
	FailureCodeDatabaseConnection  FailureCode = "database_connection"
	FailureCodeLockAcquisitionFail FailureCode = "lock_acquisition_failed"
	FailureCodeHeartbeatFailed     FailureCode = "heartbeat_failed"
	FailureCodeContextCanceled     FailureCode = "context_canceled"

	// Validation failures
	FailureCodeInvalidPrompt      FailureCode = "invalid_prompt"
	FailureCodeInvalidScope       FailureCode = "invalid_scope"
	FailureCodeInvalidTools       FailureCode = "invalid_tools"
	FailureCodeMissingRequired    FailureCode = "missing_required_field"
	FailureCodeBlockedCommand     FailureCode = "blocked_command"
	FailureCodeSkipPermissions    FailureCode = "skip_permissions_denied"
	FailureCodePromptLimitExeeded FailureCode = "prompt_limit_exceeded"

	// Conflict failures
	FailureCodeScopeConflict     FailureCode = "scope_conflict"
	FailureCodeDuplicateAgent    FailureCode = "duplicate_agent"
	FailureCodeSessionConflict   FailureCode = "session_conflict"
	FailureCodeIdempotencyActive FailureCode = "idempotency_in_progress"

	// Resource failures
	FailureCodeDockerUnavailable   FailureCode = "docker_unavailable"
	FailureCodeCLINotFound         FailureCode = "cli_not_found"
	FailureCodeScenarioNotFound    FailureCode = "scenario_not_found"
	FailureCodeContainmentFailed   FailureCode = "containment_failed"
	FailureCodeAgentProcessFailed  FailureCode = "agent_process_failed"
	FailureCodeResourceExhausted   FailureCode = "resource_exhausted"
	FailureCodeAgentNotFound       FailureCode = "agent_not_found"
	FailureCodeAgentAlreadyStopped FailureCode = "agent_already_stopped"

	// Process execution failures (agent runtime errors)
	FailureCodeProcessStartFailed   FailureCode = "process_start_failed"
	FailureCodeProcessPipeFailed    FailureCode = "process_pipe_failed"
	FailureCodeProcessTimeout       FailureCode = "process_timeout"
	FailureCodeProcessCanceled      FailureCode = "process_canceled"
	FailureCodeProcessExitError     FailureCode = "process_exit_error"
	FailureCodeRegistrationFailed   FailureCode = "registration_failed"
	FailureCodeContainmentPrepFailed FailureCode = "containment_prep_failed"

	// Internal failures
	FailureCodeUnexpected FailureCode = "unexpected_error"
)

// FailureInfo provides structured information about a failure.
// This is designed to be safe for logging and user display without leaking
// sensitive implementation details.
type FailureInfo struct {
	// Category groups the failure type for decision-making.
	Category FailureCategory

	// Code provides machine-readable identification.
	Code FailureCode

	// Message is a user-safe description of what went wrong.
	// This should never contain: stack traces, file paths, SQL queries,
	// internal function names, or other implementation details.
	Message string

	// Retryable indicates whether the client should attempt retry.
	Retryable bool

	// RetryAfterSeconds suggests how long to wait before retry (0 = immediate).
	// Only meaningful when Retryable is true.
	RetryAfterSeconds int

	// RecoveryHint provides user-facing guidance on what to do next.
	RecoveryHint string

	// InternalDetails contains debugging info that should NOT be sent to clients.
	// This is for operator logs only.
	InternalDetails string
}

// Error implements the error interface.
func (f FailureInfo) Error() string {
	return f.Message
}

// IsRetryable returns true if the failure may resolve on retry.
func (f FailureInfo) IsRetryable() bool {
	return f.Retryable
}

// WithInternalDetails adds internal debugging info.
// This returns a new FailureInfo, not mutating the original.
func (f FailureInfo) WithInternalDetails(details string) FailureInfo {
	f.InternalDetails = details
	return f
}

// WithRecoveryHint adds recovery guidance for the user.
func (f FailureInfo) WithRecoveryHint(hint string) FailureInfo {
	f.RecoveryHint = hint
	return f
}

// --- Error Classification ---

// ClassifyError analyzes an error and returns structured failure information.
// This is the central decision point for error categorization.
//
// Decision criteria (in order):
//  1. Nil error → no failure
//  2. Context errors → transient (may be retried after delay)
//  3. SQL errors → transient database issues
//  4. Known agent errors → mapped to specific codes
//  5. Unknown → internal error
func ClassifyError(err error) FailureInfo {
	if err == nil {
		return FailureInfo{
			Category: FailureCategoryNone,
			Code:     "",
			Message:  "",
		}
	}

	// Check context errors first
	if errors.Is(err, context.Canceled) {
		return FailureInfo{
			Category:      FailureCategoryTransient,
			Code:          FailureCodeContextCanceled,
			Message:       "Operation was canceled",
			Retryable:     true,
			RecoveryHint:  "Please try again",
			InternalDetails: err.Error(),
		}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return FailureInfo{
			Category:          FailureCategoryTransient,
			Code:              FailureCodeDatabaseTimeout,
			Message:           "Operation timed out",
			Retryable:         true,
			RetryAfterSeconds: 2,
			RecoveryHint:      "The server is under load. Please try again in a moment.",
			InternalDetails:   err.Error(),
		}
	}

	// Check SQL errors
	if errors.Is(err, sql.ErrNoRows) {
		return FailureInfo{
			Category:        FailureCategoryResource,
			Code:            FailureCodeAgentNotFound,
			Message:         "Agent not found",
			Retryable:       false,
			RecoveryHint:    "The requested agent does not exist or has been cleaned up",
			InternalDetails: err.Error(),
		}
	}
	if errors.Is(err, sql.ErrConnDone) || errors.Is(err, sql.ErrTxDone) {
		return FailureInfo{
			Category:          FailureCategoryTransient,
			Code:              FailureCodeDatabaseConnection,
			Message:           "Database connection error",
			Retryable:         true,
			RetryAfterSeconds: 1,
			RecoveryHint:      "Please try again",
			InternalDetails:   err.Error(),
		}
	}

	// Check for known error patterns by message content
	errMsg := err.Error()

	// Scope conflicts
	if strings.Contains(errMsg, "scope conflict") {
		return FailureInfo{
			Category:        FailureCategoryConflict,
			Code:            FailureCodeScopeConflict,
			Message:         "Another agent is already working in this scope",
			Retryable:       false,
			RecoveryHint:    "Wait for the existing agent to complete, or stop it first",
			InternalDetails: errMsg,
		}
	}

	// Duplicate agent
	if strings.Contains(errMsg, "duplicate agent") {
		return FailureInfo{
			Category:        FailureCategoryConflict,
			Code:            FailureCodeDuplicateAgent,
			Message:         "An identical agent is already running",
			Retryable:       false,
			RecoveryHint:    "Wait for the existing agent to complete",
			InternalDetails: errMsg,
		}
	}

	// Agent not found
	if strings.Contains(errMsg, "agent not found") || strings.Contains(errMsg, "not found") {
		return FailureInfo{
			Category:        FailureCategoryResource,
			Code:            FailureCodeAgentNotFound,
			Message:         "Agent not found",
			Retryable:       false,
			RecoveryHint:    "The agent may have completed or been cleaned up",
			InternalDetails: errMsg,
		}
	}

	// Agent not running
	if strings.Contains(errMsg, "not running") {
		return FailureInfo{
			Category:        FailureCategoryResource,
			Code:            FailureCodeAgentAlreadyStopped,
			Message:         "Agent is not running",
			Retryable:       false,
			RecoveryHint:    "The agent has already stopped or completed",
			InternalDetails: errMsg,
		}
	}

	// Lock acquisition failed
	if strings.Contains(errMsg, "acquire lock") || strings.Contains(errMsg, "lock already held") {
		return FailureInfo{
			Category:          FailureCategoryTransient,
			Code:              FailureCodeLockAcquisitionFail,
			Message:           "Could not acquire scope lock",
			Retryable:         true,
			RetryAfterSeconds: 5,
			RecoveryHint:      "Another operation may be in progress. Please try again.",
			InternalDetails:   errMsg,
		}
	}

	// Default: internal error
	return FailureInfo{
		Category:        FailureCategoryInternal,
		Code:            FailureCodeUnexpected,
		Message:         "An unexpected error occurred",
		Retryable:       false,
		RecoveryHint:    "Please try again. If the problem persists, contact support.",
		InternalDetails: errMsg,
	}
}

// --- Factory Functions for Common Failures ---
// These provide consistent failure creation with appropriate messages.

// NewValidationFailure creates a validation category failure.
func NewValidationFailure(code FailureCode, message string) FailureInfo {
	return FailureInfo{
		Category: FailureCategoryValidation,
		Code:     code,
		Message:  message,
	}
}

// NewConflictFailure creates a conflict category failure.
func NewConflictFailure(code FailureCode, message string) FailureInfo {
	return FailureInfo{
		Category: FailureCategoryConflict,
		Code:     code,
		Message:  message,
	}
}

// NewResourceFailure creates a resource unavailable failure.
func NewResourceFailure(code FailureCode, message, recoveryHint string) FailureInfo {
	return FailureInfo{
		Category:     FailureCategoryResource,
		Code:         code,
		Message:      message,
		RecoveryHint: recoveryHint,
	}
}

// NewTransientFailure creates a transient (retryable) failure.
func NewTransientFailure(code FailureCode, message string, retryAfterSeconds int) FailureInfo {
	return FailureInfo{
		Category:          FailureCategoryTransient,
		Code:              code,
		Message:           message,
		Retryable:         true,
		RetryAfterSeconds: retryAfterSeconds,
	}
}

// NewInternalFailure creates an internal error failure.
func NewInternalFailure(code FailureCode, message string) FailureInfo {
	return FailureInfo{
		Category:     FailureCategoryInternal,
		Code:         code,
		Message:      message,
		RecoveryHint: "Please try again. If the problem persists, contact support.",
	}
}

// --- Process Execution Failure Factories ---
// These provide consistent failure creation for agent runtime errors.

// NewProcessStartFailure creates a failure for when the agent process fails to start.
func NewProcessStartFailure(err error) FailureInfo {
	return FailureInfo{
		Category:        FailureCategoryResource,
		Code:            FailureCodeProcessStartFailed,
		Message:         "Failed to start agent process",
		RecoveryHint:    "Check that the agent CLI tool is installed and accessible",
		InternalDetails: err.Error(),
	}
}

// NewProcessPipeFailure creates a failure for when setting up process I/O pipes fails.
func NewProcessPipeFailure(pipeType string, err error) FailureInfo {
	return FailureInfo{
		Category:        FailureCategoryResource,
		Code:            FailureCodeProcessPipeFailed,
		Message:         "Failed to set up agent communication channel",
		RecoveryHint:    "This may be a temporary system issue. Please try again.",
		InternalDetails: fmt.Sprintf("%s pipe error: %s", pipeType, err.Error()),
	}
}

// NewProcessTimeoutFailure creates a failure for when the agent exceeds its time limit.
func NewProcessTimeoutFailure(timeoutSeconds int) FailureInfo {
	return FailureInfo{
		Category:        FailureCategoryTransient,
		Code:            FailureCodeProcessTimeout,
		Message:         "Agent exceeded its time limit",
		Retryable:       true,
		RetryAfterSeconds: 0, // Can retry immediately with longer timeout
		RecoveryHint:    fmt.Sprintf("The agent ran longer than %d seconds. Consider increasing the timeout or breaking the task into smaller pieces.", timeoutSeconds),
		InternalDetails: fmt.Sprintf("process timed out after %d seconds", timeoutSeconds),
	}
}

// NewProcessCanceledFailure creates a failure for when the agent was manually stopped.
func NewProcessCanceledFailure() FailureInfo {
	return FailureInfo{
		Category:     FailureCategoryResource,
		Code:         FailureCodeProcessCanceled,
		Message:      "Agent was stopped",
		RecoveryHint: "The agent was manually stopped or canceled. You can spawn a new agent to continue.",
	}
}

// NewProcessExitFailure creates a failure for when the agent process exits with an error.
func NewProcessExitFailure(exitErr error, output string) FailureInfo {
	msg := "Agent process failed"
	if output != "" {
		// Truncate output for user message, keep full output in internal details
		maxLen := 200
		if len(output) > maxLen {
			msg = fmt.Sprintf("Agent process failed: %s...", output[:maxLen])
		} else {
			msg = fmt.Sprintf("Agent process failed: %s", output)
		}
	}

	return FailureInfo{
		Category:        FailureCategoryResource,
		Code:            FailureCodeProcessExitError,
		Message:         msg,
		RecoveryHint:    "Check the agent output for error details. The agent may have encountered an issue with the task or environment.",
		InternalDetails: fmt.Sprintf("exit error: %v, output: %s", exitErr, output),
	}
}

// NewRegistrationFailure creates a failure for when agent registration fails.
func NewRegistrationFailure(err error) FailureInfo {
	return FailureInfo{
		Category:        FailureCategoryResource,
		Code:            FailureCodeRegistrationFailed,
		Message:         "Failed to register agent",
		RecoveryHint:    "There may be a conflict with existing agents or a database issue. Please try again.",
		InternalDetails: err.Error(),
	}
}

// NewContainmentPrepFailure creates a failure for when containment setup fails.
func NewContainmentPrepFailure(provider string, err error) FailureInfo {
	return FailureInfo{
		Category:        FailureCategoryResource,
		Code:            FailureCodeContainmentPrepFailed,
		Message:         "Failed to prepare containerized execution",
		RecoveryHint:    "Check that Docker is running, or the agent will run without OS-level isolation.",
		InternalDetails: fmt.Sprintf("provider=%s, error=%s", provider, err.Error()),
	}
}

// --- Graceful Degradation Decisions ---

// DegradationDecision describes how to handle a partial failure.
type DegradationDecision struct {
	// ShouldContinue indicates whether the operation should proceed despite the failure.
	ShouldContinue bool

	// Reason explains why we're continuing or stopping.
	Reason string

	// WarningForUser is a message to show the user about degraded functionality.
	// Empty if no warning is needed.
	WarningForUser string

	// OriginalFailure contains the failure that triggered this decision.
	OriginalFailure *FailureInfo
}

// DecideOnContainmentFailure determines how to handle containment setup failure.
// This is the central decision point for containment degradation.
//
// Decision criteria:
//   - Docker unavailable: Continue without containment (with warning)
//   - Unknown error: Log and continue without containment
//
// We prefer degraded operation over complete failure because:
//   - Agents already have tool-level restrictions
//   - Complete failure prevents any useful work
//   - The user should be able to choose security vs. availability
func DecideOnContainmentFailure(failure FailureInfo) DegradationDecision {
	switch failure.Code {
	case FailureCodeDockerUnavailable, FailureCodeContainmentFailed:
		return DegradationDecision{
			ShouldContinue: true,
			Reason:         "containment is optional; agent will run with tool-level restrictions only",
			WarningForUser: "Running without OS-level isolation. Consider installing Docker for improved security.",
			OriginalFailure: &failure,
		}
	default:
		// Unknown containment error - still degrade gracefully
		return DegradationDecision{
			ShouldContinue: true,
			Reason:         "unknown containment error; degrading to direct execution",
			WarningForUser: "Containment setup failed. Running with limited isolation.",
			OriginalFailure: &failure,
		}
	}
}

// DecideOnHeartbeatFailure determines whether to continue after heartbeat failures.
//
// Decision criteria:
//   - Few consecutive failures: Continue with warning (transient network issue)
//   - Many consecutive failures: Stop agent to prevent orphaned locks
func DecideOnHeartbeatFailure(consecutiveFailures, maxFailures int, failure FailureInfo) DegradationDecision {
	if consecutiveFailures < maxFailures {
		return DegradationDecision{
			ShouldContinue: true,
			Reason:         fmt.Sprintf("heartbeat failure %d/%d; continuing", consecutiveFailures, maxFailures),
			OriginalFailure: &failure,
		}
	}

	return DegradationDecision{
		ShouldContinue: false,
		Reason:         fmt.Sprintf("heartbeat failed %d consecutive times; stopping agent to release locks", consecutiveFailures),
		WarningForUser: "Agent stopped due to connection issues. Your work has been saved.",
		OriginalFailure: &failure,
	}
}

// DecideOnSessionTrackingFailure determines how to handle session tracking failure.
//
// Decision criteria:
//   - Session tracking is a nice-to-have for duplicate prevention
//   - Core spawn functionality should continue if tracking fails
func DecideOnSessionTrackingFailure(failure FailureInfo) DegradationDecision {
	return DegradationDecision{
		ShouldContinue: true,
		Reason:         "session tracking is non-critical; spawn can continue",
		// No user warning - this is purely internal tracking
		OriginalFailure: &failure,
	}
}

// DecideOnRegistrationFailure determines how to handle agent registration failure.
//
// Decision criteria:
//   - Scope conflicts: Do NOT continue - would cause coordination issues
//   - Duplicate agent: Do NOT continue - would cause wasted resources
//   - Database transient: May retry after delay
//   - Other errors: Do NOT continue - registration is critical
func DecideOnRegistrationFailure(failure FailureInfo) DegradationDecision {
	switch failure.Code {
	case FailureCodeScopeConflict:
		return DegradationDecision{
			ShouldContinue:  false,
			Reason:          "scope conflict prevents safe concurrent execution",
			WarningForUser:  "Another agent is working in this area. Please wait for it to finish or stop it first.",
			OriginalFailure: &failure,
		}

	case FailureCodeDuplicateAgent:
		return DegradationDecision{
			ShouldContinue:  false,
			Reason:          "identical prompt already running",
			WarningForUser:  "An identical task is already in progress. You can monitor its progress or wait for completion.",
			OriginalFailure: &failure,
		}

	case FailureCodeDatabaseConnection, FailureCodeDatabaseTimeout:
		return DegradationDecision{
			ShouldContinue:  false,
			Reason:          "database temporarily unavailable; registration cannot proceed",
			WarningForUser:  "The service is temporarily busy. Please try again in a moment.",
			OriginalFailure: &failure,
		}

	default:
		return DegradationDecision{
			ShouldContinue:  false,
			Reason:          "registration is required before agent execution",
			WarningForUser:  failure.FormatForUser(),
			OriginalFailure: &failure,
		}
	}
}

// DecideOnProcessFailure determines how to handle agent process failure.
//
// Decision criteria:
//   - Timeout: Suggest retry with longer timeout
//   - Canceled: Normal user-initiated stop
//   - Start failed: Resource issue, check installation
//   - Exit error: Inspect output for clues
func DecideOnProcessFailure(failure FailureInfo) DegradationDecision {
	switch failure.Code {
	case FailureCodeProcessTimeout:
		return DegradationDecision{
			ShouldContinue:  false,
			Reason:          "process exceeded time limit",
			WarningForUser:  "The task took too long. Consider breaking it into smaller tasks or increasing the timeout.",
			OriginalFailure: &failure,
		}

	case FailureCodeProcessCanceled:
		return DegradationDecision{
			ShouldContinue:  false,
			Reason:          "process was manually stopped",
			WarningForUser:  "The agent was stopped. You can spawn a new one to continue the task.",
			OriginalFailure: &failure,
		}

	case FailureCodeProcessStartFailed:
		return DegradationDecision{
			ShouldContinue:  false,
			Reason:          "agent process could not start",
			WarningForUser:  "Could not start the agent. Check that the required tools are installed.",
			OriginalFailure: &failure,
		}

	case FailureCodeProcessPipeFailed:
		return DegradationDecision{
			ShouldContinue:  false,
			Reason:          "agent communication channel failed",
			WarningForUser:  "A system error occurred. Please try again.",
			OriginalFailure: &failure,
		}

	case FailureCodeProcessExitError:
		return DegradationDecision{
			ShouldContinue:  false,
			Reason:          "agent process exited with error",
			WarningForUser:  "The agent encountered an error while executing. Check the output for details.",
			OriginalFailure: &failure,
		}

	default:
		return DegradationDecision{
			ShouldContinue:  false,
			Reason:          "unknown process failure",
			WarningForUser:  failure.FormatForUser(),
			OriginalFailure: &failure,
		}
	}
}

// DecideOnOrphanCleanupFailure determines how to handle orphan cleanup failure.
//
// Decision criteria:
//   - Orphan cleanup is best-effort housekeeping
//   - Failure should not prevent server startup or operation
//   - Should be logged for operator awareness
func DecideOnOrphanCleanupFailure(failure FailureInfo, orphanCount, stillAliveCount int) DegradationDecision {
	return DegradationDecision{
		ShouldContinue: true,
		Reason: fmt.Sprintf("orphan cleanup is best-effort; found %d orphans, %d still alive",
			orphanCount, stillAliveCount),
		// No user warning - this is purely operational housekeeping
		OriginalFailure: &failure,
	}
}

// --- Safe Error Formatting ---

// FormatForUser returns a user-safe error message that doesn't leak internal details.
func (f FailureInfo) FormatForUser() string {
	var parts []string
	parts = append(parts, f.Message)
	if f.RecoveryHint != "" {
		parts = append(parts, f.RecoveryHint)
	}
	return strings.Join(parts, " ")
}

// FormatForLog returns a detailed error message for operator logs.
// This may include internal details that should not be shown to users.
func (f FailureInfo) FormatForLog() string {
	details := fmt.Sprintf("category=%s code=%s message=%q",
		f.Category, f.Code, f.Message)
	if f.InternalDetails != "" {
		details += fmt.Sprintf(" internal=%q", f.InternalDetails)
	}
	return details
}

// ToMap returns the failure as a map suitable for JSON serialization in API responses.
// Sensitive details are excluded.
func (f FailureInfo) ToMap() map[string]interface{} {
	m := map[string]interface{}{
		"category": string(f.Category),
		"code":     string(f.Code),
		"message":  f.Message,
	}
	if f.Retryable {
		m["retryable"] = true
		if f.RetryAfterSeconds > 0 {
			m["retryAfterSeconds"] = f.RetryAfterSeconds
		}
	}
	if f.RecoveryHint != "" {
		m["recoveryHint"] = f.RecoveryHint
	}
	return m
}

// ToHTTPStatus maps the failure category to an appropriate HTTP status code.
// This is the central decision point for HTTP response codes.
//
// Decision criteria:
//   - Validation failures → 400 Bad Request (client error, fix input)
//   - Conflict failures → 409 Conflict (retry after resolving conflict)
//   - Resource failures → depends on code:
//       - Not found → 404
//       - Already stopped → 400 (invalid action)
//       - Others → 503 Service Unavailable
//   - Transient failures → 503 Service Unavailable (retry later)
//   - Internal failures → 500 Internal Server Error
func (f FailureInfo) ToHTTPStatus() int {
	switch f.Category {
	case FailureCategoryNone:
		return 200 // No failure

	case FailureCategoryValidation:
		return 400 // Bad Request - client needs to fix input

	case FailureCategoryConflict:
		return 409 // Conflict - wait for existing operation

	case FailureCategoryResource:
		// Some resource errors are really "not found"
		switch f.Code {
		case FailureCodeAgentNotFound, FailureCodeScenarioNotFound:
			return 404 // Not Found
		case FailureCodeAgentAlreadyStopped:
			return 400 // Bad Request - can't stop an already stopped agent
		default:
			return 503 // Service Unavailable - resource not available
		}

	case FailureCategoryTransient:
		return 503 // Service Unavailable - try again later

	case FailureCategoryInternal:
		return 500 // Internal Server Error

	default:
		return 500 // Default to internal error for unknown categories
	}
}

// ToLogFields returns a map suitable for structured logging.
// This includes internal details for debugging.
func (f FailureInfo) ToLogFields() map[string]interface{} {
	fields := map[string]interface{}{
		"failure_category": string(f.Category),
		"failure_code":     string(f.Code),
		"failure_message":  f.Message,
		"failure_retryable": f.Retryable,
	}
	if f.InternalDetails != "" {
		fields["failure_internal_details"] = f.InternalDetails
	}
	if f.RecoveryHint != "" {
		fields["failure_recovery_hint"] = f.RecoveryHint
	}
	if f.RetryAfterSeconds > 0 {
		fields["failure_retry_after_seconds"] = f.RetryAfterSeconds
	}
	return fields
}
