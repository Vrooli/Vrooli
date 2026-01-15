package pipeline

import (
	"context"
)

// newStageResult creates a new StageResult with running status.
func newStageResult(stageName string, tp TimeProvider) *StageResult {
	return &StageResult{
		Stage:     stageName,
		Status:    StatusRunning,
		StartedAt: tp.Now(),
		Logs:      []string{},
	}
}

// checkCancellation checks if the context is cancelled.
// Returns true if cancelled (result is updated and should be returned).
func checkCancellation(ctx context.Context, result *StageResult, tp TimeProvider) bool {
	select {
	case <-ctx.Done():
		result.Status = StatusCancelled
		result.CompletedAt = tp.Now()
		result.Error = "stage cancelled"
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
}

// skipStage marks a stage as skipped with the given reason.
func skipStage(result *StageResult, tp TimeProvider, reason string) {
	result.Status = StatusSkipped
	result.CompletedAt = tp.Now()
	result.Logs = append(result.Logs, reason)
}

// completeStage marks a stage as completed with optional details.
func completeStage(result *StageResult, tp TimeProvider, details interface{}) {
	result.Status = StatusCompleted
	result.CompletedAt = tp.Now()
	result.Details = details
}
