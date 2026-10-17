package services

import (
	"context"
	"sync"
	"time"

	"agent-inbox/integrations"
)

// AsyncTrackerService manages background polling for async tool operations.
type AsyncTrackerService struct {
	mu sync.RWMutex

	// Active operations being tracked
	operations map[string]*AsyncOperation // toolCallID -> operation

	// ID-based subscription tracking
	subscriptions map[string]*Subscription // subscriptionID -> Subscription
	chatSubs      map[string][]string      // chatID -> []subscriptionID

	// Completion callbacks for AI conversation resumption (chatID -> callback channel)
	// When an async operation completes, an event is sent to allow the AI to continue
	completionCallbacks map[string]chan<- AsyncCompletionEvent

	// Cancellation for active pollers
	cancelFuncs map[string]context.CancelFunc // toolCallID -> cancel

	// Dependencies
	toolExecutor *integrations.ToolExecutor

	// Optional persistence layer for crash recovery
	// If nil, operations are tracked in-memory only (original behavior)
	repo AsyncOperationRepository
}

// Subscription represents an active subscriber for async status updates.
// Use SubscribeWithID to create and UnsubscribeByID to remove.
type Subscription struct {
	ID      string
	ChatID  string
	Channel chan AsyncStatusUpdate
}

// AsyncCompletionEvent is sent when an async operation reaches a terminal state.
// This is used to notify the AI conversation loop that results are available.
type AsyncCompletionEvent struct {
	ToolCallID string      `json:"tool_call_id"`
	ChatID     string      `json:"chat_id"`
	ToolName   string      `json:"tool_name"`
	Scenario   string      `json:"scenario"`
	Status     string      `json:"status"`           // "completed", "failed", "timeout", "cancelled"
	Result     interface{} `json:"result,omitempty"` // Final result if successful
	Error      string      `json:"error,omitempty"`  // Error message if failed
}

// AsyncBehavior describes how the runtime should track a long-running tool call.
//
// This intentionally lives in agent-inbox services instead of the old provider
// tool proto schema. Runtime tracking remains, while provider tool discovery is
// owned by Search Hub.
type AsyncBehavior struct {
	StatusPolling        *StatusPolling        `json:"status_polling,omitempty"`
	CompletionConditions *CompletionConditions `json:"completion_conditions,omitempty"`
	ProgressTracking     *ProgressTracking     `json:"progress_tracking,omitempty"`
	Cancellation         *CancellationBehavior `json:"cancellation,omitempty"`
}

// StatusPolling defines the status command and polling cadence for an operation.
type StatusPolling struct {
	StatusTool             string          `json:"status_tool,omitempty"`
	OperationIdField       string          `json:"operation_id_field,omitempty"`
	StatusToolIdParam      string          `json:"status_tool_id_param,omitempty"`
	PollIntervalSeconds    int32           `json:"poll_interval_seconds,omitempty"`
	MaxPollDurationSeconds int32           `json:"max_poll_duration_seconds,omitempty"`
	Backoff                *PollingBackoff `json:"backoff,omitempty"`
}

// PollingBackoff controls exponential polling backoff.
type PollingBackoff struct {
	InitialIntervalSeconds int32   `json:"initial_interval_seconds,omitempty"`
	MaxIntervalSeconds     int32   `json:"max_interval_seconds,omitempty"`
	Multiplier             float32 `json:"multiplier,omitempty"`
}

// CompletionConditions defines how a status response reaches a terminal state.
type CompletionConditions struct {
	StatusField       string   `json:"status_field,omitempty"`
	SuccessValues     []string `json:"success_values,omitempty"`
	FailureValues     []string `json:"failure_values,omitempty"`
	PendingValues     []string `json:"pending_values,omitempty"`
	ErrorField        string   `json:"error_field,omitempty"`
	ErrorDetailsField string   `json:"error_details_field,omitempty"`
	ResultField       string   `json:"result_field,omitempty"`
}

// ProgressTracking maps status response fields onto operation progress.
type ProgressTracking struct {
	ProgressField           string `json:"progress_field,omitempty"`
	MessageField            string `json:"message_field,omitempty"`
	PhaseField              string `json:"phase_field,omitempty"`
	CurrentStepField        string `json:"current_step_field,omitempty"`
	TotalStepsField         string `json:"total_steps_field,omitempty"`
	EstimatedRemainingField string `json:"estimated_remaining_field,omitempty"`
}

// CancellationBehavior defines how to cancel an external operation.
type CancellationBehavior struct {
	CancelTool           string `json:"cancel_tool,omitempty"`
	CancelToolIdParam    string `json:"cancel_tool_id_param,omitempty"`
	Graceful             bool   `json:"graceful,omitempty"`
	CancelTimeoutSeconds int32  `json:"cancel_timeout_seconds,omitempty"`
}

// AsyncOperation represents a tracked async tool execution.
type AsyncOperation struct {
	ToolCallID        string         `json:"tool_call_id"`
	ChatID            string         `json:"chat_id"`
	ToolName          string         `json:"tool_name"`
	Scenario          string         `json:"scenario"`
	ExternalRunID     string         `json:"external_run_id"`
	AsyncBehavior     *AsyncBehavior `json:"-"`
	Status            string         `json:"status"`
	Progress          *int           `json:"progress,omitempty"`
	Message           string         `json:"message,omitempty"`
	Phase             string         `json:"phase,omitempty"`
	Result            interface{}    `json:"result,omitempty"`
	Error             string         `json:"error,omitempty"`
	ConsecutiveErrors int            `json:"-"` // Track consecutive poll failures (not serialized)
	LastPollError     string         `json:"-"` // Most recent poll error message (not serialized)
	StartedAt         time.Time      `json:"started_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
	CompletedAt       *time.Time     `json:"completed_at,omitempty"`
}

// AsyncStatusUpdate represents a status update pushed to subscribers.
type AsyncStatusUpdate struct {
	ToolCallID string      `json:"tool_call_id"`
	ChatID     string      `json:"chat_id"`
	ToolName   string      `json:"tool_name"`
	Status     string      `json:"status"`
	Progress   *int        `json:"progress,omitempty"`
	Message    string      `json:"message,omitempty"`
	Phase      string      `json:"phase,omitempty"`
	Result     interface{} `json:"result,omitempty"`
	Error      string      `json:"error,omitempty"`
	IsTerminal bool        `json:"is_terminal"`
	UpdatedAt  time.Time   `json:"updated_at"`
}

// OperationSnapshot holds immutable operation fields for safe concurrent access.
// These fields are set once in StartTracking and never modified after.
// Reading these fields doesn't require holding the mutex.
type OperationSnapshot struct {
	ToolCallID    string
	ChatID        string
	ToolName      string
	Scenario      string
	ExternalRunID string
	AsyncBehavior *AsyncBehavior
	StartedAt     time.Time

	// Backoff configuration (extracted from AsyncBehavior for convenience)
	BackoffInitial    time.Duration // Initial polling interval
	BackoffMax        time.Duration // Maximum polling interval after backoff
	BackoffMultiplier float64       // Multiplier applied after each poll
}

// NewAsyncTrackerService creates a new AsyncTrackerService.
// The repository parameter is optional - if nil, operations are tracked in-memory only.
func NewAsyncTrackerService(executor *integrations.ToolExecutor, repo AsyncOperationRepository) *AsyncTrackerService {
	return &AsyncTrackerService{
		operations:          make(map[string]*AsyncOperation),
		subscriptions:       make(map[string]*Subscription),
		chatSubs:            make(map[string][]string),
		completionCallbacks: make(map[string]chan<- AsyncCompletionEvent),
		cancelFuncs:         make(map[string]context.CancelFunc),
		toolExecutor:        executor,
		repo:                repo,
	}
}

// SetRepository sets the optional persistence repository.
// Can be called after construction if the repository isn't available at creation time.
func (s *AsyncTrackerService) SetRepository(repo AsyncOperationRepository) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.repo = repo
}
