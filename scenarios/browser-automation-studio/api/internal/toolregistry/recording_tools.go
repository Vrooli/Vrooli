// Package toolregistry provides tool definitions for recording sessions.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// RecordingToolProvider provides recording session tools.
type RecordingToolProvider struct{}

// NewRecordingToolProvider creates a new RecordingToolProvider.
func NewRecordingToolProvider() *RecordingToolProvider {
	return &RecordingToolProvider{}
}

// Name returns the provider name.
func (p *RecordingToolProvider) Name() string {
	return "bas-recording"
}

// Categories returns the categories for recording tools.
func (p *RecordingToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		NewCategory("recording", "Recording", "Tools for recording browser sessions and converting them to workflows", "record", 5),
	}
}

// Tools returns the recording tool definitions.
func (p *RecordingToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.createRecordingSessionTool(),
		p.getRecordedActionsTool(),
		p.generateWorkflowFromRecordingTool(),
	}
}

// createRecordingSessionTool defines the create_recording_session tool (async).
func (p *RecordingToolProvider) createRecordingSessionTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"url":        StringParamWithFormat("Initial URL to navigate to when starting the recording session", "uri"),
			"project_id": StringParamWithFormat("Project ID to associate the recording with (optional)", "uuid"),
			"profile_id": StringParamWithFormat("Session profile ID to use for browser state (optional)", "uuid"),
		},
		[]string{"url"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     60,
		RateLimitPerMinute: 5,
		CostEstimate:       "high",
		LongRunning:        true,
		Idempotent:         false,
		ModifiesState:      true,
		Tags:               []string{"recording", "capture", "async"},
	})

	meta.AsyncBehavior = NewAsyncBehavior(AsyncBehaviorOpts{
		StatusTool:             "get_recording_session",
		OperationIdField:       "session_id",
		StatusToolIdParam:      "session_id",
		PollIntervalSeconds:    2,
		MaxPollDurationSeconds: 1800, // 30 minutes max recording
		StatusField:            "status",
		SuccessValues:          []string{"stopped", "completed"},
		FailureValues:          []string{"failed", "error"},
		PendingValues:          []string{"recording", "active", "starting"},
		ErrorField:             "error_message",
		ResultField:            "result",
	})

	return NewToolDefinition(
		"create_recording_session",
		"Start a live browser recording session. Opens a browser window and captures all user interactions. The session can later be converted to a workflow.",
		"recording",
		params,
		meta,
	)
}

// getRecordedActionsTool defines the get_recorded_actions tool.
func (p *RecordingToolProvider) getRecordedActionsTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"session_id": StringParamWithFormat("The recording session ID", "uuid"),
		},
		[]string{"session_id"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     30,
		RateLimitPerMinute: 30,
		CostEstimate:       "low",
		Idempotent:         true,
		Tags:               []string{"recording", "actions"},
	})

	return NewToolDefinition(
		"get_recorded_actions",
		"Get the list of recorded actions from a recording session. Returns clicks, typing, navigation, and other browser interactions.",
		"recording",
		params,
		meta,
	)
}

// generateWorkflowFromRecordingTool defines the generate_workflow_from_recording tool (async).
func (p *RecordingToolProvider) generateWorkflowFromRecordingTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"session_id":    StringParamWithFormat("The recording session ID to convert", "uuid"),
			"workflow_name": StringParam("Name for the generated workflow"),
			"project_id":    StringParamWithFormat("Project ID to save the workflow to (optional)", "uuid"),
		},
		[]string{"session_id", "workflow_name"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     60,
		RateLimitPerMinute: 10,
		CostEstimate:       "medium",
		LongRunning:        true,
		Idempotent:         false,
		ModifiesState:      true,
		Tags:               []string{"recording", "workflow", "generation"},
	})

	return NewToolDefinition(
		"generate_workflow_from_recording",
		"Convert a recording session into a reusable workflow. Analyzes recorded actions and generates an optimized workflow definition.",
		"recording",
		params,
		meta,
	)
}
