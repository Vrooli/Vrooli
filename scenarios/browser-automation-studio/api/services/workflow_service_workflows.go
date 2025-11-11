package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/database"
)

func (s *WorkflowService) CreateWorkflow(ctx context.Context, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
	return s.CreateWorkflowWithProject(ctx, nil, name, folderPath, flowDefinition, aiPrompt)
}

// CreateWorkflowWithProject creates a new workflow with optional project association
func (s *WorkflowService) CreateWorkflowWithProject(ctx context.Context, projectID *uuid.UUID, name, folderPath string, flowDefinition map[string]any, aiPrompt string) (*database.Workflow, error) {
	if projectID == nil {
		return nil, fmt.Errorf("workflows must belong to a project so they can be synchronized with the filesystem")
	}

	project, err := s.repo.GetProject(ctx, *projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve project for workflow creation: %w", err)
	}

	if existing, err := s.repo.GetWorkflowByProjectAndName(ctx, *projectID, name); err == nil && existing != nil {
		return nil, ErrWorkflowNameConflict
	} else if err != nil && !errors.Is(err, sql.ErrNoRows) && !errors.Is(err, database.ErrNotFound) {
		return nil, fmt.Errorf("failed to check existing workflows: %w", err)
	}

	now := time.Now().UTC()
	workflow := &database.Workflow{
		ID:                    uuid.New(),
		ProjectID:             projectID,
		Name:                  name,
		FolderPath:            folderPath,
		CreatedAt:             now,
		UpdatedAt:             now,
		Tags:                  []string{},
		Version:               1,
		LastChangeSource:      fileSourceManual,
		LastChangeDescription: "Initial workflow creation",
	}

	if aiPrompt != "" {
		generated, genErr := s.generateWorkflowFromPrompt(ctx, aiPrompt)
		if genErr != nil {
			return nil, fmt.Errorf("ai workflow generation failed: %w", genErr)
		}
		workflow.FlowDefinition = generated
	} else if flowDefinition != nil {
		workflow.FlowDefinition = database.JSONMap(flowDefinition)
	} else {
		workflow.FlowDefinition = database.JSONMap{
			"nodes": []any{},
			"edges": []any{},
		}
	}
	workflow.FlowDefinition = sanitizeWorkflowDefinition(workflow.FlowDefinition)

	if err := s.repo.CreateWorkflow(ctx, workflow); err != nil {
		return nil, err
	}

	version := &database.WorkflowVersion{
		WorkflowID:        workflow.ID,
		Version:           workflow.Version,
		FlowDefinition:    workflow.FlowDefinition,
		ChangeDescription: "Initial workflow creation",
		CreatedBy:         fileSourceManual,
	}
	if err := s.repo.CreateWorkflowVersion(ctx, version); err != nil {
		return nil, err
	}

	nodes := toInterfaceSlice(workflow.FlowDefinition["nodes"])
	edges := toInterfaceSlice(workflow.FlowDefinition["edges"])
	absPath, relPath, err := s.writeWorkflowFile(project, workflow, nodes, edges, "")
	if err != nil {
		return nil, err
	}
	s.cacheWorkflowPath(workflow.ID, absPath, relPath)

	return workflow, nil
}

// ListWorkflows lists workflows with optional filtering
func (s *WorkflowService) ListWorkflows(ctx context.Context, folderPath string, limit, offset int) ([]*database.Workflow, error) {
	// When no specific folder is requested we eagerly synchronize every project so filesystem edits are reflected.
	if strings.TrimSpace(folderPath) == "" {
		projects, err := s.repo.ListProjects(ctx, 1000, 0)
		if err != nil {
			return nil, err
		}
		for _, project := range projects {
			if err := s.syncProjectWorkflows(ctx, project.ID); err != nil {
				s.log.WithError(err).WithField("project_id", project.ID).Warn("Failed to synchronize workflows before listing")
			}
		}
	}
	workflows, err := s.repo.ListWorkflows(ctx, folderPath, limit, offset)
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

// GetWorkflow gets a workflow by ID
func (s *WorkflowService) GetWorkflow(ctx context.Context, id uuid.UUID) (*database.Workflow, error) {
	workflow, err := s.repo.GetWorkflow(ctx, id)
	if err != nil {
		return nil, err
	}

	if workflow.ProjectID != nil {
		if err := s.syncProjectWorkflows(ctx, *workflow.ProjectID); err != nil {
			return nil, err
		}
		// Re-fetch in case synchronization updated the row or removed it.
		workflow, err = s.repo.GetWorkflow(ctx, id)
		if err != nil {
			return nil, err
		}
	}

	if err := s.ensureWorkflowChangeMetadata(ctx, workflow); err != nil {
		return nil, err
	}

	return workflow, nil
}

// UpdateWorkflow persists workflow edits originating from the UI, CLI, or filesystem-triggered autosave. The
// filesystem is the source of truth, so we synchronise before applying updates and increment the workflow version so
// executions can record the lineage they ran against.
func (s *WorkflowService) UpdateWorkflow(ctx context.Context, workflowID uuid.UUID, input WorkflowUpdateInput) (*database.Workflow, error) {
	current, err := s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if current.ProjectID == nil {
		return nil, fmt.Errorf("workflow %s is not associated with a project; cannot update file-backed workflows", workflowID)
	}

	if err := s.syncProjectWorkflows(ctx, *current.ProjectID); err != nil {
		return nil, err
	}

	// Reload after sync in case the filesystem introduced a new revision.
	current, err = s.repo.GetWorkflow(ctx, workflowID)
	if err != nil {
		return nil, err
	}

	if input.ExpectedVersion == nil {
		sourceLabel := strings.TrimSpace(input.Source)
		if strings.EqualFold(sourceLabel, fileSourceAutosave) {
			expected := current.Version
			input.ExpectedVersion = &expected
			if s.log != nil {
				s.log.WithFields(logrus.Fields{
					"workflow_id": workflowID,
					"version":     current.Version,
				}).Warn("Autosave update missing expected version; defaulting to current version")
			}
		}
	}

	if input.ExpectedVersion != nil && current.Version != *input.ExpectedVersion {
		return nil, fmt.Errorf("%w: expected %d, found %d", ErrWorkflowVersionConflict, *input.ExpectedVersion, current.Version)
	}

	originalName := current.Name
	originalFolder := current.FolderPath
	s.coalesceWorkflowChangeMetadata(current)
	existingFlowHash := hashWorkflowDefinition(current.FlowDefinition)

	metadataChanged := false

	if name := strings.TrimSpace(input.Name); name != "" && name != current.Name {
		current.Name = name
		metadataChanged = true
	}

	trimmedDescription := strings.TrimSpace(input.Description)
	if trimmedDescription != current.Description {
		current.Description = trimmedDescription
		metadataChanged = true
	}

	if folder := strings.TrimSpace(input.FolderPath); folder != "" {
		normalizedFolder := normalizeFolderPath(folder)
		if normalizedFolder != current.FolderPath {
			current.FolderPath = normalizedFolder
			metadataChanged = true
		}
	}

	if input.Tags != nil {
		tags := pq.StringArray(input.Tags)
		if !equalStringArrays(current.Tags, tags) {
			current.Tags = tags
			metadataChanged = true
		}
	}

	flowPayloadProvided := input.FlowDefinition != nil || len(input.Nodes) > 0 || len(input.Edges) > 0
	flowChanged := false
	updatedDefinition := current.FlowDefinition

	if flowPayloadProvided {
		definitionCandidate := input.FlowDefinition
		if definitionCandidate == nil {
			definitionCandidate = map[string]any{}
		}
		if _, ok := definitionCandidate["nodes"]; !ok && len(input.Nodes) > 0 {
			definitionCandidate["nodes"] = input.Nodes
		}
		if _, ok := definitionCandidate["edges"]; !ok && len(input.Edges) > 0 {
			definitionCandidate["edges"] = input.Edges
		}

		normalized, normErr := normalizeFlowDefinition(definitionCandidate)
		if normErr != nil {
			return nil, fmt.Errorf("invalid workflow definition: %w", normErr)
		}

		updatedDefinition = sanitizeWorkflowDefinition(database.JSONMap(normalized))
		incomingHash := hashWorkflowDefinition(updatedDefinition)
		if incomingHash != existingFlowHash {
			flowChanged = true
		}
	}

	if !metadataChanged && !flowChanged {
		return current, nil
	}

	if flowPayloadProvided {
		current.FlowDefinition = updatedDefinition
	}

	source := firstNonEmpty(strings.TrimSpace(input.Source), fileSourceManual)

	changeDesc := strings.TrimSpace(input.ChangeDescription)
	if changeDesc == "" {
		switch {
		case flowChanged && metadataChanged:
			changeDesc = "Flow and metadata update"
		case flowChanged:
			if strings.EqualFold(source, fileSourceAutosave) {
				changeDesc = "Autosave"
			} else {
				changeDesc = "Workflow updated"
			}
		default:
			changeDesc = "Metadata update"
		}
	}

	current.LastChangeSource = source
	current.LastChangeDescription = changeDesc
	current.Version++
	current.UpdatedAt = time.Now().UTC()
	if current.CreatedAt.IsZero() {
		current.CreatedAt = current.UpdatedAt
	}

	if err := s.repo.UpdateWorkflow(ctx, current); err != nil {
		return nil, err
	}

	version := &database.WorkflowVersion{
		WorkflowID:        current.ID,
		Version:           current.Version,
		FlowDefinition:    current.FlowDefinition,
		ChangeDescription: changeDesc,
		CreatedBy:         source,
	}
	if err := s.repo.CreateWorkflowVersion(ctx, version); err != nil {
		return nil, err
	}

	project, err := s.repo.GetProject(ctx, *current.ProjectID)
	if err != nil {
		return nil, err
	}

	nodes := toInterfaceSlice(current.FlowDefinition["nodes"])
	edges := toInterfaceSlice(current.FlowDefinition["edges"])

	cacheEntry, hasEntry := s.lookupWorkflowPath(current.ID)
	preferredPath := ""
	if hasEntry && originalName == current.Name && originalFolder == current.FolderPath {
		preferredPath = cacheEntry.RelativePath
	}

	absPath, relPath, err := s.writeWorkflowFile(project, current, nodes, edges, preferredPath)
	if err != nil {
		return nil, err
	}
	s.cacheWorkflowPath(current.ID, absPath, relPath)

	if hasEntry && preferredPath == "" && cacheEntry.RelativePath != "" && cacheEntry.RelativePath != relPath {
		if removeErr := os.Remove(cacheEntry.AbsolutePath); removeErr != nil && !errors.Is(removeErr, os.ErrNotExist) {
			s.log.WithError(removeErr).WithField("path", cacheEntry.AbsolutePath).Warn("Failed to remove stale workflow file after rename")
		}
	}

	return current, nil
}
