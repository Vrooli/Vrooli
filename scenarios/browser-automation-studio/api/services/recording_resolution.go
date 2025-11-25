package services

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// resolveProject finds an existing project or creates a new one for the recording.
// It attempts to match by explicit ID, then by name from opts or manifest, then creates a default.
// Returns the project, whether it was newly created, and any error.
func (s *RecordingService) resolveProject(ctx context.Context, manifest *recordingManifest, opts RecordingImportOptions) (*database.Project, bool, error) {
	// First priority: explicit project ID from options
	if opts.ProjectID != nil {
		if project, err := s.repo.GetProject(ctx, *opts.ProjectID); err == nil {
			return project, false, nil
		}
	}

	// Second priority: match by name from options or manifest
	candidateNames := []string{}
	if opts.ProjectName != "" {
		candidateNames = append(candidateNames, opts.ProjectName)
	}
	if manifest.ProjectName != "" {
		candidateNames = append(candidateNames, manifest.ProjectName)
	}
	candidateNames = append(candidateNames, "Demo Browser Automations")
	candidateNames = append(candidateNames, "Extension Recordings")

	for _, name := range candidateNames {
		project, err := s.repo.GetProjectByName(ctx, name)
		if err == nil && project != nil {
			return project, false, nil
		}
	}

	// Last resort: create a new project for extension recordings
	project := &database.Project{
		ID:         uuid.New(),
		Name:       "Extension Recordings",
		FolderPath: filepath.Join("scenarios", "browser-automation-studio", "data", "projects", "extension-recordings"),
		CreatedAt:  time.Now().UTC(),
		UpdatedAt:  time.Now().UTC(),
	}

	if err := os.MkdirAll(project.FolderPath, 0o755); err != nil && s.log != nil {
		s.log.WithError(err).Warn("Failed to ensure extension recordings project directory exists")
	}

	if err := s.repo.CreateProject(ctx, project); err != nil {
		return nil, false, fmt.Errorf("failed to create recordings project: %w", err)
	}

	return project, true, nil
}

// resolveWorkflow finds an existing workflow or creates a new one for the recording.
// It attempts to match by explicit ID, then by name from opts or manifest, then creates a default.
// Returns the workflow, whether it was newly created, and any error.
func (s *RecordingService) resolveWorkflow(ctx context.Context, manifest *recordingManifest, project *database.Project, opts RecordingImportOptions) (*database.Workflow, bool, error) {
	// First priority: explicit workflow ID from options
	if opts.WorkflowID != nil {
		if workflow, err := s.repo.GetWorkflow(ctx, *opts.WorkflowID); err == nil {
			return workflow, false, nil
		}
	}

	// Second priority: match by name from options or manifest
	candidateNames := []string{}
	if opts.WorkflowName != "" {
		candidateNames = append(candidateNames, opts.WorkflowName)
	}
	if manifest.WorkflowName != "" {
		candidateNames = append(candidateNames, manifest.WorkflowName)
	}

	for _, name := range candidateNames {
		if name == "" {
			continue
		}
		workflow, err := s.repo.GetWorkflowByName(ctx, name, defaultRecordingWorkflowFolder)
		if err == nil && workflow != nil {
			return workflow, false, nil
		}
	}

	// Last resort: create a new workflow for this recording
	workflow := &database.Workflow{
		ID:         uuid.New(),
		ProjectID:  &project.ID,
		Name:       deriveWorkflowName(manifest, opts),
		FolderPath: defaultRecordingWorkflowFolder,
		FlowDefinition: database.JSONMap{
			"nodes": []any{},
			"edges": []any{},
		},
		Description: "Imported Chrome extension recording",
		Version:     1,
		CreatedBy:   "extension",
		CreatedAt:   time.Now().UTC(),
		UpdatedAt:   time.Now().UTC(),
	}

	if err := s.repo.CreateWorkflow(ctx, workflow); err != nil {
		return nil, false, fmt.Errorf("failed to create recording workflow: %w", err)
	}

	return workflow, true, nil
}
