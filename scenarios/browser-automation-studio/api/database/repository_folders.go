package database

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// Folder repository methods

func (r *repository) CreateFolder(ctx context.Context, folder *WorkflowFolder) error {
	query := `
		INSERT INTO workflow_folders (id, path, parent_id, name, description)
		VALUES (:id, :path, :parent_id, :name, :description)`

	// Generate ID if not set
	if folder.ID == uuid.Nil {
		folder.ID = uuid.New()
	}

	_, err := r.db.NamedExecContext(ctx, query, folder)
	if err != nil {
		r.log.WithError(err).Error("Failed to create folder")
		return fmt.Errorf("failed to create folder: %w", err)
	}

	return nil
}

func (r *repository) GetFolder(ctx context.Context, path string) (*WorkflowFolder, error) {
	query := `SELECT * FROM workflow_folders WHERE path = $1`

	var folder WorkflowFolder
	err := r.db.GetContext(ctx, &folder, query, path)
	if err != nil {
		r.log.WithError(err).WithField("path", path).Error("Failed to get folder")
		return nil, fmt.Errorf("failed to get folder: %w", err)
	}

	return &folder, nil
}

func (r *repository) ListFolders(ctx context.Context) ([]*WorkflowFolder, error) {
	query := `SELECT * FROM workflow_folders ORDER BY path ASC`

	var folders []*WorkflowFolder
	err := r.db.SelectContext(ctx, &folders, query)
	if err != nil {
		r.log.WithError(err).Error("Failed to list folders")
		return nil, fmt.Errorf("failed to list folders: %w", err)
	}

	return folders, nil
}
