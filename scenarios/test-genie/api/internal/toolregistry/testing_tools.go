// Package toolregistry provides the tool discovery service for test-genie.
//
// This file defines the test execution and scenario management tools.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// TestingToolProvider provides the test execution and scenario management tools.
type TestingToolProvider struct{}

// NewTestingToolProvider creates a new TestingToolProvider.
func NewTestingToolProvider() *TestingToolProvider {
	return &TestingToolProvider{}
}

// Name returns the provider identifier.
func (p *TestingToolProvider) Name() string {
	return "test-genie-testing"
}

// Categories returns the tool categories for test execution and scenario management.
func (p *TestingToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:          "test_execution",
			Name:        "Test Execution",
			Description: "Tools for running tests and viewing results",
			Icon:        "play",
		},
		{
			Id:          "scenario_management",
			Name:        "Scenario Management",
			Description: "Tools for managing and inspecting scenarios",
			Icon:        "folder",
		},
	}
}

// Tools returns the tool definitions for test execution and scenario management.
func (p *TestingToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.runTestSuiteTool(),
		p.runScenarioTestsTool(),
		p.runUiSmokeTool(),
		p.getExecutionTool(),
		p.listExecutionsTool(),
		p.listScenariosTool(),
		p.getScenarioTool(),
		p.listScenarioFilesTool(),
		p.listPhasesTool(),
	}
}

func (p *TestingToolProvider) runTestSuiteTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "run_test_suite",
		Description: "Execute a full test suite for a scenario. Runs through configured test phases and returns aggregated results. Use this when you want to run comprehensive tests on a scenario.",
		Category:    "test_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario": {
					Type:        "string",
					Description: "Name of the scenario to test (e.g., 'agent-manager', 'test-genie')",
				},
				"phases": {
					Type:        "array",
					Description: "Optional list of specific phases to run. If not provided, runs all enabled phases.",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"extra_args": {
					Type:        "array",
					Description: "Optional extra arguments to pass to the test runner",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
			},
			Required: []string{"scenario"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     900, // 15 minutes for full test suites
			RateLimitPerMinute: 5,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         true,
			Tags:               []string{"testing", "async", "scenario"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Run all tests for agent-manager",
					map[string]interface{}{
						"scenario": "agent-manager",
					},
				),
				NewToolExample(
					"Run only unit and integration tests",
					map[string]interface{}{
						"scenario": "test-genie",
						"phases":   []string{"unit", "integration"},
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "get_execution",
					OperationIdField:       "execution_id",
					StatusToolIdParam:      "execution_id",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 900,
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"passed", "completed"},
					FailureValues: []string{"failed", "error"},
					PendingValues: []string{"running", "pending"},
				},
			},
		},
	}
}

func (p *TestingToolProvider) runScenarioTestsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "run_scenario_tests",
		Description: "Run tests directly for a scenario using its native test runner. This bypasses the orchestrated phase system and runs tests directly.",
		Category:    "test_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario": {
					Type:        "string",
					Description: "Name of the scenario to test",
				},
				"preferred_runner": {
					Type:        "string",
					Description: "Preferred test runner to use (e.g., 'go', 'pnpm', 'bats'). Auto-detected if not specified.",
				},
				"extra_args": {
					Type:        "array",
					Description: "Extra arguments to pass to the test runner",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
			},
			Required: []string{"scenario"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     600,
			RateLimitPerMinute: 10,
			CostEstimate:       "medium",
			LongRunning:        true,
			Idempotent:         true,
			Tags:               []string{"testing", "scenario"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Run Go tests for a scenario",
					map[string]interface{}{
						"scenario":         "ecosystem-manager",
						"preferred_runner": "go",
					},
				),
			},
		},
	}
}

func (p *TestingToolProvider) runUiSmokeTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "run_ui_smoke",
		Description: "Run UI smoke tests for a scenario. Tests that the UI loads and basic interactions work. Requires browserless service to be running.",
		Category:    "test_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"scenario": {
					Type:        "string",
					Description: "Name of the scenario to test",
				},
				"ui_url": {
					Type:        "string",
					Description: "URL of the UI to test. Auto-detected from scenario config if not specified.",
				},
				"browserless_url": {
					Type:        "string",
					Description: "URL of the browserless service. Uses default if not specified.",
				},
				"timeout_ms": {
					Type:        "integer",
					Default:     IntValue(30000),
					Description: "Timeout in milliseconds for the smoke test",
				},
			},
			Required: []string{"scenario"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     120,
			RateLimitPerMinute: 10,
			CostEstimate:       "medium",
			LongRunning:        true,
			Idempotent:         true,
			Tags:               []string{"testing", "ui", "smoke"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Run UI smoke test with custom timeout",
					map[string]interface{}{
						"scenario":   "agent-inbox",
						"timeout_ms": 60000,
					},
				),
			},
		},
	}
}

func (p *TestingToolProvider) getExecutionTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_execution",
		Description: "Get details and status of a test execution by ID. Use this to check the progress and results of a running or completed test suite.",
		Category:    "test_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"execution_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The ID of the test execution to retrieve",
				},
			},
			Required: []string{"execution_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"status", "monitoring"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get status of a running test",
					map[string]interface{}{
						"execution_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				),
			},
		},
	}
}

func (p *TestingToolProvider) listExecutionsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_executions",
		Description: "List recent test executions. Optionally filter by scenario name.",
		Category:    "test_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"limit": {
					Type:        "integer",
					Default:     IntValue(20),
					Description: "Maximum number of executions to return",
				},
				"scenario": {
					Type:        "string",
					Description: "Optional filter by scenario name",
				},
			},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"list", "history"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List last 10 executions for a scenario",
					map[string]interface{}{
						"limit":    10,
						"scenario": "agent-manager",
					},
				),
			},
		},
	}
}

func (p *TestingToolProvider) listScenariosTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_scenarios",
		Description: "List all available scenarios that can be tested. Returns scenario names and basic metadata.",
		Category:    "scenario_management",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"list", "scenario"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all scenarios",
					map[string]interface{}{},
				),
			},
		},
	}
}

func (p *TestingToolProvider) getScenarioTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_scenario",
		Description: "Get detailed information about a specific scenario including its configuration, dependencies, and test setup.",
		Category:    "scenario_management",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"name": {
					Type:        "string",
					Description: "Name of the scenario to retrieve",
				},
			},
			Required: []string{"name"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"scenario", "info"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get scenario details",
					map[string]interface{}{
						"name": "agent-manager",
					},
				),
			},
		},
	}
}

func (p *TestingToolProvider) listScenarioFilesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_scenario_files",
		Description: "List files in a scenario directory. Useful for understanding the scenario structure and finding test files.",
		Category:    "scenario_management",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"name": {
					Type:        "string",
					Description: "Name of the scenario",
				},
				"recursive": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Whether to list files recursively",
				},
				"include_hidden": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Whether to include hidden files (starting with .)",
				},
			},
			Required: []string{"name"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"scenario", "files"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List files in scenario",
					map[string]interface{}{
						"name":      "test-genie",
						"recursive": true,
					},
				),
			},
		},
	}
}

func (p *TestingToolProvider) listPhasesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_phases",
		Description: "List available test phases and their descriptions. Phases define different types of tests (unit, integration, e2e, etc.)",
		Category:    "test_execution",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"phases", "config"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all test phases",
					map[string]interface{}{},
				),
			},
		},
	}
}
