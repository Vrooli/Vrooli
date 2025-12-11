package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// FindStaleExecutions returns executions that are in 'running' or 'pending' status
// but have not received a heartbeat within the staleThreshold duration.
// These executions likely represent interrupted runs that need to be recovered.
func (r *repository) FindStaleExecutions(ctx context.Context, staleThreshold time.Duration) ([]*Execution, error) {
	cutoff := time.Now().Add(-staleThreshold)

	query := r.db.Rebind(`
		SELECT * FROM executions
		WHERE status IN ('running', 'pending')
		AND (
			last_heartbeat IS NULL AND started_at < ?
			OR last_heartbeat < ?
		)
		ORDER BY started_at DESC`)

	var executions []*Execution
	err := r.db.SelectContext(ctx, &executions, query, cutoff, cutoff)
	if err != nil {
		r.log.WithError(err).Error("Failed to find stale executions")
		return nil, fmt.Errorf("failed to find stale executions: %w", err)
	}

	return executions, nil
}

// MarkExecutionInterrupted updates an execution's status to 'failed' with an
// interruption reason. This is used during recovery to mark stale executions
// that were abandoned due to crashes or restarts.
func (r *repository) MarkExecutionInterrupted(ctx context.Context, id uuid.UUID, reason string) error {
	now := time.Now()
	query := r.db.Rebind(`
		UPDATE executions
		SET status = 'failed',
		    error = ?,
		    completed_at = ?,
		    result = jsonb_set(COALESCE(result, '{}'::jsonb), '{interrupted}', 'true')
		WHERE id = ?
		AND status IN ('running', 'pending')`)

	result, err := r.db.ExecContext(ctx, query, reason, now, id)
	if err != nil {
		r.log.WithError(err).WithField("id", id).Error("Failed to mark execution interrupted")
		return fmt.Errorf("failed to mark execution interrupted: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		r.log.WithField("id", id).Warn("No execution updated - may have already completed")
	}

	return nil
}

// GetLastSuccessfulStepIndex returns the highest step_index that completed successfully
// for the given execution. Returns -1 if no steps have completed successfully.
// This is used to determine where to resume an interrupted execution.
func (r *repository) GetLastSuccessfulStepIndex(ctx context.Context, executionID uuid.UUID) (int, error) {
	query := r.db.Rebind(`
		SELECT COALESCE(MAX(step_index), -1)
		FROM execution_steps
		WHERE execution_id = ?
		AND status = 'completed'`)

	var lastIndex int
	err := r.db.GetContext(ctx, &lastIndex, query, executionID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return -1, nil
		}
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get last successful step index")
		return -1, fmt.Errorf("failed to get last successful step index: %w", err)
	}

	return lastIndex, nil
}

// UpdateExecutionCheckpoint updates the execution's progress checkpoint.
// This should be called after each successful step to enable resumption.
func (r *repository) UpdateExecutionCheckpoint(ctx context.Context, executionID uuid.UUID, stepIndex int, progress int) error {
	now := time.Now()
	query := r.db.Rebind(`
		UPDATE executions
		SET progress = ?,
		    current_step = ?,
		    last_heartbeat = ?
		WHERE id = ?`)

	stepName := fmt.Sprintf("step_%d", stepIndex)
	_, err := r.db.ExecContext(ctx, query, progress, stepName, now, executionID)
	if err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to update execution checkpoint")
		return fmt.Errorf("failed to update execution checkpoint: %w", err)
	}

	return nil
}

// GetCompletedSteps returns all steps that completed successfully for an execution,
// ordered by step_index. This is used to restore state when resuming an interrupted execution.
func (r *repository) GetCompletedSteps(ctx context.Context, executionID uuid.UUID) ([]*ExecutionStep, error) {
	query := r.db.Rebind(`
		SELECT * FROM execution_steps
		WHERE execution_id = ?
		AND status = 'completed'
		ORDER BY step_index ASC`)

	var steps []*ExecutionStep
	if err := r.db.SelectContext(ctx, &steps, query, executionID); err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to get completed steps")
		return nil, fmt.Errorf("failed to get completed steps: %w", err)
	}

	return steps, nil
}

// GetResumableExecution returns an execution if it can be resumed.
// An execution can be resumed if:
// - It has a terminal status (failed, cancelled) with interrupted=true in result
// - It has at least one completed step
// Returns ErrNotFound if the execution doesn't exist or cannot be resumed.
func (r *repository) GetResumableExecution(ctx context.Context, id uuid.UUID) (*Execution, int, error) {
	execution, err := r.GetExecution(ctx, id)
	if err != nil {
		return nil, -1, err
	}

	// Check if this was an interrupted execution
	if execution.Status != "failed" && execution.Status != "cancelled" {
		return nil, -1, fmt.Errorf("execution status %q is not resumable (must be failed or cancelled)", execution.Status)
	}

	// Check if it's marked as interrupted (can be resumed)
	if execution.Result != nil {
		if interrupted, ok := execution.Result["interrupted"].(bool); ok && interrupted {
			// Get the last successful step index
			lastStep, stepErr := r.GetLastSuccessfulStepIndex(ctx, id)
			if stepErr != nil {
				return nil, -1, stepErr
			}
			if lastStep < 0 {
				return nil, -1, fmt.Errorf("no completed steps found, cannot resume")
			}
			return execution, lastStep, nil
		}
	}

	return nil, -1, fmt.Errorf("execution was not interrupted, cannot resume")
}
