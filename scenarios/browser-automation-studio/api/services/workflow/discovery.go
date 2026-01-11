package workflow

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/browser-automation-studio/database"
)

// DiscoveredWorkflow represents a JSON file containing workflow structure.
type DiscoveredWorkflow struct {
	// AbsolutePath is the full path to the file on disk.
	AbsolutePath string
	// RelativePath is the path relative to the project root.
	RelativePath string
	// IsNativeFormat indicates the file is in workflows/ with .workflow.json extension.
	IsNativeFormat bool
	// WorkflowsRelPath is the path relative to the workflows directory (only set for native format).
	WorkflowsRelPath string
}

// DiscoverWorkflows walks the entire project and finds all JSON files with a "nodes" array.
// This matches the validation detection logic (detectWorkflows in projects/service.go).
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

		// Check if this file looks like a workflow (has "nodes" array)
		if !isWorkflowJSONFile(path) {
			return nil
		}

		// Determine if this is native format (in workflows/ directory with .workflow.json extension)
		isNative := false
		workflowsRelPath := ""
		if strings.HasPrefix(path, workflowsDir+string(os.PathSeparator)) ||
			path == workflowsDir {
			if strings.HasSuffix(strings.ToLower(d.Name()), workflowFileExt) {
				isNative = true
				workflowsRelPath, _ = filepath.Rel(workflowsDir, path)
			}
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

// isWorkflowJSONFile checks if a JSON file contains a "nodes" array (workflow structure).
// This matches the validation detection logic (isWorkflowJSONFile in projects/service.go).
func isWorkflowJSONFile(path string) bool {
	data, err := os.ReadFile(path)
	if err != nil {
		return false
	}

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

// hasNodesArray checks if a map contains a "nodes" key with an array value.
func hasNodesArray(content map[string]any) bool {
	if nodes, ok := content["nodes"]; ok {
		if _, isArray := nodes.([]any); isArray {
			return true
		}
	}
	return false
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

// DiscoverNativeWorkflows returns only native format workflows in workflows/ directory.
func DiscoverNativeWorkflows(project *database.ProjectIndex) ([]DiscoveredWorkflow, error) {
	if project == nil {
		return nil, nil
	}

	workflowsDir := ProjectWorkflowsDir(project)
	info, err := os.Stat(workflowsDir)
	if err != nil || !info.IsDir() {
		return nil, nil
	}

	var discovered []DiscoveredWorkflow

	err = filepath.WalkDir(workflowsDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(d.Name()), workflowFileExt) {
			return nil
		}

		relPath, _ := filepath.Rel(workflowsDir, path)
		projectRelPath, _ := filepath.Rel(project.FolderPath, path)

		discovered = append(discovered, DiscoveredWorkflow{
			AbsolutePath:     path,
			RelativePath:     projectRelPath,
			IsNativeFormat:   true,
			WorkflowsRelPath: relPath,
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	return discovered, nil
}
