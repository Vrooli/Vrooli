// Package toolexecution provides the tool execution service for system-monitor.
//
// This package implements the Tool Execution Protocol, enabling external consumers
// (like agent-inbox) to execute tools exposed by this scenario.
package toolexecution

// ExecuteRequest represents a request to execute a tool.
type ExecuteRequest struct {
	// ToolName is the name of the tool to execute.
	ToolName string `json:"tool_name"`

	// Arguments contains the tool-specific parameters.
	Arguments map[string]interface{} `json:"arguments"`
}

// ExecutionResult represents the result of executing a tool.
type ExecutionResult struct {
	// Success indicates if the tool executed successfully.
	Success bool `json:"success"`

	// Result contains the tool-specific result data.
	// For async tools, this may contain initial response before polling begins.
	Result interface{} `json:"result,omitempty"`

	// Error contains the error message if Success is false.
	Error string `json:"error,omitempty"`

	// Code contains a machine-readable error code.
	// Common codes: unknown_tool, invalid_args, not_found, conflict, internal_error
	Code string `json:"code,omitempty"`

	// IsAsync indicates if this is an asynchronous operation that requires polling.
	IsAsync bool `json:"is_async"`

	// RunID contains the operation ID for async operations (e.g., investigation_id).
	// Use this with the status polling tool to check progress.
	RunID string `json:"run_id,omitempty"`

	// Status contains the current status for async operations.
	// Values: queued, in_progress, completed, failed, cancelled, stopped
	Status string `json:"status,omitempty"`
}

// ErrorCodes for standardized error responses.
const (
	ErrorCodeUnknownTool   = "unknown_tool"
	ErrorCodeInvalidArgs   = "invalid_args"
	ErrorCodeNotFound      = "not_found"
	ErrorCodeConflict      = "conflict"
	ErrorCodeInternalError = "internal_error"
	ErrorCodeCooldown      = "cooldown"
	ErrorCodeUnavailable   = "unavailable"
)

// NewSuccessResult creates a successful execution result.
func NewSuccessResult(result interface{}) *ExecutionResult {
	return &ExecutionResult{
		Success: true,
		Result:  result,
	}
}

// NewAsyncResult creates an async execution result.
func NewAsyncResult(runID string, status string, result interface{}) *ExecutionResult {
	return &ExecutionResult{
		Success: true,
		IsAsync: true,
		RunID:   runID,
		Status:  status,
		Result:  result,
	}
}

// NewErrorResult creates a failed execution result.
func NewErrorResult(code string, message string) *ExecutionResult {
	return &ExecutionResult{
		Success: false,
		Error:   message,
		Code:    code,
	}
}
