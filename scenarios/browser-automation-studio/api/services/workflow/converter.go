package workflow

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

// ConvertResult contains the result of converting an external workflow.
type ConvertResult struct {
	Workflow   *basapi.WorkflowSummary
	SourcePath string
}

// ConvertExternalWorkflow converts an external workflow JSON file to native BAS format.
// External workflows have structure like:
//
//	{
//	  "metadata": { "name": "...", "description": "...", "labels": {...} },
//	  "nodes": [...],
//	  "edges": [...]
//	}
//
// Native format is WorkflowSummary proto with flow_definition containing nodes/edges.
func ConvertExternalWorkflow(project *database.ProjectIndex, content []byte, sourceRelPath string) (*ConvertResult, error) {
	if project == nil {
		return nil, errors.New("project is nil")
	}
	if len(content) == 0 {
		return nil, errors.New("empty content")
	}

	var raw map[string]any
	if err := json.Unmarshal(content, &raw); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	// Extract metadata
	name, description, tags := extractMetadata(raw, sourceRelPath)

	// Build flow definition from nodes and edges
	flowDef, err := buildFlowDefinition(raw)
	if err != nil {
		return nil, fmt.Errorf("build flow definition: %w", err)
	}

	// Determine folder path from source directory structure
	folderPath := inferFolderPath(sourceRelPath)

	// Generate new workflow ID
	workflowID := uuid.New()
	now := autocontracts.NowTimestamp()

	summary := &basapi.WorkflowSummary{
		Id:             workflowID.String(),
		ProjectId:      project.ID.String(),
		Name:           name,
		FolderPath:     folderPath,
		Description:    description,
		Tags:           tags,
		Version:        1,
		CreatedAt:      now,
		UpdatedAt:      now,
		FlowDefinition: flowDef,
	}

	return &ConvertResult{
		Workflow:   summary,
		SourcePath: sourceRelPath,
	}, nil
}

// extractMetadata extracts name, description, and tags from workflow JSON.
// Handles both external format (metadata object) and flat format.
func extractMetadata(raw map[string]any, fallbackPath string) (name, description string, tags []string) {
	// Try metadata object first (external format)
	if meta, ok := raw["metadata"].(map[string]any); ok {
		name = strings.TrimSpace(anyToString(meta["name"]))
		description = strings.TrimSpace(anyToString(meta["description"]))

		// Extract labels as tags
		if labels, ok := meta["labels"].(map[string]any); ok {
			for k, v := range labels {
				tag := fmt.Sprintf("%s:%s", k, anyToString(v))
				tags = append(tags, tag)
			}
		}
	}

	// Try flat fields (partial native format)
	if name == "" {
		name = strings.TrimSpace(anyToString(raw["name"]))
	}
	if description == "" {
		description = strings.TrimSpace(anyToString(raw["description"]))
	}
	if len(tags) == 0 {
		tags = stringSliceFromAny(raw["tags"])
	}

	// Fallback to filename if no name found
	if name == "" {
		base := filepath.Base(fallbackPath)
		name = strings.TrimSuffix(base, filepath.Ext(base))
		// Also remove typed suffixes like .action, .flow, .case
		name = strings.TrimSuffix(name, ".action")
		name = strings.TrimSuffix(name, ".flow")
		name = strings.TrimSuffix(name, ".case")
	}

	return name, description, tags
}

// buildFlowDefinition creates a WorkflowDefinitionV2 from the raw workflow JSON.
func buildFlowDefinition(raw map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	def := &basworkflows.WorkflowDefinitionV2{}

	// Check if there's already a nested definition
	if flowDef, ok := raw["flow_definition"].(map[string]any); ok {
		return marshalToFlowDefinition(flowDef)
	}
	if defV2, ok := raw["definition_v2"].(map[string]any); ok {
		return marshalToFlowDefinition(defV2)
	}

	// Build from top-level nodes and edges
	defMap := make(map[string]any)

	if nodes, ok := raw["nodes"].([]any); ok {
		defMap["nodes"] = nodes
	}
	if edges, ok := raw["edges"].([]any); ok {
		defMap["edges"] = edges
	}
	if settings, ok := raw["settings"].(map[string]any); ok {
		defMap["settings"] = settings
	}

	if len(defMap) > 0 {
		return marshalToFlowDefinition(defMap)
	}

	return def, nil
}

// marshalToFlowDefinition converts a map to WorkflowDefinitionV2 via JSON/protojson.
func marshalToFlowDefinition(defMap map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	defBytes, err := json.Marshal(defMap)
	if err != nil {
		return nil, err
	}

	var def basworkflows.WorkflowDefinitionV2
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(defBytes, &def); err != nil {
		return nil, err
	}

	return &def, nil
}

// inferFolderPath derives a logical folder path from the source file's directory.
// For example: "actions/dismiss-tutorial.json" -> "/actions"
func inferFolderPath(sourceRelPath string) string {
	dir := filepath.Dir(sourceRelPath)
	if dir == "" || dir == "." {
		return defaultWorkflowFolder
	}

	// Normalize to forward slashes and ensure leading slash
	normalized := strings.ReplaceAll(dir, "\\", "/")
	if !strings.HasPrefix(normalized, "/") {
		normalized = "/" + normalized
	}

	return normalized
}

// WriteConvertedWorkflow writes a converted workflow to the native workflows/ directory
// and returns the paths used.
func WriteConvertedWorkflow(project *database.ProjectIndex, result *ConvertResult) (absPath, relPath string, err error) {
	if project == nil || result == nil || result.Workflow == nil {
		return "", "", errors.New("invalid input")
	}

	// Write to the standard workflows/ directory
	return WriteWorkflowSummaryFile(project, result.Workflow, "")
}
