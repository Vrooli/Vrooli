// Package adapters provides adapter implementations that connect
// the import usecase interfaces to the actual database repository.
package adapters

import (
	"context"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
)

// ProjectAdapter wraps Repository to implement shared.ProjectIndexer.
type ProjectAdapter struct {
	repo database.Repository
}

// NewProjectAdapter creates a new ProjectAdapter.
func NewProjectAdapter(repo database.Repository) *ProjectAdapter {
	return &ProjectAdapter{repo: repo}
}

// GetProjectByID retrieves a project by ID.
func (a *ProjectAdapter) GetProjectByID(ctx context.Context, id uuid.UUID) (*shared.ProjectIndexData, error) {
	project, err := a.repo.GetProject(ctx, id)
	if err != nil {
		return nil, err
	}
	return convertProjectToShared(project), nil
}

// GetProjectByFolderPath retrieves a project by its folder path.
func (a *ProjectAdapter) GetProjectByFolderPath(ctx context.Context, folderPath string) (*shared.ProjectIndexData, error) {
	project, err := a.repo.GetProjectByFolderPath(ctx, folderPath)
	if err != nil {
		return nil, err
	}
	return convertProjectToShared(project), nil
}

// ListProjects lists all projects with pagination.
func (a *ProjectAdapter) ListProjects(ctx context.Context, limit, offset int) ([]*shared.ProjectIndexData, error) {
	projects, err := a.repo.ListProjects(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	result := make([]*shared.ProjectIndexData, len(projects))
	for i, p := range projects {
		result[i] = convertProjectToShared(p)
	}
	return result, nil
}

// CreateProject creates a new project.
func (a *ProjectAdapter) CreateProject(ctx context.Context, project *shared.ProjectIndexData) error {
	dbProject := &database.ProjectIndex{
		ID:         project.ID,
		Name:       project.Name,
		FolderPath: project.FolderPath,
	}
	return a.repo.CreateProject(ctx, dbProject)
}

// convertProjectToShared converts a database project to shared.ProjectIndexData.
func convertProjectToShared(p *database.ProjectIndex) *shared.ProjectIndexData {
	if p == nil {
		return nil
	}
	return &shared.ProjectIndexData{
		ID:          p.ID,
		Name:        p.Name,
		Description: "", // Description not stored in DB currently
		FolderPath:  p.FolderPath,
	}
}

// Ensure ProjectAdapter implements ProjectIndexer
var _ shared.ProjectIndexer = (*ProjectAdapter)(nil)

// WorkflowAdapter wraps Repository to implement shared.WorkflowIndexer.
type WorkflowAdapter struct {
	repo database.Repository
}

// NewWorkflowAdapter creates a new WorkflowAdapter.
func NewWorkflowAdapter(repo database.Repository) *WorkflowAdapter {
	return &WorkflowAdapter{repo: repo}
}

// CreateWorkflowIndex creates a new workflow index entry.
func (a *WorkflowAdapter) CreateWorkflowIndex(ctx context.Context, projectID uuid.UUID, workflow *shared.WorkflowIndexData) error {
	// Get project to get folder path
	project, err := a.repo.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	dbWorkflow := &database.WorkflowIndex{
		ID:         workflow.ID,
		ProjectID:  &projectID,
		Name:       workflow.Name,
		FolderPath: project.FolderPath,
		FilePath:   workflow.FilePath,
		Version:    workflow.Version,
	}
	return a.repo.CreateWorkflow(ctx, dbWorkflow)
}

// GetWorkflowByFilePath retrieves a workflow by its file path within a project.
func (a *WorkflowAdapter) GetWorkflowByFilePath(ctx context.Context, projectID uuid.UUID, filePath string) (*shared.WorkflowIndexData, error) {
	// Get project to get folder path
	project, err := a.repo.GetProject(ctx, projectID)
	if err != nil {
		return nil, err
	}

	// Get workflow by name (file path is not directly queryable, use the folder path filter)
	workflows, err := a.repo.ListWorkflowsByProject(ctx, projectID, 1000, 0)
	if err != nil {
		return nil, err
	}

	// Find by file path
	for _, w := range workflows {
		if w.FilePath == filePath {
			return convertWorkflowToShared(w, project.FolderPath), nil
		}
	}

	return nil, nil // Not found
}

// GetWorkflowByID retrieves a workflow by ID.
func (a *WorkflowAdapter) GetWorkflowByID(ctx context.Context, id uuid.UUID) (*shared.WorkflowIndexData, error) {
	workflow, err := a.repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}
	return convertWorkflowToShared(workflow, ""), nil
}

// UpdateWorkflowIndex updates an existing workflow index entry.
func (a *WorkflowAdapter) UpdateWorkflowIndex(ctx context.Context, workflow *shared.WorkflowIndexData) error {
	existing, err := a.repo.GetWorkflow(ctx, workflow.ID)
	if err != nil {
		return err
	}

	existing.Name = workflow.Name
	existing.FilePath = workflow.FilePath
	existing.Version = workflow.Version
	return a.repo.UpdateWorkflow(ctx, existing)
}

// ListWorkflowsByProject lists all workflows for a project.
func (a *WorkflowAdapter) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID) ([]*shared.WorkflowIndexData, error) {
	workflows, err := a.repo.ListWorkflowsByProject(ctx, projectID, 1000, 0)
	if err != nil {
		return nil, err
	}

	result := make([]*shared.WorkflowIndexData, len(workflows))
	for i, w := range workflows {
		result[i] = convertWorkflowToShared(w, "")
	}
	return result, nil
}

// DeleteWorkflowIndex deletes a workflow index entry.
func (a *WorkflowAdapter) DeleteWorkflowIndex(ctx context.Context, id uuid.UUID) error {
	return a.repo.DeleteWorkflow(ctx, id)
}

// convertWorkflowToShared converts a database workflow to shared.WorkflowIndexData.
func convertWorkflowToShared(w *database.WorkflowIndex, _ string) *shared.WorkflowIndexData {
	if w == nil {
		return nil
	}
	result := &shared.WorkflowIndexData{
		ID:       w.ID,
		Name:     w.Name,
		FilePath: w.FilePath,
		Version:  w.Version,
	}
	if w.ProjectID != nil {
		result.ProjectID = *w.ProjectID
	}
	return result
}

// Ensure WorkflowAdapter implements WorkflowIndexer
var _ shared.WorkflowIndexer = (*WorkflowAdapter)(nil)

// WorkflowSyncService defines the interface for workflow sync operations.
// This is implemented by workflow.CatalogService.
type WorkflowSyncService interface {
	SyncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error
}

// WorkflowSyncAdapter wraps a WorkflowSyncService to implement shared.WorkflowSyncer.
type WorkflowSyncAdapter struct {
	service WorkflowSyncService
}

// NewWorkflowSyncAdapter creates a new WorkflowSyncAdapter.
func NewWorkflowSyncAdapter(service WorkflowSyncService) *WorkflowSyncAdapter {
	return &WorkflowSyncAdapter{service: service}
}

// SyncProjectWorkflows synchronizes the workflow DB index for a project from the filesystem.
func (a *WorkflowSyncAdapter) SyncProjectWorkflows(ctx context.Context, projectID uuid.UUID) error {
	return a.service.SyncProjectWorkflows(ctx, projectID)
}

// Ensure WorkflowSyncAdapter implements WorkflowSyncer
var _ shared.WorkflowSyncer = (*WorkflowSyncAdapter)(nil)
