package main

import "strings"

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
	StartedAt       string          `json:"started_at,omitempty"`
	EndedAt         string          `json:"ended_at,omitempty"`
	CreatedAt       string          `json:"created_at,omitempty"`
	UpdatedAt       string          `json:"updated_at,omitempty"`
}

// Wire event oneof data structs.

type wireMessageData struct {
	Content string `json:"content,omitempty"`
}

type wireToolCallData struct {
	ToolName string `json:"tool_name,omitempty"`
	Input    string `json:"input,omitempty"`
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

type agentTaskData struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	ScopePath   string `json:"scope_path,omitempty"`
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

// ============================================================================
// API types — UI-facing (camelCase JSON tags)
// ============================================================================

// AgentProfile represents an agent configuration profile.
type AgentProfile struct {
	ID          string `json:"id"`
	Key         string `json:"key,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Model       string `json:"model,omitempty"`
	RunnerType  string `json:"runnerType,omitempty"`
}

// AgentProfileListResponse wraps a list of profiles.
type AgentProfileListResponse struct {
	Profiles []AgentProfile `json:"profiles"`
	Total    int            `json:"total"`
}

// AgentRunRequest is the composite request sent from the UI to create a Task + Run.
type AgentRunRequest struct {
	ScenarioSlug string `json:"scenarioSlug"`
	Prompt       string `json:"prompt"`
	ProfileID    string `json:"profileId,omitempty"`
	ProfileKey   string `json:"profileKey,omitempty"`
}

// AgentRunCreateResponse is returned after creating a Task + Run.
type AgentRunCreateResponse struct {
	RunID  string `json:"runId"`
	TaskID string `json:"taskId"`
}

// AgentRunSummary describes what an agent run accomplished.
type AgentRunSummary struct {
	FilesModified []string `json:"filesModified,omitempty"`
	FilesCreated  []string `json:"filesCreated,omitempty"`
	FilesDeleted  []string `json:"filesDeleted,omitempty"`
	TokensUsed    int      `json:"tokensUsed,omitempty"`
	TurnsUsed     int      `json:"turnsUsed,omitempty"`
	CostEstimate  float64  `json:"costEstimate,omitempty"`
}

// AgentRunActions describes what actions are available for a run.
type AgentRunActions struct {
	CanInvestigate               bool `json:"canInvestigate,omitempty"`
	CanApplyInvestigation        bool `json:"canApplyInvestigation,omitempty"`
	CanDelete                    bool `json:"canDelete,omitempty"`
	CanStop                      bool `json:"canStop"`
	CanRetry                     bool `json:"canRetry"`
	CanContinue                  bool `json:"canContinue"`
	CanApprove                   bool `json:"canApprove"`
	CanReject                    bool `json:"canReject"`
	CanReview                    bool `json:"canReview,omitempty"`
	CanExtractRecommendations    bool `json:"canExtractRecommendations,omitempty"`
	CanRegenerateRecommendations bool `json:"canRegenerateRecommendations,omitempty"`
}

// AgentRun represents the state of an agent execution.
type AgentRun struct {
	ID              string           `json:"id"`
	TaskID          string           `json:"taskId,omitempty"`
	SessionID       string           `json:"sessionId,omitempty"`
	Status          string           `json:"status"`
	Phase           string           `json:"phase,omitempty"`
	ProgressPercent int              `json:"progressPercent,omitempty"`
	ErrorMsg        string           `json:"errorMsg,omitempty"`
	ApprovalState   string           `json:"approvalState,omitempty"`
	Summary         *AgentRunSummary `json:"summary,omitempty"`
	Actions         *AgentRunActions `json:"actions,omitempty"`
	CreatedAt       string           `json:"createdAt"`
	StartedAt       string           `json:"startedAt,omitempty"`
	EndedAt         string           `json:"endedAt,omitempty"`
}

// AgentRunListResponse wraps a list of runs.
type AgentRunListResponse struct {
	Runs  []AgentRun `json:"runs"`
	Total int        `json:"total"`
}

// AgentRunEvent represents a single event in an agent run's event stream.
type AgentRunEvent struct {
	ID        string      `json:"id"`
	RunID     string      `json:"runId"`
	Sequence  int64       `json:"sequence"`
	EventType string      `json:"eventType"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// AgentRunEventsResponse wraps a list of events.
type AgentRunEventsResponse struct {
	Events []AgentRunEvent `json:"events"`
}

// AgentRunDiffFile represents a single file in a diff.
type AgentRunDiffFile struct {
	Path       string `json:"path"`
	ChangeType string `json:"changeType"`
	Additions  int    `json:"additions"`
	Deletions  int    `json:"deletions"`
	IsBinary   bool   `json:"isBinary,omitempty"`
	Patch      string `json:"patch,omitempty"`
}

// AgentRunDiffResponse wraps the diff output for a run.
type AgentRunDiffResponse struct {
	RunID   string             `json:"runId"`
	Content string             `json:"content,omitempty"`
	Files   []AgentRunDiffFile `json:"files"`
}

// AgentContinueRequest sends a follow-up message to an agent run.
type AgentContinueRequest struct {
	Message string `json:"message"`
}

// AgentContinueResponse is the response from continuing a run.
type AgentContinueResponse struct {
	Success bool      `json:"success"`
	Run     *AgentRun `json:"run,omitempty"`
}

// AgentApproveRequest approves a run's changes.
type AgentApproveRequest struct {
	Actor     string `json:"actor,omitempty"`
	CommitMsg string `json:"commitMsg,omitempty"`
}

// AgentApproveResponse is the response from approving a run.
type AgentApproveResponse struct {
	Success      bool   `json:"success"`
	FilesApplied int    `json:"filesApplied,omitempty"`
	CommitHash   string `json:"commitHash,omitempty"`
	Message      string `json:"message,omitempty"`
}

// AgentRejectRequest rejects a run's changes.
type AgentRejectRequest struct {
	Actor  string `json:"actor,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// AgentRejectResponse is the response from rejecting a run.
type AgentRejectResponse struct {
	Status string `json:"status"`
}

// AgentStopResponse is the response from stopping a run.
type AgentStopResponse struct {
	Status string `json:"status"`
}

// ============================================================================
// Conversion functions: wire → API
// ============================================================================

func wireRunSummaryToAPI(w *wireRunSummary) *AgentRunSummary {
	if w == nil {
		return nil
	}
	return &AgentRunSummary{
		FilesModified: w.FilesModified,
		FilesCreated:  w.FilesCreated,
		FilesDeleted:  w.FilesDeleted,
		TokensUsed:    w.TokensUsed,
		TurnsUsed:     w.TurnsUsed,
		CostEstimate:  w.CostEstimate,
	}
}

func wireRunActionsToAPI(w *wireRunActions) *AgentRunActions {
	if w == nil {
		return nil
	}
	return &AgentRunActions{
		CanInvestigate:               w.CanInvestigate,
		CanApplyInvestigation:        w.CanApplyInvestigation,
		CanDelete:                    w.CanDelete,
		CanStop:                      w.CanStop,
		CanRetry:                     w.CanRetry,
		CanContinue:                  w.CanContinue,
		CanApprove:                   w.CanApprove,
		CanReject:                    w.CanReject,
		CanReview:                    w.CanReview,
		CanExtractRecommendations:    w.CanExtractRecommendations,
		CanRegenerateRecommendations: w.CanRegenerateRecommendations,
	}
}

func wireRunToAPI(w *wireRun) AgentRun {
	return AgentRun{
		ID:              w.ID,
		TaskID:          w.TaskID,
		SessionID:       w.SessionID,
		Status:          normalizeEnum(w.Status, "RUN_STATUS_"),
		Phase:           normalizeEnum(w.Phase, "RUN_PHASE_"),
		ProgressPercent: w.ProgressPercent,
		ErrorMsg:        w.ErrorMsg,
		ApprovalState:   normalizeEnum(w.ApprovalState, "APPROVAL_STATE_"),
		Summary:         wireRunSummaryToAPI(w.Summary),
		Actions:         wireRunActionsToAPI(w.Actions),
		CreatedAt:       w.CreatedAt,
		StartedAt:       w.StartedAt,
		EndedAt:         w.EndedAt,
	}
}

func wireRunEventToAPI(w *wireRunEvent) AgentRunEvent {
	evt := AgentRunEvent{
		ID:        w.ID,
		RunID:     w.RunID,
		Sequence:  w.Sequence,
		EventType: normalizeEventType(w.EventType),
		Timestamp: w.Timestamp,
	}

	// Flatten oneof data into a map for the UI.
	data := map[string]interface{}{}
	populated := false

	if w.Message != nil {
		data["content"] = w.Message.Content
		populated = true
	}
	if w.ToolCall != nil {
		data["name"] = w.ToolCall.ToolName
		if w.ToolCall.Input != "" {
			data["input"] = w.ToolCall.Input
		}
		populated = true
	}
	if w.ToolResult != nil {
		data["result"] = w.ToolResult.Output
		if w.ToolResult.ToolName != "" {
			data["name"] = w.ToolResult.ToolName
		}
		populated = true
	}
	if w.Error != nil {
		data["message"] = w.Error.Message
		if w.Error.Code != "" {
			data["code"] = w.Error.Code
		}
		populated = true
	}
	if w.Status != nil {
		if w.Status.NewStatus != "" {
			data["newStatus"] = normalizeEnum(w.Status.NewStatus, "RUN_STATUS_")
		}
		if w.Status.OldStatus != "" {
			data["oldStatus"] = normalizeEnum(w.Status.OldStatus, "RUN_STATUS_")
		}
		populated = true
	}
	if w.Log != nil {
		data["message"] = w.Log.Message
		if w.Log.Level != "" {
			data["level"] = w.Log.Level
		}
		populated = true
	}
	if w.Progress != nil {
		data["percent"] = w.Progress.Percent
		if w.Progress.Message != "" {
			data["message"] = w.Progress.Message
		}
		populated = true
	}

	if populated {
		evt.Data = data
	}
	return evt
}

func wireFileDiffToAPI(w *wireFileDiff) AgentRunDiffFile {
	return AgentRunDiffFile{
		Path:       w.Path,
		ChangeType: w.ChangeType,
		Additions:  w.Additions,
		Deletions:  w.Deletions,
		IsBinary:   w.IsBinary,
		Patch:      w.Patch,
	}
}

func wireProfileToAPI(w *wireAgentProfile) AgentProfile {
	return AgentProfile{
		ID:          w.ID,
		Key:         w.ProfileKey,
		Name:        w.Name,
		Description: w.Description,
		Model:       w.Model,
		RunnerType:  runnerTypeToString(w.RunnerType),
	}
}
