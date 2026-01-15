package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/database"
)

// DiscoveredWorkflow represents a JSON file containing workflow structure.
type DiscoveredWorkflow struct {
	// AbsolutePath is the full path to the file on disk.
	AbsolutePath string
	// RelativePath is the path relative to the project root.
	RelativePath string
	// IsNativeFormat indicates the file content has a valid UUID in the "id" field.
	// This is content-based detection, not location or extension based.
	IsNativeFormat bool
	// WorkflowsRelPath is the path relative to the workflows directory (set if file is in workflows/).
	WorkflowsRelPath string
}

// DiscoverWorkflows walks the entire project and finds all JSON files with a "nodes" array.
// Uses content-based detection to determine native vs external format:
// - Native format: has valid UUID in "id" field
// - External format: has "nodes" array but no valid "id" UUID
func DiscoverWorkflows(project *database.ProjectIndex, maxDepth int) ([]DiscoveredWorkflow, error) {
	if project == nil {
		return nil, nil
	}

	projectRoot := strings.TrimSpace(project.FolderPath)
	if projectRoot == "" {
		return nil, nil
	}

	if maxDepth <= 0 {
		maxDepth = 4
	}

	var discovered []DiscoveredWorkflow
	workflowsDir := ProjectWorkflowsDir(project)

	err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip hidden directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		// Calculate depth from project root
		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return nil
		}

		// Enforce depth limit for directories
		depth := strings.Count(relPath, string(os.PathSeparator))
		if d.IsDir() && depth >= maxDepth {
			return filepath.SkipDir
		}

		// Only process JSON files
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}

		// Read file content once for all checks
		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		// Check if this file looks like a workflow (has "nodes" array)
		if !isWorkflowContent(content) {
			return nil
		}

		// Content-based format detection: native format has valid UUID in "id" field
		isNative := isNativeWorkflowFormat(content)

		// WorkflowsRelPath is set if file is in workflows/ directory (for path resolution)
		workflowsRelPath := ""
		if strings.HasPrefix(path, workflowsDir+string(os.PathSeparator)) || path == workflowsDir {
			workflowsRelPath, _ = filepath.Rel(workflowsDir, path)
		}

		discovered = append(discovered, DiscoveredWorkflow{
			AbsolutePath:     path,
			RelativePath:     relPath,
			IsNativeFormat:   isNative,
			WorkflowsRelPath: workflowsRelPath,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return discovered, nil
}

// isWorkflowContent checks if JSON content contains a "nodes" array (workflow structure).
// This works on raw bytes to avoid reading the file twice during discovery.
func isWorkflowContent(data []byte) bool {
	var content map[string]any
	if err := json.Unmarshal(data, &content); err != nil {
		return false
	}

	// Check for "nodes" array - the key indicator of a workflow file.
	// This can be at the top level or nested in flow_definition/definition_v2.
	if hasNodesArray(content) {
		return true
	}

	// Check nested flow_definition
	if flowDef, ok := content["flow_definition"].(map[string]any); ok {
		if hasNodesArray(flowDef) {
			return true
		}
	}

	// Check nested definition_v2
	if defV2, ok := content["definition_v2"].(map[string]any); ok {
		if hasNodesArray(defV2) {
			return true
		}
	}

	return false
}

// isWorkflowJSONFile checks if a JSON file contains a "nodes" array (workflow structure).
// This is a convenience wrapper that reads the file first.
func isWorkflowJSONFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	return isWorkflowContent(data)
}

// hasNodesArray checks if a map contains a "nodes" key with an array value.
func hasNodesArray(content map[string]any) bool {
	if nodes, ok := content["nodes"]; ok {
		if _, isArray := nodes.([]any); isArray {
			return true
		}
	}
	return false
}

// isNativeWorkflowFormat checks if workflow content is in native format by verifying
// the presence of a valid UUID in the "id" field. This is content-based detection,
// not location or extension based.
func isNativeWorkflowFormat(content []byte) bool {
	var summary struct {
		ID        string `json:"id"`
		ProjectID string `json:"project_id"`
	}
	if err := json.Unmarshal(content, &summary); err != nil {
		return false
	}
	if summary.ID == "" {
		return false
	}
	_, err := uuid.Parse(summary.ID)
	return err == nil
}

// DiscoverExternalWorkflows returns only non-native workflows that need conversion.
func DiscoverExternalWorkflows(project *database.ProjectIndex, maxDepth int) ([]DiscoveredWorkflow, error) {
	all, err := DiscoverWorkflows(project, maxDepth)
	if err != nil {
		return nil, err
	}

	var external []DiscoveredWorkflow
	for _, d := range all {
		if !d.IsNativeFormat {
			external = append(external, d)
		}
	}
	return external, nil
}

// DiscoverNativeWorkflows returns only workflows in native format (have valid UUID in "id" field).
// This uses content-based detection rather than location/extension.
func DiscoverNativeWorkflows(project *database.ProjectIndex, maxDepth int) ([]DiscoveredWorkflow, error) {
	all, err := DiscoverWorkflows(project, maxDepth)
	if err != nil {
		return nil, err
	}

	var native []DiscoveredWorkflow
	for _, d := range all {
		if d.IsNativeFormat {
			native = append(native, d)
		}
	}
	return native, nil
}

// IsV1WorkflowFormat checks if workflow content uses V1 format (node.type + node.data pattern).
// V1 format is legacy; V2 format has node.action with typed action definitions.
// Returns true if the workflow uses V1 format and needs conversion.
func IsV1WorkflowFormat(content []byte) bool {
	var doc map[string]any
	if err := json.Unmarshal(content, &doc); err != nil {
		return false
	}

	// Get nodes from document (can be at top level or in flow_definition/definition_v2)
	var nodes []any
	if n, ok := doc["nodes"].([]any); ok {
		nodes = n
	} else if flowDef, ok := doc["flow_definition"].(map[string]any); ok {
		if n, ok := flowDef["nodes"].([]any); ok {
			nodes = n
		}
	} else if defV2, ok := doc["definition_v2"].(map[string]any); ok {
		if n, ok := defV2["nodes"].([]any); ok {
			nodes = n
		}
	}

	if len(nodes) == 0 {
		return false // Empty workflow, consider it V2
	}

	// Check first node for V1 pattern (type + data, no action)
	firstNode, ok := nodes[0].(map[string]any)
	if !ok {
		return false
	}

	_, hasType := firstNode["type"]
	_, hasData := firstNode["data"]
	_, hasAction := firstNode["action"]

	// V1 if it has type+data but no action
	return (hasType || hasData) && !hasAction
}

// CountV1Workflows scans a project directory and counts how many workflows use V1 format.
// This is used during import to warn users about files that will be converted.
func CountV1Workflows(project *database.ProjectIndex, maxDepth int) (int, error) {
	if project == nil {
		return 0, nil
	}

	projectRoot := strings.TrimSpace(project.FolderPath)
	if projectRoot == "" {
		return 0, nil
	}

	if maxDepth <= 0 {
		maxDepth = 4
	}

	count := 0

	err := filepath.WalkDir(projectRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		// Skip hidden directories
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return filepath.SkipDir
		}

		// Calculate depth
		relPath, err := filepath.Rel(projectRoot, path)
		if err != nil {
			return nil
		}
		depth := strings.Count(relPath, string(os.PathSeparator))
		if d.IsDir() && depth >= maxDepth {
			return filepath.SkipDir
		}

		// Only process JSON files
		if d.IsDir() || !strings.HasSuffix(strings.ToLower(d.Name()), ".json") {
			return nil
		}

		content, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		if !isWorkflowContent(content) {
			return nil
		}

		if IsV1WorkflowFormat(content) {
			count++
		}

		return nil
	})

	return count, err
}
