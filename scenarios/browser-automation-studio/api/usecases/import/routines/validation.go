package routines

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/browser-automation-studio/usecases/import/shared"
)

// Validator validates workflow files.
type Validator struct{}

// NewValidator creates a new Validator.
func NewValidator() *Validator {
	return &Validator{}
}

// ValidateWorkflowJSON validates workflow JSON content and extracts metadata.
func (v *Validator) ValidateWorkflowJSON(content []byte) (*shared.WorkflowValidationResult, error) {
	result := &shared.WorkflowValidationResult{
		IsValid:  true,
		Errors:   []shared.ValidationIssue{},
		Warnings: []shared.ValidationIssue{},
	}

	// Parse JSON
	var workflow map[string]interface{}
	if err := json.Unmarshal(content, &workflow); err != nil {
		result.IsValid = false
		result.Errors = append(result.Errors, shared.ValidationIssue{
			Code:    "INVALID_JSON",
			Message: fmt.Sprintf("Invalid JSON: %v", err),
		})
		return result, nil
	}

	// Check for required fields
	if _, ok := workflow["nodes"]; !ok {
		result.Warnings = append(result.Warnings, shared.ValidationIssue{
			Code:    "MISSING_NODES",
			Message: "Workflow has no nodes array",
			Path:    "nodes",
		})
	}

	// Count nodes and edges
	if nodes, ok := workflow["nodes"].([]interface{}); ok {
		result.NodeCount = len(nodes)

		// Check for start/end nodes
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				nodeType, _ := nodeMap["type"].(string)
				if strings.ToLower(nodeType) == "start" || strings.ToLower(nodeType) == "trigger" {
					result.HasStartNode = true
				}
				if strings.ToLower(nodeType) == "end" || strings.ToLower(nodeType) == "finish" {
					result.HasEndNode = true
				}
			}
		}
	}

	if edges, ok := workflow["edges"].([]interface{}); ok {
		result.EdgeCount = len(edges)
	}

	// Add warnings for missing start/end
	if !result.HasStartNode {
		result.Warnings = append(result.Warnings, shared.ValidationIssue{
			Code:    "NO_START_NODE",
			Message: "Workflow has no start/trigger node",
		})
	}
	if !result.HasEndNode {
		result.Warnings = append(result.Warnings, shared.ValidationIssue{
			Code:    "NO_END_NODE",
			Message: "Workflow has no end/finish node",
		})
	}

	return result, nil
}

// ExtractWorkflowPreview extracts preview metadata from workflow JSON.
func (v *Validator) ExtractWorkflowPreview(content []byte) (*WorkflowPreview, error) {
	var workflow map[string]interface{}
	if err := json.Unmarshal(content, &workflow); err != nil {
		return nil, fmt.Errorf("invalid JSON: %w", err)
	}

	preview := &WorkflowPreview{
		Version: 1,
	}

	// Extract ID
	if id, ok := workflow["id"].(string); ok {
		preview.ID = id
	}

	// Extract name
	if name, ok := workflow["name"].(string); ok {
		preview.Name = name
	}

	// Extract description
	if desc, ok := workflow["description"].(string); ok {
		preview.Description = desc
	}

	// Extract version
	if version, ok := workflow["version"].(float64); ok {
		preview.Version = int(version)
	}

	// Extract tags
	if tags, ok := workflow["tags"].([]interface{}); ok {
		for _, tag := range tags {
			if tagStr, ok := tag.(string); ok {
				preview.Tags = append(preview.Tags, tagStr)
			}
		}
	}

	// Count nodes and check for start/end
	if nodes, ok := workflow["nodes"].([]interface{}); ok {
		preview.NodeCount = len(nodes)
		for _, node := range nodes {
			if nodeMap, ok := node.(map[string]interface{}); ok {
				nodeType, _ := nodeMap["type"].(string)
				if strings.ToLower(nodeType) == "start" || strings.ToLower(nodeType) == "trigger" {
					preview.HasStartNode = true
				}
				if strings.ToLower(nodeType) == "end" || strings.ToLower(nodeType) == "finish" {
					preview.HasEndNode = true
				}
			}
		}
	}

	// Count edges
	if edges, ok := workflow["edges"].([]interface{}); ok {
		preview.EdgeCount = len(edges)
	}

	return preview, nil
}

// ValidateFilePath validates a file path for import.
func (v *Validator) ValidateFilePath(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("file path is required")
	}

	// Check for path traversal
	if strings.Contains(path, "..") {
		return fmt.Errorf("path traversal not allowed")
	}

	// Must end with .workflow.json or .json
	lower := strings.ToLower(path)
	if !strings.HasSuffix(lower, ".workflow.json") && !strings.HasSuffix(lower, ".json") {
		return fmt.Errorf("file must be a JSON file")
	}

	return nil
}

// Ensure Validator implements the interface
var _ shared.WorkflowValidator = (*Validator)(nil)
