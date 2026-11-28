// Package recovery provides startup recovery for executions left active by a
// previous API process.
package recovery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/executionartifactlock"
)

// Service handles recovery of interrupted executions on startup.
type Service struct {
	repo           RecoveryRepository
	log            *logrus.Logger
	recordingsRoot string
}

// RecoveryRepository captures the subset of database operations required for recovery.
type RecoveryRepository interface {
	ListExecutions(ctx context.Context, query database.ExecutionQuery) ([]*database.ExecutionIndex, int, error)
	GetExecution(ctx context.Context, id uuid.UUID) (*database.ExecutionIndex, error)
	UpdateExecutionStatus(ctx context.Context, id uuid.UUID, status string, errorMessage *string, completedAt *time.Time, updatedAt time.Time) error
}

// NewService creates a recovery service using the canonical execution-artifact root.
func NewService(repo RecoveryRepository, log *logrus.Logger, recordingsRoot string) *Service {
	return &Service{repo: repo, log: log, recordingsRoot: recordingsRoot}
}

// RecoveryResult captures how many previous-process executions were finalized.
type RecoveryResult struct {
	TotalActive           int `json:"total_active"`
	Recovered             int `json:"recovered"`
	Failed                int `json:"failed"`
	Skipped               int `json:"skipped"`
	ArtifactTempsRemoved  int `json:"artifact_temps_removed"`
	ArtifactCleanupFailed int `json:"artifact_cleanup_failed"`
}

const interruptedExecutionErrorPrefix = "Execution interrupted due to service restart at "

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
	} else {
		s.log.WithField("count", len(activeExecutions)).Info("Found active executions from previous process")
	}
	recoveredAt := time.Now()

	for _, exec := range activeExecutions {
		if err := ctx.Err(); err != nil {
			result.Skipped += len(activeExecutions) - result.Recovered - result.Failed - result.Skipped
			return result, errors.Join(err, recoveryFailureSummary(result))
		}
		if exec == nil || exec.ID == uuid.Nil {
			result.Failed++
			continue
		}
		recovered := false
		updateErr := executionartifactlock.WithLock(ctx, s.recordingsRoot, exec.ID, func(lockCtx context.Context) error {
			fresh, err := s.repo.GetExecution(lockCtx, exec.ID)
			if err != nil {
				return err
			}
			if fresh == nil || (fresh.Status != database.ExecutionStatusRunning && fresh.Status != database.ExecutionStatusPending) {
				return nil
			}
			exec = fresh
			exec.Status = database.ExecutionStatusFailed
			exec.ErrorMessage = fmt.Sprintf("%s%s", interruptedExecutionErrorPrefix,
				recoveredAt.Format(time.RFC3339))
			exec.CompletedAt = &recoveredAt
			exec.UpdatedAt = recoveredAt

			errMsg := exec.ErrorMessage
			if err := s.repo.UpdateExecutionStatus(lockCtx, exec.ID, exec.Status, &errMsg, exec.CompletedAt, exec.UpdatedAt); err != nil {
				return err
			}
			recovered = true
			return nil
		})
		if updateErr != nil {
			s.log.WithError(updateErr).WithField("execution_id", exec.ID).Error("Failed to mark execution interrupted")
			result.Failed++
		} else if !recovered {
			result.Skipped++
			s.log.WithField("execution_id", exec.ID).Debug("Execution was no longer active during recovery")
		} else {
			result.Recovered++
			s.log.WithFields(logrus.Fields{
				"execution_id": exec.ID,
				"workflow_id":  exec.WorkflowID,
			}).Info("Recovered interrupted execution")
		}

	}
	if err := ctx.Err(); err != nil {
		return result, errors.Join(err, recoveryFailureSummary(result))
	}
	cleanupErr := s.cleanupRecoveredArtifactTemps(ctx, result)
	if result.Failed > 0 {
		return result, errors.Join(fmt.Errorf("failed to recover %d of %d interrupted executions", result.Failed, result.TotalActive), cleanupErr)
	}
	if cleanupErr != nil {
		return result, cleanupErr
	}

	return result, nil
}

func recoveryFailureSummary(result *RecoveryResult) error {
	if result == nil || result.Failed == 0 {
		return nil
	}
	return fmt.Errorf("failed to recover %d of %d interrupted executions", result.Failed, result.TotalActive)
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

// cleanupRecoveredArtifactTemps removes only atomic-write staging files for
// executions whose persisted service-restart failure proves the old writer was
// abandoned. It runs under the same cross-process execution lock as writers,
// then rechecks the owner row before touching the execution UUID directory.
func (s *Service) cleanupRecoveredArtifactTemps(ctx context.Context, result *RecoveryResult) error {
	failedExecutions, _, err := s.repo.ListExecutions(ctx, database.ExecutionQuery{Status: database.ExecutionStatusFailed})
	if err != nil {
		result.ArtifactCleanupFailed++
		return fmt.Errorf("find recovered executions for artifact cleanup: %w", err)
	}
	var cleanupErrors []error
	for _, execution := range failedExecutions {
		if execution == nil || execution.ID == uuid.Nil || !strings.HasPrefix(execution.ErrorMessage, interruptedExecutionErrorPrefix) {
			continue
		}
		removed := 0
		lockErr := executionartifactlock.WithLock(ctx, s.recordingsRoot, execution.ID, func(lockCtx context.Context) error {
			fresh, err := s.repo.GetExecution(lockCtx, execution.ID)
			if err != nil {
				return err
			}
			if fresh == nil || fresh.Status != database.ExecutionStatusFailed || !strings.HasPrefix(fresh.ErrorMessage, interruptedExecutionErrorPrefix) {
				return nil
			}
			removed, err = removeAbandonedArtifactTempFiles(s.recordingsRoot, execution.ID)
			return err
		})
		if lockErr != nil {
			result.ArtifactCleanupFailed++
			cleanupErrors = append(cleanupErrors, fmt.Errorf("cleanup artifacts for execution %s: %w", execution.ID, lockErr))
			continue
		}
		result.ArtifactTempsRemoved += removed
	}
	return errors.Join(cleanupErrors...)
}
