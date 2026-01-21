package pipeline

import (
	"context"
	"fmt"
	"time"
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

// failStage marks a stage as failed with the given error message.
func failStage(result *StageResult, tp TimeProvider, errMsg string) {
	result.Status = StatusFailed
	result.CompletedAt = tp.Now()
	result.Error = errMsg
	appendError(result, "Stage %s failed: %s", result.Stage, errMsg)
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
