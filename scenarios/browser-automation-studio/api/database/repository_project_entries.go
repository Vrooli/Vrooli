package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

func (r *repository) UpsertProjectEntry(ctx context.Context, entry *ProjectEntry) error {
	if entry == nil {
		return fmt.Errorf("project entry is nil")
	}
	if entry.ID == uuid.Nil {
		entry.ID = uuid.New()
	}
	if entry.ProjectID == uuid.Nil {
		return fmt.Errorf("project_id is required")
	}
	if entry.Path == "" {
		return fmt.Errorf("path is required")
	}
	if entry.Kind == "" {
		return fmt.Errorf("kind is required")
	}

	query := `
		INSERT INTO project_entries (id, project_id, path, kind, workflow_id, metadata, created_at, updated_at)
		VALUES (:id, :project_id, :path, :kind, :workflow_id, :metadata, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (project_id, path) DO UPDATE SET
			kind = EXCLUDED.kind,
			workflow_id = EXCLUDED.workflow_id,
			metadata = EXCLUDED.metadata,
			updated_at = CURRENT_TIMESTAMP
	`

	_, err := r.db.NamedExecContext(ctx, query, entry)
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{
			"project_id": entry.ProjectID,
			"path":       entry.Path,
		}).Error("Failed to upsert project entry")
		return fmt.Errorf("upsert project entry: %w", err)
	}
	return nil
}

func (r *repository) GetProjectEntry(ctx context.Context, projectID uuid.UUID, path string) (*ProjectEntry, error) {
	query := r.db.Rebind(`SELECT * FROM project_entries WHERE project_id = ? AND path = ?`)

	var entry ProjectEntry
	err := r.db.GetContext(ctx, &entry, query, projectID, path)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrNotFound
		}
		r.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"path":       path,
		}).Error("Failed to get project entry")
		return nil, fmt.Errorf("get project entry: %w", err)
	}
	return &entry, nil
}

func (r *repository) DeleteProjectEntry(ctx context.Context, projectID uuid.UUID, path string) error {
	query := r.db.Rebind(`DELETE FROM project_entries WHERE project_id = ? AND path = ?`)
	_, err := r.db.ExecContext(ctx, query, projectID, path)
	if err != nil {
		r.log.WithError(err).WithFields(logrus.Fields{
			"project_id": projectID,
			"path":       path,
		}).Error("Failed to delete project entry")
		return fmt.Errorf("delete project entry: %w", err)
	}
	return nil
}

func (r *repository) DeleteProjectEntries(ctx context.Context, projectID uuid.UUID) error {
	query := r.db.Rebind(`DELETE FROM project_entries WHERE project_id = ?`)
	_, err := r.db.ExecContext(ctx, query, projectID)
	if err != nil {
		r.log.WithError(err).WithField("project_id", projectID).Error("Failed to delete project entries")
		return fmt.Errorf("delete project entries: %w", err)
	}
	return nil
}

func (r *repository) ListProjectEntries(ctx context.Context, projectID uuid.UUID) ([]*ProjectEntry, error) {
	query := r.db.Rebind(`SELECT * FROM project_entries WHERE project_id = ? ORDER BY path ASC`)

	var entries []*ProjectEntry
	err := r.db.SelectContext(ctx, &entries, query, projectID)
	if err != nil {
		r.log.WithError(err).WithField("project_id", projectID).Error("Failed to list project entries")
		return nil, fmt.Errorf("list project entries: %w", err)
	}
	return entries, nil
}
