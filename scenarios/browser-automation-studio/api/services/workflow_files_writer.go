package services

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

func (s *WorkflowService) writeWorkflowFile(project *database.Project, workflow *database.Workflow, nodes, edges []any, preferredPath string) (string, string, error) {
	workflow.FlowDefinition = sanitizeWorkflowDefinition(workflow.FlowDefinition)
	nodes = toInterfaceSlice(workflow.FlowDefinition["nodes"])
	edges = toInterfaceSlice(workflow.FlowDefinition["edges"])
	desiredAbs, desiredRel := s.desiredWorkflowFilePath(project, workflow)
	targetAbs := desiredAbs
	targetRel := desiredRel

	if preferredPath != "" {
		targetAbs = filepath.Join(s.projectWorkflowsDir(project), filepath.FromSlash(preferredPath))
		targetRel = preferredPath
	}

	// Ensure directory exists before attempting to write.
	dir := filepath.Dir(targetAbs)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", "", fmt.Errorf("failed to create workflow directory: %w", err)
	}

	s.coalesceWorkflowChangeMetadata(workflow)

	payload := map[string]any{
		"id":          workflow.ID.String(),
		"name":        workflow.Name,
		"folder_path": workflow.FolderPath,
		"description": workflow.Description,
		"tags":        []string(workflow.Tags),
		"version":     workflow.Version,
		"flow_definition": map[string]any{
			"nodes": nodes,
			"edges": edges,
		},
		"nodes":      nodes,
		"edges":      edges,
		"updated_at": workflow.UpdatedAt.UTC().Format(time.RFC3339),
		"created_at": workflow.CreatedAt.UTC().Format(time.RFC3339),
	}
	if workflow.CreatedBy != "" {
		payload["created_by"] = workflow.CreatedBy
	}
	if strings.TrimSpace(workflow.LastChangeDescription) != "" {
		payload["change_description"] = workflow.LastChangeDescription
	}
	if strings.TrimSpace(workflow.LastChangeSource) != "" {
		payload["source"] = workflow.LastChangeSource
	}

	bytes, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", "", fmt.Errorf("failed to marshal workflow file: %w", err)
	}

	tmpFile := targetAbs + ".tmp"
	if err := os.WriteFile(tmpFile, bytes, 0o644); err != nil {
		return "", "", fmt.Errorf("failed to write workflow temp file: %w", err)
	}

	if err := os.Rename(tmpFile, targetAbs); err != nil {
		return "", "", fmt.Errorf("failed to finalize workflow file write: %w", err)
	}

	return targetAbs, targetRel, nil
}

func (s *WorkflowService) listAllProjectWorkflows(ctx context.Context, projectID uuid.UUID) ([]*database.Workflow, error) {
	var all []*database.Workflow
	offset := 0
	for {
		batch, err := s.repo.ListWorkflowsByProject(ctx, projectID, projectWorkflowPageSize, offset)
		if err != nil {
			return nil, err
		}
		if len(batch) == 0 {
			break
		}
		all = append(all, batch...)
		if len(batch) < projectWorkflowPageSize {
			break
		}
		offset += len(batch)
	}
	return all, nil
}
