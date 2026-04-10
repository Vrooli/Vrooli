// Package domain defines the core domain types for the Agent Inbox scenario.
//
// Protocol types (ToolManifest, ToolDefinition, etc.) are defined in the proto
// schema at packages/proto/schemas/agent-inbox/v1/ and imported as toolspb.
//
// This file defines agent-inbox specific types for tool configuration and
// aggregation that are not part of the protocol.
package domain

import (
	"time"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// -----------------------------------------------------------------------------
// Tool Configuration Types (agent-inbox specific)
// -----------------------------------------------------------------------------

// ApprovalOverride represents the three-state approval setting.
// Users can override the tool's default approval requirement.
type ApprovalOverride string

const (
	// ApprovalDefault uses the tool's metadata.requires_approval setting.
	ApprovalDefault ApprovalOverride = ""
	// ApprovalRequire always requires approval before execution.
	ApprovalRequire ApprovalOverride = "require"
	// ApprovalSkip never requires approval (auto-execute).
	ApprovalSkip ApprovalOverride = "skip"
)

// ToolConfiguration represents user preferences for a tool.
// This can be global (ChatID empty) or per-chat.
type ToolConfiguration struct {
	ID               string           `json:"id"`
	ChatID           string           `json:"chat_id,omitempty"` // Empty for global configuration
	Scenario         string           `json:"scenario"`          // e.g., "agent-manager"
	ToolName         string           `json:"tool_name"`
	Enabled          bool             `json:"enabled"`
	ApprovalOverride ApprovalOverride `json:"approval_override,omitempty"` // Three-state: "", "require", "skip"
	CreatedAt        time.Time        `json:"created_at"`
	UpdatedAt        time.Time        `json:"updated_at"`
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

	// Tool is the proto tool definition.
	Tool *toolspb.ToolDefinition `json:"tool"`

	// Enabled indicates if this tool is currently enabled.
	Enabled bool `json:"enabled"`

	// Source indicates where the enabled state came from.
	Source ToolConfigurationScope `json:"source"`

	// RequiresApproval is the effective approval requirement.
	// This considers: user override > global override > tool metadata default.
	RequiresApproval bool `json:"requires_approval"`

	// ApprovalSource indicates where the approval setting came from.
	// Empty string means using tool's default metadata.
	ApprovalSource ToolConfigurationScope `json:"approval_source,omitempty"`

	// ApprovalOverride is the user's configured approval override.
	// Empty string means using tool's default, "require" forces approval,
	// "skip" bypasses approval.
	ApprovalOverride ApprovalOverride `json:"approval_override,omitempty"`
}

// ToolSet represents a collection of tools from multiple scenarios.
// This is the aggregated view of all available tools.
type ToolSet struct {
	// Scenarios lists all scenarios that contributed tools.
	Scenarios []*toolspb.ScenarioInfo `json:"scenarios"`

	// Tools is the flat list of all effective tools.
	Tools []EffectiveTool `json:"tools"`

	// Categories aggregates all categories from all scenarios.
	Categories []*toolspb.ToolCategory `json:"categories"`

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

// DiscoveryResult represents the result of a tool discovery sync operation.
// Returned by the SyncTools endpoint to inform the UI about what changed.
type DiscoveryResult struct {
	// ScenariosWithTools is the number of scenarios that have tool endpoints.
	ScenariosWithTools int `json:"scenarios_with_tools"`

	// NewScenarios lists scenarios that were newly discovered (not in previous cache).
	NewScenarios []string `json:"new_scenarios"`

	// RemovedScenarios lists scenarios that were in previous cache but not found now.
	RemovedScenarios []string `json:"removed_scenarios"`

	// TotalTools is the total number of tools across all discovered scenarios.
	TotalTools int `json:"total_tools"`
}

// -----------------------------------------------------------------------------
// Helper Methods for EffectiveTool
// -----------------------------------------------------------------------------

// GetName returns the tool name, handling nil cases.
func (e *EffectiveTool) GetName() string {
	if e.Tool == nil {
		return ""
	}
	return e.Tool.Name
}

// GetDescription returns the tool description, handling nil cases.
func (e *EffectiveTool) GetDescription() string {
	if e.Tool == nil {
		return ""
	}
	return e.Tool.Description
}

// GetMetadata returns the tool metadata, handling nil cases.
func (e *EffectiveTool) GetMetadata() *toolspb.ToolMetadata {
	if e.Tool == nil {
		return nil
	}
	return e.Tool.Metadata
}

// IsLongRunning returns true if the tool is marked as long-running.
func (e *EffectiveTool) IsLongRunning() bool {
	meta := e.GetMetadata()
	if meta == nil {
		return false
	}
	return meta.LongRunning
}

// HasAsyncBehavior returns true if the tool has async behavior configured.
func (e *EffectiveTool) HasAsyncBehavior() bool {
	meta := e.GetMetadata()
	if meta == nil {
		return false
	}
	return meta.AsyncBehavior != nil
}

// GetAsyncBehavior returns the async behavior config, or nil if not set.
func (e *EffectiveTool) GetAsyncBehavior() *toolspb.AsyncBehavior {
	meta := e.GetMetadata()
	if meta == nil {
		return nil
	}
	return meta.AsyncBehavior
}
