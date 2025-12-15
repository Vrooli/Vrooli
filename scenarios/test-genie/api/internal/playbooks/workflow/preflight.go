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
	// Check for nodes array
	nodes, ok := workflow["nodes"]
	if !ok {
		result.Errors = append(result.Errors, PreflightIssue{
			Severity: "error",
			Code:     "PF_MISSING_NODES",
			Message:  "Workflow missing required 'nodes' field",
			Hint:     "Add a 'nodes' array to the workflow",
		})
	} else if _, ok := nodes.([]any); !ok {
		result.Errors = append(result.Errors, PreflightIssue{
			Severity: "error",
			Code:     "PF_INVALID_NODES",
			Message:  "Workflow 'nodes' field must be an array",
		})
	}

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

// Patterns for token detection
var (
	preflightSelectorPattern = regexp.MustCompile(`@selector/([A-Za-z0-9_.-]+)`)
	preflightParamsPattern   = regexp.MustCompile(`\$\{@params/([A-Za-z0-9_./]+)\}`)
)

// scanDefinition recursively scans a workflow definition for tokens and dependencies.
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
		nodeType, _ := nodeMap["type"].(string)
		nodePointer := fmt.Sprintf("%s/nodes/%d", pointer, i)

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
