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
type TokenCounts struct {
	Fixtures  int `json:"fixtures"`
	Selectors int `json:"selectors"`
	Seeds     int `json:"seeds"`
}

// PreflightValidator validates workflows before resolution.
type PreflightValidator struct {
	scenarioDir    string
	fixturesDir    string
	loadedFixtures map[string]bool
	// NOTE: Selector validation removed - BAS handles @selector/ resolution natively.
	// See: scenarios/browser-automation-studio/api/automation/compiler/compiler.go:939-1073
}

// NewPreflightValidator creates a new preflight validator.
func NewPreflightValidator(scenarioDir string) *PreflightValidator {
	return &PreflightValidator{
		scenarioDir: scenarioDir,
		fixturesDir: filepath.Join(scenarioDir, "test", "playbooks", "__subflows"),
	}
}

// Validate performs preflight validation on a workflow file.
// It checks that all referenced fixtures and selectors exist without actually resolving them.
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

	// Load available fixtures
	if err := v.loadAvailableFixtures(); err != nil {
		result.Errors = append(result.Errors, PreflightIssue{
			Severity: "error",
			Code:     "PF_FIXTURES_LOAD_FAILED",
			Message:  fmt.Sprintf("Failed to load fixtures directory: %v", err),
			Hint:     "Check that test/playbooks/__subflows directory is accessible",
		})
	}

	// NOTE: Selector manifest loading removed - BAS handles @selector/ validation natively.

	// Scan workflow for tokens
	v.scanDefinition(workflow, result, "")

	// Determine validity
	result.Valid = len(result.Errors) == 0

	return result, nil
}

// loadAvailableFixtures scans the fixtures directory for available fixture IDs.
func (v *PreflightValidator) loadAvailableFixtures() error {
	v.loadedFixtures = make(map[string]bool)

	if _, err := os.Stat(v.fixturesDir); os.IsNotExist(err) {
		return nil // No fixtures directory is OK
	}

	return filepath.WalkDir(v.fixturesDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || filepath.Ext(path) != ".json" {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return nil // Skip unreadable files
		}

		var doc map[string]any
		if err := json.Unmarshal(data, &doc); err != nil {
			return nil // Skip invalid JSON
		}

		// Extract fixture ID
		metadata, _ := doc["metadata"].(map[string]any)
		if metadata == nil {
			return nil
		}

		fixtureID, _ := metadata["fixture_id"].(string)
		if fixtureID == "" {
			fixtureID, _ = metadata["fixtureId"].(string)
		}
		if fixtureID != "" {
			v.loadedFixtures[fixtureID] = true
		}

		return nil
	})
}

// Patterns for token detection
var (
	preflightFixturePattern  = regexp.MustCompile(`@fixture/([A-Za-z0-9_.-]+)`)
	preflightSelectorPattern = regexp.MustCompile(`@selector/([A-Za-z0-9_.-]+)`)
	preflightSeedPattern     = regexp.MustCompile(`@seed/([A-Za-z0-9_.-]+)`)
)

// scanDefinition recursively scans a workflow definition for tokens.
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

		// Check workflowId for fixture references
		if workflowID, ok := data["workflowId"].(string); ok {
			v.checkFixtureReference(workflowID, nodeID, nodeType, nodePointer+"/data/workflowId", result)
		}

		// Check navigate nodes for scenario references
		if nodeType == "navigate" {
			v.checkScenarioReference(data, nodeID, nodePointer, result)
		}

		// Scan all string fields for selector and seed references
		for field, value := range data {
			if strVal, ok := value.(string); ok {
				fieldPointer := fmt.Sprintf("%s/data/%s", nodePointer, field)
				v.checkSelectorReferences(strVal, nodeID, nodeType, field, fieldPointer, result)
				v.checkSeedReferences(strVal, nodeID, nodeType, field, fieldPointer, result)
			}
		}

		// Recursively check nested workflow definitions
		if nestedDef, ok := data["workflowDefinition"].(map[string]any); ok {
			v.scanDefinition(nestedDef, result, nodePointer+"/data/workflowDefinition")
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

// checkFixtureReference validates a fixture reference.
func (v *PreflightValidator) checkFixtureReference(ref string, nodeID, nodeType, pointer string, result *PreflightResult) {
	if !strings.HasPrefix(ref, "@fixture/") {
		return
	}

	match := preflightFixturePattern.FindStringSubmatch(ref)
	if match == nil {
		result.Errors = append(result.Errors, PreflightIssue{
			Severity: "error",
			Code:     "PF_FIXTURE_INVALID_SYNTAX",
			Message:  fmt.Sprintf("Invalid fixture reference syntax: %s", ref),
			NodeID:   nodeID,
			NodeType: nodeType,
			Pointer:  pointer,
			Hint:     "Use format @fixture/fixture-id or @fixture/fixture-id(param=value)",
		})
		return
	}

	fixtureID := match[1]
	result.TokenCounts.Fixtures++

	if v.loadedFixtures != nil && !v.loadedFixtures[fixtureID] {
		// Find similar fixtures for suggestions
		suggestions := v.findSimilarFixtures(fixtureID)
		hint := fmt.Sprintf("Create fixture in test/playbooks/__subflows/%s.json", fixtureID)
		if len(suggestions) > 0 {
			hint = fmt.Sprintf("Did you mean: %s? Or create: test/playbooks/__subflows/%s.json",
				strings.Join(suggestions, ", "), fixtureID)
		}

		result.Errors = append(result.Errors, PreflightIssue{
			Severity: "error",
			Code:     "PF_FIXTURE_NOT_FOUND",
			Message:  fmt.Sprintf("Fixture @fixture/%s not found", fixtureID),
			NodeID:   nodeID,
			NodeType: nodeType,
			Field:    "workflowId",
			Pointer:  pointer,
			Hint:     hint,
		})
	}
}

// checkSelectorReferences counts selector references for informational purposes.
// NOTE: Selector validation is handled by BAS compiler, not test-genie preflight.
// See: scenarios/browser-automation-studio/api/automation/compiler/compiler.go:939-1073
func (v *PreflightValidator) checkSelectorReferences(value, nodeID, nodeType, field, pointer string, result *PreflightResult) {
	matches := preflightSelectorPattern.FindAllStringSubmatch(value, -1)
	result.TokenCounts.Selectors += len(matches)
	// Validation delegated to BAS - selectors are resolved at compile time by BAS compiler
}

// checkSeedReferences notes seed references (warnings only - can't validate statically).
func (v *PreflightValidator) checkSeedReferences(value, nodeID, nodeType, field, pointer string, result *PreflightResult) {
	matches := preflightSeedPattern.FindAllStringSubmatch(value, -1)
	for _, match := range matches {
		seedKey := match[1]
		result.TokenCounts.Seeds++

		// Seeds can only be validated at runtime, so we just count them
		// and emit a warning if many are found (indicates heavy seed dependency)
		if result.TokenCounts.Seeds == 1 {
			result.Warnings = append(result.Warnings, PreflightIssue{
				Severity: "warning",
				Code:     "PF_SEED_RUNTIME_DEPENDENCY",
				Message:  fmt.Sprintf("Workflow uses @seed/%s which requires runtime seed state", seedKey),
				NodeID:   nodeID,
				NodeType: nodeType,
				Field:    field,
				Pointer:  pointer,
				Hint:     "Seeds are resolved at runtime from test/playbooks/__seeds/state.json",
			})
		}
	}
}

// findSimilarFixtures finds fixtures with similar names for suggestions.
func (v *PreflightValidator) findSimilarFixtures(target string) []string {
	var suggestions []string
	targetParts := strings.Split(strings.ToLower(target), "-")

	for fixtureID := range v.loadedFixtures {
		fixtureParts := strings.Split(strings.ToLower(fixtureID), "-")
		for _, tp := range targetParts {
			for _, fp := range fixtureParts {
				if tp == fp || strings.Contains(fp, tp) || strings.Contains(tp, fp) {
					suggestions = append(suggestions, "@fixture/"+fixtureID)
					break
				}
			}
		}
	}

	// Deduplicate and limit
	seen := make(map[string]bool)
	var unique []string
	for _, s := range suggestions {
		if !seen[s] && len(unique) < 3 {
			seen[s] = true
			unique = append(unique, s)
		}
	}
	sort.Strings(unique)
	return unique
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
		aggregated.TokenCounts.Fixtures += result.TokenCounts.Fixtures
		aggregated.TokenCounts.Selectors += result.TokenCounts.Selectors
		aggregated.TokenCounts.Seeds += result.TokenCounts.Seeds

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
