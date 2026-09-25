// Package recovery provides startup recovery for executions left active by a
// previous API process.
package recovery

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

// Service handles recovery of interrupted executions on startup.
type Service struct {
	repo RecoveryRepository
	log  *logrus.Logger
}

// RecoveryRepository captures the subset of database operations required for recovery.
type RecoveryRepository interface {
	ListExecutions(ctx context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error)
	UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error
}

// NewService creates a new recovery service.
func NewService(repo RecoveryRepository, log *logrus.Logger) *Service {
	return &Service{repo: repo, log: log}
}

// RecoveryResult captures how many previous-process executions were finalized.
type RecoveryResult struct {
	TotalActive int `json:"total_active"`
	Recovered   int `json:"recovered"`
	Failed      int `json:"failed"`
}

// RecoverInterruptedExecutions finalizes all rows left active by the previous
// API process. Call this before serving requests or starting the scheduler so
// every persisted active row belongs to the process being replaced.
func (s *Service) RecoverInterruptedExecutions(ctx context.Context) (*RecoveryResult, error) {
	activeExecutions, err := s.findActiveExecutions(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to find active executions: %w", err)
	}

	result := &RecoveryResult{TotalActive: len(activeExecutions)}

	if len(activeExecutions) == 0 {
		s.log.Info("No interrupted executions found during recovery")
		return result, nil
	}

	s.log.WithField("count", len(activeExecutions)).Info("Found active executions from previous process")
	recoveredAt := time.Now()

	for _, exec := range activeExecutions {
		exec.Status = database.ExecutionStatusFailed
		exec.ErrorMessage = fmt.Sprintf("Execution interrupted due to service restart at %s",
			recoveredAt.Format(time.RFC3339))
		exec.CompletedAt = &recoveredAt
		exec.UpdatedAt = recoveredAt

		errMsg := exec.ErrorMessage
		if updateErr := s.repo.UpdateExecutionStatus(ctx, exec.ID, exec.Status, &errMsg, exec.CompletedAt, exec.UpdatedAt); updateErr != nil {
			s.log.WithError(updateErr).WithField("execution_id", exec.ID).Error("Failed to mark execution interrupted")
			result.Failed++
		} else {
			result.Recovered++
			s.log.WithFields(logrus.Fields{
				"execution_id": exec.ID,
				"workflow_id":  exec.WorkflowID,
			}).Info("Recovered interrupted execution")
		}

	}
	if result.Failed > 0 {
		return result, fmt.Errorf("failed to recover %d of %d interrupted executions", result.Failed, result.TotalActive)
	}

	return result, nil
}

// findActiveExecutions finds all rows that are still active. Startup recovery
// is a process-boundary operation, so UpdatedAt does not determine ownership.
func (s *Service) findActiveExecutions(ctx context.Context) ([]*database.ExecutionIndex, error) {
	var active []*database.ExecutionIndex
	for _, status := range []string{database.ExecutionStatusRunning, database.ExecutionStatusPending} {
		activeExecutions, _, err := s.repo.ListExecutions(ctx, database.ExecutionQuery{Status: status})
		if err != nil {
			return nil, err
		}
		active = append(active, activeExecutions...)
	}
	return active, nil
}
