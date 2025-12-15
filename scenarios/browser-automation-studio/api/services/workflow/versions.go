package workflow

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// fileWorkflowVersion represents a workflow version stored on disk.
// Versions are stored in a ".versions" subdirectory alongside workflow files.
type fileWorkflowVersion struct {
	Version           int              `json:"version"`
	WorkflowID        uuid.UUID        `json:"workflow_id"`
	CreatedAt         time.Time        `json:"created_at"`
	CreatedBy         string           `json:"created_by"`
	ChangeDescription string           `json:"change_description"`
	FlowDefinition    database.JSONMap `json:"flow_definition"`
}

// newWorkflowVersionSummaryFromFile creates a WorkflowVersionSummary from file-based version data.
func newWorkflowVersionSummaryFromFile(version *fileWorkflowVersion) *WorkflowVersionSummary {
	if version == nil {
		return nil
	}
	definition := cloneJSONMap(version.FlowDefinition)
	hash := hashWorkflowDefinition(definition)
	nodes, edges := workflowDefinitionStats(definition)
	return &WorkflowVersionSummary{
		Version:           version.Version,
		WorkflowID:        version.WorkflowID,
		CreatedAt:         version.CreatedAt,
		CreatedBy:         version.CreatedBy,
		ChangeDescription: version.ChangeDescription,
		DefinitionHash:    hash,
		NodeCount:         nodes,
		EdgeCount:         edges,
		FlowDefinition:    definition,
	}
}

// getVersionsDir returns the directory path where versions are stored for a workflow.
func (s *WorkflowService) getVersionsDir(workflow *database.Workflow) string {
	if workflow.FilePath != "" {
		// Versions stored alongside the workflow file
		dir := filepath.Dir(workflow.FilePath)
		return filepath.Join(dir, ".versions", workflow.ID.String())
	}
	// Fallback for workflows without a file path
	return filepath.Join(workflow.FolderPath, ".versions", workflow.ID.String())
}

// listFileVersions reads all version files from disk for a workflow.
func (s *WorkflowService) listFileVersions(workflow *database.Workflow) ([]*fileWorkflowVersion, error) {
	versionsDir := s.getVersionsDir(workflow)
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*fileWorkflowVersion{}, nil
		}
		return nil, fmt.Errorf("read versions directory: %w", err)
	}

	versions := make([]*fileWorkflowVersion, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		versionPath := filepath.Join(versionsDir, entry.Name())
		version, err := s.readVersionFile(versionPath)
		if err != nil {
			if s.log != nil {
				s.log.WithError(err).WithField("path", versionPath).Warn("Failed to read version file")
			}
			continue
		}
		versions = append(versions, version)
	}

	// Sort by version number descending (newest first)
	sort.Slice(versions, func(i, j int) bool {
		return versions[i].Version > versions[j].Version
	})

	return versions, nil
}

// readVersionFile reads a single version file from disk.
func (s *WorkflowService) readVersionFile(path string) (*fileWorkflowVersion, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var version fileWorkflowVersion
	if err := json.Unmarshal(data, &version); err != nil {
		return nil, fmt.Errorf("parse version file: %w", err)
	}
	return &version, nil
}

// getFileVersion retrieves a specific version from disk.
func (s *WorkflowService) getFileVersion(workflow *database.Workflow, versionNum int) (*fileWorkflowVersion, error) {
	versionsDir := s.getVersionsDir(workflow)
	versionPath := filepath.Join(versionsDir, fmt.Sprintf("v%d.json", versionNum))
	return s.readVersionFile(versionPath)
}

// ListWorkflowVersions returns version metadata for a workflow ordered by newest first.
// Reads versions from disk files stored alongside the workflow.
func (s *WorkflowService) ListWorkflowVersions(ctx context.Context, workflowID uuid.UUID, limit, offset int) ([]*WorkflowVersionSummary, error) {
	if limit <= 0 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}

	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows), errors.Is(err, database.ErrNotFound):
			return nil, database.ErrNotFound
		default:
			return nil, err
		}
	}

	versions, err := s.listFileVersions(workflow)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	if offset >= len(versions) {
		return []*WorkflowVersionSummary{}, nil
	}
	end := offset + limit
	if end > len(versions) {
		end = len(versions)
	}
	versions = versions[offset:end]

	results := make([]*WorkflowVersionSummary, 0, len(versions))
	for _, version := range versions {
		if version == nil {
			continue
		}
		summary := newWorkflowVersionSummaryFromFile(version)
		if summary == nil {
			continue
		}
		if strings.TrimSpace(summary.CreatedBy) == "" {
			summary.CreatedBy = fileSourceManual
		}
		results = append(results, summary)
	}

	return results, nil
}

// GetWorkflowVersion retrieves a specific workflow version summary from disk.
func (s *WorkflowService) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*WorkflowVersionSummary, error) {
	if version <= 0 {
		return nil, fmt.Errorf("invalid workflow version: %d", version)
	}

	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	fileVersion, err := s.getFileVersion(workflow, version)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrWorkflowVersionNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrWorkflowVersionNotFound, err)
	}
	return newWorkflowVersionSummaryFromFile(fileVersion), nil
}

// RestoreWorkflowVersion restores the specified version and creates a new version entry that mirrors the change.
func (s *WorkflowService) RestoreWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int, description string) (*database.Workflow, error) {
	if version <= 0 {
		return nil, fmt.Errorf("invalid workflow version: %d", version)
	}

	workflow, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows), errors.Is(err, database.ErrNotFound):
			return nil, ErrWorkflowVersionNotFound
		default:
			return nil, err
		}
	}

	if workflow.ProjectID == nil {
		return nil, ErrWorkflowRestoreProjectMismatch
	}

	fileVersion, err := s.getFileVersion(workflow, version)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrWorkflowVersionNotFound
		}
		return nil, fmt.Errorf("%w: %v", ErrWorkflowVersionNotFound, err)
	}

	restoreDefinition := cloneJSONMap(fileVersion.FlowDefinition)
	definitionMap := map[string]any{}
	for k, v := range restoreDefinition {
		definitionMap[k] = v
	}

	currentVersion := workflow.Version
	if strings.TrimSpace(description) == "" {
		description = fmt.Sprintf("Restored from version %d", version)
	}

	input := WorkflowUpdateInput{
		Name:              workflow.Name,
		FolderPath:        workflow.FolderPath,
		FlowDefinition:    definitionMap,
		ChangeDescription: description,
		Source:            "version-restore",
		ExpectedVersion:   &currentVersion,
	}

	updated, err := s.UpdateWorkflow(ctx, workflowID, input)
	if err != nil {
		return nil, err
	}

	return updated, nil
}

// DeleteProjectWorkflows deletes a set of workflows within a project boundary
func (s *WorkflowService) DeleteProjectWorkflows(ctx context.Context, projectID uuid.UUID, workflowIDs []uuid.UUID) error {
	if len(workflowIDs) == 0 {
		return nil
	}

	project, err := s.repo.GetProject(ctx, projectID)
	if err != nil {
		return err
	}

	if err := s.syncProjectWorkflows(ctx, projectID); err != nil {
		return err
	}

	// Collect workflow info before deletion for file cleanup
	workflows := make(map[uuid.UUID]*database.Workflow, len(workflowIDs))
	for _, workflowID := range workflowIDs {
		if wf, err := s.repo.GetWorkflow(ctx, workflowID); err == nil {
			// Verify workflow belongs to the project
			if wf.ProjectID == nil || *wf.ProjectID != projectID {
				continue
			}
			workflows[workflowID] = wf
		}
	}

	// Delete workflows from database (one at a time since we don't have batch delete)
	for workflowID := range workflows {
		if err := s.repo.DeleteWorkflow(ctx, workflowID); err != nil {
			if s.log != nil {
				s.log.WithError(err).WithField("workflow_id", workflowID).Warn("Failed to delete workflow from database")
			}
			// Continue with other deletions
		}
	}

	// Clean up workflow files from disk
	for workflowID := range workflows {
		entry, hasEntry := s.lookupWorkflowPath(workflowID)
		if hasEntry {
			if removeErr := os.Remove(entry.AbsolutePath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				s.log.WithError(removeErr).WithField("path", entry.AbsolutePath).Warn("Failed to remove workflow file during deletion")
			}
			s.removeWorkflowPath(workflowID)
			continue
		}

		if wf, ok := workflows[workflowID]; ok {
			desiredAbs, _ := s.desiredWorkflowFilePath(project, wf)
			if removeErr := os.Remove(desiredAbs); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
				s.log.WithError(removeErr).WithField("path", desiredAbs).Warn("Failed to remove inferred workflow file during deletion")
			}
		}
	}

	return nil
}

// ensureWorkflowChangeMetadata is a no-op in the simplified schema.
// Change metadata (source, description) is stored in the workflow JSON file, not the index.
func (s *WorkflowService) ensureWorkflowChangeMetadata(_ context.Context, _ *database.Workflow) error {
	// In the new architecture, change metadata is stored in the workflow JSON file on disk,
	// not in the database index. This method is retained for interface compatibility.
	return nil
}

// coalesceWorkflowChangeMetadata is a no-op in the simplified schema.
// Change metadata is stored in the workflow JSON file, not the database index.
func (s *WorkflowService) coalesceWorkflowChangeMetadata(_ *database.Workflow) bool {
	// In the new architecture, change metadata is stored in the workflow JSON file on disk,
	// not in the database index. This method is retained for interface compatibility.
	return false
}
