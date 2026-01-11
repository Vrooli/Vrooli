package workflow

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	autoengine "github.com/vrooli/browser-automation-studio/automation/engine"
	"github.com/vrooli/browser-automation-studio/database"
	"github.com/vrooli/browser-automation-studio/internal/paths"
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

const seedProjectName = "Demo Browser Automations"

// EnsureSeedProject makes sure the default demo project exists so CLI/tests have a stable target.
// The project is persisted as a DB index row (for queries) plus on-disk proto metadata (source of truth).
func (s *WorkflowService) EnsureSeedProject(ctx context.Context) (*database.ProjectIndex, error) {
	if s == nil {
		return nil, errors.New("workflow service is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	if existing, err := s.repo.GetProjectByName(ctx, seedProjectName); err == nil && existing != nil {
		return existing, nil
	} else if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, fmt.Errorf("lookup seed project: %w", err)
	}

	demoFolder := paths.ResolveDemoProjectFolder(s.log)
	if abs, err := filepath.Abs(demoFolder); err == nil {
		demoFolder = abs
	}
	demoFolder, err := paths.ValidateAndPrepareFolderPath(demoFolder, s.log)
	if err != nil {
		return nil, fmt.Errorf("prepare seed project folder: %w", err)
	}

	// If the folder is already indexed under a different name, reuse it.
	if existing, err := s.repo.GetProjectByFolderPath(ctx, demoFolder); err == nil && existing != nil {
		if strings.TrimSpace(existing.Name) == "" || existing.Name != seedProjectName {
			existing.Name = seedProjectName
			_ = s.repo.UpdateProject(ctx, existing)
		}
		_ = persistProjectProto(existing, "")
		return existing, nil
	} else if err != nil && !errors.Is(err, database.ErrNotFound) {
		return nil, fmt.Errorf("lookup seed project by folder: %w", err)
	}

	project := &database.ProjectIndex{
		Name:       seedProjectName,
		FolderPath: demoFolder,
	}
	if err := s.CreateProject(ctx, project, ""); err != nil {
		return nil, fmt.Errorf("create seed project: %w", err)
	}
	return project, nil
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

// DeleteProject deletes a project and all its workflows.
// If deleteFiles is true, it also deletes project files from disk:
// - The workflows/ directory
// - The .bas/ directory (import manifest, etc.)
// - The project.bas.json file
// The project folder itself is preserved.
func (s *WorkflowService) DeleteProject(ctx context.Context, id uuid.UUID, deleteFiles bool) error {
	// Get project first if we need to delete files
	var project *database.ProjectIndex
	if deleteFiles {
		var err error
		project, err = s.repo.GetProject(ctx, id)
		if err != nil {
			// If project not found, nothing to delete from disk
			if err == database.ErrNotFound {
				return s.repo.DeleteProject(ctx, id)
			}
			return err
		}
	}

	// Delete from database
	if err := s.repo.DeleteProject(ctx, id); err != nil {
		return err
	}

	// Delete files if requested
	if deleteFiles && project != nil && project.FolderPath != "" {
		s.deleteProjectFiles(project)
		// Clear manifest cache
		ClearManifestCache(id)
	}

	return nil
}

// deleteProjectFiles removes BAS-related files from the project folder.
// It deletes:
// - {project}/workflows/ directory
// - {project}/.bas/ directory
// - {project}/project.bas.json file
func (s *WorkflowService) deleteProjectFiles(project *database.ProjectIndex) {
	if project == nil || project.FolderPath == "" {
		return
	}

	// Delete workflows directory
	workflowsDir := ProjectWorkflowsDir(project)
	if err := os.RemoveAll(workflowsDir); err != nil && !os.IsNotExist(err) {
		s.log.WithError(err).WithField("path", workflowsDir).Warn("Failed to delete workflows directory")
	}

	// Delete .bas directory
	basDir := filepath.Join(project.FolderPath, ".bas")
	if err := os.RemoveAll(basDir); err != nil && !os.IsNotExist(err) {
		s.log.WithError(err).WithField("path", basDir).Warn("Failed to delete .bas directory")
	}

	// Delete project.json
	projectFile := filepath.Join(project.FolderPath, projectMetadataFileName)
	if err := os.Remove(projectFile); err != nil && !os.IsNotExist(err) {
		s.log.WithError(err).WithField("path", projectFile).Warn("Failed to delete project.json")
	}

	s.log.WithField("project_id", project.ID).Info("Deleted project files from disk")
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
