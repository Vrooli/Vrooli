package main

import "time"

// =============================================================================
// Profile Types
// =============================================================================

// Profile represents an agent configuration.
type Profile struct {
	ID                   string   `json:"id"`
	Name                 string   `json:"name"`
	Description          string   `json:"description,omitempty"`
	RunnerType           string   `json:"runnerType"`
	Model                string   `json:"model,omitempty"`
	MaxTurns             int      `json:"maxTurns,omitempty"`
	Timeout              string   `json:"timeout,omitempty"`
	AllowedTools         []string `json:"allowedTools,omitempty"`
	DeniedTools          []string `json:"deniedTools,omitempty"`
	SkipPermissionPrompt bool     `json:"skipPermissionPrompt,omitempty"`
	RequiresSandbox      bool     `json:"requiresSandbox"`
	RequiresApproval     bool     `json:"requiresApproval"`
	AllowedPaths         []string `json:"allowedPaths,omitempty"`
	DeniedPaths          []string `json:"deniedPaths,omitempty"`
	CreatedBy            string   `json:"createdBy,omitempty"`
	CreatedAt            string   `json:"createdAt,omitempty"`
	UpdatedAt            string   `json:"updatedAt,omitempty"`
}

// CreateProfileRequest contains the fields for creating a profile.
type CreateProfileRequest struct {
	Name                 string   `json:"name"`
	Description          string   `json:"description,omitempty"`
	RunnerType           string   `json:"runnerType"`
	Model                string   `json:"model,omitempty"`
	MaxTurns             int      `json:"maxTurns,omitempty"`
	Timeout              string   `json:"timeout,omitempty"`
	AllowedTools         []string `json:"allowedTools,omitempty"`
	DeniedTools          []string `json:"deniedTools,omitempty"`
	SkipPermissionPrompt bool     `json:"skipPermissionPrompt,omitempty"`
	RequiresSandbox      bool     `json:"requiresSandbox"`
	RequiresApproval     bool     `json:"requiresApproval"`
	AllowedPaths         []string `json:"allowedPaths,omitempty"`
	DeniedPaths          []string `json:"deniedPaths,omitempty"`
	CreatedBy            string   `json:"createdBy,omitempty"`
}

// =============================================================================
// Task Types
// =============================================================================

// Task represents a work item to be executed.
type Task struct {
	ID                 string              `json:"id"`
	Title              string              `json:"title"`
	Description        string              `json:"description,omitempty"`
	ScopePath          string              `json:"scopePath,omitempty"`
	ProjectRoot        string              `json:"projectRoot,omitempty"`
	PhasePromptIDs     []string            `json:"phasePromptIds,omitempty"`
	ContextAttachments []ContextAttachment `json:"contextAttachments,omitempty"`
	Status             string              `json:"status"`
	CreatedBy          string              `json:"createdBy,omitempty"`
	CreatedAt          string              `json:"createdAt,omitempty"`
	UpdatedAt          string              `json:"updatedAt,omitempty"`
}

// ContextAttachment represents additional context for a task.
type ContextAttachment struct {
	Type    string `json:"type"` // "file", "link", "note"
	Path    string `json:"path,omitempty"`
	URL     string `json:"url,omitempty"`
	Content string `json:"content,omitempty"`
}

// CreateTaskRequest contains the fields for creating a task.
type CreateTaskRequest struct {
	Title              string              `json:"title"`
	Description        string              `json:"description,omitempty"`
	ScopePath          string              `json:"scopePath,omitempty"`
	ProjectRoot        string              `json:"projectRoot,omitempty"`
	ContextAttachments []ContextAttachment `json:"contextAttachments,omitempty"`
	CreatedBy          string              `json:"createdBy,omitempty"`
}

// =============================================================================
// Run Types
// =============================================================================

// Run represents an execution instance.
type Run struct {
	ID              string      `json:"id"`
	TaskID          string      `json:"taskId"`
	AgentProfileID  string      `json:"agentProfileId"`
	SandboxID       string      `json:"sandboxId,omitempty"`
	RunMode         string      `json:"runMode"`
	Status          string      `json:"status"`
	Phase           string      `json:"phase,omitempty"`
	ProgressPercent int         `json:"progressPercent,omitempty"`
	StartedAt       string      `json:"startedAt,omitempty"`
	EndedAt         string      `json:"endedAt,omitempty"`
	Summary         *RunSummary `json:"summary,omitempty"`
	ErrorMsg        string      `json:"errorMsg,omitempty"`
	ExitCode        *int        `json:"exitCode,omitempty"`
	ApprovalState   string      `json:"approvalState,omitempty"`
	ApprovedBy      string      `json:"approvedBy,omitempty"`
	ApprovedAt      string      `json:"approvedAt,omitempty"`
	DiffPath        string      `json:"diffPath,omitempty"`
	LogPath         string      `json:"logPath,omitempty"`
	ChangedFiles    int         `json:"changedFiles,omitempty"`
	TotalSizeBytes  int64       `json:"totalSizeBytes,omitempty"`
	CreatedAt       string      `json:"createdAt,omitempty"`
	UpdatedAt       string      `json:"updatedAt,omitempty"`
}

// RunSummary contains execution summary information.
type RunSummary struct {
	Description  string  `json:"description,omitempty"`
	TurnsUsed    int     `json:"turnsUsed,omitempty"`
	TokensUsed   int     `json:"tokensUsed,omitempty"`
	CostEstimate float64 `json:"costEstimate,omitempty"`
}

// CreateRunRequest contains the fields for creating a run.
type CreateRunRequest struct {
	TaskID         string `json:"taskId"`
	AgentProfileID string `json:"agentProfileId"`
	Prompt         string `json:"prompt,omitempty"`
	RunMode        string `json:"runMode,omitempty"`
	ForceInPlace   bool   `json:"forceInPlace,omitempty"`
	IdempotencyKey string `json:"idempotencyKey,omitempty"`
}

// ApproveRequest contains the fields for approving a run.
type ApproveRequest struct {
	Actor     string `json:"actor"`
	CommitMsg string `json:"commitMsg,omitempty"`
	Force     bool   `json:"force,omitempty"`
}

// RejectRequest contains the fields for rejecting a run.
type RejectRequest struct {
	Actor  string `json:"actor"`
	Reason string `json:"reason"`
}

// ApproveResult contains the approval outcome.
type ApproveResult struct {
	Success    bool   `json:"success"`
	Applied    int    `json:"applied"`
	Remaining  int    `json:"remaining"`
	IsPartial  bool   `json:"isPartial"`
	CommitHash string `json:"commitHash,omitempty"`
	AppliedAt  string `json:"appliedAt,omitempty"`
	ErrorMsg   string `json:"errorMsg,omitempty"`
}

// =============================================================================
// Event Types
// =============================================================================

// RunEvent represents an event during run execution.
type RunEvent struct {
	ID        string    `json:"id"`
	RunID     string    `json:"runId"`
	Sequence  int64     `json:"sequence"`
	EventType string    `json:"eventType"`
	Timestamp time.Time `json:"timestamp"`
	Data      any       `json:"data,omitempty"`
}

// =============================================================================
// Health Types
// =============================================================================

// HealthResponse represents the API health status.
type HealthResponse struct {
	Status       string              `json:"status"`
	Service      string              `json:"service"`
	Timestamp    string              `json:"timestamp"`
	Readiness    bool                `json:"readiness"`
	Dependencies *HealthDependencies `json:"dependencies,omitempty"`
	ActiveRuns   int                 `json:"activeRuns"`
	QueuedTasks  int                 `json:"queuedTasks"`
}

// HealthDependencies contains dependency health status.
type HealthDependencies struct {
	Database *DependencyStatus            `json:"database,omitempty"`
	Sandbox  *DependencyStatus            `json:"sandbox,omitempty"`
	Runners  map[string]*DependencyStatus `json:"runners,omitempty"`
}

// DependencyStatus describes a dependency's health.
type DependencyStatus struct {
	Connected bool    `json:"connected"`
	LatencyMs *int64  `json:"latency_ms,omitempty"`
	Error     *string `json:"error,omitempty"`
}

// RunnerStatus describes a runner's availability.
type RunnerStatus struct {
	Type      string `json:"type"`
	Available bool   `json:"available"`
	Message   string `json:"message,omitempty"`
}
