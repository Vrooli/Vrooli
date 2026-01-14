// Package toolregistry provides the tool discovery service for system-monitor.
//
// This file defines the metrics tools that are exposed via the Tool Discovery Protocol.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// MetricsToolProvider provides the system metrics monitoring tools.
type MetricsToolProvider struct{}

// NewMetricsToolProvider creates a new MetricsToolProvider.
func NewMetricsToolProvider() *MetricsToolProvider {
	return &MetricsToolProvider{}
}

// Name returns the provider identifier.
func (p *MetricsToolProvider) Name() string {
	return "system-monitor-metrics"
}

// Categories returns the tool categories for metrics tools.
func (p *MetricsToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:          "metrics",
			Name:        "System Metrics",
			Description: "Tools for querying real-time system metrics and health status",
			Icon:        "activity",
		},
	}
}

// Tools returns the tool definitions for metrics tools.
func (p *MetricsToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.getMetricsTool(),
		p.getDetailedMetricsTool(),
		p.getProcessesTool(),
		p.getSystemHealthTool(),
	}
}

func (p *MetricsToolProvider) getMetricsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_metrics",
		Description: "Get a real-time snapshot of system metrics including CPU usage, memory usage, network connections, and GPU utilization. Use this for a quick overview of system resource usage.",
		Category:    "metrics",
		Parameters: NewObjectParams(
			map[string]*toolspb.ParameterSchema{
				"fresh": NewBoolParamWithDefault(
					"Force fresh metrics collection, bypassing any cache. Set to true for the most current data.",
					true,
				),
			},
			nil, // No required parameters
		),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"metrics", "monitoring", "real-time"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get current system metrics",
					map[string]interface{}{},
				),
				NewToolExample(
					"Force fresh metrics collection",
					map[string]interface{}{"fresh": true},
				),
			},
		},
	}
}

func (p *MetricsToolProvider) getDetailedMetricsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_detailed_metrics",
		Description: "Get a comprehensive breakdown of system metrics including detailed CPU stats, memory breakdown (swap, growth patterns), network states, disk usage, and GPU information. Use this when you need in-depth system analysis.",
		Category:    "metrics",
		Parameters:  NewEmptyParams(),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"metrics", "monitoring", "detailed", "analysis"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get comprehensive system metrics",
					map[string]interface{}{},
				),
			},
		},
	}
}

func (p *MetricsToolProvider) getProcessesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_processes",
		Description: "Get information about running processes including their CPU and memory usage. Useful for identifying resource-hungry processes or troubleshooting performance issues.",
		Category:    "metrics",
		Parameters: NewObjectParams(
			map[string]*toolspb.ParameterSchema{
				"sort_by": NewStringParamWithEnum(
					"Sort processes by this metric",
					[]string{"cpu", "memory"},
					"cpu",
				),
				"limit": NewIntParamWithDefault(
					"Maximum number of processes to return",
					20,
				),
			},
			nil, // No required parameters
		),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"processes", "monitoring", "troubleshooting"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List top CPU-consuming processes",
					map[string]interface{}{"sort_by": "cpu", "limit": 10},
				),
				NewToolExample(
					"List memory-hungry processes",
					map[string]interface{}{"sort_by": "memory", "limit": 5},
				),
			},
		},
	}
}

func (p *MetricsToolProvider) getSystemHealthTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_system_health",
		Description: "Get the overall health status of the system including service dependencies (PostgreSQL, Redis, QuestDB, Node-RED, Ollama), uptime, and readiness state. Use this to verify system availability before operations.",
		Category:    "metrics",
		Parameters:  NewEmptyParams(),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"health", "status", "dependencies", "readiness"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check system health and dependencies",
					map[string]interface{}{},
				),
			},
		},
	}
}
