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

	// Progress tracking (for resumption and visibility)
	Phase            RunPhase   `json:"phase" db:"phase"`
	LastCheckpointID *uuid.UUID `json:"lastCheckpointId,omitempty" db:"last_checkpoint_id"`
	LastHeartbeat    *time.Time `json:"lastHeartbeat,omitempty" db:"last_heartbeat"`
	ProgressPercent  int        `json:"progressPercent" db:"progress_percent"`

	// Idempotency (for replay safety)
	IdempotencyKey string `json:"idempotencyKey,omitempty" db:"idempotency_key"`

	// Results
	Summary  *RunSummary `json:"summary,omitempty" db:"summary"`
	ErrorMsg string      `json:"errorMsg,omitempty" db:"error_msg"`
	ExitCode *int        `json:"exitCode,omitempty" db:"exit_code"`

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

// IsResumable returns whether this run can be resumed from its current state.
func (r *Run) IsResumable() bool {
	// Can only resume runs that are in a non-terminal state
	switch r.Status {
	case RunStatusComplete, RunStatusFailed, RunStatusCancelled:
		return false
	}
	// Check if the phase supports resumption
	return r.Phase.CanResumeFromPhase()
}

// IsStale returns whether this run appears to have stalled.
func (r *Run) IsStale(staleDuration time.Duration) bool {
	if r.LastHeartbeat == nil {
		// No heartbeat recorded, check based on started time
		if r.StartedAt == nil {
			return false
		}
		return time.Since(*r.StartedAt) > staleDuration
	}
	return time.Since(*r.LastHeartbeat) > staleDuration
}

// UpdateProgress updates the run's progress tracking fields.
func (r *Run) UpdateProgress(phase RunPhase, percent int) {
	r.Phase = phase
	r.ProgressPercent = percent
	now := time.Now()
	r.LastHeartbeat = &now
	r.UpdatedAt = now
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
	ApprovalStateNone              ApprovalState = "none"
	ApprovalStatePending           ApprovalState = "pending"
	ApprovalStatePartiallyApproved ApprovalState = "partially_approved"
	ApprovalStateApproved          ApprovalState = "approved"
	ApprovalStateRejected          ApprovalState = "rejected"
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
//
// TAGGED UNION PATTERN:
// RunEvent uses a tagged union for type-safe event payloads. Each event type
// has a specific payload struct, ensuring you can only set relevant fields.
//
// Usage:
//   event := NewLogEvent(runID, "info", "Starting execution")
//   event := NewToolCallEvent(runID, "Read", map[string]interface{}{"path": "/foo"})
//
// The Data field contains a type-specific payload that can be type-asserted:
//   if log, ok := event.Data.(*LogEventData); ok { ... }

// RunEvent represents a single event in a run's event stream.
type RunEvent struct {
	ID        uuid.UUID    `json:"id" db:"id"`
	RunID     uuid.UUID    `json:"runId" db:"run_id"`
	Sequence  int64        `json:"sequence" db:"sequence"`
	EventType RunEventType `json:"eventType" db:"event_type"`
	Timestamp time.Time    `json:"timestamp" db:"timestamp"`
	Data      EventPayload `json:"data" db:"data"`
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

// =============================================================================
// EVENT PAYLOAD INTERFACE (Tagged Union)
// =============================================================================

// EventPayload is the interface for all event-specific data.
// Each event type has a corresponding struct implementing this interface.
type EventPayload interface {
	// EventType returns the type of this payload for serialization.
	EventType() RunEventType

	// isEventPayload is a marker method to prevent external implementations.
	isEventPayload()
}

// =============================================================================
// LOG EVENT
// =============================================================================

// LogEventData contains data for log events (debug, info, warn, error messages).
type LogEventData struct {
	Level   string `json:"level"`   // debug, info, warn, error
	Message string `json:"message"` // The log message
}

func (d *LogEventData) EventType() RunEventType { return EventTypeLog }
func (d *LogEventData) isEventPayload()         {}

// NewLogEvent creates a new log event.
func NewLogEvent(runID uuid.UUID, level, message string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeLog,
		Timestamp: time.Now(),
		Data:      &LogEventData{Level: level, Message: message},
	}
}

// =============================================================================
// MESSAGE EVENT
// =============================================================================

// MessageEventData contains data for conversation messages (user, assistant, system).
type MessageEventData struct {
	Role    string `json:"role"`    // user, assistant, system
	Content string `json:"content"` // Message content
}

func (d *MessageEventData) EventType() RunEventType { return EventTypeMessage }
func (d *MessageEventData) isEventPayload()         {}

// NewMessageEvent creates a new message event.
func NewMessageEvent(runID uuid.UUID, role, content string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeMessage,
		Timestamp: time.Now(),
		Data:      &MessageEventData{Role: role, Content: content},
	}
}

// =============================================================================
// TOOL CALL EVENT
// =============================================================================

// ToolCallEventData contains data for tool invocation events.
type ToolCallEventData struct {
	ToolName string                 `json:"toolName"` // Name of the tool being called
	Input    map[string]interface{} `json:"input"`    // Tool input parameters
}

func (d *ToolCallEventData) EventType() RunEventType { return EventTypeToolCall }
func (d *ToolCallEventData) isEventPayload()         {}

// NewToolCallEvent creates a new tool call event.
func NewToolCallEvent(runID uuid.UUID, toolName string, input map[string]interface{}) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeToolCall,
		Timestamp: time.Now(),
		Data:      &ToolCallEventData{ToolName: toolName, Input: input},
	}
}

// =============================================================================
// TOOL RESULT EVENT
// =============================================================================

// ToolResultEventData contains data for tool result events.
type ToolResultEventData struct {
	ToolName string `json:"toolName"`        // Name of the tool that was called
	Output   string `json:"output"`          // Tool output (success)
	Error    string `json:"error,omitempty"` // Error message (if failed)
	Success  bool   `json:"success"`         // Whether the tool call succeeded
}

func (d *ToolResultEventData) EventType() RunEventType { return EventTypeToolResult }
func (d *ToolResultEventData) isEventPayload()         {}

// NewToolResultEvent creates a new tool result event.
func NewToolResultEvent(runID uuid.UUID, toolName, output string, err error) *RunEvent {
	data := &ToolResultEventData{
		ToolName: toolName,
		Output:   output,
		Success:  err == nil,
	}
	if err != nil {
		data.Error = err.Error()
	}
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeToolResult,
		Timestamp: time.Now(),
		Data:      data,
	}
}

// =============================================================================
// STATUS EVENT
// =============================================================================

// StatusEventData contains data for status transition events.
type StatusEventData struct {
	OldStatus string `json:"oldStatus"`        // Previous status
	NewStatus string `json:"newStatus"`        // New status
	Reason    string `json:"reason,omitempty"` // Why the transition happened
}

func (d *StatusEventData) EventType() RunEventType { return EventTypeStatus }
func (d *StatusEventData) isEventPayload()         {}

// NewStatusEvent creates a new status change event.
func NewStatusEvent(runID uuid.UUID, oldStatus, newStatus, reason string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeStatus,
		Timestamp: time.Now(),
		Data:      &StatusEventData{OldStatus: oldStatus, NewStatus: newStatus, Reason: reason},
	}
}

// =============================================================================
// METRIC EVENT
// =============================================================================

// MetricEventData contains data for metric/telemetry events.
type MetricEventData struct {
	Name  string            `json:"name"`           // Metric name (e.g., "tokens_used")
	Value float64           `json:"value"`          // Metric value
	Unit  string            `json:"unit,omitempty"` // Unit (e.g., "tokens", "ms", "bytes")
	Tags  map[string]string `json:"tags,omitempty"` // Additional tags for grouping
}

func (d *MetricEventData) EventType() RunEventType { return EventTypeMetric }
func (d *MetricEventData) isEventPayload()         {}

// NewMetricEvent creates a new metric event.
func NewMetricEvent(runID uuid.UUID, name string, value float64, unit string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeMetric,
		Timestamp: time.Now(),
		Data:      &MetricEventData{Name: name, Value: value, Unit: unit},
	}
}

// =============================================================================
// ARTIFACT EVENT
// =============================================================================

// ArtifactEventData contains data for artifact creation events.
type ArtifactEventData struct {
	Type     string `json:"type"`               // Artifact type (diff, log, screenshot, etc.)
	Path     string `json:"path"`               // Path to the artifact
	Size     int64  `json:"size,omitempty"`     // Size in bytes
	MimeType string `json:"mimeType,omitempty"` // MIME type
}

func (d *ArtifactEventData) EventType() RunEventType { return EventTypeArtifact }
func (d *ArtifactEventData) isEventPayload()         {}

// NewArtifactEvent creates a new artifact event.
func NewArtifactEvent(runID uuid.UUID, artifactType, path string, size int64) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeArtifact,
		Timestamp: time.Now(),
		Data:      &ArtifactEventData{Type: artifactType, Path: path, Size: size},
	}
}

// =============================================================================
// ERROR EVENT
// =============================================================================

// ErrorEventData contains data for error events.
type ErrorEventData struct {
	Code       string         `json:"code"`                 // Machine-readable error code
	Message    string         `json:"message"`              // Human-readable error message
	Retryable  bool           `json:"retryable"`            // Whether the error is retryable
	Recovery   RecoveryAction `json:"recovery,omitempty"`   // Suggested recovery action
	StackTrace string         `json:"stackTrace,omitempty"` // Optional stack trace
}

func (d *ErrorEventData) EventType() RunEventType { return EventTypeError }
func (d *ErrorEventData) isEventPayload()         {}

// NewErrorEvent creates a new error event.
func NewErrorEvent(runID uuid.UUID, code, message string, retryable bool) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeError,
		Timestamp: time.Now(),
		Data:      &ErrorEventData{Code: code, Message: message, Retryable: retryable},
	}
}

// NewErrorEventFromDomainError creates an error event from a DomainError.
func NewErrorEventFromDomainError(runID uuid.UUID, err DomainError) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeError,
		Timestamp: time.Now(),
		Data: &ErrorEventData{
			Code:      string(err.Code()),
			Message:   err.Error(),
			Retryable: err.Retryable(),
			Recovery:  err.Recovery(),
		},
	}
}

// =============================================================================
// LEGACY SUPPORT (RunEventData)
// =============================================================================
// RunEventData is kept for backward compatibility during migration.
// New code should use the specific event data types above.

// RunEventData contains the event-specific payload (DEPRECATED: use specific types).
// This struct is retained for JSON unmarshaling compatibility with existing data.
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

// Implement EventPayload interface for backward compatibility
func (d RunEventData) EventType() RunEventType {
	// Infer type from which fields are populated
	if d.Level != "" || (d.Message != "" && d.Role == "") {
		return EventTypeLog
	}
	if d.Role != "" {
		return EventTypeMessage
	}
	if d.ToolName != "" && d.ToolInput != nil {
		return EventTypeToolCall
	}
	if d.ToolOutput != "" || d.ToolError != "" {
		return EventTypeToolResult
	}
	if d.OldStatus != "" || d.NewStatus != "" {
		return EventTypeStatus
	}
	if d.MetricName != "" {
		return EventTypeMetric
	}
	if d.ArtifactType != "" {
		return EventTypeArtifact
	}
	if d.ErrorCode != "" || d.ErrorMessage != "" {
		return EventTypeError
	}
	return EventTypeLog // default fallback
}
func (d RunEventData) isEventPayload() {}

// ToTypedPayload converts legacy RunEventData to the appropriate typed payload.
func (d RunEventData) ToTypedPayload() EventPayload {
	switch d.EventType() {
	case EventTypeLog:
		return &LogEventData{Level: d.Level, Message: d.Message}
	case EventTypeMessage:
		return &MessageEventData{Role: d.Role, Content: d.Content}
	case EventTypeToolCall:
		return &ToolCallEventData{ToolName: d.ToolName, Input: d.ToolInput}
	case EventTypeToolResult:
		var err string
		if d.ToolError != "" {
			err = d.ToolError
		}
		return &ToolResultEventData{ToolName: d.ToolName, Output: d.ToolOutput, Error: err, Success: err == ""}
	case EventTypeStatus:
		return &StatusEventData{OldStatus: d.OldStatus, NewStatus: d.NewStatus}
	case EventTypeMetric:
		return &MetricEventData{Name: d.MetricName, Value: d.MetricValue}
	case EventTypeArtifact:
		return &ArtifactEventData{Type: d.ArtifactType, Path: d.ArtifactPath}
	case EventTypeError:
		return &ErrorEventData{Code: d.ErrorCode, Message: d.ErrorMessage}
	default:
		return &LogEventData{Message: d.Message}
	}
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
	RequireSandbox          *bool `json:"requireSandbox,omitempty"`
	AllowInPlace            *bool `json:"allowInPlace,omitempty"`
	InPlaceRequiresApproval *bool `json:"inPlaceRequiresApproval,omitempty"`

	// Approval requirements
	RequireApproval     *bool    `json:"requireApproval,omitempty"`
	AutoApprovePatterns []string `json:"autoApprovePatterns,omitempty"`

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

// =============================================================================
// IDEMPOTENCY & REPLAY SAFETY
// =============================================================================
// These types enable safe retries, resumption, and replay of operations.
// See: idempotency-replay-safety-hardening.md

// IdempotencyRecord tracks whether an operation has been performed.
// This prevents duplicate work when operations are retried.
type IdempotencyRecord struct {
	// Key uniquely identifies the operation (e.g., "run-create:task-{taskID}:profile-{profileID}:ts-{timestamp}")
	Key string `json:"key" db:"key"`

	// Status indicates the operation outcome
	Status IdempotencyStatus `json:"status" db:"status"`

	// EntityID is the ID of the created/affected entity (if applicable)
	EntityID *uuid.UUID `json:"entityId,omitempty" db:"entity_id"`

	// EntityType identifies what was created (e.g., "Run", "Task")
	EntityType string `json:"entityType,omitempty" db:"entity_type"`

	// CreatedAt is when this record was created
	CreatedAt time.Time `json:"createdAt" db:"created_at"`

	// ExpiresAt is when this record can be garbage collected
	ExpiresAt time.Time `json:"expiresAt" db:"expires_at"`

	// Response contains the cached response (JSON) for successful operations
	Response []byte `json:"response,omitempty" db:"response"`
}

// IdempotencyStatus indicates the state of an idempotent operation.
type IdempotencyStatus string

const (
	// IdempotencyStatusPending - Operation started but not completed
	IdempotencyStatusPending IdempotencyStatus = "pending"

	// IdempotencyStatusComplete - Operation completed successfully
	IdempotencyStatusComplete IdempotencyStatus = "complete"

	// IdempotencyStatusFailed - Operation failed (may be retried)
	IdempotencyStatusFailed IdempotencyStatus = "failed"
)

// =============================================================================
// PROGRESS & CHECKPOINT TRACKING
// =============================================================================
// These types enable safe interruption and resumption of runs.
// See: progress-continuity-interruption-resilience.md

// RunPhase represents the current phase of run execution.
// This enables resumption from the correct point after interruption.
type RunPhase string

const (
	// RunPhaseQueued - Run created but not started
	RunPhaseQueued RunPhase = "queued"

	// RunPhaseInitializing - Setting up workspace and acquiring resources
	RunPhaseInitializing RunPhase = "initializing"

	// RunPhaseSandboxCreating - Creating sandbox (if sandboxed mode)
	RunPhaseSandboxCreating RunPhase = "sandbox_creating"

	// RunPhaseRunnerAcquiring - Acquiring and validating runner
	RunPhaseRunnerAcquiring RunPhase = "runner_acquiring"

	// RunPhaseExecuting - Agent is actively executing
	RunPhaseExecuting RunPhase = "executing"

	// RunPhaseCollectingResults - Gathering results and artifacts
	RunPhaseCollectingResults RunPhase = "collecting_results"

	// RunPhaseAwaitingReview - Execution complete, awaiting approval
	RunPhaseAwaitingReview RunPhase = "awaiting_review"

	// RunPhaseApplying - Applying approved changes
	RunPhaseApplying RunPhase = "applying"

	// RunPhaseCleaningUp - Releasing resources and cleaning up
	RunPhaseCleaningUp RunPhase = "cleaning_up"

	// RunPhaseCompleted - Run is finished (terminal)
	RunPhaseCompleted RunPhase = "completed"
)

// CanResumeFromPhase returns whether a run can be resumed from this phase.
func (p RunPhase) CanResumeFromPhase() bool {
	switch p {
	case RunPhaseQueued, RunPhaseInitializing, RunPhaseSandboxCreating,
		RunPhaseRunnerAcquiring, RunPhaseExecuting:
		return true
	default:
		return false
	}
}

// IsTerminal returns whether this phase represents a completed run.
func (p RunPhase) IsTerminal() bool {
	return p == RunPhaseCompleted
}

// RunCheckpoint captures the state needed to resume a run.
type RunCheckpoint struct {
	// RunID is the run this checkpoint belongs to
	RunID uuid.UUID `json:"runId" db:"run_id"`

	// Phase is the current execution phase
	Phase RunPhase `json:"phase" db:"phase"`

	// StepWithinPhase tracks progress within a phase (0-indexed)
	StepWithinPhase int `json:"stepWithinPhase" db:"step_within_phase"`

	// SandboxID is set after sandbox creation
	SandboxID *uuid.UUID `json:"sandboxId,omitempty" db:"sandbox_id"`

	// WorkDir is set after workspace setup
	WorkDir string `json:"workDir,omitempty" db:"work_dir"`

	// LockID is set after acquiring scope lock
	LockID *uuid.UUID `json:"lockId,omitempty" db:"lock_id"`

	// LastEventSequence is the last event sequence number persisted
	LastEventSequence int64 `json:"lastEventSequence" db:"last_event_sequence"`

	// LastHeartbeat is when we last confirmed progress
	LastHeartbeat time.Time `json:"lastHeartbeat" db:"last_heartbeat"`

	// RetryCount tracks how many times this phase has been retried
	RetryCount int `json:"retryCount" db:"retry_count"`

	// SavedAt is when this checkpoint was created
	SavedAt time.Time `json:"savedAt" db:"saved_at"`

	// Metadata contains phase-specific state that may be needed for resumption
	Metadata map[string]string `json:"metadata,omitempty" db:"metadata"`
}

// NewCheckpoint creates a checkpoint for the current run state.
func NewCheckpoint(runID uuid.UUID, phase RunPhase) *RunCheckpoint {
	now := time.Now()
	return &RunCheckpoint{
		RunID:         runID,
		Phase:         phase,
		LastHeartbeat: now,
		SavedAt:       now,
		Metadata:      make(map[string]string),
	}
}

// Update creates an updated checkpoint with new phase information.
func (c *RunCheckpoint) Update(phase RunPhase, step int) *RunCheckpoint {
	now := time.Now()
	return &RunCheckpoint{
		RunID:             c.RunID,
		Phase:             phase,
		StepWithinPhase:   step,
		SandboxID:         c.SandboxID,
		WorkDir:           c.WorkDir,
		LockID:            c.LockID,
		LastEventSequence: c.LastEventSequence,
		LastHeartbeat:     now,
		RetryCount:        c.RetryCount,
		SavedAt:           now,
		Metadata:          c.Metadata,
	}
}

// WithSandbox adds sandbox information to the checkpoint.
func (c *RunCheckpoint) WithSandbox(sandboxID uuid.UUID, workDir string) *RunCheckpoint {
	cp := *c
	cp.SandboxID = &sandboxID
	cp.WorkDir = workDir
	cp.SavedAt = time.Now()
	return &cp
}

// WithLock adds lock information to the checkpoint.
func (c *RunCheckpoint) WithLock(lockID uuid.UUID) *RunCheckpoint {
	cp := *c
	cp.LockID = &lockID
	cp.SavedAt = time.Now()
	return &cp
}

// WithEventSequence updates the last persisted event sequence.
func (c *RunCheckpoint) WithEventSequence(seq int64) *RunCheckpoint {
	cp := *c
	cp.LastEventSequence = seq
	cp.SavedAt = time.Now()
	return &cp
}

// IncrementRetry increments the retry count for the current phase.
func (c *RunCheckpoint) IncrementRetry() *RunCheckpoint {
	cp := *c
	cp.RetryCount++
	cp.SavedAt = time.Now()
	return &cp
}

// =============================================================================
// TEMPORAL FLOW & HEARTBEAT
// =============================================================================
// These types support time-based coordination and health monitoring.
// See: temporal-flow-audit.md

// HeartbeatConfig defines heartbeat behavior for long-running operations.
type HeartbeatConfig struct {
	// Interval is how often to send heartbeats
	Interval time.Duration `json:"interval"`

	// Timeout is how long without a heartbeat before considering dead
	Timeout time.Duration `json:"timeout"`

	// MaxMissedBeats is the number of missed heartbeats before termination
	MaxMissedBeats int `json:"maxMissedBeats"`
}

// DefaultHeartbeatConfig returns sensible defaults for heartbeat monitoring.
func DefaultHeartbeatConfig() HeartbeatConfig {
	return HeartbeatConfig{
		Interval:       30 * time.Second,
		Timeout:        2 * time.Minute,
		MaxMissedBeats: 3,
	}
}

// RunProgress represents the current progress of a run for display.
type RunProgress struct {
	// Phase is the current execution phase
	Phase RunPhase `json:"phase"`

	// PhaseDescription is a human-readable description
	PhaseDescription string `json:"phaseDescription"`

	// PercentComplete is an estimate of overall progress (0-100)
	PercentComplete int `json:"percentComplete"`

	// CurrentAction describes what's happening now
	CurrentAction string `json:"currentAction,omitempty"`

	// ElapsedTime is how long the run has been active
	ElapsedTime time.Duration `json:"elapsedTime"`

	// EstimatedRemaining is an estimate of time left (if known)
	EstimatedRemaining *time.Duration `json:"estimatedRemaining,omitempty"`

	// LastUpdate is when progress was last reported
	LastUpdate time.Time `json:"lastUpdate"`
}

// PhaseToProgress converts a phase to approximate progress percentage.
func PhaseToProgress(phase RunPhase) int {
	switch phase {
	case RunPhaseQueued:
		return 0
	case RunPhaseInitializing:
		return 5
	case RunPhaseSandboxCreating:
		return 15
	case RunPhaseRunnerAcquiring:
		return 25
	case RunPhaseExecuting:
		return 50 // This phase takes most of the time
	case RunPhaseCollectingResults:
		return 85
	case RunPhaseAwaitingReview:
		return 90
	case RunPhaseApplying:
		return 95
	case RunPhaseCleaningUp:
		return 98
	case RunPhaseCompleted:
		return 100
	default:
		return 0
	}
}

// PhaseDescription returns a human-readable description of the phase.
func (p RunPhase) Description() string {
	switch p {
	case RunPhaseQueued:
		return "Waiting to start"
	case RunPhaseInitializing:
		return "Initializing execution environment"
	case RunPhaseSandboxCreating:
		return "Creating isolated workspace"
	case RunPhaseRunnerAcquiring:
		return "Acquiring agent runner"
	case RunPhaseExecuting:
		return "Agent is executing"
	case RunPhaseCollectingResults:
		return "Collecting results and artifacts"
	case RunPhaseAwaitingReview:
		return "Awaiting approval"
	case RunPhaseApplying:
		return "Applying approved changes"
	case RunPhaseCleaningUp:
		return "Cleaning up resources"
	case RunPhaseCompleted:
		return "Completed"
	default:
		return "Unknown phase"
	}
}
