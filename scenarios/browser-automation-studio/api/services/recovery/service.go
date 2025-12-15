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
//
// In the new architecture, execution details are stored in JSON files.
// We find stale executions by looking for "running" or "pending" executions
// that haven't been updated within the stale threshold.
func (s *Service) RecoverStaleExecutions(ctx context.Context) (*RecoveryResult, error) {
	startedAt := time.Now()
	result := &RecoveryResult{
		StartedAt: startedAt,
	}

	// Find all stale executions by listing running/pending executions
	// and checking their updated_at timestamp
	staleThreshold := time.Now().Add(-s.staleThreshold)
	staleExecutions, err := s.findStaleExecutions(ctx, staleThreshold)
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

		// In the new architecture, step details are in result JSON files.
		// We mark the execution as failed with an interrupt reason.
		// Resumability should be determined by checking the result file.
		recovery.LastStepIndex = -1 // Unknown without reading result file
		recovery.Resumable = false   // Conservative default

		// Mark the execution as failed/interrupted
		exec.Status = database.ExecutionStatusFailed
		exec.ErrorMessage = fmt.Sprintf("Execution interrupted due to service restart at %s",
			startedAt.Format(time.RFC3339))
		now := time.Now()
		exec.CompletedAt = &now
		exec.UpdatedAt = now

		if updateErr := s.repo.UpdateExecution(ctx, exec); updateErr != nil {
			s.log.WithError(updateErr).WithField("execution_id", exec.ID).Error("Failed to mark execution interrupted")
			recovery.Error = updateErr.Error()
			recovery.RecoveryAction = "error_marking_interrupted"
			result.Failed++
		} else {
			recovery.RecoveryAction = "marked_interrupted"
			result.Recovered++
			s.log.WithFields(logrus.Fields{
				"execution_id": exec.ID,
				"workflow_id":  exec.WorkflowID,
			}).Info("Recovered stale execution")
		}

		result.Executions = append(result.Executions, recovery)
	}

	result.CompletedAt = time.Now()
	result.DurationMs = time.Since(startedAt).Milliseconds()
	return result, nil
}

// findStaleExecutions finds executions that are in running/pending state
// but haven't been updated since the stale threshold.
func (s *Service) findStaleExecutions(ctx context.Context, staleThreshold time.Time) ([]*database.ExecutionIndex, error) {
	// List all executions and filter for stale ones
	// In a production system, this should be a database query with filters
	allExecutions, err := s.repo.ListExecutions(ctx, nil, 1000, 0)
	if err != nil {
		return nil, err
	}

	var stale []*database.ExecutionIndex
	for _, exec := range allExecutions {
		// Check if execution is in a running state
		if exec.Status != database.ExecutionStatusRunning && exec.Status != database.ExecutionStatusPending {
			continue
		}
		// Check if it hasn't been updated recently
		if exec.UpdatedAt.Before(staleThreshold) {
			stale = append(stale, exec)
		}
	}
	return stale, nil
}
