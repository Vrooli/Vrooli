package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Project repository methods

func (r *repository) CreateProject(ctx context.Context, project *Project) error {
	query := `
		INSERT INTO projects (id, name, description, folder_path, created_at, updated_at)
		VALUES (:id, :name, :description, :folder_path, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)`

	// Generate ID if not set
	if project.ID == uuid.Nil {
		project.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, project)
	if err != nil {
		r.log.WithError(err).Error("Failed to create project")
		return fmt.Errorf("failed to create project: %w", err)
	}

	return nil
}

func (r *repository) GetProject(ctx context.Context, id uuid.UUID) (*Project, error) {
	query := r.db.Rebind(`SELECT * FROM projects WHERE id = ?`)

	var project Project
	err := r.db.GetContext(ctx, &project, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		r.log.WithError(err).WithField("id", id).Error("Failed to get project")
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	return &project, nil
}

func (r *repository) GetProjectByName(ctx context.Context, name string) (*Project, error) {
	query := r.db.Rebind(`SELECT * FROM projects WHERE name = ?`)

	var project Project
	err := r.db.GetContext(ctx, &project, query, name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.WithError(err).WithField("name", name).Error("Failed to get project by name")
		return nil, fmt.Errorf("failed to get project by name: %w", err)
	}

	return &project, nil
}

func (r *repository) GetProjectByFolderPath(ctx context.Context, folderPath string) (*Project, error) {
	query := r.db.Rebind(`SELECT * FROM projects WHERE folder_path = ?`)

	var project Project
	err := r.db.GetContext(ctx, &project, query, folderPath)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.WithError(err).WithField("folder_path", folderPath).Error("Failed to get project by folder path")
		return nil, fmt.Errorf("failed to get project by folder path: %w", err)
	}

	return &project, nil
}

func (r *repository) UpdateProject(ctx context.Context, project *Project) error {
	query := `
		UPDATE projects
		SET name = :name, description = :description, folder_path = :folder_path, updated_at = CURRENT_TIMESTAMP
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, project)
	if err != nil {
		r.log.WithError(err).WithField("id", project.ID).Error("Failed to update project")
		return fmt.Errorf("failed to update project: %w", err)
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

func (r *repository) DeleteProject(ctx context.Context, id uuid.UUID) error {
	// Start a transaction to ensure consistency
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to start transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete all workflows in the project first
	deleteWorkflows := r.db.Rebind(`DELETE FROM workflows WHERE project_id = ?`)
	_, err = tx.ExecContext(ctx, deleteWorkflows, id)
	if err != nil {
		r.log.WithError(err).WithField("project_id", id).Error("Failed to delete project workflows")
		return fmt.Errorf("failed to delete project workflows: %w", err)
	}

	// Delete the project
	deleteProject := r.db.Rebind(`DELETE FROM projects WHERE id = ?`)
	result, err := tx.ExecContext(ctx, deleteProject, id)
	if err != nil {
		r.log.WithError(err).WithField("id", id).Error("Failed to delete project")
		return fmt.Errorf("failed to delete project: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	if rowsAffected == 0 {
		return ErrNotFound
	}

	// Commit transaction
	if err = tx.Commit(); err != nil {
		r.log.WithError(err).Error("Failed to commit project deletion")
		return fmt.Errorf("failed to commit project deletion: %w", err)
	}

	return nil
}

func (r *repository) ListProjects(ctx context.Context, limit, offset int) ([]*Project, error) {
	query := r.db.Rebind(`SELECT * FROM projects ORDER BY updated_at DESC LIMIT ? OFFSET ?`)

	var projects []*Project
	err := r.db.SelectContext(ctx, &projects, query, limit, offset)
	if err != nil {
		r.log.WithError(err).Error("Failed to list projects")
		return nil, fmt.Errorf("failed to list projects: %w", err)
	}

	return projects, nil
}

func (r *repository) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	stats := make(map[string]any)

	// Get workflow count
	var workflowCount int
	countWorkflows := r.db.Rebind(`SELECT COUNT(*) FROM workflows WHERE project_id = ?`)
	err := r.db.GetContext(ctx, &workflowCount, countWorkflows, projectID)
	if err != nil {
		r.log.WithError(err).WithField("project_id", projectID).Error("Failed to get workflow count")
		return nil, fmt.Errorf("failed to get workflow count: %w", err)
	}
	stats["workflow_count"] = workflowCount

	// Get execution count
	var executionCount int
	executionQuery := r.db.Rebind(`
		SELECT COUNT(e.*) FROM executions e
		JOIN workflows w ON e.workflow_id = w.id
		WHERE w.project_id = ?`)
	err = r.db.GetContext(ctx, &executionCount, executionQuery, projectID)
	if err != nil {
		r.log.WithError(err).WithField("project_id", projectID).Error("Failed to get execution count")
		return nil, fmt.Errorf("failed to get execution count: %w", err)
	}
	stats["execution_count"] = executionCount

	// Get last execution date
	var lastExecution *time.Time
	lastExecQuery := r.db.Rebind(`
		SELECT MAX(e.started_at) FROM executions e
		JOIN workflows w ON e.workflow_id = w.id
		WHERE w.project_id = ?`)
	err = r.db.GetContext(ctx, &lastExecution, lastExecQuery, projectID)
	if err != nil && err.Error() != "sql: no rows in result set" {
		r.log.WithError(err).WithField("project_id", projectID).Error("Failed to get last execution date")
		return nil, fmt.Errorf("failed to get last execution date: %w", err)
	}
	stats["last_execution"] = lastExecution

	return stats, nil
}

func (r *repository) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*ProjectStats, error) {
	results := make(map[uuid.UUID]*ProjectStats, len(projectIDs))
	if len(projectIDs) == 0 {
		return results, nil
	}

	query, args, err := sqlx.In(`
SELECT p.id AS project_id,
       COUNT(DISTINCT w.id) AS workflow_count,
       COUNT(DISTINCT e.id) AS execution_count,
       MAX(e.started_at) AS last_execution
FROM projects p
LEFT JOIN workflows w ON w.project_id = p.id
LEFT JOIN executions e ON e.workflow_id = w.id
WHERE p.id IN (?)
GROUP BY p.id`, projectIDs)
	if err != nil {
		return nil, fmt.Errorf("build project stats query: %w", err)
	}
	query = r.db.Rebind(query)

	var rows []*ProjectStats
	if err := r.db.SelectContext(ctx, &rows, query, args...); err != nil {
		r.log.WithError(err).Error("Failed to get bulk project stats")
		return nil, fmt.Errorf("failed to get bulk project stats: %w", err)
	}

	for _, row := range rows {
		if row == nil {
			continue
		}
		results[row.ProjectID] = row
	}

	return results, nil
}
