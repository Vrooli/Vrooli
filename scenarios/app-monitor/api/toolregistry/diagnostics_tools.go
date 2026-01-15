// Package toolregistry provides tool definitions for app-monitor.
//
// This file defines diagnostic and health check tools.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// DiagnosticsToolProvider provides diagnostic and health check tools.
type DiagnosticsToolProvider struct{}

// NewDiagnosticsToolProvider creates a new DiagnosticsToolProvider.
func NewDiagnosticsToolProvider() *DiagnosticsToolProvider {
	return &DiagnosticsToolProvider{}
}

// Name returns the provider identifier.
func (p *DiagnosticsToolProvider) Name() string {
	return "app-monitor-diagnostics"
}

// Categories returns the tool categories for diagnostics tools.
func (p *DiagnosticsToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "diagnostics",
			Name:         "Diagnostics & Health",
			Description:  "Tools for checking app health, running diagnostics, and identifying issues",
			Icon:         "stethoscope",
			DisplayOrder: 3,
		},
	}
}

// Tools returns the diagnostics tool definitions.
func (p *DiagnosticsToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.getAppDiagnosticsTool(),
		p.getAppHealthTool(),
		p.checkIframeBridgeTool(),
		p.checkLocalhostUsageTool(),
		p.getFallbackDiagnosticsTool(),
		p.getAppCompletenessTool(),
	}
}

// getAppDiagnosticsTool returns the complete diagnostics tool.
func (p *DiagnosticsToolProvider) getAppDiagnosticsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_app_diagnostics",
		Description: "Get complete diagnostics bundle for an application. Includes health checks, scenario status, iframe bridge validation, localhost usage scan, and recommendations.",
		Category:    "diagnostics",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
			},
			Required: []string{"app_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     120,
			RateLimitPerMinute: 20,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"diagnostics", "health", "complete"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get full diagnostics for agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}

// getAppHealthTool returns the health check tool.
func (p *DiagnosticsToolProvider) getAppHealthTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_app_health",
		Description: "Check health status of an application by hitting its health endpoint and checking process status.",
		Category:    "diagnostics",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
			},
			Required: []string{"app_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"diagnostics", "health"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check health of agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}

// checkIframeBridgeTool returns the iframe bridge validation tool.
func (p *DiagnosticsToolProvider) checkIframeBridgeTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "check_iframe_bridge",
		Description: "Validate iframe bridge rules for an application. Checks for security issues and quality violations in the scenario's frontend code via scenario-auditor.",
		Category:    "diagnostics",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
			},
			Required: []string{"app_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"diagnostics", "security", "iframe"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check iframe bridge rules for agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}

// checkLocalhostUsageTool returns the localhost usage scanning tool.
func (p *DiagnosticsToolProvider) checkLocalhostUsageTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "check_localhost_usage",
		Description: "Scan an application for hardcoded localhost references. Identifies URLs that should use environment variables or service discovery instead.",
		Category:    "diagnostics",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
			},
			Required: []string{"app_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"diagnostics", "localhost", "code-quality"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check for hardcoded localhost in agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}

// getFallbackDiagnosticsTool returns the browserless diagnostics tool (async).
func (p *DiagnosticsToolProvider) getFallbackDiagnosticsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_fallback_diagnostics",
		Description: "Get browser-based diagnostics for an application using browserless. Captures console logs, network requests, and screenshots. Use when iframe bridge is unavailable or for deep debugging.",
		Category:    "diagnostics",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
				"url": {
					Type:        "string",
					Description: "Optional URL to diagnose (defaults to app's UI URL)",
				},
				"timeout_ms": {
					Type:        "integer",
					Default:     IntValue(30000),
					Description: "Maximum time to wait for page load in milliseconds",
				},
			},
			Required: []string{"app_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     120,
			RateLimitPerMinute: 10,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"diagnostics", "browser", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get browser diagnostics for agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "get_app_diagnostics",
					OperationIdField:       "app_id",
					StatusToolIdParam:      "app_id",
					PollIntervalSeconds:    3,
					MaxPollDurationSeconds: 120,
				},
			},
		},
	}
}

// getAppCompletenessTool returns the completeness score tool.
func (p *DiagnosticsToolProvider) getAppCompletenessTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_app_completeness",
		Description: "Get the completeness score for an application (0-100). Evaluates documentation, tests, configuration, and implementation quality.",
		Category:    "diagnostics",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
			},
			Required: []string{"app_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"diagnostics", "quality", "completeness"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get completeness score for agent-inbox",
					map[string]interface{}{
						"app_id": "agent-inbox",
					},
				),
			},
		},
	}
}
