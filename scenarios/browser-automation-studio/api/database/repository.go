package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
)

// Repository defines the interface for database operations.
// The database serves as an INDEX for queryable data.
// Actual workflow definitions and execution details live on disk as JSON files.
type Repository interface {
	// Project operations
	CreateProject(ctx context.Context, project *ProjectIndex) error
	GetProject(ctx context.Context, id uuid.UUID) (*ProjectIndex, error)
	GetProjectByName(ctx context.Context, name string) (*ProjectIndex, error)
	GetProjectByFolderPath(ctx context.Context, folderPath string) (*ProjectIndex, error)
	UpdateProject(ctx context.Context, project *ProjectIndex) error
	DeleteProject(ctx context.Context, id uuid.UUID) error
	ListProjects(ctx context.Context, limit, offset int) ([]*ProjectIndex, error)
	GetProjectStats(ctx context.Context, projectID uuid.UUID) (*ProjectStats, error)
	GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*ProjectStats, error)

	// Workflow index operations (actual data is on disk)
	CreateWorkflow(ctx context.Context, workflow *WorkflowIndex) error
	GetWorkflow(ctx context.Context, id uuid.UUID) (*WorkflowIndex, error)
	GetWorkflowByName(ctx context.Context, name, folderPath string) (*WorkflowIndex, error)
	UpdateWorkflow(ctx context.Context, workflow *WorkflowIndex) error
	DeleteWorkflow(ctx context.Context, id uuid.UUID) error
	ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*WorkflowIndex, error)
	ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*WorkflowIndex, error)

	// Execution index operations (detailed results are on disk)
	CreateExecution(ctx context.Context, execution *ExecutionIndex) error
	GetExecution(ctx context.Context, id uuid.UUID) (*ExecutionIndex, error)
	UpdateExecution(ctx context.Context, execution *ExecutionIndex) error
	DeleteExecution(ctx context.Context, id uuid.UUID) error
	ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*ExecutionIndex, error)
	ListExecutionsByStatus(ctx context.Context, status string, limit, offset int) ([]*ExecutionIndex, error)

	// Schedule operations (must be in DB for cron queries)
	CreateSchedule(ctx context.Context, schedule *ScheduleIndex) error
	GetSchedule(ctx context.Context, id uuid.UUID) (*ScheduleIndex, error)
	UpdateSchedule(ctx context.Context, schedule *ScheduleIndex) error
	DeleteSchedule(ctx context.Context, id uuid.UUID) error
	ListSchedules(ctx context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*ScheduleIndex, error)
	GetActiveSchedulesDue(ctx context.Context, before time.Time) ([]*ScheduleIndex, error)
	UpdateScheduleNextRun(ctx context.Context, id uuid.UUID, nextRun time.Time) error
	UpdateScheduleLastRun(ctx context.Context, id uuid.UUID, lastRun time.Time) error

	// Settings operations (key-value store)
	GetSetting(ctx context.Context, key string) (string, error)
	SetSetting(ctx context.Context, key, value string) error
	DeleteSetting(ctx context.Context, key string) error
}

// repository implements the Repository interface
type repository struct {
	db  *DB
	log *logrus.Logger
}

// NewRepository creates a new repository instance
func NewRepository(db *DB, log *logrus.Logger) Repository {
	return &repository{
		db:  db,
		log: log,
	}
}

// Compile-time interface enforcement
var _ Repository = (*repository)(nil)

// ============================================================================
// Project Operations
// ============================================================================

func (r *repository) CreateProject(ctx context.Context, project *ProjectIndex) error {
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}

	query := `INSERT INTO projects (id, name, folder_path) VALUES (:id, :name, :folder_path)`
	_, err := r.db.NamedExecContext(ctx, query, project)
	if err != nil {
		r.log.WithError(err).Error("Failed to create project")
		return fmt.Errorf("failed to create project: %w", err)
	}
	return nil
}

func (r *repository) GetProject(ctx context.Context, id uuid.UUID) (*ProjectIndex, error) {
	query := r.db.Rebind(`SELECT * FROM projects WHERE id = ?`)
	var project ProjectIndex
	if err := r.db.GetContext(ctx, &project, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get project: %w", err)
	}
	return &project, nil
}

func (r *repository) GetProjectByName(ctx context.Context, name string) (*ProjectIndex, error) {
	query := r.db.Rebind(`SELECT * FROM projects WHERE name = ?`)
	var project ProjectIndex
	if err := r.db.GetContext(ctx, &project, query, name); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get project by name: %w", err)
	}
	return &project, nil
}

func (r *repository) GetProjectByFolderPath(ctx context.Context, folderPath string) (*ProjectIndex, error) {
	query := r.db.Rebind(`SELECT * FROM projects WHERE folder_path = ?`)
	var project ProjectIndex
	if err := r.db.GetContext(ctx, &project, query, folderPath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get project by folder path: %w", err)
	}
	return &project, nil
}

func (r *repository) UpdateProject(ctx context.Context, project *ProjectIndex) error {
	query := `UPDATE projects SET name = :name, folder_path = :folder_path WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, query, project)
	if err != nil {
		return fmt.Errorf("failed to update project: %w", err)
	}
	return nil
}

func (r *repository) DeleteProject(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM projects WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}
	return nil
}

func (r *repository) ListProjects(ctx context.Context, limit, offset int) ([]*ProjectIndex, error) {
	query := r.db.Rebind(`SELECT * FROM projects ORDER BY updated_at DESC LIMIT ? OFFSET ?`)
	var projects []*ProjectIndex
	if err := r.db.SelectContext(ctx, &projects, query, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}
	return projects, nil
}

func (r *repository) GetProjectStats(ctx context.Context, projectID uuid.UUID) (*ProjectStats, error) {
	statsByID, err := r.GetProjectsStats(ctx, []uuid.UUID{projectID})
	if err != nil {
		return nil, err
	}
	if stats, ok := statsByID[projectID]; ok {
		return stats, nil
	}
	return &ProjectStats{ProjectID: projectID}, nil
}

func (r *repository) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*ProjectStats, error) {
	result := make(map[uuid.UUID]*ProjectStats, len(projectIDs))
	if len(projectIDs) == 0 {
		return result, nil
	}

	query, args, err := sqlx.In(`
		SELECT
			p.id AS project_id,
			COUNT(DISTINCT w.id) AS workflow_count,
			COUNT(DISTINCT e.id) AS execution_count,
			MAX(e.started_at) AS last_execution
		FROM projects p
		LEFT JOIN workflows w ON w.project_id = p.id
		LEFT JOIN executions e ON e.workflow_id = w.id
		WHERE p.id IN (?)
		GROUP BY p.id
	`, projectIDs)
	if err != nil {
		return nil, fmt.Errorf("prepare project stats query: %w", err)
	}

	query = r.db.Rebind(query)
	var stats []*ProjectStats
	if err := r.db.SelectContext(ctx, &stats, query, args...); err != nil {
		return nil, fmt.Errorf("failed to get project stats: %w", err)
	}

	for _, row := range stats {
		if row == nil {
			continue
		}
		result[row.ProjectID] = row
	}
	for _, id := range projectIDs {
		if _, ok := result[id]; !ok {
			result[id] = &ProjectStats{ProjectID: id}
		}
	}
	return result, nil
}

// ============================================================================
// Workflow Index Operations
// ============================================================================

func (r *repository) CreateWorkflow(ctx context.Context, workflow *WorkflowIndex) error {
	if workflow.ID == uuid.Nil {
		workflow.ID = uuid.New()
	}
	if workflow.Version == 0 {
		workflow.Version = 1
	}

	query := `INSERT INTO workflows (id, project_id, name, folder_path, file_path, version)
	          VALUES (:id, :project_id, :name, :folder_path, :file_path, :version)`
	_, err := r.db.NamedExecContext(ctx, query, workflow)
	if err != nil {
		r.log.WithError(err).Error("Failed to create workflow")
		return fmt.Errorf("failed to create workflow: %w", err)
	}
	return nil
}

func (r *repository) GetWorkflow(ctx context.Context, id uuid.UUID) (*WorkflowIndex, error) {
	query := r.db.Rebind(`SELECT * FROM workflows WHERE id = ?`)
	var workflow WorkflowIndex
	if err := r.db.GetContext(ctx, &workflow, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get workflow: %w", err)
	}
	return &workflow, nil
}

func (r *repository) GetWorkflowByName(ctx context.Context, name, folderPath string) (*WorkflowIndex, error) {
	query := r.db.Rebind(`SELECT * FROM workflows WHERE name = ? AND folder_path = ?`)
	var workflow WorkflowIndex
	if err := r.db.GetContext(ctx, &workflow, query, name, folderPath); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get workflow by name: %w", err)
	}
	return &workflow, nil
}

func (r *repository) UpdateWorkflow(ctx context.Context, workflow *WorkflowIndex) error {
	query := `UPDATE workflows SET project_id = :project_id, name = :name, folder_path = :folder_path,
	          file_path = :file_path, version = :version WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, query, workflow)
	if err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}
	return nil
}

func (r *repository) DeleteWorkflow(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM workflows WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete workflow: %w", err)
	}
	return nil
}

func (r *repository) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*WorkflowIndex, error) {
	var query string
	var args []any

	if folderPath != "" {
		query = r.db.Rebind(`SELECT * FROM workflows WHERE folder_path = ? ORDER BY updated_at DESC LIMIT ? OFFSET ?`)
		args = []any{folderPath, limit, offset}
	} else {
		query = r.db.Rebind(`SELECT * FROM workflows ORDER BY updated_at DESC LIMIT ? OFFSET ?`)
		args = []any{limit, offset}
	}

	var workflows []*WorkflowIndex
	if err := r.db.SelectContext(ctx, &workflows, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}
	return workflows, nil
}

func (r *repository) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*WorkflowIndex, error) {
	query := r.db.Rebind(`SELECT * FROM workflows WHERE project_id = ? ORDER BY updated_at DESC LIMIT ? OFFSET ?`)
	var workflows []*WorkflowIndex
	if err := r.db.SelectContext(ctx, &workflows, query, projectID, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list workflows by project: %w", err)
	}
	return workflows, nil
}

// ============================================================================
// Execution Index Operations
// ============================================================================

func (r *repository) CreateExecution(ctx context.Context, execution *ExecutionIndex) error {
	if execution.ID == uuid.Nil {
		execution.ID = uuid.New()
	}

	query := `INSERT INTO executions (id, workflow_id, status, started_at, error_message, result_path)
	          VALUES (:id, :workflow_id, :status, :started_at, :error_message, :result_path)`
	_, err := r.db.NamedExecContext(ctx, query, execution)
	if err != nil {
		r.log.WithError(err).Error("Failed to create execution")
		return fmt.Errorf("failed to create execution: %w", err)
	}
	return nil
}

func (r *repository) GetExecution(ctx context.Context, id uuid.UUID) (*ExecutionIndex, error) {
	query := r.db.Rebind(`SELECT * FROM executions WHERE id = ?`)
	var execution ExecutionIndex
	if err := r.db.GetContext(ctx, &execution, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get execution: %w", err)
	}
	return &execution, nil
}

func (r *repository) UpdateExecution(ctx context.Context, execution *ExecutionIndex) error {
	query := `UPDATE executions SET status = :status, completed_at = :completed_at,
	          error_message = :error_message, result_path = :result_path WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, query, execution)
	if err != nil {
		return fmt.Errorf("failed to update execution: %w", err)
	}
	return nil
}

func (r *repository) DeleteExecution(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM executions WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete execution: %w", err)
	}
	return nil
}

func (r *repository) ListExecutions(ctx context.Context, workflowID *uuid.UUID, limit, offset int) ([]*ExecutionIndex, error) {
	var query string
	var args []any

	if workflowID != nil {
		query = r.db.Rebind(`SELECT * FROM executions WHERE workflow_id = ? ORDER BY started_at DESC LIMIT ? OFFSET ?`)
		args = []any{*workflowID, limit, offset}
	} else {
		query = r.db.Rebind(`SELECT * FROM executions ORDER BY started_at DESC LIMIT ? OFFSET ?`)
		args = []any{limit, offset}
	}

	var executions []*ExecutionIndex
	if err := r.db.SelectContext(ctx, &executions, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list executions: %w", err)
	}
	return executions, nil
}

func (r *repository) ListExecutionsByStatus(ctx context.Context, status string, limit, offset int) ([]*ExecutionIndex, error) {
	query := r.db.Rebind(`SELECT * FROM executions WHERE status = ? ORDER BY started_at DESC LIMIT ? OFFSET ?`)
	var executions []*ExecutionIndex
	if err := r.db.SelectContext(ctx, &executions, query, status, limit, offset); err != nil {
		return nil, fmt.Errorf("failed to list executions by status: %w", err)
	}
	return executions, nil
}

// ============================================================================
// Schedule Operations
// ============================================================================

func (r *repository) CreateSchedule(ctx context.Context, schedule *ScheduleIndex) error {
	if schedule.ID == uuid.Nil {
		schedule.ID = uuid.New()
	}
	if schedule.Timezone == "" {
		schedule.Timezone = "UTC"
	}

	query := `INSERT INTO schedules (id, workflow_id, name, cron_expression, timezone, is_active, parameters_json, next_run_at, last_run_at)
	          VALUES (:id, :workflow_id, :name, :cron_expression, :timezone, :is_active, :parameters_json, :next_run_at, :last_run_at)`
	_, err := r.db.NamedExecContext(ctx, query, schedule)
	if err != nil {
		r.log.WithError(err).Error("Failed to create schedule")
		return fmt.Errorf("failed to create schedule: %w", err)
	}
	return nil
}

func (r *repository) GetSchedule(ctx context.Context, id uuid.UUID) (*ScheduleIndex, error) {
	query := r.db.Rebind(`SELECT * FROM schedules WHERE id = ?`)
	var schedule ScheduleIndex
	if err := r.db.GetContext(ctx, &schedule, query, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("failed to get schedule: %w", err)
	}
	return &schedule, nil
}

func (r *repository) UpdateSchedule(ctx context.Context, schedule *ScheduleIndex) error {
	query := `UPDATE schedules SET workflow_id = :workflow_id, name = :name, cron_expression = :cron_expression,
	          timezone = :timezone, is_active = :is_active, parameters_json = :parameters_json,
	          next_run_at = :next_run_at, last_run_at = :last_run_at WHERE id = :id`
	_, err := r.db.NamedExecContext(ctx, query, schedule)
	if err != nil {
		return fmt.Errorf("failed to update schedule: %w", err)
	}
	return nil
}

func (r *repository) DeleteSchedule(ctx context.Context, id uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM schedules WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete schedule: %w", err)
	}
	return nil
}

func (r *repository) ListSchedules(ctx context.Context, workflowID *uuid.UUID, activeOnly bool, limit, offset int) ([]*ScheduleIndex, error) {
	var query string
	var args []any

	if workflowID != nil && activeOnly {
		query = r.db.Rebind(`SELECT * FROM schedules WHERE workflow_id = ? AND is_active = true ORDER BY next_run_at ASC NULLS LAST LIMIT ? OFFSET ?`)
		args = []any{*workflowID, limit, offset}
	} else if workflowID != nil {
		query = r.db.Rebind(`SELECT * FROM schedules WHERE workflow_id = ? ORDER BY next_run_at ASC NULLS LAST LIMIT ? OFFSET ?`)
		args = []any{*workflowID, limit, offset}
	} else if activeOnly {
		query = r.db.Rebind(`SELECT * FROM schedules WHERE is_active = true ORDER BY next_run_at ASC NULLS LAST LIMIT ? OFFSET ?`)
		args = []any{limit, offset}
	} else {
		query = r.db.Rebind(`SELECT * FROM schedules ORDER BY next_run_at ASC NULLS LAST LIMIT ? OFFSET ?`)
		args = []any{limit, offset}
	}

	var schedules []*ScheduleIndex
	if err := r.db.SelectContext(ctx, &schedules, query, args...); err != nil {
		return nil, fmt.Errorf("failed to list schedules: %w", err)
	}
	return schedules, nil
}

func (r *repository) GetActiveSchedulesDue(ctx context.Context, before time.Time) ([]*ScheduleIndex, error) {
	query := r.db.Rebind(`SELECT * FROM schedules WHERE is_active = true AND next_run_at <= ? ORDER BY next_run_at ASC`)
	var schedules []*ScheduleIndex
	if err := r.db.SelectContext(ctx, &schedules, query, before); err != nil {
		return nil, fmt.Errorf("failed to get active schedules due: %w", err)
	}
	return schedules, nil
}

func (r *repository) UpdateScheduleNextRun(ctx context.Context, id uuid.UUID, nextRun time.Time) error {
	query := r.db.Rebind(`UPDATE schedules SET next_run_at = ? WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, nextRun, id)
	if err != nil {
		return fmt.Errorf("failed to update schedule next run: %w", err)
	}
	return nil
}

func (r *repository) UpdateScheduleLastRun(ctx context.Context, id uuid.UUID, lastRun time.Time) error {
	query := r.db.Rebind(`UPDATE schedules SET last_run_at = ? WHERE id = ?`)
	_, err := r.db.ExecContext(ctx, query, lastRun, id)
	if err != nil {
		return fmt.Errorf("failed to update schedule last run: %w", err)
	}
	return nil
}

// ============================================================================
// Settings Operations
// ============================================================================

func (r *repository) GetSetting(ctx context.Context, key string) (string, error) {
	query := r.db.Rebind(`SELECT value FROM settings WHERE key = ?`)
	var value string
	if err := r.db.GetContext(ctx, &value, query, key); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNotFound
		}
		return "", fmt.Errorf("failed to get setting: %w", err)
	}
	return value, nil
}

func (r *repository) SetSetting(ctx context.Context, key, value string) error {
	// Use UPSERT pattern
	query := r.db.Rebind(`
		INSERT INTO settings (key, value) VALUES (?, ?)
		ON CONFLICT (key) DO UPDATE SET value = EXCLUDED.value
	`)
	_, err := r.db.ExecContext(ctx, query, key, value)
	if err != nil {
		return fmt.Errorf("failed to set setting: %w", err)
	}
	return nil
}

func (r *repository) DeleteSetting(ctx context.Context, key string) error {
	query := r.db.Rebind(`DELETE FROM settings WHERE key = ?`)
	_, err := r.db.ExecContext(ctx, query, key)
	if err != nil {
		return fmt.Errorf("failed to delete setting: %w", err)
	}
	return nil
}
