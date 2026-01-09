// Package domain defines the core domain entities for agent-manager.
//
// Protocol types for Tool Discovery (ToolManifest, ToolDefinition, etc.) are
// defined in the proto schema at packages/proto/schemas/agent-inbox/v1/ and
// imported from the generated Go code.
//
// This file defines agent-manager specific types for tool execution results.
package domain

import (
	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// Re-export proto types for backward compatibility with existing imports.
// This allows gradual migration without breaking existing code.
type (
	// Protocol types from proto
	ToolManifest    = toolspb.ToolManifest
	ScenarioInfo    = toolspb.ScenarioInfo
	ToolDefinition  = toolspb.ToolDefinition
	ToolParameters  = toolspb.ToolParameters
	ParameterSchema = toolspb.ParameterSchema
	ToolMetadata    = toolspb.ToolMetadata
	ToolExample     = toolspb.ToolExample
	ToolCategory    = toolspb.ToolCategory
	AsyncBehavior   = toolspb.AsyncBehavior
	StatusPolling   = toolspb.StatusPolling
	// Add more as needed
)

// ToolProtocolVersion is the current version of the Tool Discovery Protocol.
// This is re-exported from the proto package constant.
const ToolProtocolVersion = "1.0"

// -----------------------------------------------------------------------------
// Tool Result Types (agent-manager specific, not part of protocol)
// -----------------------------------------------------------------------------

// ToolResult represents the result of executing a tool.
type ToolResult struct {
	// Success indicates if the tool executed successfully.
	Success bool `json:"success"`

	// Data contains the tool-specific result data.
	Data interface{} `json:"data,omitempty"`

	// Error contains error information if Success is false.
	Error *ToolError `json:"error,omitempty"`

	// Metadata contains execution metadata.
	Metadata *ToolResultMetadata `json:"metadata,omitempty"`
}

// ToolError provides structured error information.
type ToolError struct {
	// Code is a machine-readable error code.
	Code string `json:"code"`

	// Message is a human-readable error message.
	Message string `json:"message"`

	// Retryable indicates if the operation can be retried.
	Retryable bool `json:"retryable"`

	// Details provides additional context.
	Details map[string]interface{} `json:"details,omitempty"`
}

// ToolResultMetadata contains execution metadata.
type ToolResultMetadata struct {
	// ExecutionTime is how long the tool took to execute.
	ExecutionTimeMs int64 `json:"execution_time_ms"`

	// ExternalID is an identifier for async operations (e.g., run_id).
	ExternalID string `json:"external_id,omitempty"`

	// FollowUpActions suggests what the caller might do next.
	FollowUpActions []string `json:"follow_up_actions,omitempty"`
}
