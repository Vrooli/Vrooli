// Package domain defines the core domain entities for agent-manager.
//
// This package contains the central concepts that agent-manager operates on:
// - AgentProfile: defines HOW an agent runs (runner config, permissions)
// - Task: defines WHAT needs to be done (scope, context, requirements)
// - Run: a concrete execution linking Task to AgentProfile within a sandbox
// - RunEvent: append-only event stream capturing all agent activity
// - Policy: rules governing execution, approval, and resource access
package domain

import (
	"time"

	"github.com/google/uuid"
)

// -----------------------------------------------------------------------------
// AgentProfile - Defines HOW an agent runs
// -----------------------------------------------------------------------------

// AgentProfile defines the configuration for running an agent.
// This is a reusable definition that can be applied to many tasks.
type AgentProfile struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`

	// Runner configuration
	RunnerType RunnerType    `json:"runnerType" db:"runner_type"`
	Model      string        `json:"model,omitempty" db:"model"`
	MaxTurns   int           `json:"maxTurns,omitempty" db:"max_turns"`
	Timeout    time.Duration `json:"timeout,omitempty" db:"timeout_ms"`

	// Tool permissions
	AllowedTools []string `json:"allowedTools,omitempty" db:"allowed_tools"`
	DeniedTools  []string `json:"deniedTools,omitempty" db:"denied_tools"`

	// Execution flags
	SkipPermissionPrompt bool `json:"skipPermissionPrompt,omitempty" db:"skip_permission_prompt"`

	// Default policies (can be overridden per task)
	RequiresSandbox  bool `json:"requiresSandbox" db:"requires_sandbox"`
	RequiresApproval bool `json:"requiresApproval" db:"requires_approval"`

	// Path restrictions
	AllowedPaths []string `json:"allowedPaths,omitempty" db:"allowed_paths"`
	DeniedPaths  []string `json:"deniedPaths,omitempty" db:"denied_paths"`

	// Metadata
	CreatedBy string    `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// RunnerType identifies which agent runner to use.
type RunnerType string

const (
	RunnerTypeClaudeCode RunnerType = "claude-code"
	RunnerTypeCodex      RunnerType = "codex"
	RunnerTypeOpenCode   RunnerType = "opencode"
)

// ValidRunnerTypes returns all valid runner types.
func ValidRunnerTypes() []RunnerType {
	return []RunnerType{
		RunnerTypeClaudeCode,
		RunnerTypeCodex,
		RunnerTypeOpenCode,
	}
}

// IsValid checks if the runner type is valid.
func (r RunnerType) IsValid() bool {
	for _, valid := range ValidRunnerTypes() {
		if r == valid {
			return true
		}
	}
	return false
}

// -----------------------------------------------------------------------------
// Task - Defines WHAT needs to be done
// -----------------------------------------------------------------------------

// Task defines a unit of work to be performed by an agent.
type Task struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Title       string    `json:"title" db:"title"`
	Description string    `json:"description,omitempty" db:"description"`

	// Scope defines where the agent can operate
	ScopePath   string `json:"scopePath" db:"scope_path"`
	ProjectRoot string `json:"projectRoot,omitempty" db:"project_root"`

	// Multi-phase execution support
	PhasePromptIDs []uuid.UUID `json:"phasePromptIds,omitempty" db:"phase_prompt_ids"`

	// Context attachments (files, links, notes)
	ContextAttachments []ContextAttachment `json:"contextAttachments,omitempty" db:"context_attachments"`

	// Status tracking
	Status TaskStatus `json:"status" db:"status"`

	// Ownership
	CreatedBy string    `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	TaskStatusQueued      TaskStatus = "queued"
	TaskStatusRunning     TaskStatus = "running"
	TaskStatusNeedsReview TaskStatus = "needs_review"
	TaskStatusApproved    TaskStatus = "approved"
	TaskStatusRejected    TaskStatus = "rejected"
	TaskStatusFailed      TaskStatus = "failed"
	TaskStatusCancelled   TaskStatus = "cancelled"
)

// ContextAttachment represents additional context for a task.
type ContextAttachment struct {
	Type    string `json:"type"` // "file", "link", "note"
	Path    string `json:"path,omitempty"`
	URL     string `json:"url,omitempty"`
	Content string `json:"content,omitempty"`
}

// -----------------------------------------------------------------------------
// Run - A concrete execution attempt
// -----------------------------------------------------------------------------

// Run represents a single execution attempt of a task using a specific agent profile.
type Run struct {
	ID             uuid.UUID `json:"id" db:"id"`
	TaskID         uuid.UUID `json:"taskId" db:"task_id"`
	AgentProfileID uuid.UUID `json:"agentProfileId" db:"agent_profile_id"`

	// Sandbox integration
	SandboxID *uuid.UUID `json:"sandboxId,omitempty" db:"sandbox_id"`
	RunMode   RunMode    `json:"runMode" db:"run_mode"`

	// Execution state
	Status    RunStatus  `json:"status" db:"status"`
	StartedAt *time.Time `json:"startedAt,omitempty" db:"started_at"`
	EndedAt   *time.Time `json:"endedAt,omitempty" db:"ended_at"`

	// Results
	Summary   *RunSummary `json:"summary,omitempty" db:"summary"`
	ErrorMsg  string      `json:"errorMsg,omitempty" db:"error_msg"`
	ExitCode  *int        `json:"exitCode,omitempty" db:"exit_code"`

	// Approval workflow
	ApprovalState ApprovalState `json:"approvalState" db:"approval_state"`
	ApprovedBy    string        `json:"approvedBy,omitempty" db:"approved_by"`
	ApprovedAt    *time.Time    `json:"approvedAt,omitempty" db:"approved_at"`

	// Artifacts
	DiffPath       string `json:"diffPath,omitempty" db:"diff_path"`
	LogPath        string `json:"logPath,omitempty" db:"log_path"`
	ChangedFiles   int    `json:"changedFiles" db:"changed_files"`
	TotalSizeBytes int64  `json:"totalSizeBytes" db:"total_size_bytes"`

	// Metadata
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

// RunMode indicates whether the run uses sandbox isolation.
type RunMode string

const (
	RunModeSandboxed RunMode = "sandboxed"
	RunModeInPlace   RunMode = "in_place"
)

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

// ApprovalState represents the approval workflow state.
type ApprovalState string

const (
	ApprovalStateNone             ApprovalState = "none"
	ApprovalStatePending          ApprovalState = "pending"
	ApprovalStatePartiallyApproved ApprovalState = "partially_approved"
	ApprovalStateApproved         ApprovalState = "approved"
	ApprovalStateRejected         ApprovalState = "rejected"
)

// RunSummary contains the structured summary from an agent run.
type RunSummary struct {
	Description   string   `json:"description,omitempty"`
	FilesModified []string `json:"filesModified,omitempty"`
	FilesCreated  []string `json:"filesCreated,omitempty"`
	FilesDeleted  []string `json:"filesDeleted,omitempty"`
	TokensUsed    int      `json:"tokensUsed,omitempty"`
	TurnsUsed     int      `json:"turnsUsed,omitempty"`
	CostEstimate  float64  `json:"costEstimate,omitempty"`
}

// -----------------------------------------------------------------------------
// RunEvent - Append-only event stream
// -----------------------------------------------------------------------------

// RunEvent represents a single event in a run's event stream.
type RunEvent struct {
	ID        uuid.UUID     `json:"id" db:"id"`
	RunID     uuid.UUID     `json:"runId" db:"run_id"`
	Sequence  int64         `json:"sequence" db:"sequence"`
	EventType RunEventType  `json:"eventType" db:"event_type"`
	Timestamp time.Time     `json:"timestamp" db:"timestamp"`
	Data      RunEventData  `json:"data" db:"data"`
}

// RunEventType categorizes the event.
type RunEventType string

const (
	EventTypeLog        RunEventType = "log"
	EventTypeMessage    RunEventType = "message"
	EventTypeToolCall   RunEventType = "tool_call"
	EventTypeToolResult RunEventType = "tool_result"
	EventTypeStatus     RunEventType = "status"
	EventTypeMetric     RunEventType = "metric"
	EventTypeArtifact   RunEventType = "artifact"
	EventTypeError      RunEventType = "error"
)

// RunEventData contains the event-specific payload.
type RunEventData struct {
	// For log events
	Level   string `json:"level,omitempty"`
	Message string `json:"message,omitempty"`

	// For message events
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`

	// For tool_call events
	ToolName  string                 `json:"toolName,omitempty"`
	ToolInput map[string]interface{} `json:"toolInput,omitempty"`

	// For tool_result events
	ToolOutput string `json:"toolOutput,omitempty"`
	ToolError  string `json:"toolError,omitempty"`

	// For status events
	OldStatus string `json:"oldStatus,omitempty"`
	NewStatus string `json:"newStatus,omitempty"`

	// For metric events
	MetricName  string  `json:"metricName,omitempty"`
	MetricValue float64 `json:"metricValue,omitempty"`

	// For artifact events
	ArtifactType string `json:"artifactType,omitempty"`
	ArtifactPath string `json:"artifactPath,omitempty"`

	// For error events
	ErrorCode    string `json:"errorCode,omitempty"`
	ErrorMessage string `json:"errorMessage,omitempty"`
}

// -----------------------------------------------------------------------------
// Policy - Rules governing execution
// -----------------------------------------------------------------------------

// Policy defines rules for agent execution, approval, and resource access.
type Policy struct {
	ID          uuid.UUID `json:"id" db:"id"`
	Name        string    `json:"name" db:"name"`
	Description string    `json:"description,omitempty" db:"description"`
	Priority    int       `json:"priority" db:"priority"` // Higher priority wins

	// Scope matching
	ScopePattern string `json:"scopePattern,omitempty" db:"scope_pattern"` // Glob pattern

	// Execution rules
	Rules PolicyRules `json:"rules" db:"rules"`

	// Metadata
	CreatedBy string    `json:"createdBy,omitempty" db:"created_by"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
	Enabled   bool      `json:"enabled" db:"enabled"`
}

// PolicyRules contains the actual policy constraints.
type PolicyRules struct {
	// Sandbox requirements
	RequireSandbox    *bool `json:"requireSandbox,omitempty"`
	AllowInPlace      *bool `json:"allowInPlace,omitempty"`
	InPlaceRequiresApproval *bool `json:"inPlaceRequiresApproval,omitempty"`

	// Approval requirements
	RequireApproval       *bool `json:"requireApproval,omitempty"`
	AutoApprovePatterns   []string `json:"autoApprovePatterns,omitempty"`

	// Concurrency limits
	MaxConcurrentRuns     *int `json:"maxConcurrentRuns,omitempty"`
	MaxConcurrentPerScope *int `json:"maxConcurrentPerScope,omitempty"`

	// Resource limits
	MaxFilesChanged    *int   `json:"maxFilesChanged,omitempty"`
	MaxTotalSizeBytes  *int64 `json:"maxTotalSizeBytes,omitempty"`
	MaxExecutionTimeMs *int64 `json:"maxExecutionTimeMs,omitempty"`

	// Runner restrictions
	AllowedRunners []RunnerType `json:"allowedRunners,omitempty"`
	DeniedRunners  []RunnerType `json:"deniedRunners,omitempty"`
}

// -----------------------------------------------------------------------------
// ScopeLock - Concurrency control
// -----------------------------------------------------------------------------

// ScopeLock represents an exclusive lock on a path scope.
type ScopeLock struct {
	ID          uuid.UUID `json:"id" db:"id"`
	RunID       uuid.UUID `json:"runId" db:"run_id"`
	ScopePath   string    `json:"scopePath" db:"scope_path"`
	ProjectRoot string    `json:"projectRoot" db:"project_root"`
	AcquiredAt  time.Time `json:"acquiredAt" db:"acquired_at"`
	ExpiresAt   time.Time `json:"expiresAt" db:"expires_at"`
}
