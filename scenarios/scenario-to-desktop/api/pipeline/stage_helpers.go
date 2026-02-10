package pipeline

import (
	"context"
	"fmt"
	"time"

	"scenario-to-desktop-api/shared/errors"
)

// LogLevel represents the severity of a log entry.
type LogLevel string

const (
	// LogLevelInfo indicates informational messages about normal operation.
	LogLevelInfo LogLevel = "INFO"
	// LogLevelWarn indicates potential issues that don't prevent operation.
	LogLevelWarn LogLevel = "WARN"
	// LogLevelError indicates errors that affect operation.
	LogLevelError LogLevel = "ERROR"
	// LogLevelDebug indicates detailed debugging information.
	LogLevelDebug LogLevel = "DEBUG"
)

// logEntry creates a structured log entry with timestamp and severity.
// Format: "[TIMESTAMP] [LEVEL] message"
// This format is parseable by UI components for display and filtering.
func logEntry(level LogLevel, message string) string {
	ts := time.Now().UTC().Format(time.RFC3339)
	return fmt.Sprintf("[%s] [%s] %s", ts, level, message)
}

// appendLog adds a structured log entry to the result's logs.
func appendLog(result *StageResult, level LogLevel, format string, args ...interface{}) {
	message := fmt.Sprintf(format, args...)
	result.Logs = append(result.Logs, logEntry(level, message))
}

// appendInfo adds an INFO log entry.
func appendInfo(result *StageResult, format string, args ...interface{}) {
	appendLog(result, LogLevelInfo, format, args...)
}

// appendWarn adds a WARN log entry.
func appendWarn(result *StageResult, format string, args ...interface{}) {
	appendLog(result, LogLevelWarn, format, args...)
}

// appendError adds an ERROR log entry.
func appendError(result *StageResult, format string, args ...interface{}) {
	appendLog(result, LogLevelError, format, args...)
}

// newStageResult creates a new StageResult with running status.
func newStageResult(stageName string, tp TimeProvider) *StageResult {
	result := &StageResult{
		Stage:     stageName,
		Status:    StatusRunning,
		StartedAt: tp.Now(),
		Logs:      []string{},
	}
	appendInfo(result, "Stage %s starting", stageName)
	return result
}

// checkCancellation checks if the context is cancelled.
// Returns true if cancelled (result is updated and should be returned).
func checkCancellation(ctx context.Context, result *StageResult, tp TimeProvider) bool {
	select {
	case <-ctx.Done():
		result.Status = StatusCancelled
		result.CompletedAt = tp.Now()
		result.Error = "stage cancelled"
		appendWarn(result, "Stage %s cancelled by user request", result.Stage)
		return true
	default:
		return false
	}
}

// failStage marks a stage as failed with a DomainError.
// This provides rich error information including recovery guidance.
func failStage(result *StageResult, tp TimeProvider, domainErr *errors.DomainError) {
	result.Status = StatusFailed
	result.CompletedAt = tp.Now()
	// Include the underlying cause in the user-facing error (redacted) so operators
	// can converge without spelunking server logs.
	cause := ""
	if domainErr != nil && domainErr.Cause != nil {
		cause = errors.Redact(domainErr.Cause.Error())
	}
	if cause != "" {
		result.Error = domainErr.Message + ": " + cause
	} else {
		result.Error = domainErr.Message
	}
	result.ErrorInfo = domainErrorToStageErrorInfo(domainErr)
	appendError(result, "Stage %s failed: %s", result.Stage, result.Error)
}

// domainErrorToStageErrorInfo converts a DomainError to StageErrorInfo.
func domainErrorToStageErrorInfo(de *errors.DomainError) *StageErrorInfo {
	if de == nil {
		return nil
	}

	details := de.Details
	if details == nil {
		details = map[string]interface{}{}
	} else {
		// Copy to avoid mutating the original DomainError.
		copied := make(map[string]interface{}, len(details)+2)
		for k, v := range details {
			copied[k] = v
		}
		details = copied
	}

	if de.Cause != nil {
		// Provide a redacted, stable field for debugging. Avoid dumping raw nested objects.
		details["_cause"] = errors.Redact(de.Cause.Error())
		details["_cause_type"] = fmt.Sprintf("%T", de.Cause)
	}

	info := &StageErrorInfo{
		Code:         string(de.Code),
		Message:      de.Message,
		Domain:       de.Domain,
		Details:      details,
		Recovery:     string(de.GetRecovery()),
		RecoveryHint: de.RecoveryHint,
		ManualSteps:  de.ManualSteps,
	}

	if de.RetryStrategy != nil {
		info.RetryStrategy = &RetryStrategyInfo{
			MaxAttempts:       de.RetryStrategy.MaxAttempts,
			BackoffMs:         de.RetryStrategy.BackoffMs,
			BackoffMultiplier: de.RetryStrategy.BackoffMultiplier,
		}
	}

	if de.AutoFix != nil {
		info.AutoFix = &AutoFixInfo{
			Command:     de.AutoFix.Command,
			Description: de.AutoFix.Description,
			Safe:        de.AutoFix.Safe,
		}
	}

	if de.Diagnostic != nil {
		info.Diagnostic = &DiagnosticInfo{
			System: de.Diagnostic.System,
		}
		if de.Diagnostic.Process != nil {
			info.Diagnostic.Process = &ProcessDiagnosticInfo{
				PID:        de.Diagnostic.Process.PID,
				ExitCode:   de.Diagnostic.Process.ExitCode,
				RuntimeMs:  de.Diagnostic.Process.RuntimeMs,
				LastOutput: de.Diagnostic.Process.LastOutput,
			}
		}
	}

	return info
}

// skipStage marks a stage as skipped with the given reason.
func skipStage(result *StageResult, tp TimeProvider, reason string) {
	result.Status = StatusSkipped
	result.CompletedAt = tp.Now()
	appendInfo(result, "Stage %s skipped: %s", result.Stage, reason)
}

// completeStage marks a stage as completed with optional details.
func completeStage(result *StageResult, tp TimeProvider, details interface{}) {
	result.Status = StatusCompleted
	result.CompletedAt = tp.Now()
	result.Details = details
	appendInfo(result, "Stage %s completed successfully", result.Stage)
}

// WaitErrorKind classifies what type of wait error occurred.
type WaitErrorKind string

const (
	// WaitErrorCancelled indicates the operation was cancelled by user or timeout.
	WaitErrorCancelled WaitErrorKind = "cancelled"
	// WaitErrorTimeout indicates the operation exceeded the allowed time.
	WaitErrorTimeout WaitErrorKind = "timeout"
	// WaitErrorStore indicates the status store is not available.
	WaitErrorStore WaitErrorKind = "store_unavailable"
	// WaitErrorOther indicates an unclassified error.
	WaitErrorOther WaitErrorKind = "other"
)

// WaitError wraps a wait error with classification.
type WaitError struct {
	Kind       WaitErrorKind
	EntityType string
	EntityID   string
	Timeout    time.Duration
	Cause      error
}

// Error implements the error interface.
func (e *WaitError) Error() string {
	switch e.Kind {
	case WaitErrorCancelled:
		return fmt.Sprintf("%s %s cancelled", e.EntityType, e.EntityID)
	case WaitErrorTimeout:
		return fmt.Sprintf("%s %s timed out after %v", e.EntityType, e.EntityID, e.Timeout)
	case WaitErrorStore:
		return fmt.Sprintf("%s status store unavailable", e.EntityType)
	default:
		if e.Cause != nil {
			return fmt.Sprintf("%s %s failed: %v", e.EntityType, e.EntityID, e.Cause)
		}
		return fmt.Sprintf("%s %s failed", e.EntityType, e.EntityID)
	}
}

// Unwrap returns the underlying cause for errors.Is/As support.
func (e *WaitError) Unwrap() error {
	return e.Cause
}

// PollerConfig configures a Poller for status polling operations.
type PollerConfig struct {
	// EntityType describes what's being polled (e.g., "build", "smoke test").
	EntityType string
	// Timeout is the maximum duration to wait for completion.
	Timeout time.Duration
	// PollInterval is how often to check for status changes.
	PollInterval time.Duration
	// LogInterval is how often to log "still waiting" messages (in poll cycles).
	// A value of 10 with a 2-second poll interval logs every ~20 seconds.
	LogInterval int
}

// Poller provides a generic polling mechanism for async operations.
// It consolidates the polling pattern used across build, smoketest, and deploy stages.
type Poller[T any] struct {
	Config PollerConfig
	// GetStatus retrieves the current status. Returns (status, found).
	GetStatus func(id string) (T, bool)
	// IsComplete returns true if the status indicates the operation is done.
	IsComplete func(T) bool
}

// Wait polls until completion, cancellation, or timeout.
// Returns the final status and any error that occurred.
func (p *Poller[T]) Wait(ctx context.Context, entityID string) (T, *WaitError) {
	var zero T

	timeout := time.After(p.Config.Timeout)
	ticker := time.NewTicker(p.Config.PollInterval)
	defer ticker.Stop()

	notFoundCount := 0
	logInterval := p.Config.LogInterval
	if logInterval <= 0 {
		logInterval = 10 // Default: log every 10 polls
	}

	for {
		select {
		case <-ctx.Done():
			return zero, &WaitError{
				Kind:       WaitErrorCancelled,
				EntityType: p.Config.EntityType,
				EntityID:   entityID,
			}
		case <-timeout:
			return zero, &WaitError{
				Kind:       WaitErrorTimeout,
				EntityType: p.Config.EntityType,
				EntityID:   entityID,
				Timeout:    p.Config.Timeout,
			}
		case <-ticker.C:
			status, ok := p.GetStatus(entityID)
			if !ok {
				// Status not yet registered, keep waiting
				notFoundCount++
				if notFoundCount%logInterval == 0 {
					fmt.Printf("%s status not yet registered after %d polls, still waiting for %s...\n",
						p.Config.EntityType, notFoundCount, entityID)
				}
				continue
			}

			if p.IsComplete(status) {
				return status, nil
			}
			// Still running, continue polling
		}
	}
}

// StageExecutor provides a common pattern for stage execution.
// It handles result creation, cancellation checks, and service nil checks.
type StageExecutor struct {
	StageName    string
	TimeProvider TimeProvider
}

// NewStageExecutor creates a new stage executor.
func NewStageExecutor(stageName string, tp TimeProvider) *StageExecutor {
	return &StageExecutor{
		StageName:    stageName,
		TimeProvider: tp,
	}
}

// Execute runs a stage with common setup and teardown.
// It creates the result, checks for cancellation, and validates the service.
// The fn callback receives the result and should return the final result.
// If service is nil, the stage fails with the provided error.
func (e *StageExecutor) Execute(ctx context.Context, service interface{}, serviceErr *errors.DomainError, fn func(*StageResult) *StageResult) *StageResult {
	result := newStageResult(e.StageName, e.TimeProvider)

	// Check for cancellation before any work
	if checkCancellation(ctx, result, e.TimeProvider) {
		return result
	}

	// Validate service is configured
	if service == nil {
		failStage(result, e.TimeProvider, serviceErr)
		return result
	}

	// Execute the stage logic
	return fn(result)
}
