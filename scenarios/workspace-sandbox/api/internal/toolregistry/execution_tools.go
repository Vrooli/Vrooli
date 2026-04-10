// Package toolregistry provides the tool discovery service for workspace-sandbox.
//
// This file defines the command execution tools (Tier 2) that are exposed
// via the Tool Discovery Protocol.
package toolregistry

import (
	"context"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// ExecutionToolProvider provides the command execution tools.
// These tools enable running commands and managing processes in sandboxes.
type ExecutionToolProvider struct{}

// NewExecutionToolProvider creates a new ExecutionToolProvider.
func NewExecutionToolProvider() *ExecutionToolProvider {
	return &ExecutionToolProvider{}
}

// Name returns the provider identifier.
func (p *ExecutionToolProvider) Name() string {
	return "workspace-sandbox-execution"
}

// Categories returns the tool categories for command execution.
func (p *ExecutionToolProvider) Categories(_ context.Context) []*toolspb.ToolCategory {
	return []*toolspb.ToolCategory{
		{
			Id:          "command_execution",
			Name:        "Command Execution",
			Description: "Tools for executing commands and managing processes in sandboxes",
			Icon:        "terminal",
		},
	}
}

// Tools returns the tool definitions for command execution.
func (p *ExecutionToolProvider) Tools(_ context.Context) []*toolspb.ToolDefinition {
	return []*toolspb.ToolDefinition{
		p.executeCommandTool(),
		p.startProcessTool(),
		p.getProcessStatusTool(),
		p.listProcessesTool(),
		p.stopProcessTool(),
		p.getProcessLogsTool(),
	}
}

func (p *ExecutionToolProvider) executeCommandTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "execute_command",
		Description: "Execute a command synchronously in a sandbox and wait for completion. Returns stdout, stderr, and exit code. Use for short-running commands where you need the output immediately.",
		Category:    "command_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox to execute the command in.",
				},
				"command": {
					Type:        "string",
					Description: "The command to execute (e.g., 'npm', 'python', 'go').",
				},
				"args": {
					Type:        "array",
					Description: "Command arguments as an array of strings.",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"working_dir": {
					Type:        "string",
					Description: "Working directory relative to the sandbox root. Defaults to sandbox root if not specified.",
				},
				"env": {
					Type:        "object",
					Description: "Environment variables to set for the command execution.",
				},
				"timeout_sec": {
					Type:        "integer",
					Default:     IntValue(60),
					Description: "Maximum time to wait for command completion in seconds.",
				},
				"isolation_profile": {
					Type:        "string",
					Enum:        []string{"full", "restricted", "vrooli-aware"},
					Default:     StringValue("restricted"),
					Description: "Security isolation profile to use. 'restricted' is recommended for untrusted code.",
				},
			},
			Required: []string{"sandbox_id", "command"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     120,
			RateLimitPerMinute: 60,
			CostEstimate:       "medium",
			LongRunning:        false,
			Idempotent:         false,
			Tags:               []string{"execution", "command", "sync"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Run npm install",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"command":    "npm",
						"args":       []string{"install"},
					},
				),
				NewToolExample(
					"Run tests with custom timeout",
					map[string]interface{}{
						"sandbox_id":  "550e8400-e29b-41d4-a716-446655440000",
						"command":     "npm",
						"args":        []string{"test"},
						"timeout_sec": 300,
					},
				),
			},
		},
	}
}

func (p *ExecutionToolProvider) startProcessTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "start_process",
		Description: "Start a background process in a sandbox. Returns immediately with a PID that can be used to monitor the process. Use for long-running commands like development servers.",
		Category:    "command_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox to start the process in.",
				},
				"command": {
					Type:        "string",
					Description: "The command to execute.",
				},
				"args": {
					Type:        "array",
					Description: "Command arguments as an array of strings.",
					Items: &toolspb.ParameterSchema{
						Type: "string",
					},
				},
				"working_dir": {
					Type:        "string",
					Description: "Working directory relative to the sandbox root.",
				},
				"env": {
					Type:        "object",
					Description: "Environment variables to set for the process.",
				},
				"isolation_profile": {
					Type:        "string",
					Enum:        []string{"full", "restricted", "vrooli-aware"},
					Default:     StringValue("restricted"),
					Description: "Security isolation profile to use.",
				},
			},
			Required: []string{"sandbox_id", "command"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "medium",
			LongRunning:        true,
			Idempotent:         false,
			Tags:               []string{"execution", "process", "async", "background"},
			AsyncBehavior: &toolspb.AsyncBehavior{
				StatusPolling: &toolspb.StatusPolling{
					StatusTool:             "get_process_status",
					OperationIdField:       "pid",
					StatusToolIdParam:      "pid",
					PollIntervalSeconds:    2,
					MaxPollDurationSeconds: 3600,
				},
				CompletionConditions: &toolspb.CompletionConditions{
					StatusField:   "status",
					SuccessValues: []string{"exited"},
					FailureValues: []string{"killed", "error", "failed"},
					PendingValues: []string{"running"},
				},
				Cancellation: &toolspb.CancellationBehavior{
					CancelTool:        "stop_process",
					CancelToolIdParam: "pid",
				},
			},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Start a development server",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"command":    "npm",
						"args":       []string{"run", "dev"},
					},
				),
			},
		},
	}
}

func (p *ExecutionToolProvider) getProcessStatusTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_process_status",
		Description: "Get the current status of a process running in a sandbox. Returns whether the process is still running and its exit code if completed.",
		Category:    "command_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox containing the process.",
				},
				"pid": {
					Type:        "integer",
					Description: "The process ID to check.",
				},
			},
			Required: []string{"sandbox_id", "pid"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 120,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"execution", "process", "status"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Check process status",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"pid":        12345,
					},
				),
			},
		},
	}
}

func (p *ExecutionToolProvider) listProcessesTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "list_processes",
		Description: "List all processes (running and recently completed) in a sandbox.",
		Category:    "command_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox to list processes for.",
				},
			},
			Required: []string{"sandbox_id"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     10,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"execution", "process", "list"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"List all processes in a sandbox",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
					},
				),
			},
		},
	}
}

func (p *ExecutionToolProvider) stopProcessTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "stop_process",
		Description: "Stop (kill) a running process in a sandbox. Sends SIGTERM first, then SIGKILL if the process doesn't exit gracefully.",
		Category:    "command_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox containing the process.",
				},
				"pid": {
					Type:        "integer",
					Description: "The process ID to stop.",
				},
			},
			Required: []string{"sandbox_id", "pid"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     30,
			RateLimitPerMinute: 30,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"execution", "process", "stop", "kill"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Stop a running process",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"pid":        12345,
					},
				),
			},
		},
	}
}

func (p *ExecutionToolProvider) getProcessLogsTool() *toolspb.ToolDefinition {
	return &toolspb.ToolDefinition{
		Name:        "get_process_logs",
		Description: "Get the stdout and stderr output from a process. Useful for checking command output or debugging.",
		Category:    "command_execution",
		Parameters: &toolspb.ToolParameters{
			Type: "object",
			Properties: map[string]*toolspb.ParameterSchema{
				"sandbox_id": {
					Type:        "string",
					Format:      "uuid",
					Description: "The sandbox containing the process.",
				},
				"pid": {
					Type:        "integer",
					Description: "The process ID to get logs for.",
				},
				"tail_lines": {
					Type:        "integer",
					Default:     IntValue(100),
					Description: "Number of lines to return from the end of the log.",
				},
				"stream": {
					Type:        "string",
					Enum:        []string{"stdout", "stderr", "combined"},
					Default:     StringValue("combined"),
					Description: "Which output stream to return.",
				},
			},
			Required: []string{"sandbox_id", "pid"},
		},
		Metadata: &toolspb.ToolMetadata{
			EnabledByDefault:   true,
			RequiresApproval:   false,
			TimeoutSeconds:     15,
			RateLimitPerMinute: 60,
			CostEstimate:       "low",
			LongRunning:        false,
			Idempotent:         true,
			Tags:               []string{"execution", "process", "logs", "output"},
			Examples: []*toolspb.ToolExample{
				NewToolExample(
					"Get last 50 lines of process output",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"pid":        12345,
						"tail_lines": 50,
					},
				),
				NewToolExample(
					"Get only stderr output",
					map[string]interface{}{
						"sandbox_id": "550e8400-e29b-41d4-a716-446655440000",
						"pid":        12345,
						"stream":     "stderr",
					},
				),
			},
		},
	}
}
