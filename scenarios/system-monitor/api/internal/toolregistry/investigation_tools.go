// Package toolregistry provides the tool discovery service for system-monitor.
//
// This file defines the investigation tools that are exposed via the Tool Discovery Protocol.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// InvestigationToolProvider provides the AI-driven investigation tools.
type InvestigationToolProvider struct{}

// NewInvestigationToolProvider creates a new InvestigationToolProvider.
func NewInvestigationToolProvider() *InvestigationToolProvider {
	return &InvestigationToolProvider{}
}

// Name returns the provider identifier.
func (p *InvestigationToolProvider) Name() string {
	return "system-monitor-investigation"
}

// Categories returns the tool categories for investigation tools.
func (p *InvestigationToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:          "investigation",
			Name:        "Investigation",
			Description: "Tools for AI-driven anomaly detection and root cause analysis",
			Icon:        "search",
		},
		{
			Id:          "reports",
			Name:        "Reports",
			Description: "Tools for generating system analysis reports",
			Icon:        "file-text",
		},
	}
}

// Tools returns the tool definitions for investigation tools.
func (p *InvestigationToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.triggerInvestigationTool(),
		p.checkInvestigationStatusTool(),
		p.getLatestInvestigationTool(),
		p.stopInvestigationTool(),
		p.generateReportTool(),
	}
}

func (p *InvestigationToolProvider) triggerInvestigationTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "trigger_investigation",
		Description: "Trigger an AI-driven investigation to analyze system anomalies and perform root cause analysis. The investigation runs asynchronously using agent-manager. Use check_investigation_status to poll for progress.",
		Category:    "investigation",
		Parameters: NewObjectParams(
			map[string]*toolspb.ParameterSchema{
				"auto_fix": NewBoolParamWithDefault(
					"If true, the investigation agent may automatically apply fixes for detected issues. If false (default), it only reports findings.",
					false,
				),
				"note": NewStringParam(
					"Optional note or specific focus area for the investigation. Example: 'Focus on memory issues' or 'Check disk I/O patterns'",
				),
			},
			nil, // No required parameters
		),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false, // Manual approval via auto_fix=false by default
			TimeoutSeconds:     30,    // Initial response timeout, actual investigation runs async
			RateLimitPerMinute: 5,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         false,
			Tags:               []string{"investigation", "anomaly", "ai", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Start a read-only investigation",
					map[string]interface{}{},
				),
				NewToolExample(
					"Start investigation with auto-fix enabled",
					map[string]interface{}{"auto_fix": true},
				),
				NewToolExample(
					"Investigate specific issue",
					map[string]interface{}{
						"note": "High memory usage detected, focus on memory leaks",
					},
				),
			},
			// Define async behavior for status polling
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "check_investigation_status",
					OperationIdField:       "investigation_id",
					StatusToolIdParam:      "investigation_id",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 1800, // 30 min max
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"completed"},
					FailureValues: []string{"failed", "cancelled", "stopped"},
					PendingValues: []string{"queued", "in_progress"},
					ErrorField:    "error",
					ResultField:   "findings",
				},
				ProgressTracking: &toolspb.ProgressTracking{
					ProgressField: "progress",
					MessageField:  "findings",
				},
				Cancellation: &toolspb.CancellationBehavior{
					CancelTool:        "stop_investigation",
					CancelToolIdParam: "investigation_id",
				},
			},
		},
	}
}

func (p *InvestigationToolProvider) checkInvestigationStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "check_investigation_status",
		Description: "Check the current status and progress of a running or completed investigation. Returns status, progress percentage, findings, and details.",
		Category:    "investigation",
		Parameters: NewObjectParams(
			map[string]*toolspb.ParameterSchema{
				"investigation_id": NewStringParam(
					"The ID of the investigation to check. This is returned when triggering an investigation.",
				),
			},
			[]string{"investigation_id"},
		),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"investigation", "status", "monitoring"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check investigation progress",
					map[string]interface{}{
						"investigation_id": "inv_1234567890",
					},
				),
			},
		},
	}
}

func (p *InvestigationToolProvider) getLatestInvestigationTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_latest_investigation",
		Description: "Get the most recent investigation's status and findings. Useful for checking if there's an active investigation or reviewing the last completed analysis.",
		Category:    "investigation",
		Parameters:  NewEmptyParams(),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"investigation", "status", "latest"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get the most recent investigation",
					map[string]interface{}{},
				),
			},
		},
	}
}

func (p *InvestigationToolProvider) stopInvestigationTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "stop_investigation",
		Description: "Stop a running investigation. Use this to cancel an investigation that is taking too long or is no longer needed.",
		Category:    "investigation",
		Parameters: NewObjectParams(
			map[string]*toolspb.ParameterSchema{
				"investigation_id": NewStringParam(
					"The ID of the investigation to stop.",
				),
			},
			[]string{"investigation_id"},
		),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"investigation", "cancellation", "lifecycle"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Stop a running investigation",
					map[string]interface{}{
						"investigation_id": "inv_1234567890",
					},
				),
			},
		},
	}
}

func (p *InvestigationToolProvider) generateReportTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "generate_report",
		Description: "Generate a comprehensive system analysis report. Reports include executive summary, metrics overview, anomaly history, performance trends, and recommendations.",
		Category:    "reports",
		Parameters: NewObjectParams(
			map[string]*toolspb.ParameterSchema{
				"type": NewStringParamWithEnum(
					"Type of report to generate. Daily reports cover the last 24 hours, weekly reports cover the last 7 days.",
					[]string{"daily", "weekly"},
					"daily",
				),
			},
			nil, // No required parameters
		),
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     60,
			RateLimitPerMinute: 10,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"reports", "analysis", "summary"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Generate a daily system report",
					map[string]interface{}{"type": "daily"},
				),
				NewToolExample(
					"Generate a weekly system report",
					map[string]interface{}{"type": "weekly"},
				),
			},
		},
	}
}
