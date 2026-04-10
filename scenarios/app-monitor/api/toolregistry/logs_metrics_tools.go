// Package toolregistry provides tool definitions for app-monitor.
//
// This file defines log retrieval and metrics tools.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// LogsMetricsToolProvider provides log and metrics tools.
type LogsMetricsToolProvider struct{}

// NewLogsMetricsToolProvider creates a new LogsMetricsToolProvider.
func NewLogsMetricsToolProvider() *LogsMetricsToolProvider {
	return &LogsMetricsToolProvider{}
}

// Name returns the provider identifier.
func (p *LogsMetricsToolProvider) Name() string {
	return "app-monitor-logs-metrics"
}

// Categories returns the tool categories for logs and metrics tools.
func (p *LogsMetricsToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "logs_metrics",
			Name:         "Logs & Metrics",
			Description:  "Tools for retrieving application logs and performance metrics",
			Icon:         "chart-line",
			DisplayOrder: 4,
		},
	}
}

// Tools returns the logs and metrics tool definitions.
func (p *LogsMetricsToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.getAppLogsTool(),
		p.getAppMetricsTool(),
		p.getSystemMetricsTool(),
		p.searchLogsTool(),
	}
}

// getAppLogsTool returns the app logs retrieval tool.
func (p *LogsMetricsToolProvider) getAppLogsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_app_logs",
		Description: "Get logs for an application. Returns both lifecycle logs (startup/shutdown) and background step logs. Useful for debugging issues and monitoring behavior.",
		Category:    "logs_metrics",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
				"lines": {
					Type:        "integer",
					Default:     IntValue(100),
					Description: "Number of log lines to retrieve (default: 100)",
				},
				"log_type": {
					Type:        "string",
					Description: "Type of logs: 'all', 'lifecycle', or 'background' (default: all)",
					Enum:        []string{"all", "lifecycle", "background"},
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
			Tags:               []string{"logs", "debugging"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get last 50 lines of agent-inbox logs",
					map[string]interface{}{
						"app_id": "agent-inbox",
						"lines":  50,
					},
				),
				NewToolExample(
					"Get lifecycle logs only",
					map[string]interface{}{
						"app_id":   "agent-inbox",
						"log_type": "lifecycle",
					},
				),
			},
		},
	}
}

// getAppMetricsTool returns the app metrics retrieval tool.
func (p *LogsMetricsToolProvider) getAppMetricsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_app_metrics",
		Description: "Get performance metrics history for an application. Returns time-series data including CPU, memory, disk, and network usage.",
		Category:    "logs_metrics",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_id": {
					Type:        "string",
					Description: "The application identifier (scenario name)",
				},
				"duration": {
					Type:        "string",
					Default:     StringValue("1h"),
					Description: "Time duration to retrieve metrics for (e.g., '1h', '24h', '7d')",
				},
			},
			Required: []string{"app_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"metrics", "performance"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get last hour of metrics for agent-inbox",
					map[string]interface{}{
						"app_id":   "agent-inbox",
						"duration": "1h",
					},
				),
			},
		},
	}
}

// getSystemMetricsTool returns the system-wide metrics tool.
func (p *LogsMetricsToolProvider) getSystemMetricsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_system_metrics",
		Description: "Get system-wide performance metrics including CPU, memory, disk, and network usage for the entire Vrooli server.",
		Category:    "logs_metrics",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
			Required:   []string{},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      false,
			Tags:               []string{"metrics", "system", "performance"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get current system metrics",
					map[string]interface{}{},
				),
			},
		},
	}
}

// searchLogsTool returns the log search tool.
func (p *LogsMetricsToolProvider) searchLogsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "search_logs",
		Description: "Search logs for an application by name. Alternative to get_app_logs that uses the app name directly.",
		Category:    "logs_metrics",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"app_name": {
					Type:        "string",
					Description: "The application name (scenario name)",
				},
				"lines": {
					Type:        "integer",
					Default:     IntValue(100),
					Description: "Number of log lines to retrieve (default: 100)",
				},
			},
			Required: []string{"app_name"},
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
			Tags:               []string{"logs", "search"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Search logs for agent-inbox",
					map[string]interface{}{
						"app_name": "agent-inbox",
						"lines":    50,
					},
				),
			},
		},
	}
}
