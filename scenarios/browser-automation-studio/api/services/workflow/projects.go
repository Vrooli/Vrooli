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
	stats, err := s.GetProjectsStats(ctx, []uuid.UUID{projectID})
	if err != nil {
		return nil, err
	}
	if value, ok := stats[projectID]; ok {
		return value, nil
	}
	return map[string]any{
		"workflow_count":  0,
		"execution_count": 0,
		"last_execution":  (*time.Time)(nil),
	}, nil
}

func (s *WorkflowService) GetProjectsStats(ctx context.Context, projectIDs []uuid.UUID) (map[uuid.UUID]map[string]any, error) {
	stats, err := s.repo.GetProjectsStats(ctx, projectIDs)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]map[string]any, len(projectIDs))
	for _, projectID := range projectIDs {
		if summary, ok := stats[projectID]; ok && summary != nil {
			result[projectID] = map[string]any{
				"workflow_count":  summary.WorkflowCount,
				"execution_count": summary.ExecutionCount,
				"last_execution":  summary.LastExecution,
			}
		} else {
			result[projectID] = map[string]any{
				"workflow_count":  0,
				"execution_count": 0,
				"last_execution":  (*time.Time)(nil),
			}
		}
	}
	return result, nil
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
