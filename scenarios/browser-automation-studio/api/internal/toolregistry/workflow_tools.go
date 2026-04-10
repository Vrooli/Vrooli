// Package toolregistry provides tool definitions for workflow execution.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// WorkflowToolProvider provides workflow execution tools.
type WorkflowToolProvider struct{}

// NewWorkflowToolProvider creates a new WorkflowToolProvider.
func NewWorkflowToolProvider() *WorkflowToolProvider {
	return &WorkflowToolProvider{}
}

// Name returns the provider name.
func (p *WorkflowToolProvider) Name() string {
	return "bas-workflow"
}

// Categories returns the categories for workflow tools.
func (p *WorkflowToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		NewCategory("workflow_execution", "Workflow Execution", "Tools for executing and monitoring browser automation workflows", "play", 1),
		NewCategory("execution_status", "Execution Status", "Tools for checking execution status and results", "info", 2),
	}
}

// Tools returns the workflow tool definitions.
func (p *WorkflowToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.executeWorkflowTool(),
		p.getExecutionTool(),
		p.getExecutionTimelineTool(),
		p.stopExecutionTool(),
		p.listWorkflowsTool(),
		p.listExecutionsTool(),
	}
}

// executeWorkflowTool defines the execute_workflow tool (async, long-running).
func (p *WorkflowToolProvider) executeWorkflowTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"workflow_id": StringParamWithFormat("The unique ID of the workflow to execute", "uuid"),
			"parameters":  ObjectParam("Optional parameters to pass to the workflow (key-value pairs)"),
		},
		[]string{"workflow_id"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     60, // Initial response timeout
		RateLimitPerMinute: 10,
		CostEstimate:       "high",
		LongRunning:        true,
		Idempotent:         false,
		ModifiesState:      true,
		Tags:               []string{"workflow", "execution", "async"},
	})

	meta.AsyncBehavior = NewAsyncBehavior(AsyncBehaviorOpts{
		StatusTool:             "get_execution",
		OperationIdField:       "execution_id",
		StatusToolIdParam:      "execution_id",
		PollIntervalSeconds:    5,
		MaxPollDurationSeconds: 3600, // 1 hour
		StatusField:            "status",
		SuccessValues:          []string{"completed", "passed"},
		FailureValues:          []string{"failed", "stopped", "error"},
		PendingValues:          []string{"pending", "running", "queued"},
		ErrorField:             "error_message",
		ResultField:            "result",
		CancelTool:             "stop_execution",
		CancelToolIdParam:      "execution_id",
	})

	return NewToolDefinition(
		"execute_workflow",
		"Execute a browser automation workflow. Returns an execution ID that can be used to monitor progress. Workflows can automate web interactions, form filling, data extraction, and more.",
		"workflow_execution",
		params,
		meta,
	)
}

// getExecutionTool defines the get_execution tool (sync, used for status polling).
func (p *WorkflowToolProvider) getExecutionTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"execution_id": StringParamWithFormat("The execution ID returned from execute_workflow", "uuid"),
		},
		[]string{"execution_id"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     30,
		RateLimitPerMinute: 60,
		CostEstimate:       "low",
		Idempotent:         true,
		Tags:               []string{"status", "monitoring"},
	})

	return NewToolDefinition(
		"get_execution",
		"Get the current status and details of a workflow execution. Use this to monitor progress of running executions or retrieve results of completed ones.",
		"execution_status",
		params,
		meta,
	)
}

// getExecutionTimelineTool defines the get_execution_timeline tool.
func (p *WorkflowToolProvider) getExecutionTimelineTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"execution_id": StringParamWithFormat("The execution ID to get timeline for", "uuid"),
			"limit":        IntParamWithDefault("Maximum number of timeline entries to return", 100),
			"offset":       IntParamWithDefault("Number of entries to skip", 0),
		},
		[]string{"execution_id"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     30,
		RateLimitPerMinute: 30,
		CostEstimate:       "low",
		Idempotent:         true,
		Tags:               []string{"timeline", "monitoring", "debugging"},
	})

	return NewToolDefinition(
		"get_execution_timeline",
		"Get the timeline of events for a workflow execution. Includes step-by-step actions, screenshots, and any errors encountered.",
		"execution_status",
		params,
		meta,
	)
}

// stopExecutionTool defines the stop_execution tool.
func (p *WorkflowToolProvider) stopExecutionTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"execution_id": StringParamWithFormat("The execution ID to stop", "uuid"),
		},
		[]string{"execution_id"},
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     30,
		RateLimitPerMinute: 20,
		CostEstimate:       "low",
		Idempotent:         true,
		ModifiesState:      true,
		Tags:               []string{"control", "cancel"},
	})

	return NewToolDefinition(
		"stop_execution",
		"Stop a running workflow execution. The execution will be marked as stopped and any in-progress browser actions will be terminated.",
		"workflow_execution",
		params,
		meta,
	)
}

// listWorkflowsTool defines the list_workflows tool.
func (p *WorkflowToolProvider) listWorkflowsTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"project_id": StringParamWithFormat("Filter workflows by project ID (optional)", "uuid"),
			"limit":      IntParamWithDefault("Maximum number of workflows to return", 50),
			"offset":     IntParamWithDefault("Number of workflows to skip", 0),
		},
		[]string{}, // No required params
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     30,
		RateLimitPerMinute: 30,
		CostEstimate:       "low",
		Idempotent:         true,
		Tags:               []string{"listing", "discovery"},
	})

	return NewToolDefinition(
		"list_workflows",
		"List available workflows. Can be filtered by project. Returns workflow names, IDs, and metadata.",
		"workflow_execution",
		params,
		meta,
	)
}

// listExecutionsTool defines the list_executions tool.
func (p *WorkflowToolProvider) listExecutionsTool() *toolspb.ToolDefinition {
	params := NewToolParameters(
		map[string]*toolspb.ParameterSchema{
			"workflow_id": StringParamWithFormat("Filter executions by workflow ID (optional)", "uuid"),
			"project_id":  StringParamWithFormat("Filter executions by project ID (optional)", "uuid"),
			"status":      EnumParam("Filter by execution status (optional)", []string{"pending", "running", "completed", "failed", "stopped"}),
			"limit":       IntParamWithDefault("Maximum number of executions to return", 50),
			"offset":      IntParamWithDefault("Number of executions to skip", 0),
		},
		[]string{}, // No required params
	)

	meta := NewToolMetadata(ToolMetadataOpts{
		EnabledByDefault:   true,
		RequiresApproval:   false,
		TimeoutSeconds:     30,
		RateLimitPerMinute: 30,
		CostEstimate:       "low",
		Idempotent:         true,
		Tags:               []string{"listing", "history"},
	})

	return NewToolDefinition(
		"list_executions",
		"List workflow executions. Can be filtered by workflow, project, or status. Returns execution history with status and timing information.",
		"execution_status",
		params,
		meta,
	)
}
