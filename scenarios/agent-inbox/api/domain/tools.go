// Package domain defines the core domain types for the Agent Inbox scenario.
//
// This file defines the Tool Discovery Protocol types for consuming tools from
// external scenarios. These types enable agent-inbox to discover and configure
// tools dynamically at runtime.
//
// DESIGN PRINCIPLES:
// - Compatible with the Tool Discovery Protocol v1.0
// - Supports both global and per-chat tool configurations
// - Enables UI generation through rich metadata
package domain

import "time"

// ToolProtocolVersion is the version of the Tool Discovery Protocol we support.
const ToolProtocolVersion = "1.0"

// -----------------------------------------------------------------------------
// Tool Discovery Protocol Types (consumed from external scenarios)
// -----------------------------------------------------------------------------

// ToolManifest is the response from GET /api/v1/tools on a scenario.
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

// ScenarioInfo provides metadata about a scenario exposing tools.
type ScenarioInfo struct {
	// Name is the scenario identifier (e.g., "agent-manager").
	Name string `json:"name"`

	// Version is the scenario version (e.g., "1.0.0").
	Version string `json:"version"`

	// Description is a human-readable description of the scenario.
	Description string `json:"description"`

	// BaseURL is the API base URL for this scenario.
	BaseURL string `json:"base_url,omitempty"`
}

// ToolDefinition describes a single tool that can be invoked.
// The structure is compatible with OpenAI's function-calling format.
type ToolDefinition struct {
	// Name is the unique identifier for this tool within the scenario.
	Name string `json:"name"`

	// Description explains what the tool does. This is shown to the LLM.
	Description string `json:"description"`

	// Category groups related tools together.
	Category string `json:"category,omitempty"`

	// Parameters defines the input schema in JSON Schema format.
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
}

// ToolCategory provides metadata for grouping tools.
type ToolCategory struct {
	// ID is the category identifier used in ToolDefinition.Category.
	ID string `json:"id"`

	// Name is the human-readable category name.
	Name string `json:"name"`

	// Description explains what tools in this category do.
	Description string `json:"description,omitempty"`

	// Icon is an optional icon identifier for UI.
	Icon string `json:"icon,omitempty"`
}

// -----------------------------------------------------------------------------
// Tool Configuration Types (agent-inbox specific)
// -----------------------------------------------------------------------------

// ToolConfiguration represents user preferences for a tool.
// This can be global (ChatID empty) or per-chat.
type ToolConfiguration struct {
	ID         string    `json:"id"`
	ChatID     string    `json:"chat_id,omitempty"` // Empty for global configuration
	Scenario   string    `json:"scenario"`          // e.g., "agent-manager"
	ToolName   string    `json:"tool_name"`
	Enabled    bool      `json:"enabled"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// ToolConfigurationScope defines the scope of a tool configuration.
type ToolConfigurationScope string

const (
	// ScopeGlobal applies to all chats by default.
	ScopeGlobal ToolConfigurationScope = "global"
	// ScopeChat applies to a specific chat only.
	ScopeChat ToolConfigurationScope = "chat"
)

// EffectiveTool represents a tool with its effective enabled state.
// This merges the tool definition with user configuration.
type EffectiveTool struct {
	// Scenario is the source scenario name.
	Scenario string `json:"scenario"`

	// Tool is the tool definition.
	Tool ToolDefinition `json:"tool"`

	// Enabled indicates if this tool is currently enabled.
	Enabled bool `json:"enabled"`

	// Source indicates where the enabled state came from.
	Source ToolConfigurationScope `json:"source"`
}

// ToolSet represents a collection of tools from multiple scenarios.
// This is the aggregated view of all available tools.
type ToolSet struct {
	// Scenarios lists all scenarios that contributed tools.
	Scenarios []ScenarioInfo `json:"scenarios"`

	// Tools is the flat list of all effective tools.
	Tools []EffectiveTool `json:"tools"`

	// Categories aggregates all categories from all scenarios.
	Categories []ToolCategory `json:"categories"`

	// GeneratedAt is when this tool set was assembled.
	GeneratedAt time.Time `json:"generated_at"`
}

// ScenarioStatus represents the health status of a scenario's tool endpoint.
type ScenarioStatus struct {
	// Scenario is the scenario name.
	Scenario string `json:"scenario"`

	// Available indicates if the scenario's tool endpoint is reachable.
	Available bool `json:"available"`

	// LastChecked is when availability was last verified.
	LastChecked time.Time `json:"last_checked"`

	// ToolCount is the number of tools provided (when available).
	ToolCount int `json:"tool_count,omitempty"`

	// Error contains the error message if not available.
	Error string `json:"error,omitempty"`
}

// -----------------------------------------------------------------------------
// OpenAI Function Format Conversion
// -----------------------------------------------------------------------------

// ToOpenAIFunction converts a ToolDefinition to OpenAI function format.
// This is used when sending tools to the LLM via OpenRouter.
func (t *ToolDefinition) ToOpenAIFunction() map[string]interface{} {
	return map[string]interface{}{
		"type": "function",
		"function": map[string]interface{}{
			"name":        t.Name,
			"description": t.Description,
			"parameters":  t.Parameters.ToMap(),
		},
	}
}

// ToMap converts ToolParameters to a generic map for JSON encoding.
func (p *ToolParameters) ToMap() map[string]interface{} {
	props := make(map[string]interface{})
	for name, schema := range p.Properties {
		props[name] = schema.ToMap()
	}

	result := map[string]interface{}{
		"type":       p.Type,
		"properties": props,
	}

	if len(p.Required) > 0 {
		result["required"] = p.Required
	}

	return result
}

// ToMap converts ParameterSchema to a generic map for JSON encoding.
func (s *ParameterSchema) ToMap() map[string]interface{} {
	result := map[string]interface{}{
		"type": s.Type,
	}

	if s.Description != "" {
		result["description"] = s.Description
	}
	if len(s.Enum) > 0 {
		result["enum"] = s.Enum
	}
	if s.Default != nil {
		result["default"] = s.Default
	}
	if s.Format != "" {
		result["format"] = s.Format
	}
	if s.Items != nil {
		result["items"] = s.Items.ToMap()
	}
	if len(s.Properties) > 0 {
		props := make(map[string]interface{})
		for name, prop := range s.Properties {
			props[name] = prop.ToMap()
		}
		result["properties"] = props
	}

	return result
}
