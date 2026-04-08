package main

import (
	"encoding/json"
	"strings"
)

// ============================================================================
// Enum normalization
// ============================================================================

// normalizeEnum strips a proto enum prefix and lowercases the remainder.
// e.g. "RUN_STATUS_RUNNING" with prefix "RUN_STATUS_" → "running".
func normalizeEnum(protoValue, prefix string) string {
	if protoValue == "" {
		return ""
	}
	stripped := protoValue
	if prefix != "" {
		stripped = strings.TrimPrefix(protoValue, prefix)
	}
	return strings.ToLower(stripped)
}

// normalizeEventType handles the special case where proto "RUN_EVENT_TYPE_STATUS"
// maps to "status_change" (the UI convention).
func normalizeEventType(protoValue string) string {
	const prefix = "RUN_EVENT_TYPE_"
	if protoValue == "RUN_EVENT_TYPE_STATUS" {
		return "status_change"
	}
	norm := normalizeEnum(protoValue, prefix)
	if norm != "" {
		return norm
	}
	// Already lowercase / no prefix — pass through.
	return protoValue
}

// runnerTypeToString converts proto runner-type strings to UI-friendly form.
func runnerTypeToString(wire string) string {
	switch wire {
	case "RUNNER_TYPE_CLAUDE_CODE":
		return "claude-code"
	case "RUNNER_TYPE_CUSTOM":
		return "custom"
	default:
		if wire == "" {
			return ""
		}
		return strings.ToLower(strings.TrimPrefix(wire, "RUNNER_TYPE_"))
	}
}

// ============================================================================
// Wire types — match proto-JSON (snake_case, enum strings, wrappers)
// ============================================================================

type wireRunSummary struct {
	FilesModified []string `json:"files_modified,omitempty"`
	FilesCreated  []string `json:"files_created,omitempty"`
	FilesDeleted  []string `json:"files_deleted,omitempty"`
	TokensUsed    int      `json:"tokens_used,omitempty"`
	TurnsUsed     int      `json:"turns_used,omitempty"`
	CostEstimate  float64  `json:"cost_estimate,omitempty"`
}

type wireRunActions struct {
	CanInvestigate               bool `json:"can_investigate,omitempty"`
	CanApplyInvestigation        bool `json:"can_apply_investigation,omitempty"`
	CanDelete                    bool `json:"can_delete,omitempty"`
	CanStop                      bool `json:"can_stop,omitempty"`
	CanRetry                     bool `json:"can_retry,omitempty"`
	CanContinue                  bool `json:"can_continue,omitempty"`
	CanApprove                   bool `json:"can_approve,omitempty"`
	CanReject                    bool `json:"can_reject,omitempty"`
	CanReview                    bool `json:"can_review,omitempty"`
	CanExtractRecommendations    bool `json:"can_extract_recommendations,omitempty"`
	CanRegenerateRecommendations bool `json:"can_regenerate_recommendations,omitempty"`
}

type wireRun struct {
	ID              string          `json:"id"`
	TaskID          string          `json:"task_id,omitempty"`
	Status          string          `json:"status,omitempty"`
	Phase           string          `json:"phase,omitempty"`
	ProgressPercent int             `json:"progress_percent,omitempty"`
	ErrorMsg        string          `json:"error_msg,omitempty"`
	Summary         *wireRunSummary `json:"summary,omitempty"`
	Actions         *wireRunActions `json:"actions,omitempty"`
	SessionID       string          `json:"session_id,omitempty"`
	ApprovalState   string          `json:"approval_state,omitempty"`
	PromptPreview   string          `json:"prompt_preview,omitempty"`
	SandboxID       string          `json:"sandbox_id,omitempty"`
	StartedAt       string          `json:"started_at,omitempty"`
	EndedAt         string          `json:"ended_at,omitempty"`
	CreatedAt       string          `json:"created_at,omitempty"`
	UpdatedAt       string          `json:"updated_at,omitempty"`
}

// Wire event oneof data structs.

type wireMessageData struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

type wireToolCallData struct {
	ToolName string          `json:"tool_name,omitempty"`
	Input    json.RawMessage `json:"input,omitempty"`
}

type wireToolResultData struct {
	ToolName string `json:"tool_name,omitempty"`
	Output   string `json:"output,omitempty"`
}

type wireErrorData struct {
	Message string `json:"message,omitempty"`
	Code    string `json:"code,omitempty"`
}

type wireStatusData struct {
	OldStatus string `json:"old_status,omitempty"`
	NewStatus string `json:"new_status,omitempty"`
}

type wireLogData struct {
	Message string `json:"message,omitempty"`
	Level   string `json:"level,omitempty"`
}

type wireProgressData struct {
	Percent int    `json:"percent,omitempty"`
	Message string `json:"message,omitempty"`
}

type wireRunEvent struct {
	ID         string              `json:"id"`
	RunID      string              `json:"run_id,omitempty"`
	EventType  string              `json:"event_type,omitempty"`
	Timestamp  string              `json:"timestamp,omitempty"`
	Sequence   int64               `json:"sequence,omitempty,string"`
	Message    *wireMessageData    `json:"message,omitempty"`
	ToolCall   *wireToolCallData   `json:"tool_call,omitempty"`
	ToolResult *wireToolResultData `json:"tool_result,omitempty"`
	Error      *wireErrorData      `json:"error,omitempty"`
	Status     *wireStatusData     `json:"status,omitempty"`
	Log        *wireLogData        `json:"log,omitempty"`
	Progress   *wireProgressData   `json:"progress,omitempty"`
}

type wireFileDiff struct {
	Path       string `json:"path"`
	ChangeType string `json:"change_type,omitempty"`
	Additions  int    `json:"additions,omitempty"`
	Deletions  int    `json:"deletions,omitempty"`
	IsBinary   bool   `json:"is_binary,omitempty"`
	Patch      string `json:"patch,omitempty"`
}

type wireRunDiff struct {
	RunID       string         `json:"run_id,omitempty"`
	Content     string         `json:"content,omitempty"`
	Files       []wireFileDiff `json:"files,omitempty"`
	GeneratedAt string         `json:"generated_at,omitempty"`
}

type wireAgentProfile struct {
	ID          string `json:"id"`
	Name        string `json:"name,omitempty"`
	ProfileKey  string `json:"profile_key,omitempty"`
	Description string `json:"description,omitempty"`
	RunnerType  string `json:"runner_type,omitempty"`
	Model       string `json:"model,omitempty"`
	MaxTurns    int    `json:"max_turns,omitempty"`
}

// Wire response wrappers.

type wireGetRunResponse struct {
	Run wireRun `json:"run"`
}

type wireListRunsResponse struct {
	Runs  []wireRun `json:"runs"`
	Total int       `json:"total,omitempty"`
}

type wireGetRunEventsResponse struct {
	Events []wireRunEvent `json:"events"`
}

type wireGetRunDiffResponse struct {
	Diff *wireRunDiff `json:"diff,omitempty"`
}

type wireContinueRunResponse struct {
	Success   bool     `json:"success,omitempty"`
	Run       *wireRun `json:"run,omitempty"`
	Error     string   `json:"error,omitempty"`
	ErrorCode string   `json:"error_code,omitempty"`
}

type wireApproveResult struct {
	Success      bool   `json:"success,omitempty"`
	FilesApplied int    `json:"files_applied,omitempty"`
	CommitHash   string `json:"commit_hash,omitempty"`
	Message      string `json:"message,omitempty"`
}

type wireApproveRunResponse struct {
	Result *wireApproveResult `json:"result,omitempty"`
}

type wireRejectRunResponse struct {
	Status string `json:"status,omitempty"`
}

type wireStopRunResponse struct {
	Status string `json:"status,omitempty"`
}

type wireListProfilesResponse struct {
	Profiles []wireAgentProfile `json:"profiles"`
	Total    int                `json:"total,omitempty"`
}

type wireEnsureProfileResponse struct {
	Profile *wireAgentProfile `json:"profile,omitempty"`
	Created bool              `json:"created,omitempty"`
	Updated bool              `json:"updated,omitempty"`
}

// Wire request types (outbound, snake_case).

type agentContextAttachment struct {
	Type         string `json:"type"`
	AttachmentID string `json:"attachment_id"`
	Label        string `json:"label,omitempty"`
}

type agentTaskData struct {
	Title              string                   `json:"title"`
	Description        string                   `json:"description,omitempty"`
	ScopePath          string                   `json:"scope_path,omitempty"`
	ContextAttachments []agentContextAttachment `json:"context_attachments,omitempty"`
}

type agentTaskCreateRequest struct {
	Task agentTaskData `json:"task"`
}

type agentTaskCreateResponse struct {
	Task struct {
		ID string `json:"id"`
	} `json:"task"`
}

type agentProfileRef struct {
	ProfileKey string `json:"profile_key"`
}

type agentRunCreateInternalRequest struct {
	TaskID         string           `json:"task_id"`
	RunMode        int              `json:"run_mode,omitempty"`
	AgentProfileID string           `json:"agent_profile_id,omitempty"`
	ProfileRef     *agentProfileRef `json:"profile_ref,omitempty"`
	Tag            string           `json:"tag,omitempty"`
}

type agentRunCreateInternalResponse struct {
	Run struct {
		ID string `json:"id"`
	} `json:"run"`
}

// API types and conversion functions are in agent_manager_model_api.go
// and agent_manager_model_convert.go respectively.
