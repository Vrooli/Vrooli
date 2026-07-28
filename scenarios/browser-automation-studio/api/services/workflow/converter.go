package workflow

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/database"
	workflowingress "github.com/vrooli/browser-automation-studio/internal/compat"
	basapi "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/api"
	basworkflows "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/workflows"
	"google.golang.org/protobuf/encoding/protojson"
)

// ConvertResult contains the result of converting an external workflow.
type ConvertResult struct {
	Workflow   *basapi.WorkflowSummary
	SourcePath string
}

// ExtractExternalWorkflowMetadata extracts name and folder path from external workflow content
// without performing a full conversion. This is used for deduplication lookup before conversion.
func ExtractExternalWorkflowMetadata(content []byte, sourceRelPath string) (name, folderPath string, err error) {
	if len(content) == 0 {
		return "", "", errors.New("empty content")
	}

	var raw map[string]any
	if err := json.Unmarshal(content, &raw); err != nil {
		return "", "", fmt.Errorf("invalid JSON: %w", err)
	}

	extractedName, _, _ := extractMetadata(raw, sourceRelPath)
	extractedFolderPath := inferFolderPath(sourceRelPath)

	return extractedName, extractedFolderPath, nil
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
//
// If existingID is provided, it will be used instead of generating a new UUID.
// This enables deduplication when re-syncing external workflows that already exist
// in the database with the same name and folder path.
func ConvertExternalWorkflow(project *database.ProjectIndex, content []byte, sourceRelPath string, existingID *uuid.UUID) (*ConvertResult, error) {
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

	// Use existing ID if provided (for deduplication), otherwise generate new
	var workflowID uuid.UUID
	if existingID != nil {
		workflowID = *existingID
	} else {
		workflowID = uuid.New()
	}
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

// marshalToFlowDefinition delegates map-shaped input to the sole workflow
// ingress adapter. The workflow service receives only typed V2 values after it.
func marshalToFlowDefinition(defMap map[string]any) (*basworkflows.WorkflowDefinitionV2, error) {
	raw, err := json.Marshal(defMap)
	if err != nil {
		return nil, err
	}
	raw, err = workflowingress.NormalizeExternalWorkflowDefinitionBytes(raw)
	if err != nil {
		return nil, err
	}
	definition := &basworkflows.WorkflowDefinitionV2{}
	if err := (protojson.UnmarshalOptions{DiscardUnknown: false}).Unmarshal(raw, definition); err != nil {
		return nil, err
	}
	return definition, nil
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

// WriteConvertedWorkflow writes a converted workflow in-place (same directory as source file)
// and returns the paths used.
func WriteConvertedWorkflow(project *database.ProjectIndex, result *ConvertResult) (absPath, relPath string, err error) {
	if project == nil || result == nil || result.Workflow == nil {
		return "", "", errors.New("invalid input")
	}
	if result.SourcePath == "" {
		return "", "", errors.New("source path is required for in-place conversion")
	}

	// Generate the native filename based on workflow name and ID
	slug := sanitizeWorkflowSlug(result.Workflow.Name)
	id, _ := uuid.Parse(result.Workflow.Id)
	fileName := fmt.Sprintf("%s--%s%s", slug, shortID(id), workflowFileExt)

	// Write in-place: same directory as source file, with native filename
	sourceDir := filepath.Dir(result.SourcePath)
	relPath = filepath.Join(sourceDir, fileName)
	absPath = filepath.Join(project.FolderPath, relPath)

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
		return "", "", fmt.Errorf("failed to create directory: %w", err)
	}

	// Marshal workflow to JSON
	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(result.Workflow)
	if err != nil {
		return "", "", fmt.Errorf("marshal workflow: %w", err)
	}

	// Pretty-print the JSON
	var buf bytes.Buffer
	if jsonErr := json.Indent(&buf, raw, "", "  "); jsonErr == nil {
		raw = buf.Bytes()
	}

	// Write atomically via temp file
	tmp := absPath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return "", "", fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, absPath); err != nil {
		return "", "", fmt.Errorf("rename temp file: %w", err)
	}

	return absPath, filepath.ToSlash(relPath), nil
}

// WriteWorkflowInPlace writes a workflow to the exact path specified, overwriting the original file.
// This is used for in-place conversion of external workflow formats.
func WriteWorkflowInPlace(absPath string, workflow *basapi.WorkflowSummary) error {
	if workflow == nil {
		return errors.New("workflow is nil")
	}

	// Marshal workflow to JSON using protojson
	raw, err := protojson.MarshalOptions{UseProtoNames: true, EmitUnpopulated: false}.Marshal(workflow)
	if err != nil {
		return fmt.Errorf("marshal workflow: %w", err)
	}

	// Pretty-print the JSON
	var buf bytes.Buffer
	if jsonErr := json.Indent(&buf, raw, "", "  "); jsonErr == nil {
		raw = buf.Bytes()
	}

	// Write atomically via temp file
	tmp := absPath + ".tmp"
	if err := os.WriteFile(tmp, raw, 0o644); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := os.Rename(tmp, absPath); err != nil {
		os.Remove(tmp) // Clean up on rename failure
		return fmt.Errorf("rename temp file: %w", err)
	}

	return nil
}
