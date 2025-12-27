// Package domain defines the core domain entities for agent-manager.
//
// This file defines the Tool Discovery Protocol types. These types enable
// scenarios to expose their capabilities to other scenarios (like agent-inbox)
// in a standardized, discoverable way.
//
// DESIGN PRINCIPLES:
// - OpenAI function-calling compatible for broad LLM compatibility
// - Self-describing with rich metadata for UI generation
// - Versioned protocol for backwards compatibility
// - Category-based organization for logical grouping
package domain

import "time"

// ProtocolVersion is the current version of the Tool Discovery Protocol.
// Consumers should check this to ensure compatibility.
const ProtocolVersion = "1.0"

// -----------------------------------------------------------------------------
// Tool Discovery Protocol Types
// -----------------------------------------------------------------------------

// ToolManifest is the top-level response from GET /api/v1/tools.
// It contains scenario metadata and all available tools.
type ToolManifest struct {
	// ProtocolVersion indicates the protocol version for compatibility checks.
	ProtocolVersion string `json:"protocol_version"`

	// Scenario provides metadata about the scenario exposing these tools.
	Scenario ScenarioInfo `json:"scenario"`

	// Tools is the list of available tools.
	Tools []ToolDefinition `json:"tools"`

	// Categories provides optional grouping metadata for tools.
	Categories []ToolCategory `json:"categories,omitempty"`

	// GeneratedAt is when this manifest was generated.
	GeneratedAt time.Time `json:"generated_at"`
}

// ScenarioInfo provides metadata about the scenario exposing tools.
type ScenarioInfo struct {
	// Name is the scenario identifier (e.g., "agent-manager").
	Name string `json:"name"`

	// Version is the scenario version (e.g., "1.0.0").
	Version string `json:"version"`

	// Description is a human-readable description of the scenario.
	Description string `json:"description"`

	// BaseURL is the API base URL for this scenario (optional, for remote scenarios).
	BaseURL string `json:"base_url,omitempty"`
}

// ToolDefinition describes a single tool that can be invoked.
// The structure is compatible with OpenAI's function-calling format.
type ToolDefinition struct {
	// Name is the unique identifier for this tool within the scenario.
	// Must be a valid identifier (alphanumeric + underscores, no spaces).
	Name string `json:"name"`

	// Description explains what the tool does. This is shown to the LLM.
	Description string `json:"description"`

	// Category groups related tools together (e.g., "agent_lifecycle", "analysis").
	Category string `json:"category,omitempty"`

	// Parameters defines the input schema in JSON Schema format.
	// This is compatible with OpenAI's function-calling parameter format.
	Parameters ToolParameters `json:"parameters"`

	// Metadata provides additional information for UI and orchestration.
	Metadata ToolMetadata `json:"metadata"`
}

// ToolParameters defines the input schema for a tool.
// This follows JSON Schema format for OpenAI compatibility.
type ToolParameters struct {
	// Type is always "object" for function parameters.
	Type string `json:"type"`

	// Properties defines each parameter.
	Properties map[string]ParameterSchema `json:"properties"`

	// Required lists parameter names that must be provided.
	Required []string `json:"required,omitempty"`

	// AdditionalProperties controls whether extra properties are allowed.
	AdditionalProperties bool `json:"additionalProperties,omitempty"`
}

// ParameterSchema defines a single parameter's schema.
type ParameterSchema struct {
	// Type is the JSON Schema type (string, number, integer, boolean, array, object).
	Type string `json:"type"`

	// Description explains the parameter's purpose.
	Description string `json:"description,omitempty"`

	// Enum restricts values to a specific set.
	Enum []string `json:"enum,omitempty"`

	// Default provides a default value.
	Default interface{} `json:"default,omitempty"`

	// Items defines array element schema (for type: array).
	Items *ParameterSchema `json:"items,omitempty"`

	// Properties defines nested properties (for type: object).
	Properties map[string]ParameterSchema `json:"properties,omitempty"`

	// Minimum/Maximum for numeric constraints.
	Minimum *float64 `json:"minimum,omitempty"`
	Maximum *float64 `json:"maximum,omitempty"`

	// MinLength/MaxLength for string constraints.
	MinLength *int `json:"minLength,omitempty"`
	MaxLength *int `json:"maxLength,omitempty"`

	// Pattern for regex validation.
	Pattern string `json:"pattern,omitempty"`

	// Format for semantic validation (e.g., "uri", "email", "uuid").
	Format string `json:"format,omitempty"`
}

// ToolMetadata provides additional information for orchestration and UI.
type ToolMetadata struct {
	// EnabledByDefault indicates if this tool should be active by default.
	EnabledByDefault bool `json:"enabled_by_default"`

	// RequiresApproval indicates if human approval is needed before execution.
	RequiresApproval bool `json:"requires_approval"`

	// TimeoutSeconds is the default timeout for this tool.
	TimeoutSeconds int `json:"timeout_seconds,omitempty"`

	// RateLimitPerMinute limits how often this tool can be called.
	RateLimitPerMinute int `json:"rate_limit_per_minute,omitempty"`

	// CostEstimate describes the relative cost (low, medium, high, variable).
	CostEstimate string `json:"cost_estimate,omitempty"`

	// Tags are additional labels for filtering/categorization.
	Tags []string `json:"tags,omitempty"`

	// LongRunning indicates this tool may take a long time to complete.
	LongRunning bool `json:"long_running,omitempty"`

	// Idempotent indicates the tool can be safely retried.
	Idempotent bool `json:"idempotent,omitempty"`

	// Examples provides usage examples for documentation.
	Examples []ToolExample `json:"examples,omitempty"`
}

// ToolExample provides a usage example for documentation.
type ToolExample struct {
	// Description explains what this example demonstrates.
	Description string `json:"description"`

	// Input is example parameter values.
	Input map[string]interface{} `json:"input"`

	// Output is example response (optional).
	Output interface{} `json:"output,omitempty"`
}

// ToolCategory provides metadata for grouping tools.
type ToolCategory struct {
	// ID is the category identifier used in ToolDefinition.Category.
	ID string `json:"id"`

	// Name is the human-readable category name.
	Name string `json:"name"`

	// Description explains what tools in this category do.
	Description string `json:"description,omitempty"`

	// Icon is an optional icon identifier for UI (e.g., "code", "search").
	Icon string `json:"icon,omitempty"`
}

// -----------------------------------------------------------------------------
// Tool Result Types (for standardized responses)
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
