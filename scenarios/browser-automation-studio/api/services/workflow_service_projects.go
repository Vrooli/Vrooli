package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func (s *WorkflowService) CheckHealth() string {
	status := "healthy"

	if err := s.browserless.CheckBrowserlessHealth(); err != nil {
		s.log.WithError(err).Warn("Browserless health check failed")
		status = "degraded"
	}

	return status
}

// CreateProject creates a new project
func (s *WorkflowService) CreateProject(ctx context.Context, project *database.Project) error {
	project.CreatedAt = time.Now()
	project.UpdatedAt = time.Now()
	return s.repo.CreateProject(ctx, project)
}

// GetProject gets a project by ID
func (s *WorkflowService) GetProject(ctx context.Context, id uuid.UUID) (*database.Project, error) {
	return s.repo.GetProject(ctx, id)
}

// GetProjectByName gets a project by name
func (s *WorkflowService) GetProjectByName(ctx context.Context, name string) (*database.Project, error) {
	return s.repo.GetProjectByName(ctx, name)
}

// GetProjectByFolderPath gets a project by folder path
func (s *WorkflowService) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.Project, error) {
	return s.repo.GetProjectByFolderPath(ctx, folderPath)
}

// UpdateProject updates a project
func (s *WorkflowService) UpdateProject(ctx context.Context, project *database.Project) error {
	project.UpdatedAt = time.Now()
	return s.repo.UpdateProject(ctx, project)
}

// DeleteProject deletes a project and all its workflows
func (s *WorkflowService) DeleteProject(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteProject(ctx, id)
}

// ListProjects lists all projects
func (s *WorkflowService) ListProjects(ctx context.Context, limit, offset int) ([]*database.Project, error) {
	return s.repo.ListProjects(ctx, limit, offset)
}

// GetProjectStats gets statistics for a project
func (s *WorkflowService) GetProjectStats(ctx context.Context, projectID uuid.UUID) (map[string]any, error) {
	return s.repo.GetProjectStats(ctx, projectID)
}

// ListWorkflowsByProject lists workflows for a specific project
func (s *WorkflowService) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.Workflow, error) {
	if err := s.syncProjectWorkflows(ctx, projectID); err != nil {
		return nil, err
	}
	workflows, err := s.repo.ListWorkflowsByProject(ctx, projectID, limit, offset)
	if err != nil {
		return nil, err
	}
	for _, wf := range workflows {
		if wf == nil {
			continue
		}
		if err := s.ensureWorkflowChangeMetadata(ctx, wf); err != nil {
			s.log.WithError(err).WithField("workflow_id", wf.ID).Warn("Failed to normalize workflow change metadata")
		}
	}
	return workflows, nil
}
