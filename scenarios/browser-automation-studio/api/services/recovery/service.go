// Package recovery provides startup recovery for interrupted executions.
// It detects stale executions that were abandoned due to crashes, restarts,
// or other interruptions and marks them appropriately for user visibility.
package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// DefaultStaleThreshold is the duration after which an execution without
// a heartbeat is considered stale and should be recovered.
const DefaultStaleThreshold = 5 * time.Minute

// Service handles recovery of interrupted executions on startup.
type Service struct {
	repo           database.Repository
	log            *logrus.Logger
	staleThreshold time.Duration
}

// Option configures the recovery service.
type Option func(*Service)

// WithStaleThreshold sets a custom threshold for considering executions stale.
func WithStaleThreshold(d time.Duration) Option {
	return func(s *Service) {
		if d > 0 {
			s.staleThreshold = d
		}
	}
}

// NewService creates a new recovery service.
func NewService(repo database.Repository, log *logrus.Logger, opts ...Option) *Service {
	s := &Service{
		repo:           repo,
		log:            log,
		staleThreshold: DefaultStaleThreshold,
	}
	for _, opt := range opts {
		opt(s)
	}
	return s
}

// RecoveryResult captures the outcome of a recovery operation.
type RecoveryResult struct {
	TotalStale  int                 `json:"total_stale"`
	Recovered   int                 `json:"recovered"`
	Failed      int                 `json:"failed"`
	Resumable   int                 `json:"resumable"`
	Executions  []ExecutionRecovery `json:"executions,omitempty"`
	StartedAt   time.Time           `json:"started_at"`
	CompletedAt time.Time           `json:"completed_at"`
	DurationMs  int64               `json:"duration_ms"`
}

// ExecutionRecovery captures recovery details for a single execution.
type ExecutionRecovery struct {
	ExecutionID    string `json:"execution_id"`
	WorkflowID     string `json:"workflow_id"`
	Status         string `json:"status"`
	LastStepIndex  int    `json:"last_step_index"`
	Resumable      bool   `json:"resumable"`
	Error          string `json:"error,omitempty"`
	RecoveryAction string `json:"recovery_action"`
}

// RecoverStaleExecutions finds and marks all stale executions as interrupted.
// This should be called during startup to clean up any executions that were
// abandoned due to process termination.
func (s *Service) RecoverStaleExecutions(ctx context.Context) (*RecoveryResult, error) {
	startedAt := time.Now()
	result := &RecoveryResult{
		StartedAt: startedAt,
	}

	// Find all stale executions
	staleExecutions, err := s.repo.FindStaleExecutions(ctx, s.staleThreshold)
	if err != nil {
		return nil, fmt.Errorf("failed to find stale executions: %w", err)
	}

	result.TotalStale = len(staleExecutions)

	if len(staleExecutions) == 0 {
		s.log.Info("No stale executions found during recovery")
		result.CompletedAt = time.Now()
		result.DurationMs = time.Since(startedAt).Milliseconds()
		return result, nil
	}

	s.log.WithField("count", len(staleExecutions)).Info("Found stale executions during recovery")

	for _, exec := range staleExecutions {
		recovery := ExecutionRecovery{
			ExecutionID: exec.ID.String(),
			WorkflowID:  exec.WorkflowID.String(),
			Status:      exec.Status,
		}

		// Check if the execution has any completed steps (making it resumable)
		lastStepIndex, stepErr := s.repo.GetLastSuccessfulStepIndex(ctx, exec.ID)
		if stepErr != nil {
			s.log.WithError(stepErr).WithField("execution_id", exec.ID).Warn("Failed to get last step index")
			recovery.Error = stepErr.Error()
			recovery.RecoveryAction = "error_checking_steps"
			result.Failed++
			result.Executions = append(result.Executions, recovery)
			continue
		}

		recovery.LastStepIndex = lastStepIndex
		recovery.Resumable = lastStepIndex >= 0

		if recovery.Resumable {
			result.Resumable++
		}

		// Mark the execution as interrupted
		interruptReason := fmt.Sprintf("Execution interrupted due to service restart. Last successful step: %d. Resumable: %v",
			lastStepIndex, recovery.Resumable)

		if markErr := s.repo.MarkExecutionInterrupted(ctx, exec.ID, interruptReason); markErr != nil {
			s.log.WithError(markErr).WithField("execution_id", exec.ID).Error("Failed to mark execution interrupted")
			recovery.Error = markErr.Error()
			recovery.RecoveryAction = "error_marking_interrupted"
			result.Failed++
		} else {
			recovery.RecoveryAction = "marked_interrupted"
			result.Recovered++
			s.log.WithFields(logrus.Fields{
				"execution_id":    exec.ID,
				"workflow_id":     exec.WorkflowID,
				"last_step_index": lastStepIndex,
				"resumable":       recovery.Resumable,
			}).Info("Recovered stale execution")
		}

		result.Executions = append(result.Executions, recovery)
	}

	result.CompletedAt = time.Now()
	result.DurationMs = time.Since(startedAt).Milliseconds()

	s.log.WithFields(logrus.Fields{
		"total_stale": result.TotalStale,
		"recovered":   result.Recovered,
		"resumable":   result.Resumable,
		"failed":      result.Failed,
		"duration_ms": result.DurationMs,
	}).Info("Stale execution recovery completed")

	return result, nil
}

// CanResume checks if an execution can be resumed from its last checkpoint.
// Returns the step index to resume from, or -1 if not resumable.
func (s *Service) CanResume(ctx context.Context, executionID string) (int, bool, error) {
	// For now, resumption is tracked but not implemented.
	// This method provides the foundation for future resume functionality.
	return -1, false, nil
}
