// Package integrations provides clients for external services.
// This file contains type definitions, interfaces, and constants for the
// agent-manager integration.
package integrations

import (
	"context"
	"mime/multipart"
	"time"
)

// UploadResponse contains the server response after uploading an attachment.
type UploadResponse struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
	StoragePath string `json:"storage_path"`
	URL         string `json:"url"`
}

// AgentManagerClientInterface defines the operations used by agent mode handlers.
// This interface enables dependency injection for testing.
type AgentManagerClientInterface interface {
	StartAgentChat(ctx context.Context, message string, cfg AgentChatConfig) (*AgentChatSession, error)
	ContinueChat(ctx context.Context, runID, message string, attachmentIDs []string) error
	GetEvents(ctx context.Context, runID string, afterSequence int64) ([]*TranslatedEvent, error)
	GetRunStatus(ctx context.Context, runID string) (*AgentRunStatus, error)
	StopRun(ctx context.Context, runID string) error
	ListRuns(ctx context.Context, opts ListRunsOptions) (*ListRunsResult, error)
	UploadAttachment(ctx context.Context, file multipart.File, header *multipart.FileHeader) (*UploadResponse, error)
}

// RunnerType identifies which agent runner to use.
type RunnerType string

const (
	RunnerTypeClaudeCode RunnerType = "claude-code"
	RunnerTypeCodex      RunnerType = "codex"
	RunnerTypeOpenCode   RunnerType = "opencode"
)

// AgentChatConfig contains configuration for starting an agent chat.
type AgentChatConfig struct {
	// RunnerType specifies which runner to use (claude-code, codex, opencode)
	RunnerType RunnerType `json:"runner_type"`

	// ProjectPath is the directory where the agent will operate
	ProjectPath string `json:"project_path"`

	// Model optionally specifies the model to use (e.g., "claude-opus-4")
	Model string `json:"model,omitempty"`

	// MaxTurns optionally limits the number of conversation turns
	MaxTurns int `json:"max_turns,omitempty"`
}

// AgentChatSession contains the session info returned after starting agent mode.
type AgentChatSession struct {
	TaskID    string `json:"task_id"`
	RunID     string `json:"run_id"`
	SessionID string `json:"session_id,omitempty"` // Runner-specific session ID for continuation
}

// RunStatus represents the current state of a run.
type RunStatus string

const (
	RunStatusPending     RunStatus = "pending"
	RunStatusStarting    RunStatus = "starting"
	RunStatusRunning     RunStatus = "running"
	RunStatusNeedsReview RunStatus = "needs_review"
	RunStatusComplete    RunStatus = "complete"
	RunStatusFailed      RunStatus = "failed"
	RunStatusCancelled   RunStatus = "cancelled"
)

// IsActiveRunStatus returns true if the status indicates the run is still in progress.
func IsActiveRunStatus(status string) bool {
	switch RunStatus(status) {
	case RunStatusPending, RunStatusStarting, RunStatusRunning, RunStatusNeedsReview:
		return true
	default:
		return false
	}
}

// IsTerminalRunStatus returns true if the status indicates the run has finished.
func IsTerminalRunStatus(status string) bool {
	return !IsActiveRunStatus(status) && status != ""
}

// ListRunsOptions contains filter/pagination parameters for listing runs.
type ListRunsOptions struct {
	Status    string `json:"status,omitempty"`
	TagPrefix string `json:"tag_prefix,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
}

// AgentRunSummary is a lightweight representation of a run for list views.
type AgentRunSummary struct {
	RunID           string    `json:"run_id"`
	TaskID          string    `json:"task_id"`
	Tag             string    `json:"tag,omitempty"`
	Status          RunStatus `json:"status"`
	Phase           string    `json:"phase,omitempty"`
	ProgressPercent int       `json:"progress_percent"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ListRunsResult contains paginated run listing results.
type ListRunsResult struct {
	Runs    []AgentRunSummary `json:"runs"`
	Total   int               `json:"total"`
	HasMore bool              `json:"has_more"`
}

// AgentRunStatus contains the status info for a run.
type AgentRunStatus struct {
	RunID           string    `json:"run_id"`
	Status          RunStatus `json:"status"`
	Phase           string    `json:"phase"`
	ProgressPercent int       `json:"progress_percent"`
	SessionID       string    `json:"session_id,omitempty"`
	ErrorMsg        string    `json:"error_msg,omitempty"`
}

// TranslatedEvent represents an agent-manager event translated for inbox rendering.
type TranslatedEvent struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // message, tool_call, tool_result, status, error, log, metric, artifact, message_deleted, etc.
	Role      string    `json:"role"` // user, assistant, system, tool
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	Sequence  int64     `json:"sequence"`

	// Tool fields (for tool_call and tool_result types)
	ToolName    string `json:"tool_name,omitempty"`
	ToolCallID  string `json:"tool_call_id,omitempty"` // Correlation ID linking tool_call <-> tool_result
	ToolInput   string `json:"tool_input,omitempty"`   // JSON string of tool input
	ToolOutput  string `json:"tool_output,omitempty"`  // Tool result output
	ToolSuccess bool   `json:"tool_success,omitempty"`

	// Status fields (for status type)
	RunStatus string `json:"run_status,omitempty"`
	Phase     string `json:"phase,omitempty"`
	Progress  int    `json:"progress,omitempty"`

	// Compaction fields (for compaction type)
	CompactionTrigger           string `json:"compaction_trigger,omitempty"`
	CompactionFocus             string `json:"compaction_focus,omitempty"`
	CompactionMessagesCompacted int64  `json:"compaction_messages_compacted,omitempty"`
	CompactionTokensBefore      int64  `json:"compaction_tokens_before,omitempty"`
	CompactionTokensAfter       int64  `json:"compaction_tokens_after,omitempty"`
	CompactionOriginalCommand   string `json:"compaction_original_command,omitempty"`

	// RawData holds the original event data as JSON for event types that
	// don't have dedicated fields. The UI can display this in a generic card.
	RawData string `json:"raw_data,omitempty"`
}
