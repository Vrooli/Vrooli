package workflow

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
	"google.golang.org/protobuf/encoding/protojson"
)

// writeWorkflowFile writes a workflow to disk in V1 format.
// For V2 format, use writeWorkflowFileWithSchema with SchemaV2.
func (s *WorkflowService) writeWorkflowFile(project *database.Project, workflow *database.Workflow, nodes, edges []any, preferredPath string) (string, string, error) {
	return s.writeWorkflowFileWithSchema(project, workflow, nodes, edges, preferredPath, SchemaV1)
}

// writeWorkflowFileWithSchema writes a workflow to disk in the specified schema format.
// - SchemaV1: Legacy React Flow nodes/edges format
// - SchemaV2: Unified proto-based format with definition_v2 field
func (s *WorkflowService) writeWorkflowFileWithSchema(project *database.Project, workflow *database.Workflow, nodes, edges []any, preferredPath string, schemaVersion SchemaVersion) (string, string, error) {
	workflow.FlowDefinition = sanitizeWorkflowDefinition(workflow.FlowDefinition)
	nodes = ToInterfaceSlice(workflow.FlowDefinition["nodes"])
	edges = ToInterfaceSlice(workflow.FlowDefinition["edges"])
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

	// Build base payload common to both V1 and V2
	payload := map[string]any{
		"id":          workflow.ID.String(),
		"name":        workflow.Name,
		"folder_path": workflow.FolderPath,
		"description": workflow.Description,
		"tags":        []string(workflow.Tags),
		"version":     workflow.Version,
		"updated_at":  workflow.UpdatedAt.UTC().Format(time.RFC3339),
		"created_at":  workflow.CreatedAt.UTC().Format(time.RFC3339),
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

	if schemaVersion == SchemaV2 {
		// V2 format: Convert V1 nodes/edges to V2 definition and include both formats
		payload["schema_version"] = string(SchemaV2)

		// Extract metadata and settings from flow definition
		metadata := extractMapOrNil(workflow.FlowDefinition, "metadata")
		settings := extractMapOrNil(workflow.FlowDefinition, "settings")

		// Convert to V2 definition
		v2Def := V1ToV2Definition(nodes, edges, metadata, settings)
		if v2Def != nil {
			// Use protojson for proper JSON serialization of proto messages
			v2JSON, err := protojson.MarshalOptions{
				EmitUnpopulated: false,
				UseProtoNames:   true,
			}.Marshal(v2Def)
			if err == nil {
				var v2Map map[string]any
				if json.Unmarshal(v2JSON, &v2Map) == nil {
					payload["definition_v2"] = v2Map
				}
			}
		}

		// Also include V1 nodes/edges for backward compatibility
		payload["flow_definition"] = map[string]any{
			"nodes": nodes,
			"edges": edges,
		}
		payload["nodes"] = nodes
		payload["edges"] = edges
	} else {
		// V1 format: Legacy format with nodes/edges at top level
		payload["flow_definition"] = map[string]any{
			"nodes": nodes,
			"edges": edges,
		}
		payload["nodes"] = nodes
		payload["edges"] = edges
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

// extractMapOrNil safely extracts a map[string]any from a JSONMap, returning nil if not present.
func extractMapOrNil(m database.JSONMap, key string) map[string]any {
	if m == nil {
		return nil
	}
	if v, ok := m[key].(map[string]any); ok {
		return v
	}
	return nil
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
