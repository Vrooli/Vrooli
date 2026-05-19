// Package toolexecution implements the Tool Execution Protocol for browser-automation-studio.
//
// This package provides the dispatcher and result envelope used by the
// ToolsService Connect-RPC handler (handlers/tools) to invoke individual
// tools. The transport surface lives in proto:
// packages/proto/schemas/browser-automation-studio/v1/tools/tools.proto
// (ToolsService.Execute).
package toolexecution

// ExecuteRequest is the request body for tool execution.
type ExecuteRequest struct {
	ToolName  string                 `json:"tool_name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ExecutionResult is the standardized response for tool execution.
type ExecutionResult struct {
	// Success indicates whether the tool executed successfully
	Success bool `json:"success"`

	// Result contains the tool output (for successful executions)
	Result interface{} `json:"result,omitempty"`

	// Error contains the error message (for failed executions)
	Error string `json:"error,omitempty"`

	// Code is a machine-readable error code (for failed executions)
	// Common codes: unknown_tool, invalid_args, not_found, conflict, internal_error
	Code string `json:"code,omitempty"`

	// IsAsync indicates whether this is a long-running operation
	IsAsync bool `json:"is_async"`

	// RunID is the unique identifier for async operations (for polling status)
	RunID string `json:"run_id,omitempty"`

	// Status is the current status for async operations (pending, running, completed, failed)
	Status string `json:"status,omitempty"`
}

// Error codes used in ExecutionResult.Code
const (
	CodeUnknownTool   = "unknown_tool"
	CodeInvalidArgs   = "invalid_args"
	CodeNotFound      = "not_found"
	CodeConflict      = "conflict"
	CodeInternalError = "internal_error"
	CodeValidation    = "validation_error"
)

// Async status values used in ExecutionResult.Status
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
)

// Helper functions for creating common responses

// SuccessResult creates a successful synchronous result.
func SuccessResult(result interface{}) *ExecutionResult {
	return &ExecutionResult{
		Success: true,
		Result:  result,
		IsAsync: false,
	}
}

// AsyncResult creates a successful async result with a run ID.
func AsyncResult(result interface{}, runID string) *ExecutionResult {
	return &ExecutionResult{
		Success: true,
		Result:  result,
		IsAsync: true,
		RunID:   runID,
		Status:  StatusPending,
	}
}

// ErrorResult creates an error result.
func ErrorResult(message, code string) *ExecutionResult {
	return &ExecutionResult{
		Success: false,
		Error:   message,
		Code:    code,
		IsAsync: false,
	}
}
