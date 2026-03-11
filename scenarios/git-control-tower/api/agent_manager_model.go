package main

// AgentProfile represents an agent configuration profile.
type AgentProfile struct {
	ID          string `json:"id"`
	Key         string `json:"key,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Model       string `json:"model,omitempty"`
}

// AgentProfileListResponse wraps a list of profiles.
type AgentProfileListResponse struct {
	Profiles []AgentProfile `json:"profiles"`
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
	Description   string `json:"description,omitempty"`
	FilesModified int    `json:"filesModified"`
	FilesCreated  int    `json:"filesCreated"`
	FilesDeleted  int    `json:"filesDeleted"`
	TokensUsed    int    `json:"tokensUsed,omitempty"`
	CostEstimate  string `json:"costEstimate,omitempty"`
}

// AgentRunActions describes what actions are available for a run.
type AgentRunActions struct {
	CanStop     bool `json:"canStop"`
	CanContinue bool `json:"canContinue"`
	CanApprove  bool `json:"canApprove"`
	CanReject   bool `json:"canReject"`
	CanRetry    bool `json:"canRetry"`
}

// AgentRun represents the state of an agent execution.
type AgentRun struct {
	ID              string           `json:"id"`
	TaskID          string           `json:"taskId,omitempty"`
	Status          string           `json:"status"` // pending|starting|running|needs_review|complete|failed|cancelled
	Phase           string           `json:"phase,omitempty"`
	ProgressPercent int              `json:"progressPercent,omitempty"`
	ErrorMsg        string           `json:"errorMsg,omitempty"`
	Summary         *AgentRunSummary `json:"summary,omitempty"`
	Actions         *AgentRunActions `json:"actions,omitempty"`
	CreatedAt       string           `json:"createdAt"`
	StartedAt       string           `json:"startedAt,omitempty"`
	CompletedAt     string           `json:"completedAt,omitempty"`
}

// AgentRunListResponse wraps a list of runs.
type AgentRunListResponse struct {
	Runs  []AgentRun `json:"runs"`
	Count int        `json:"count"`
}

// AgentRunEvent represents a single event in an agent run's event stream.
type AgentRunEvent struct {
	ID        string      `json:"id"`
	RunID     string      `json:"runId"`
	Sequence  int         `json:"sequence"`
	EventType string      `json:"eventType"`
	Timestamp string      `json:"timestamp"`
	Data      interface{} `json:"data,omitempty"`
}

// AgentRunEventsResponse wraps a list of events.
type AgentRunEventsResponse struct {
	Events []AgentRunEvent `json:"events"`
	Count  int             `json:"count"`
}

// AgentRunDiffFile represents a single file in a diff.
type AgentRunDiffFile struct {
	Path      string `json:"path"`
	Status    string `json:"status"` // added|modified|deleted
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
	Patch     string `json:"patch,omitempty"`
}

// AgentRunDiffResponse wraps the diff output for a run.
type AgentRunDiffResponse struct {
	RunID string             `json:"runId"`
	Files []AgentRunDiffFile `json:"files"`
}

// AgentContinueRequest sends a follow-up message to an agent run.
type AgentContinueRequest struct {
	Message string `json:"message"`
}

// AgentApproveRequest approves a run's changes.
type AgentApproveRequest struct {
	Actor     string `json:"actor,omitempty"`
	CommitMsg string `json:"commitMsg,omitempty"`
}

// AgentRejectRequest rejects a run's changes.
type AgentRejectRequest struct {
	Actor  string `json:"actor,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// agentTaskCreateRequest is the internal request to create a task in agent-manager.
type agentTaskCreateRequest struct {
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	ScopePath   string `json:"scopePath,omitempty"`
}

// agentTaskCreateResponse is the internal response from creating a task.
type agentTaskCreateResponse struct {
	ID string `json:"id"`
}

// agentRunCreateInternalRequest is the internal request to create a run in agent-manager.
type agentRunCreateInternalRequest struct {
	TaskID         string `json:"taskId"`
	RunMode        string `json:"runMode"`
	AgentProfileID string `json:"agentProfileId,omitempty"`
	ProfileRef     string `json:"profileRef,omitempty"`
}

// agentRunCreateInternalResponse is the internal response from creating a run.
type agentRunCreateInternalResponse struct {
	ID string `json:"id"`
}
