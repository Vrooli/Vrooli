// Package toolregistry provides tool definitions for app-issue-tracker.
//
// This file defines AI investigation tools including async operations.
// The investigate_issue tool triggers long-running AI investigations.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// InvestigationProvider provides AI investigation tools.
type InvestigationProvider struct{}

// NewInvestigationProvider creates a new InvestigationProvider.
func NewInvestigationProvider() *InvestigationProvider {
	return &InvestigationProvider{}
}

// Name returns the provider identifier.
func (p *InvestigationProvider) Name() string {
	return "issue-tracker-investigation"
}

// Categories returns the tool categories for investigation tools.
func (p *InvestigationProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:           "investigation",
			Name:         "Issue Investigation",
			Description:  "Tools for triggering and monitoring AI-powered issue investigations",
			Icon:         "zap",
			DisplayOrder: 3,
		},
	}
}

// Tools returns the investigation tool definitions.
func (p *InvestigationProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.investigateIssueTool(),
		p.getInvestigationStatusTool(),
		p.previewInvestigationPromptTool(),
		p.getAgentConversationTool(),
		p.listRunningInvestigationsTool(),
		p.stopInvestigationTool(),
	}
}

// investigateIssueTool returns the async investigation trigger tool.
func (p *InvestigationProvider) investigateIssueTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "investigate_issue",
		Description: "Trigger an AI agent to investigate an issue. The agent will analyze the error context, research potential causes, and suggest fixes. This is a long-running operation that can take 5-30 minutes. Use get_investigation_status to monitor progress.",
		Category:    "investigation",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"issue_id": {
					Type:        "string",
					Description: "The unique identifier of the issue to investigate",
				},
				"agent_id": {
					Type:        "string",
					Default:     StringValue("unified-resolver"),
					Description: "Agent profile to use for investigation (default: unified-resolver)",
				},
				"auto_resolve": {
					Type:        "boolean",
					Default:     BoolValue(true),
					Description: "Automatically apply the fix if the agent is confident",
				},
				"force": {
					Type:        "boolean",
					Default:     BoolValue(false),
					Description: "Force start investigation even if one is already running",
				},
			},
			Required: []string{"issue_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false, // Investigation is read-mostly
			TimeoutSeconds:     30,    // Initial response timeout (returns run_id)
			RateLimitPerMinute: 10,
			CostEstimate:       "high",
			LongRunning:        true,
			Idempotent:         false,
			ModifiesState:      true,
			Tags:               []string{"investigation", "ai", "async"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Start investigation with default settings",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
					},
				),
				NewToolExample(
					"Start investigation without auto-resolve",
					map[string]interface{}{
						"issue_id":     "issue-2024-01-15-001",
						"auto_resolve": false,
					},
				),
				NewToolExample(
					"Force restart investigation",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
						"force":    true,
					},
				),
			},
			// Define async behavior for status polling
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "get_investigation_status",
					OperationIdField:       "run_id",
					StatusToolIdParam:      "issue_id",
					PollIntervalSeconds:    5,
					MaxPollDurationSeconds: 1800, // 30 minutes max
					Backoff: &toolspb.PollingBackoff{
						InitialIntervalSeconds: 5,
						MaxIntervalSeconds:     30,
						Multiplier:             1.5,
					},
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"completed"},
					FailureValues: []string{"failed", "cancelled"},
					PendingValues: []string{"active", "running", "cancelling"},
					ErrorField:    "error_message",
					ResultField:   "investigation_result",
				},
				ProgressTracking: &toolspb.ProgressTracking{
					ProgressField: "progress_percent",
					MessageField:  "current_action",
					PhaseField:    "phase",
				},
				Cancellation: &toolspb.CancellationBehavior{
					CancelTool:           "stop_investigation",
					CancelToolIdParam:    "issue_id",
					Graceful:             true,
					CancelTimeoutSeconds: 30,
				},
			},
		},
	}
}

// getInvestigationStatusTool returns the status polling tool.
func (p *InvestigationProvider) getInvestigationStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_investigation_status",
		Description: "Get the current status of an issue investigation. Returns status, progress, current action, and results if completed. Use this to poll for completion after starting investigate_issue.",
		Category:    "investigation",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"issue_id": {
					Type:        "string",
					Description: "The unique identifier of the issue to check",
				},
			},
			Required: []string{"issue_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"status", "polling", "investigation"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check investigation status",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
					},
				),
			},
		},
	}
}

// previewInvestigationPromptTool returns the prompt preview tool.
func (p *InvestigationProvider) previewInvestigationPromptTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "preview_investigation_prompt",
		Description: "Preview the prompt that would be sent to the AI agent for investigation. Useful for reviewing before starting a costly investigation.",
		Category:    "investigation",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"issue_id": {
					Type:        "string",
					Description: "The unique identifier of the issue to preview",
				},
				"agent_id": {
					Type:        "string",
					Default:     StringValue("unified-resolver"),
					Description: "Agent profile to use (affects prompt template)",
				},
			},
			Required: []string{"issue_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"preview", "prompt", "investigation"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Preview investigation prompt",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
					},
				),
			},
		},
	}
}

// getAgentConversationTool returns the conversation retrieval tool.
func (p *InvestigationProvider) getAgentConversationTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_agent_conversation",
		Description: "Get the full conversation transcript from an AI investigation. Returns the agent's analysis steps, findings, and reasoning.",
		Category:    "investigation",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"issue_id": {
					Type:        "string",
					Description: "The unique identifier of the issue",
				},
				"max_events": {
					Type:        "integer",
					Default:     IntValue(100),
					Description: "Maximum number of conversation events to return",
				},
			},
			Required: []string{"issue_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"conversation", "transcript", "investigation"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get full conversation",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
					},
				),
				NewToolExample(
					"Get recent conversation events",
					map[string]interface{}{
						"issue_id":   "issue-2024-01-15-001",
						"max_events": 20,
					},
				),
			},
		},
	}
}

// listRunningInvestigationsTool returns the running investigations list tool.
func (p *InvestigationProvider) listRunningInvestigationsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_running_investigations",
		Description: "List all currently running AI investigations. Returns issue ID, agent ID, run ID, start time, and current status for each active investigation.",
		Category:    "investigation",
		Parameters: &toolspb.ToolParameters{
			Type:       "object",
			Properties: map[string]*toolspb.ParameterSchema{},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"list", "running", "investigation"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all running investigations",
					map[string]interface{}{},
				),
			},
		},
	}
}

// stopInvestigationTool returns the investigation cancellation tool.
func (p *InvestigationProvider) stopInvestigationTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "stop_investigation",
		Description: "Stop a running AI investigation. The agent will be asked to gracefully stop and save any partial findings.",
		Category:    "investigation",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"issue_id": {
					Type:        "string",
					Description: "The unique identifier of the issue whose investigation to stop",
				},
			},
			Required: []string{"issue_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 20,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			ModifiesState:      true,
			Tags:               []string{"stop", "cancel", "investigation"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Stop investigation for an issue",
					map[string]interface{}{
						"issue_id": "issue-2024-01-15-001",
					},
				),
			},
		},
	}
}
