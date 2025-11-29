package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
)

// Export repository methods

func (r *repository) CreateExport(ctx context.Context, export *Export) error {
	query := `
		INSERT INTO exports (id, execution_id, workflow_id, name, format, settings,
		                    storage_url, thumbnail_url, file_size_bytes, duration_ms,
		                    frame_count, ai_caption, ai_caption_generated_at, status, error)
		VALUES (:id, :execution_id, :workflow_id, :name, :format, :settings,
		        :storage_url, :thumbnail_url, :file_size_bytes, :duration_ms,
		        :frame_count, :ai_caption, :ai_caption_generated_at, :status, :error)`

	// Generate ID if not set
	if export.ID == uuid.Nil {
		export.ID = uuid.New()
	}

	// Default status if not set
	if export.Status == "" {
		export.Status = "completed"
	}

	_, err := r.db.NamedExecContext(ctx, query, export)
	if err != nil {
		r.log.WithError(err).Error("Failed to create export")
		return fmt.Errorf("failed to create export: %w", err)
	}

	return nil
}

func (r *repository) GetExport(ctx context.Context, id uuid.UUID) (*Export, error) {
	query := `SELECT * FROM exports WHERE id = $1`

	var export Export
	err := r.db.GetContext(ctx, &export, query, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.WithError(err).WithField("export_id", id).Error("Failed to get export")
		return nil, fmt.Errorf("failed to get export: %w", err)
	}

	return &export, nil
}

func (r *repository) UpdateExport(ctx context.Context, export *Export) error {
	query := `
		UPDATE exports SET
			name = :name,
			settings = :settings,
			storage_url = :storage_url,
			thumbnail_url = :thumbnail_url,
			file_size_bytes = :file_size_bytes,
			duration_ms = :duration_ms,
			frame_count = :frame_count,
			ai_caption = :ai_caption,
			ai_caption_generated_at = :ai_caption_generated_at,
			status = :status,
			error = :error
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, export)
	if err != nil {
		r.log.WithError(err).WithField("export_id", export.ID).Error("Failed to update export")
		return fmt.Errorf("failed to update export: %w", err)
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

func (r *repository) DeleteExport(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM exports WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.log.WithError(err).WithField("export_id", id).Error("Failed to delete export")
		return fmt.Errorf("failed to delete export: %w", err)
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

func (r *repository) ListExports(ctx context.Context, limit, offset int) ([]*ExportWithDetails, error) {
	query := `
		SELECT
			e.*,
			COALESCE(w.name, '') as workflow_name,
			TO_CHAR(ex.started_at, 'YYYY-MM-DD HH24:MI:SS') as execution_date
		FROM exports e
		LEFT JOIN workflows w ON e.workflow_id = w.id
		LEFT JOIN executions ex ON e.execution_id = ex.id
		ORDER BY e.created_at DESC
		LIMIT $1 OFFSET $2`

	var exports []*ExportWithDetails
	err := r.db.SelectContext(ctx, &exports, query, limit, offset)
	if err != nil {
		r.log.WithError(err).Error("Failed to list exports")
		return nil, fmt.Errorf("failed to list exports: %w", err)
	}

	return exports, nil
}

func (r *repository) ListExportsByExecution(ctx context.Context, executionID uuid.UUID) ([]*Export, error) {
	query := `SELECT * FROM exports WHERE execution_id = $1 ORDER BY created_at DESC`

	var exports []*Export
	err := r.db.SelectContext(ctx, &exports, query, executionID)
	if err != nil {
		r.log.WithError(err).WithField("execution_id", executionID).Error("Failed to list exports by execution")
		return nil, fmt.Errorf("failed to list exports by execution: %w", err)
	}

	return exports, nil
}

func (r *repository) ListExportsByWorkflow(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*Export, error) {
	query := `SELECT * FROM exports WHERE workflow_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`

	var exports []*Export
	err := r.db.SelectContext(ctx, &exports, query, workflowID, limit, offset)
	if err != nil {
		r.log.WithError(err).WithField("workflow_id", workflowID).Error("Failed to list exports by workflow")
		return nil, fmt.Errorf("failed to list exports by workflow: %w", err)
	}

	return exports, nil
}
