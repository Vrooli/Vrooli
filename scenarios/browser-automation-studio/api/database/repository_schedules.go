package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Schedule repository methods for workflow scheduling

// CreateSchedule inserts a new workflow schedule into the database.
func (r *repository) CreateSchedule(ctx context.Context, schedule *WorkflowSchedule) error {
	query := `
		INSERT INTO workflow_schedules (id, workflow_id, name, description, cron_expression,
		                                timezone, is_active, parameters, next_run_at, last_run_at)
		VALUES (:id, :workflow_id, :name, :description, :cron_expression,
		        :timezone, :is_active, :parameters, :next_run_at, :last_run_at)`

	// Generate ID if not set
	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}

	// Default timezone if not set
	if schedule.Timezone == "" {
		schedule.Timezone = "UTC"
	}

	_, err := r.db.NamedExecContext(ctx, query, schedule)
	if err != nil {
		r.log.WithError(err).WithField("schedule_name", schedule.Name).Error("Failed to create schedule")
		return fmt.Errorf("failed to create schedule: %w", err)
	}

	return nil
}

// GetSchedule retrieves a schedule by ID.
func (r *repository) GetSchedule(ctx context.Context, id uuid.UUID) (*WorkflowSchedule, error) {
	query := r.db.Rebind(`SELECT * FROM workflow_schedules WHERE id = ?`)

	var schedule WorkflowSchedule
	err := r.db.GetContext(ctx, &schedule, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.WithError(err).WithField("schedule_id", id).Error("Failed to get schedule")
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}

	return &schedule, nil
}

// ListSchedules retrieves schedules with optional filtering and pagination.
// If workflowID is provided, only schedules for that workflow are returned.
// If activeOnly is true, only active schedules are returned.
func (r *repository) ListSchedules(ctx context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*WorkflowSchedule, error) {
	var schedules []*WorkflowSchedule
	var err error

	// Build query based on filters
	baseQuery := `SELECT * FROM workflow_schedules WHERE 1=1`
	args := []any{}
	argCount := 0

	if workflowID != nil {
		argCount++
		baseQuery += fmt.Sprintf(" AND workflow_id = ?")
		args = append(args, *workflowID)
	}

	if activeOnly {
		baseQuery += " AND is_active = true"
	}

	baseQuery += " ORDER BY created_at DESC"

	if limit > 0 {
		baseQuery += " LIMIT ?"
		args = append(args, limit)
	}
	if offset > 0 {
		baseQuery += " OFFSET ?"
		args = append(args, offset)
	}

	query := r.db.Rebind(baseQuery)
	err = r.db.SelectContext(ctx, &schedules, query, args...)
	if err != nil {
		r.log.WithError(err).Error("Failed to list schedules")
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}

	return schedules, nil
}

// UpdateSchedule updates an existing schedule.
func (r *repository) UpdateSchedule(ctx context.Context, schedule *WorkflowSchedule) error {
	query := `
		UPDATE workflow_schedules SET
			name = :name,
			description = :description,
			cron_expression = :cron_expression,
			timezone = :timezone,
			is_active = :is_active,
			parameters = :parameters,
			next_run_at = :next_run_at,
			last_run_at = :last_run_at
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, schedule)
	if err != nil {
		r.log.WithError(err).WithField("schedule_id", schedule.ID).Error("Failed to update schedule")
		return fmt.Errorf("failed to update schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// DeleteSchedule removes a schedule by ID.
func (r *repository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM workflow_schedules WHERE id = ?`)

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.log.WithError(err).WithField("schedule_id", id).Error("Failed to delete schedule")
		return fmt.Errorf("failed to delete schedule: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// GetActiveSchedules retrieves all schedules that are marked as active.
// This is used by the scheduler service to load schedules at startup.
func (r *repository) GetActiveSchedules(ctx context.Context) ([]*WorkflowSchedule, error) {
	query := r.db.Rebind(`
		SELECT * FROM workflow_schedules
		WHERE is_active = true
		ORDER BY next_run_at ASC NULLS LAST`)

	var schedules []*WorkflowSchedule
	err := r.db.SelectContext(ctx, &schedules, query)
	if err != nil {
		r.log.WithError(err).Error("Failed to get active schedules")
		return nil, fmt.Errorf("failed to get active schedules: %w", err)
	}

	return schedules, nil
}

// UpdateScheduleNextRun updates only the next_run_at field for a schedule.
// This is called by the scheduler after computing the next run time.
func (r *repository) UpdateScheduleNextRun(ctx context.Context, id uuid.UUID, nextRun time.Time) error {
	query := r.db.Rebind(`UPDATE workflow_schedules SET next_run_at = ? WHERE id = ?`)

	result, err := r.db.ExecContext(ctx, query, nextRun, id)
	if err != nil {
		r.log.WithError(err).WithField("schedule_id", id).Error("Failed to update schedule next run")
		return fmt.Errorf("failed to update schedule next run: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}

// UpdateScheduleLastRun updates only the last_run_at field for a schedule.
// This is called by the scheduler after executing a scheduled workflow.
func (r *repository) UpdateScheduleLastRun(ctx context.Context, id uuid.UUID, lastRun time.Time) error {
	query := r.db.Rebind(`UPDATE workflow_schedules SET last_run_at = ? WHERE id = ?`)

	result, err := r.db.ExecContext(ctx, query, lastRun, id)
	if err != nil {
		r.log.WithError(err).WithField("schedule_id", id).Error("Failed to update schedule last run")
		return fmt.Errorf("failed to update schedule last run: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return ErrNotFound
	}

	return nil
}
