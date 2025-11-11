package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func newWorkflowVersionSummary(version *database.WorkflowVersion) *WorkflowVersionSummary {
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

// ListWorkflowVersions returns version metadata for a workflow ordered by newest first.
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

	versions, err := s.repo.ListWorkflowVersions(ctx, workflowID, limit, offset)
	if err != nil {
		return nil, err
	}

	results := make([]*WorkflowVersionSummary, 0, len(versions))
	for _, version := range versions {
		if version == nil {
			continue
		}
		summary := newWorkflowVersionSummary(version)
		if summary == nil {
			continue
		}
		if strings.TrimSpace(summary.CreatedBy) == "" {
			summary.CreatedBy = firstNonEmpty(workflow.LastChangeSource, fileSourceManual)
		}
		results = append(results, summary)
	}

	return results, nil
}

// GetWorkflowVersion retrieves a specific workflow version summary.
func (s *WorkflowService) GetWorkflowVersion(ctx context.Context, workflowID uuid.UUID, version int) (*WorkflowVersionSummary, error) {
	if version <= 0 {
		return nil, fmt.Errorf("invalid workflow version: %d", version)
	}

	if _, err := s.repo.GetWorkflow(ctx, workflowID); err != nil {
		return nil, err
	}

	versionRow, err := s.repo.GetWorkflowVersion(ctx, workflowID, version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows), errors.Is(err, database.ErrNotFound):
			return nil, ErrWorkflowVersionNotFound
		default:
			return nil, fmt.Errorf("%w: %v", ErrWorkflowVersionNotFound, err)
		}
	}
	return newWorkflowVersionSummary(versionRow), nil
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

	versionRow, err := s.repo.GetWorkflowVersion(ctx, workflowID, version)
	if err != nil {
		switch {
		case errors.Is(err, sql.ErrNoRows), errors.Is(err, database.ErrNotFound):
			return nil, ErrWorkflowVersionNotFound
		default:
			return nil, fmt.Errorf("%w: %v", ErrWorkflowVersionNotFound, err)
		}
	}

	restoreDefinition := cloneJSONMap(versionRow.FlowDefinition)
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
		Description:       workflow.Description,
		FolderPath:        workflow.FolderPath,
		Tags:              append([]string(nil), []string(workflow.Tags)...),
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

	workflows := make(map[uuid.UUID]*database.Workflow, len(workflowIDs))
	for _, workflowID := range workflowIDs {
		if wf, err := s.repo.GetWorkflow(ctx, workflowID); err == nil {
			workflows[workflowID] = wf
		}
	}

	if err := s.repo.DeleteProjectWorkflows(ctx, projectID, workflowIDs); err != nil {
		return err
	}

	for _, workflowID := range workflowIDs {
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

func (s *WorkflowService) ensureWorkflowChangeMetadata(ctx context.Context, workflow *database.Workflow) error {
	if workflow == nil {
		return nil
	}
	if !s.coalesceWorkflowChangeMetadata(workflow) {
		return nil
	}
	return s.repo.UpdateWorkflow(ctx, workflow)
}

func (s *WorkflowService) coalesceWorkflowChangeMetadata(workflow *database.Workflow) bool {
	if workflow == nil {
		return false
	}
	changed := false
	if strings.TrimSpace(workflow.LastChangeSource) == "" {
		workflow.LastChangeSource = fileSourceManual
		changed = true
	}
	if strings.TrimSpace(workflow.LastChangeDescription) == "" {
		workflow.LastChangeDescription = "Manual save"
		changed = true
	}
	return changed
}
