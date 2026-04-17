package services

import (
	"agent-inbox/integrations"
	"context"
	"sync"
	"time"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
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
	toolRegistry *ToolRegistry
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

// AsyncOperation represents a tracked async tool execution.
type AsyncOperation struct {
	ToolCallID        string                 `json:"tool_call_id"`
	ChatID            string                 `json:"chat_id"`
	ToolName          string                 `json:"tool_name"`
	Scenario          string                 `json:"scenario"`
	ExternalRunID     string                 `json:"external_run_id"`
	AsyncBehavior     *toolspb.AsyncBehavior `json:"-"`
	Status            string                 `json:"status"`
	Progress          *int                   `json:"progress,omitempty"`
	Message           string                 `json:"message,omitempty"`
	Phase             string                 `json:"phase,omitempty"`
	Result            interface{}            `json:"result,omitempty"`
	Error             string                 `json:"error,omitempty"`
	ConsecutiveErrors int                    `json:"-"` // Track consecutive poll failures (not serialized)
	LastPollError     string                 `json:"-"` // Most recent poll error message (not serialized)
	StartedAt         time.Time              `json:"started_at"`
	UpdatedAt         time.Time              `json:"updated_at"`
	CompletedAt       *time.Time             `json:"completed_at,omitempty"`
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
	AsyncBehavior *toolspb.AsyncBehavior
	StartedAt     time.Time

	// Backoff configuration (extracted from AsyncBehavior for convenience)
	BackoffInitial    time.Duration // Initial polling interval
	BackoffMax        time.Duration // Maximum polling interval after backoff
	BackoffMultiplier float64       // Multiplier applied after each poll
}

// NewAsyncTrackerService creates a new AsyncTrackerService.
// The repository parameter is optional - if nil, operations are tracked in-memory only.
func NewAsyncTrackerService(registry *ToolRegistry, executor *integrations.ToolExecutor, repo AsyncOperationRepository) *AsyncTrackerService {
	return &AsyncTrackerService{
		operations:          make(map[string]*AsyncOperation),
		subscriptions:       make(map[string]*Subscription),
		chatSubs:            make(map[string][]string),
		completionCallbacks: make(map[string]chan<- AsyncCompletionEvent),
		cancelFuncs:         make(map[string]context.CancelFunc),
		toolRegistry:        registry,
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
