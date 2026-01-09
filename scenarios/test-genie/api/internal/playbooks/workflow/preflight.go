package workflow

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// PreflightIssue represents a validation issue found during preflight checks.
type PreflightIssue struct {
	Severity string `json:"severity"` // "error" or "warning"
	Code     string `json:"code"`
	Message  string `json:"message"`
	NodeID   string `json:"node_id,omitempty"`
	NodeType string `json:"node_type,omitempty"`
	Field    string `json:"field,omitempty"`
	Pointer  string `json:"pointer,omitempty"`
	Hint     string `json:"hint,omitempty"`
}

// PreflightResult contains the results of preflight validation.
type PreflightResult struct {
	Valid             bool             `json:"valid"`
	Errors            []PreflightIssue `json:"errors,omitempty"`
	Warnings          []PreflightIssue `json:"warnings,omitempty"`
	TokenCounts       TokenCounts      `json:"token_counts"`
	RequiredScenarios []string         `json:"required_scenarios,omitempty"`
}

// TokenCounts tracks how many tokens of each type were found.
// Note: These counts are informational only - BAS handles all token resolution at runtime.
type TokenCounts struct {
	Selectors    int `json:"selectors"`     // @selector/ tokens (resolved by BAS compiler)
	Subflows     int `json:"subflows"`      // Subflow nodes with workflowPath
	ParamsTokens int `json:"params_tokens"` // ${@params/...} tokens
}

// PreflightValidator validates workflows before execution.
// Since BAS handles all variable interpolation and subflow resolution natively,
// preflight validation is minimal - primarily checking basic structure and scenario dependencies.
type PreflightValidator struct {
	scenarioDir string
}

// NewPreflightValidator creates a new preflight validator.
func NewPreflightValidator(scenarioDir string) *PreflightValidator {
	return &PreflightValidator{
		scenarioDir: scenarioDir,
	}
}

// Validate performs preflight validation on a workflow file.
// Checks basic structure and extracts scenario dependencies.
// Variable interpolation and subflow resolution are delegated to BAS at runtime.
func (v *PreflightValidator) Validate(workflowPath string) (*PreflightResult, error) {
	// Ensure absolute path
	if !filepath.IsAbs(workflowPath) {
		workflowPath = filepath.Join(v.scenarioDir, filepath.FromSlash(workflowPath))
	}

	// Load workflow
	data, err := os.ReadFile(workflowPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow: %w", err)
	}

	var workflow map[string]any
	if err := json.Unmarshal(data, &workflow); err != nil {
		return nil, fmt.Errorf("failed to parse workflow JSON: %w", err)
	}

	// Initialize result
	result := &PreflightResult{
		Valid: true,
	}

	// Validate basic structure
	v.validateStructure(workflow, result)

	// Scan workflow for tokens and dependencies
	v.scanDefinition(workflow, result, "")

	// Determine validity
	result.Valid = len(result.Errors) == 0

	return result, nil
}

// validateStructure checks that the workflow has the required structure.
func (v *PreflightValidator) validateStructure(workflow map[string]any, result *PreflightResult) {
	// Check for legacy steps[] format (deprecated)
	if steps, ok := workflow["steps"]; ok {
		if _, ok := steps.([]any); ok {
			result.Errors = append(result.Errors, PreflightIssue{
				Severity: "error",
				Code:     "PF_LEGACY_STEPS_FORMAT",
				Message:  "Workflow uses deprecated 'steps' format instead of 'nodes'/'edges'",
				Hint:     "Convert to V2 proto-JSON format: replace 'steps' with 'nodes' array and add 'edges' array",
			})
			return // Don't continue checking V2 structure
		}
	}

	// Check for nodes array
	nodes, ok := workflow["nodes"]
	if !ok {
		result.Errors = append(result.Errors, PreflightIssue{
			Severity: "error",
			Code:     "PF_MISSING_NODES",
			Message:  "Workflow missing required 'nodes' field",
			Hint:     "Add a 'nodes' array to the workflow",
		})
		return
	}

	nodesArr, ok := nodes.([]any)
	if !ok {
		result.Errors = append(result.Errors, PreflightIssue{
			Severity: "error",
			Code:     "PF_INVALID_NODES",
			Message:  "Workflow 'nodes' field must be an array",
		})
		return
	}

	// Check for legacy V1 UI format (nodes have type+data instead of action)
	v.detectLegacyV1Format(nodesArr, result)

	// Check for edges array (optional but if present must be array)
	if edges, ok := workflow["edges"]; ok {
		if _, ok := edges.([]any); !ok {
			result.Errors = append(result.Errors, PreflightIssue{
				Severity: "error",
				Code:     "PF_INVALID_EDGES",
				Message:  "Workflow 'edges' field must be an array",
			})
		}
	}
}

// detectLegacyV1Format checks if nodes use the deprecated V1 UI format (type+data).
// V2 proto format uses action field with type and action-specific params.
func (v *PreflightValidator) detectLegacyV1Format(nodes []any, result *PreflightResult) {
	for i, node := range nodes {
		nodeMap, ok := node.(map[string]any)
		if !ok {
			continue
		}

		nodeID, _ := nodeMap["id"].(string)
		if nodeID == "" {
			nodeID = fmt.Sprintf("node[%d]", i)
		}

		// V1 UI format has "type" and "data" fields at node level
		// V2 proto format has "action" field containing type and params
		_, hasType := nodeMap["type"]
		_, hasData := nodeMap["data"]
		_, hasAction := nodeMap["action"]

		if hasType && hasData && !hasAction {
			nodeType, _ := nodeMap["type"].(string)
			result.Errors = append(result.Errors, PreflightIssue{
				Severity: "error",
				Code:     "PF_LEGACY_V1_FORMAT",
				Message:  fmt.Sprintf("Node '%s' uses deprecated V1 UI format (type+data)", nodeID),
				NodeID:   nodeID,
				NodeType: nodeType,
				Pointer:  fmt.Sprintf("/nodes/%d", i),
				Hint:     "Convert to V2 proto-JSON format: use 'action' field with 'type' (ACTION_TYPE_*) and action-specific params",
			})
		}
	}
}

// Patterns for token detection
var (
	preflightSelectorPattern = regexp.MustCompile(`@selector/([A-Za-z0-9_.-]+)`)
	preflightParamsPattern   = regexp.MustCompile(`\$\{@params/([A-Za-z0-9_./]+)\}`)
)

// scanDefinition recursively scans a workflow definition for tokens and dependencies.
// Supports both V2 proto format (action field) and V1 format (type/data) for backwards
// compatibility during scanning, though V1 format will generate errors in validation.
func (v *PreflightValidator) scanDefinition(definition map[string]any, result *PreflightResult, pointer string) {
	nodes, ok := definition["nodes"].([]any)
	if !ok {
		return
	}

	for i, node := range nodes {
		nodeMap, ok := node.(map[string]any)
		if !ok {
			continue
		}

		nodeID, _ := nodeMap["id"].(string)
		nodePointer := fmt.Sprintf("%s/nodes/%d", pointer, i)

		// Try V2 proto format first (action field)
		if action, ok := nodeMap["action"].(map[string]any); ok {
			v.scanV2Action(action, nodeID, nodePointer, result)
			continue
		}

		// Fall back to V1 format for scanning (will still error in validation)
		nodeType, _ := nodeMap["type"].(string)
		data, ok := nodeMap["data"].(map[string]any)
		if !ok {
			continue
		}

		// Check for subflow references (workflowPath is the new format)
		if workflowPath, ok := data["workflowPath"].(string); ok && workflowPath != "" {
			result.TokenCounts.Subflows++
			// Validate path format (should be relative, not absolute)
			if filepath.IsAbs(workflowPath) {
				result.Warnings = append(result.Warnings, PreflightIssue{
					Severity: "warning",
					Code:     "PF_ABSOLUTE_WORKFLOW_PATH",
					Message:  fmt.Sprintf("Subflow uses absolute path: %s", workflowPath),
					NodeID:   nodeID,
					NodeType: nodeType,
					Field:    "workflowPath",
					Pointer:  nodePointer + "/data/workflowPath",
					Hint:     "Use relative paths for portability (e.g., 'actions/helper.json')",
				})
			}
		}

		// Check navigate nodes for scenario references
		if nodeType == "navigate" {
			v.checkScenarioReference(data, nodeID, nodePointer, result)
		}

		// Scan all string fields for @selector/ and ${@params/...} tokens
		v.scanFieldsForTokens(data, nodeID, nodeType, nodePointer, result)

		// Recursively check nested workflow definitions
		if nestedDef, ok := data["workflowDefinition"].(map[string]any); ok {
			v.scanDefinition(nestedDef, result, nodePointer+"/data/workflowDefinition")
		}
	}
}

// scanV2Action scans a V2 proto format action for tokens and dependencies.
func (v *PreflightValidator) scanV2Action(action map[string]any, nodeID, nodePointer string, result *PreflightResult) {
	actionType, _ := action["type"].(string)
	actionPointer := nodePointer + "/action"

	// Track which keys we've already scanned to avoid double counting
	scannedKeys := make(map[string]bool)

	// Check for subflow action
	if subflow, ok := action["subflow"].(map[string]any); ok {
		scannedKeys["subflow"] = true
		if workflowPath, ok := subflow["workflow_path"].(string); ok && workflowPath != "" {
			result.TokenCounts.Subflows++
			if filepath.IsAbs(workflowPath) {
				result.Warnings = append(result.Warnings, PreflightIssue{
					Severity: "warning",
					Code:     "PF_ABSOLUTE_WORKFLOW_PATH",
					Message:  fmt.Sprintf("Subflow uses absolute path: %s", workflowPath),
					NodeID:   nodeID,
					NodeType: actionType,
					Field:    "workflow_path",
					Pointer:  actionPointer + "/subflow/workflow_path",
					Hint:     "Use relative paths for portability (e.g., 'actions/helper.json')",
				})
			}
		}
		// Scan subflow params for tokens
		if params, ok := subflow["params"].(map[string]any); ok {
			v.scanFieldsForTokens(params, nodeID, actionType, actionPointer+"/subflow/params", result)
		}
		// Check nested workflow definition
		if nestedDef, ok := subflow["workflow_definition"].(map[string]any); ok {
			v.scanDefinition(nestedDef, result, actionPointer+"/subflow/workflow_definition")
		}
	}

	// Check for navigate action with scenario reference
	if navigate, ok := action["navigate"].(map[string]any); ok {
		scannedKeys["navigate"] = true
		destType, _ := navigate["destination_type"].(string)
		if strings.Contains(strings.ToUpper(destType), "SCENARIO") {
			scenario, _ := navigate["scenario"].(string)
			scenario = strings.TrimSpace(scenario)

			if scenario == "" {
				result.Errors = append(result.Errors, PreflightIssue{
					Severity: "error",
					Code:     "PF_SCENARIO_NAME_MISSING",
					Message:  "Navigate node with destination_type=SCENARIO missing scenario name",
					NodeID:   nodeID,
					NodeType: actionType,
					Field:    "scenario",
					Pointer:  actionPointer + "/navigate/scenario",
					Hint:     "Specify the target scenario name",
				})
			} else {
				result.RequiredScenarios = append(result.RequiredScenarios, scenario)
			}
		}
		// Scan navigate fields for tokens
		v.scanFieldsForTokens(navigate, nodeID, actionType, actionPointer+"/navigate", result)
	}

	// Scan other action types for tokens (click, input, wait, assert, etc.)
	for key, value := range action {
		if key == "type" || key == "metadata" || scannedKeys[key] {
			continue
		}
		if actionParams, ok := value.(map[string]any); ok {
			v.scanFieldsForTokens(actionParams, nodeID, actionType, actionPointer+"/"+key, result)
		}
	}
}

// scanFieldsForTokens scans all string fields in data for tokens.
func (v *PreflightValidator) scanFieldsForTokens(data map[string]any, nodeID, nodeType, pointer string, result *PreflightResult) {
	for _, value := range data {
		switch val := value.(type) {
		case string:
			// Count @selector/ tokens
			selectorMatches := preflightSelectorPattern.FindAllStringSubmatch(val, -1)
			result.TokenCounts.Selectors += len(selectorMatches)

			// Count ${@params/...} tokens
			paramsMatches := preflightParamsPattern.FindAllStringSubmatch(val, -1)
			result.TokenCounts.ParamsTokens += len(paramsMatches)

		case map[string]any:
			// Recursively scan nested objects
			v.scanFieldsForTokens(val, nodeID, nodeType, pointer, result)

		case []any:
			// Recursively scan arrays
			for _, item := range val {
				if itemMap, ok := item.(map[string]any); ok {
					v.scanFieldsForTokens(itemMap, nodeID, nodeType, pointer, result)
				}
			}
		}
	}
}

// checkScenarioReference extracts scenario names from navigate nodes.
func (v *PreflightValidator) checkScenarioReference(data map[string]any, nodeID, pointer string, result *PreflightResult) {
	destType, _ := data["destinationType"].(string)
	if strings.ToLower(strings.TrimSpace(destType)) != "scenario" {
		return
	}

	scenario, _ := data["scenario"].(string)
	scenario = strings.TrimSpace(scenario)

	if scenario == "" {
		result.Errors = append(result.Errors, PreflightIssue{
			Severity: "error",
			Code:     "PF_SCENARIO_NAME_MISSING",
			Message:  "Navigate node with destinationType=scenario missing scenario name",
			NodeID:   nodeID,
			NodeType: "navigate",
			Field:    "scenario",
			Pointer:  pointer + "/data/scenario",
			Hint:     "Specify the target scenario name",
		})
		return
	}

	// Track required scenario (deduplicated in aggregation)
	result.RequiredScenarios = append(result.RequiredScenarios, scenario)
}

// ValidateAll validates multiple workflow files and aggregates results.
func (v *PreflightValidator) ValidateAll(workflowPaths []string) (*PreflightResult, error) {
	aggregated := &PreflightResult{
		Valid: true,
	}
	scenarioSet := make(map[string]bool)

	for _, path := range workflowPaths {
		result, err := v.Validate(path)
		if err != nil {
			aggregated.Errors = append(aggregated.Errors, PreflightIssue{
				Severity: "error",
				Code:     "PF_WORKFLOW_LOAD_FAILED",
				Message:  fmt.Sprintf("Failed to validate %s: %v", path, err),
				Pointer:  path,
			})
			aggregated.Valid = false
			continue
		}

		aggregated.Errors = append(aggregated.Errors, result.Errors...)
		aggregated.Warnings = append(aggregated.Warnings, result.Warnings...)
		aggregated.TokenCounts.Selectors += result.TokenCounts.Selectors
		aggregated.TokenCounts.Subflows += result.TokenCounts.Subflows
		aggregated.TokenCounts.ParamsTokens += result.TokenCounts.ParamsTokens

		// Deduplicate required scenarios
		for _, scenario := range result.RequiredScenarios {
			scenarioSet[scenario] = true
		}

		if !result.Valid {
			aggregated.Valid = false
		}
	}

	// Convert scenario set to sorted list
	for scenario := range scenarioSet {
		aggregated.RequiredScenarios = append(aggregated.RequiredScenarios, scenario)
	}
	sort.Strings(aggregated.RequiredScenarios)

	return aggregated, nil
}
