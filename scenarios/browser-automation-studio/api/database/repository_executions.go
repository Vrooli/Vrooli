package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// Execution repository methods

func (r *repository) CreateExecution(ctx context.Context, execution *Execution) error {
	query := `
		INSERT INTO executions (id, workflow_id, workflow_version, status, trigger_type,
		                       trigger_metadata, parameters, progress, current_step)
		VALUES (:id, :workflow_id, :workflow_version, :status, :trigger_type,
		        :trigger_metadata, :parameters, :progress, :current_step)`

	// Generate ID if not set
	if execution.ID == uuid.Nil {
		execution.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, execution)
	if err != nil {
		r.log.WithError(err).Error("Failed to create execution")
		return fmt.Errorf("failed to create execution: %w", err)
	}

	return nil
}

func (r *repository) GetExecution(ctx context.Context, id uuid.UUID) (*Execution, error) {
	query := `SELECT * FROM executions WHERE id = $1`

	var execution Execution
	err := r.db.GetContext(ctx, &execution, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.WithError(err).WithField("id", id).Error("Failed to get execution")
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}

	return &execution, nil
}

func (r *repository) UpdateExecution(ctx context.Context, execution *Execution) error {
	query := `
		UPDATE executions
		SET status = :status, completed_at = :completed_at, last_heartbeat = :last_heartbeat,
		    error = :error, result = :result, progress = :progress, current_step = :current_step
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, execution)
	if err != nil {
		r.log.WithError(err).WithField("id", execution.ID).Error("Failed to update execution")
		return fmt.Errorf("failed to update execution: %w", err)
	}

	return nil
}

func (r *repository) DeleteExecution(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM executions WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.log.WithError(err).WithField("id", id).Error("Failed to delete execution")
		return fmt.Errorf("failed to delete execution: %w", err)
	}

	return nil
}

func (r *repository) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*Execution, error) {
	var query string
	var args []any

	if workflowID != nil {
		query = `SELECT * FROM executions WHERE workflow_id = $1 ORDER BY started_at DESC LIMIT $2 OFFSET $3`
		args = []any{*workflowID, limit, offset}
	} else {
		query = `SELECT * FROM executions ORDER BY started_at DESC LIMIT $1 OFFSET $2`
		args = []any{limit, offset}
	}

	var executions []*Execution
	err := r.db.SelectContext(ctx, &executions, query, args...)
	if err != nil {
		r.log.WithError(err).Error("Failed to list executions")
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}

	return executions, nil
}

func (r *repository) CreateExecutionStep(ctx context.Context, step *ExecutionStep) error {
	query := `
		INSERT INTO execution_steps (
			id, execution_id, step_index, node_id, step_type, status,
			started_at, completed_at, duration_ms, error, input, output, metadata
		) VALUES (
			:id, :execution_id, :step_index, :node_id, :step_type, :status,
			:started_at, :completed_at, :duration_ms, :error, :input, :output, :metadata
		)`

	if step.ID == uuid.Nil {
		step.ID = uuid.New()
	}
	if step.StartedAt.IsZero() {
		step.StartedAt = time.Now()
	}

	_, err := r.db.NamedExecContext(ctx, query, step)
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{
			"execution_id": step.ExecutionID,
			"step_index":   step.StepIndex,
		}).Error("Failed to create execution step")
		return fmt.Errorf("failed to create execution step: %w", err)
	}

	return nil
}

func (r *repository) UpdateExecutionStep(ctx context.Context, step *ExecutionStep) error {
	query := `
		UPDATE execution_steps
		SET status = :status,
		    completed_at = :completed_at,
		    duration_ms = :duration_ms,
		    error = :error,
		    output = :output,
		    metadata = :metadata,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = :id`

	_, err := r.db.NamedExecContext(ctx, query, step)
	if err != nil {
		r.log.WithError(err).WithField("step_id", step.ID).Error("Failed to update execution step")
		return fmt.Errorf("failed to update execution step: %w", err)
	}

	return nil
}

func (r *repository) ListExecutionSteps(ctx context.Context, executionID uuid.UUID) ([]*ExecutionStep, error) {
	query := `
		SELECT * FROM execution_steps
		WHERE execution_id = $1
		ORDER BY step_index ASC`

	var steps []*ExecutionStep
	if err := r.db.SelectContext(ctx, &steps, query, executionID); err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to list execution steps")
		return nil, fmt.Errorf("failed to list execution steps: %w", err)
	}

	return steps, nil
}

func (r *repository) CreateExecutionArtifact(ctx context.Context, artifact *ExecutionArtifact) error {
	query := `
		INSERT INTO execution_artifacts (
			id, execution_id, step_id, step_index, artifact_type, label,
			storage_url, thumbnail_url, content_type, size_bytes, payload
		) VALUES (
			:id, :execution_id, :step_id, :step_index, :artifact_type, :label,
			:storage_url, :thumbnail_url, :content_type, :size_bytes, :payload
		)`

	if artifact.ID == uuid.Nil {
		artifact.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, artifact)
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{
			"execution_id": artifact.ExecutionID,
			"type":         artifact.ArtifactType,
		}).Error("Failed to create execution artifact")
		return fmt.Errorf("failed to create execution artifact: %w", err)
	}

	return nil
}

func (r *repository) ListExecutionArtifacts(ctx context.Context, executionID uuid.UUID) ([]*ExecutionArtifact, error) {
	query := `
		SELECT * FROM execution_artifacts
		WHERE execution_id = $1
		ORDER BY created_at ASC`

	var artifacts []*ExecutionArtifact
	if err := r.db.SelectContext(ctx, &artifacts, query, executionID); err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to list execution artifacts")
		return nil, fmt.Errorf("failed to list execution artifacts: %w", err)
	}

	return artifacts, nil
}
