package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/database"
)

func (s *WorkflowService) CheckHealth() string {
	ok, err := s.CheckAutomationHealth(context.Background())
	if err != nil || !ok {
		if s.log != nil && err != nil {
			s.log.WithError(err).Warn("Automation engine health check failed")
		}
		return "degraded"
	}

	return "healthy"
}

// CreateProject creates a new project index row and persists proto metadata to disk.
func (s *WorkflowService) CreateProject(ctx context.Context, project *database.ProjectIndex, description string) error {
	now := time.Now().UTC()
	project.CreatedAt = now
	project.UpdatedAt = now
	if err := s.repo.CreateProject(ctx, project); err != nil {
		return err
	}
	return persistProjectProto(project, description)
}

// GetProject gets a project by ID
func (s *WorkflowService) GetProject(ctx context.Context, id uuid.UUID) (*database.ProjectIndex, error) {
	return s.repo.GetProject(ctx, id)
}

// GetProjectByName gets a project by name
func (s *WorkflowService) GetProjectByName(ctx context.Context, name string) (*database.ProjectIndex, error) {
	return s.repo.GetProjectByName(ctx, name)
}

// GetProjectByFolderPath gets a project by folder path
func (s *WorkflowService) GetProjectByFolderPath(ctx context.Context, folderPath string) (*database.ProjectIndex, error) {
	return s.repo.GetProjectByFolderPath(ctx, folderPath)
}

// UpdateProject updates a project
func (s *WorkflowService) UpdateProject(ctx context.Context, project *database.ProjectIndex, description string) error {
	project.UpdatedAt = time.Now().UTC()
	if err := s.repo.UpdateProject(ctx, project); err != nil {
		return err
	}
	return persistProjectProto(project, description)
}

// DeleteProject deletes a project and all its workflows
func (s *WorkflowService) DeleteProject(ctx context.Context, id uuid.UUID) error {
	return s.repo.DeleteProject(ctx, id)
}

// ListProjects lists all projects
func (s *WorkflowService) ListProjects(ctx context.Context, limit, offset int) ([]*database.ProjectIndex, error) {
	return s.repo.ListProjects(ctx, limit, offset)
}

// GetProjectStats gets statistics for a project
func (s *WorkflowService) GetProjectStats(ctx context.Context, projectID uuid.UUID) (*database.ProjectStats, error) {
	return s.repo.GetProjectStats(ctx, projectID)
}

func (s *WorkflowService) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]*database.ProjectStats, error) {
	return s.repo.GetProjectsStats(ctx, projectIDs)
}

// ListWorkflowsByProject lists workflows for a specific project
func (s *WorkflowService) ListWorkflowsByProject(ctx context.Context, projectID uuid.UUID, limit, offset int) ([]*database.WorkflowIndex, error) {
	if err := s.syncProjectWorkflows(ctx, projectID); err != nil {
		return nil, err
	}
	return s.repo.ListWorkflowsByProject(ctx, projectID, limit, offset)
}

// CheckAutomationHealth reports whether the configured automation engine is reachable.
func (s *WorkflowService) CheckAutomationHealth(ctx context.Context) (bool, error) {
	if s == nil {
		return false, errors.New("workflow service not configured")
	}

	engineName := autoengine.FromEnv().Resolve("")
	factory := s.engineFactory

	if strings.EqualFold(engineName, "playwright") && strings.TrimSpace(os.Getenv("PLAYWRIGHT_DRIVER_URL")) == "" {
		return false, fmt.Errorf("playwright engine selected but PLAYWRIGHT_DRIVER_URL is not configured")
	}

	if factory == nil {
		defaultFactory, err := autoengine.DefaultFactory(s.log)
		if err != nil {
			return false, err
		}
		factory = defaultFactory
	}

	eng, err := factory.Resolve(ctx, engineName)
	if err != nil {
		return false, err
	}
	if _, err := eng.Capabilities(ctx); err != nil {
		return false, err
	}
	return true, nil
}
