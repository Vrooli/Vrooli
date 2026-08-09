package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

func (t RunEventType) IsTypedOperationalEvent() bool {
	switch t {
	case EventTypeRunnerFallbackAttempted,
		EventTypeRunnerFallbackExhausted,
		EventTypeModelFallbackAttempted,
		EventTypeModelFallbackExhausted,
		EventTypePolicyCandidateAttempt,
		EventTypeModelHealthTransition,
		EventTypeRunnerHealthTransition,
		EventTypeSandboxOperation,
		EventTypeHeartbeatMiss,
		EventTypeCheckpointFailure,
		EventTypeRetryAttempt:
		return true
	}
	return false
}

// LifecyclePhase identifies a stable transition in the spawn-to-finalize
// pipeline. The set is closed; new entries require a contract decision in
// scenarios/agent-manager/docs/internal/SEAMS.md.
type LifecyclePhase string

const (
	LifecyclePhaseSpawnEnqueued     LifecyclePhase = "spawn_enqueued"
	LifecyclePhaseSpawnStarted      LifecyclePhase = "spawn_started"
	LifecyclePhaseRunnerAcquired    LifecyclePhase = "runner_acquired"
	LifecyclePhaseRunnerExited      LifecyclePhase = "runner_exited"
	LifecyclePhaseFinalizeStarted   LifecyclePhase = "finalize_started"
	LifecyclePhaseFinalizeCompleted LifecyclePhase = "finalize_completed"
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
// TYPED OPERATIONAL EVENT
// =============================================================================

// TypedEventData carries a typed-operational payload as raw JSON bytes that
// already match the on-wire shape. It satisfies EventPayload so typed events
// can ride the existing run-event seam (Append, Stream, Get) without the
// store having to know each payload Go type.
//
// The eventlog package owns the payload struct definitions and the
// (event_type, schema_version) → Go-type dispatch; this struct is the
// transport between that package and the run-event plumbing.
type TypedEventData struct {
	// Type is the run-event type this payload belongs to. It must be one of
	// the typed-operational event categories (see RunEventType.IsTypedOperationalEvent).
	Type RunEventType `json:"-"`
	// Body is the already-marshaled JSON payload. MarshalJSON returns it
	// verbatim, and UnmarshalJSON stores the raw bytes back into it, so the
	// JSON round-trip preserves the eventlog-package payload shape exactly.
	Body json.RawMessage `json:"-"`
}

func (d *TypedEventData) EventType() RunEventType {
	if d == nil {
		return ""
	}
	return d.Type
}

func (d *TypedEventData) isEventPayload() {}

// MarshalJSON returns Body verbatim so the on-wire shape is the
// eventlog-package payload, not a wrapper.
func (d *TypedEventData) MarshalJSON() ([]byte, error) {
	if d == nil || len(d.Body) == 0 {
		return []byte("{}"), nil
	}
	return d.Body, nil
}

// UnmarshalJSON captures the raw bytes into Body for later typed decoding
// through the eventlog dispatch table.
func (d *TypedEventData) UnmarshalJSON(b []byte) error {
	d.Body = append(d.Body[:0], b...)
	return nil
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
	Role        string                  `json:"role"`                  // user, assistant, system
	Content     string                  `json:"content"`               // Message content
	Attachments []MessageAttachmentInfo `json:"attachments,omitempty"` // Image/file attachments

	// Optional provider evidence. Empty fields mean the provider did not emit
	// that fact; callers must not infer it from transcript position.
	MessageID          string `json:"messageId,omitempty"`
	ConversationID     string `json:"conversationId,omitempty"`
	TurnID             string `json:"turnId,omitempty"`
	ProviderOrigin     string `json:"providerOrigin,omitempty"`
	CompletionReason   string `json:"completionReason,omitempty"`
	Terminal           bool   `json:"terminal,omitempty"`
	ParentMessageID    string `json:"parentMessageId,omitempty"`
	ProviderEventType  string `json:"providerEventType,omitempty"`
	RawEvidenceRef     string `json:"rawEvidenceRef,omitempty"`
	EvidenceOnly       bool   `json:"evidenceOnly,omitempty"`
	EvidenceForEventID string `json:"evidenceForEventId,omitempty"`
}

// MessageAttachmentInfo stores metadata about attachments included with a message.
// Used by the UI to render image thumbnails inline.
type MessageAttachmentInfo struct {
	ID          string `json:"id"`
	FileName    string `json:"file_name"`
	ContentType string `json:"content_type"`
	URL         string `json:"url"` // Serving URL relative to API base
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

// NewProviderMessageEvent creates a message carrying only evidence the
// provider actually emitted. The evidence value is copied so codecs can reuse
// their decode structs without mutating an already-emitted event.
func NewProviderMessageEvent(runID uuid.UUID, role, content string, evidence MessageEventData) *RunEvent {
	evidence.Role = role
	evidence.Content = content
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeMessage,
		Timestamp: time.Now(),
		Data:      &evidence,
	}
}

// NewMessageEventWithAttachments creates a message event that includes attachment metadata.
func NewMessageEventWithAttachments(runID uuid.UUID, role, content string, attachments []MessageAttachmentInfo) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeMessage,
		Timestamp: time.Now(),
		Data:      &MessageEventData{Role: role, Content: content, Attachments: attachments},
	}
}

// =============================================================================
// MESSAGE DELETED EVENT
// =============================================================================

// MessageDeletedEventData marks a message event as deleted/redacted.
type MessageDeletedEventData struct {
	TargetEventID string `json:"targetEventId"`
}

func (d *MessageDeletedEventData) EventType() RunEventType { return EventTypeMessageDeleted }
func (d *MessageDeletedEventData) isEventPayload()         {}

// NewMessageDeletedEvent creates a new message deletion event.
func NewMessageDeletedEvent(runID uuid.UUID, targetEventID string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeMessageDeleted,
		Timestamp: time.Now(),
		Data:      &MessageDeletedEventData{TargetEventID: targetEventID},
	}
}

// =============================================================================
// TOOL CALL EVENT
// =============================================================================

// ToolCallEventData contains data for tool invocation events.
type ToolCallEventData struct {
	ToolName   string                 `json:"toolName"`             // Name of the tool being called
	ToolCallID string                 `json:"toolCallId,omitempty"` // Correlation ID linking to the tool_result
	Input      map[string]interface{} `json:"input"`                // Tool input parameters
}

func (d *ToolCallEventData) EventType() RunEventType { return EventTypeToolCall }
func (d *ToolCallEventData) isEventPayload()         {}

// NewToolCallEvent creates a new tool call event.
// toolCallID is the correlation ID (e.g. "toolu_01GXZ...") that links this call to its result.
func NewToolCallEvent(runID uuid.UUID, toolName, toolCallID string, input map[string]interface{}) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeToolCall,
		Timestamp: time.Now(),
		Data:      &ToolCallEventData{ToolName: toolName, ToolCallID: toolCallID, Input: input},
	}
}

// =============================================================================
// TOOL RESULT EVENT
// =============================================================================

// ToolResultEventData contains data for tool result events.
type ToolResultEventData struct {
	ToolName   string `json:"toolName"`             // Display name of the tool (e.g., "Write", "bash")
	ToolCallID string `json:"toolCallId,omitempty"` // Tool invocation ID (e.g., "toolu_01GXZ...")
	Output     string `json:"output"`               // Tool output (success)
	Error      string `json:"error,omitempty"`      // Error message (if failed)
	Success    bool   `json:"success"`              // Whether the tool call succeeded
	ExitCode   *int   `json:"exitCode,omitempty"`   // Provider-reported process exit code, when available
	DurationMS *int64 `json:"durationMs,omitempty"` // Provider-reported tool duration, when available
}

func (d *ToolResultEventData) EventType() RunEventType { return EventTypeToolResult }
func (d *ToolResultEventData) isEventPayload()         {}

// NewToolResultEvent creates a new tool result event.
// toolName is the display name (e.g., "Write"), toolCallID is the invocation ID.
func NewToolResultEvent(runID uuid.UUID, toolName, toolCallID, output string, err error) *RunEvent {
	data := &ToolResultEventData{
		ToolName:   toolName,
		ToolCallID: toolCallID,
		Output:     output,
		Success:    err == nil,
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
	Code       string                 `json:"code"`                 // Machine-readable error code
	Message    string                 `json:"message"`              // Human-readable error message
	Retryable  bool                   `json:"retryable"`            // Whether the error is retryable
	Recovery   RecoveryAction         `json:"recovery,omitempty"`   // Suggested recovery action
	StackTrace string                 `json:"stackTrace,omitempty"` // Optional stack trace
	Details    map[string]interface{} `json:"details,omitempty"`    // Structured error details (e.g., conflicting sandboxes)
}

// =============================================================================
// RATE LIMIT EVENT
// =============================================================================

// RateLimitEventData contains data for rate limit events.
type RateLimitEventData struct {
	LimitType   string     `json:"limitType"`             // Type of limit: "5_hour", "daily", "weekly", "token"
	ResetTime   *time.Time `json:"resetTime,omitempty"`   // When the limit resets
	RetryAfter  int        `json:"retryAfter,omitempty"`  // Seconds until retry is safe
	CurrentUsed int        `json:"currentUsed,omitempty"` // Current usage count
	Limit       int        `json:"limit,omitempty"`       // The limit that was hit
	Message     string     `json:"message"`               // Human-readable message
}

func (d *RateLimitEventData) EventType() RunEventType { return EventTypeError }
func (d *RateLimitEventData) isEventPayload()         {}

// NewRateLimitEvent creates a new rate limit event.
func NewRateLimitEvent(runID uuid.UUID, limitType, message string, resetTime *time.Time, retryAfter int) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeError,
		Timestamp: time.Now(),
		Data: &RateLimitEventData{
			LimitType:  limitType,
			ResetTime:  resetTime,
			RetryAfter: retryAfter,
			Message:    message,
		},
	}
}

// =============================================================================
// COST EVENT
// =============================================================================

// PayloadKind identifies the independent facts carried by metric events.
const (
	PayloadKindUsage  = "usage"
	PayloadKindCharge = "charge"
)

// UsageEventData contains provider-independent consumption. It is emitted
// even when pricing is unavailable or the run is free under its billing mode.
type UsageEventData struct {
	PayloadKind         string `json:"payloadKind"`
	InputTokens         int    `json:"inputTokens"`
	OutputTokens        int    `json:"outputTokens"`
	CacheCreationTokens int    `json:"cacheCreationTokens,omitempty"`
	CacheReadTokens     int    `json:"cacheReadTokens,omitempty"`
	Turns               int    `json:"turns,omitempty"`
	TurnIndex           int    `json:"turnIndex,omitempty"`
	// ReconciliationAuthority marks the terminal provider usage snapshot as
	// authoritative for run-total reconciliation. Per-turn usage remains
	// available for attribution but is not added again when this is present.
	ReconciliationAuthority bool             `json:"reconciliationAuthority,omitempty"`
	ServiceTier             string           `json:"serviceTier,omitempty"`
	Model                   string           `json:"model,omitempty"`
	RunnerType              string           `json:"runnerType,omitempty"`
	WebSearchRequests       int              `json:"webSearchRequests,omitempty"`
	ServerToolUseRequests   int              `json:"serverToolUseRequests,omitempty"`
	Charge                  *ChargeEventData `json:"charge,omitempty"`
}

func (d *UsageEventData) EventType() RunEventType { return EventTypeMetric }
func (d *UsageEventData) isEventPayload()         {}

// ChargeBasis explains why a charge is present, zero, or unavailable.
type ChargeBasis string

const (
	ChargeBasisMetered      ChargeBasis = "metered"
	ChargeBasisSubscription ChargeBasis = "subscription"
	ChargeBasisLocal        ChargeBasis = "local"
	ChargeBasisUnpriced     ChargeBasis = "unpriced"
	ChargeBasisUnknown      ChargeBasis = "unknown"
)

func AllChargeBases() []ChargeBasis {
	return []ChargeBasis{ChargeBasisMetered, ChargeBasisSubscription, ChargeBasisLocal, ChargeBasisUnpriced, ChargeBasisUnknown}
}

// ChargeBasisIsBillable is exhaustive by design.
func ChargeBasisIsBillable(basis ChargeBasis) bool {
	switch basis {
	case ChargeBasisMetered:
		return true
	case ChargeBasisSubscription, ChargeBasisLocal, ChargeBasisUnpriced, ChargeBasisUnknown:
		return false
	default:
		panic("unhandled charge basis: " + string(basis))
	}
}

// ChargeEventData contains billing-context-dependent charge. A nil amount
// means the run was not priced; zero is reserved for legitimate free modes.
type ChargeEventData struct {
	PayloadKind         string      `json:"payloadKind"`
	Basis               ChargeBasis `json:"basis"`
	AmountMicroUSD      *int64      `json:"amountMicroUsd"`
	InputMicroUSD       *int64      `json:"inputMicroUsd,omitempty"`
	OutputMicroUSD      *int64      `json:"outputMicroUsd,omitempty"`
	CacheReadMicroUSD   *int64      `json:"cacheReadMicroUsd,omitempty"`
	CacheCreateMicroUSD *int64      `json:"cacheCreateMicroUsd,omitempty"`
	Currency            string      `json:"currency"`
	Model               string      `json:"model,omitempty"`
	RunnerType          string      `json:"runnerType,omitempty"`
	ChargeReason        string      `json:"chargeReason,omitempty"`
}

func (d *ChargeEventData) EventType() RunEventType { return EventTypeMetric }
func (d *ChargeEventData) isEventPayload()         {}

// LegacyChargeBasis maps historical billing-source strings into the new basis
// while old event rows are normalized on read.
func LegacyChargeBasis(source string) ChargeBasis {
	switch source {
	case "runner_reported", "provider_usage_api", "pricing_table_estimate":
		return ChargeBasisMetered
	case "unknown":
		return ChargeBasisUnknown
	default:
		return ChargeBasisUnknown
	}
}

// =============================================================================
// PROGRESS EVENT
// =============================================================================

// ProgressEventData contains data for progress tracking events.
type ProgressEventData struct {
	Phase              RunPhase `json:"phase"`
	PercentComplete    int      `json:"percentComplete"`
	CurrentAction      string   `json:"currentAction,omitempty"`
	TurnsCompleted     int      `json:"turnsCompleted,omitempty"`
	TurnsTotal         int      `json:"turnsTotal,omitempty"` // 0 means unlimited
	TokensUsed         int      `json:"tokensUsed,omitempty"`
	ElapsedSeconds     float64  `json:"elapsedSeconds,omitempty"`
	EstimatedRemaining float64  `json:"estimatedRemaining,omitempty"` // seconds, -1 if unknown
}

func (d *ProgressEventData) EventType() RunEventType { return EventTypeStatus }
func (d *ProgressEventData) isEventPayload()         {}

// NewProgressEvent creates a new progress tracking event.
func NewProgressEvent(runID uuid.UUID, phase RunPhase, percent int, action string) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeStatus,
		Timestamp: time.Now(),
		Data: &ProgressEventData{
			Phase:           phase,
			PercentComplete: percent,
			CurrentAction:   action,
		},
	}
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
			Details:   err.Details(),
		},
	}
}

// =============================================================================
// COMPACTION EVENT
// =============================================================================

// CompactionEventData represents a context compaction/summarization event.
type CompactionEventData struct {
	Summary           string `json:"summary"`
	Trigger           string `json:"trigger"`         // "manual" or "auto"
	Focus             string `json:"focus,omitempty"` // Optional focus instruction
	MessagesCompacted int64  `json:"messagesCompacted"`
	TokensBefore      int64  `json:"tokensBefore"`
	TokensAfter       int64  `json:"tokensAfter"`
	OriginalCommand   string `json:"originalCommand,omitempty"`
}

func (d *CompactionEventData) EventType() RunEventType { return EventTypeCompaction }
func (d *CompactionEventData) isEventPayload()         {}

// NewCompactionEvent creates a new compaction event.
// trigger should be "manual" or "auto".
// focus is optional (empty string if not specified).
func NewCompactionEvent(
	runID uuid.UUID,
	summary string,
	trigger string,
	focus string,
	messagesCompacted int64,
	tokensBefore int64,
	tokensAfter int64,
	originalCommand string,
) *RunEvent {
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeCompaction,
		Timestamp: time.Now(),
		Data: &CompactionEventData{
			Summary:           summary,
			Trigger:           trigger,
			Focus:             focus,
			MessagesCompacted: messagesCompacted,
			TokensBefore:      tokensBefore,
			TokensAfter:       tokensAfter,
			OriginalCommand:   originalCommand,
		},
	}
}

// =============================================================================
// LIFECYCLE EVENT
// =============================================================================

// LifecycleEventData carries timing + classification fields for the
// stable spawn → finalize transitions enumerated by [LifecyclePhase].
//
// Lifecycle events are emitted only via the helpers in
// `internal/orchestration/obs/events.go`; ad-hoc construction outside
// that package is a contract violation (see SEAMS.md, contract decision
// "Lifecycle events are emitted through obs/events.go only").
type LifecycleEventData struct {
	Phase        LifecyclePhase `json:"phase"`
	Message      string         `json:"message,omitempty"`
	DurationMS   int64          `json:"durationMs,omitempty"`
	SandboxID    string         `json:"sandboxId,omitempty"`
	RunnerType   string         `json:"runnerType,omitempty"`
	LauncherType string         `json:"launcherType,omitempty"`
	QueueDepth   int            `json:"queueDepth,omitempty"`
	ActiveCount  int            `json:"activeCount,omitempty"`
	ExitCode     *int           `json:"exitCode,omitempty"`
	TerminalCode string         `json:"terminalCode,omitempty"`
}

func (d *LifecycleEventData) EventType() RunEventType { return EventTypeLifecycle }
func (d *LifecycleEventData) isEventPayload()         {}

// NewLifecycleEvent constructs a lifecycle event for the given phase.
// Caller-supplied data is shallow-copied so the event is safe to emit
// without sharing pointer state with the caller.
func NewLifecycleEvent(runID uuid.UUID, data LifecycleEventData) *RunEvent {
	d := data
	return &RunEvent{
		ID:        uuid.New(),
		RunID:     runID,
		EventType: EventTypeLifecycle,
		Timestamp: time.Now(),
		Data:      &d,
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
	RequireSandbox *bool `json:"requireSandbox,omitempty"`
	AllowInPlace   *bool `json:"allowInPlace,omitempty"`

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
