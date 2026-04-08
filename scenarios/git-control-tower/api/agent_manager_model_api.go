package main

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
	ScenarioSlug  string   `json:"scenarioSlug"`
	Prompt        string   `json:"prompt"`
	ProfileID     string   `json:"profileId,omitempty"`
	ProfileKey    string   `json:"profileKey,omitempty"`
	AttachmentIDs []string `json:"attachmentIds,omitempty"`
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
	PromptPreview   string           `json:"promptPreview,omitempty"`
	SandboxID       string           `json:"sandboxId,omitempty"`
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
	Message       string   `json:"message"`
	AttachmentIDs []string `json:"attachment_ids,omitempty"`
}

// wireUploadAttachmentResponse is the response from agent-manager's upload endpoint.
type wireUploadAttachmentResponse struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	FileSize    int64  `json:"file_size"`
	StoragePath string `json:"storage_path"`
	URL         string `json:"url"`
}

// AttachmentUploadResponse is the UI-facing response for attachment uploads.
type AttachmentUploadResponse struct {
	ID          string `json:"id"`
	FileName    string `json:"fileName"`
	ContentType string `json:"contentType"`
	FileSize    int64  `json:"fileSize"`
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
