// Package domain defines the core domain types for the Agent Inbox scenario.
// These types represent the fundamental entities in the chat management domain.
package domain

import (
	"time"
)

// Chat represents a conversation in the inbox.
// A chat contains a sequence of messages between users and AI assistants,
// and can be organized with labels, starred, or archived.
type Chat struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Preview      string    `json:"preview"`
	Model        string    `json:"model"`
	ViewMode     string    `json:"view_mode"` // "bubble" (default)
	IsRead       bool      `json:"is_read"`
	IsArchived   bool      `json:"is_archived"`
	IsStarred    bool      `json:"is_starred"`
	LabelIDs     []string  `json:"label_ids"`
	SystemPrompt string    `json:"system_prompt"`
	ToolsEnabled bool      `json:"tools_enabled"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// Message represents a single message in a chat.
// Messages can come from users, assistants, the system, or as tool responses.
type Message struct {
	ID           string     `json:"id"`
	ChatID       string     `json:"chat_id"`
	Role         string     `json:"role"` // "user", "assistant", "system", "tool"
	Content      string     `json:"content"`
	Model        string     `json:"model,omitempty"`
	TokenCount   int        `json:"token_count,omitempty"`
	ToolCallID   string     `json:"tool_call_id,omitempty"`  // For tool response messages
	ToolCalls    []ToolCall `json:"tool_calls,omitempty"`    // Tool calls requested by assistant
	ResponseID   string     `json:"response_id,omitempty"`   // OpenRouter response ID for tracking
	FinishReason string     `json:"finish_reason,omitempty"` // "stop", "tool_calls", etc.
	CreatedAt    time.Time  `json:"created_at"`
}

// ToolCall represents a tool invocation requested by the assistant.
// When an AI model decides to use a tool, it generates a ToolCall with
// the function name and arguments.
type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"` // "function"
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"` // JSON string of arguments
	} `json:"function"`
}

// ToolCallRecord stores tool call execution details in the database.
// This tracks the lifecycle of a tool call from start to completion,
// including the result or any errors.
type ToolCallRecord struct {
	ID            string    `json:"id"`
	MessageID     string    `json:"message_id"`
	ChatID        string    `json:"chat_id"`
	ToolName      string    `json:"tool_name"`
	Arguments     string    `json:"arguments"`       // JSON string
	Result        string    `json:"result"`          // JSON string or text output
	Status        string    `json:"status"`          // "pending", "running", "completed", "failed"
	ScenarioName  string    `json:"scenario_name"`   // e.g., "agent-manager", "app-issue-tracker"
	ExternalRunID string    `json:"external_run_id"` // ID in the external scenario
	StartedAt     time.Time `json:"started_at"`
	CompletedAt   time.Time `json:"completed_at,omitempty"`
	ErrorMessage  string    `json:"error_message,omitempty"`
}

// Label represents a colored label for organizing chats.
// Users can create labels and assign them to chats for organization.
type Label struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// ChatWithMessages combines a chat with its message history.
// This is used when fetching a complete conversation.
type ChatWithMessages struct {
	Chat     Chat      `json:"chat"`
	Messages []Message `json:"messages"`
}

// ViewMode constants for chat display
const (
	ViewModeBubble = "bubble"
)

// Message role constants
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
	RoleTool      = "tool"
)

// ToolCall status constants
const (
	StatusPending   = "pending"
	StatusRunning   = "running"
	StatusCompleted = "completed"
	StatusFailed    = "failed"
	StatusCancelled = "cancelled"
)

// Preview constants
const (
	// PreviewMaxLength is the maximum length for chat preview text.
	PreviewMaxLength = 100
)

// UsageRecord tracks token usage and cost for a single API call.
// This enables usage analytics, cost tracking, and billing.
type UsageRecord struct {
	ID               string    `json:"id"`
	ChatID           string    `json:"chat_id"`
	MessageID        string    `json:"message_id,omitempty"`
	Model            string    `json:"model"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	PromptCost       float64   `json:"prompt_cost"`       // Cost in USD cents
	CompletionCost   float64   `json:"completion_cost"`   // Cost in USD cents
	TotalCost        float64   `json:"total_cost"`        // Cost in USD cents
	CreatedAt        time.Time `json:"created_at"`
}

// UsageStats provides aggregated usage statistics.
type UsageStats struct {
	TotalPromptTokens     int                     `json:"total_prompt_tokens"`
	TotalCompletionTokens int                     `json:"total_completion_tokens"`
	TotalTokens           int                     `json:"total_tokens"`
	TotalCost             float64                 `json:"total_cost"` // In USD cents
	ByModel               map[string]*ModelUsage  `json:"by_model"`
	ByDay                 map[string]*DailyUsage  `json:"by_day,omitempty"`
}

// ModelUsage provides usage breakdown for a single model.
type ModelUsage struct {
	Model            string  `json:"model"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	TotalCost        float64 `json:"total_cost"`
	RequestCount     int     `json:"request_count"`
}

// DailyUsage provides usage breakdown for a single day.
type DailyUsage struct {
	Date             string  `json:"date"` // YYYY-MM-DD
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens      int     `json:"total_tokens"`
	TotalCost        float64 `json:"total_cost"`
	RequestCount     int     `json:"request_count"`
}

// ValidViewModes returns the list of valid view modes
func ValidViewModes() []string {
	return []string{ViewModeBubble}
}

// IsValidViewMode checks if a view mode string is valid
func IsValidViewMode(mode string) bool {
	return mode == ViewModeBubble
}

// ValidRoles returns the list of valid message roles
func ValidRoles() []string {
	return []string{RoleUser, RoleAssistant, RoleSystem, RoleTool}
}

// IsValidRole checks if a role string is valid
func IsValidRole(role string) bool {
	for _, r := range ValidRoles() {
		if r == role {
			return true
		}
	}
	return false
}

// TruncatePreview truncates text to PreviewMaxLength with ellipsis.
// This is the single source of truth for preview text truncation.
func TruncatePreview(text string) string {
	if len(text) <= PreviewMaxLength {
		return text
	}
	return text[:PreviewMaxLength] + "..."
}
