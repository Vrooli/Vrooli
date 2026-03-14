// Package services provides application services for the Agent Inbox scenario.
//
// This file implements the AsyncTrackerService for tracking long-running tool
// operations and providing status updates via Server-Sent Events (SSE).
//
// # ARCHITECTURE OVERVIEW
//
// The async tracking system enables tools to run asynchronously while keeping
// the AI conversation loop informed of progress and results. This is essential
// for long-running operations like code generation, browser automation, or
// data processing.
//
// ## Key Components
//
//   - AsyncTrackerService: Central coordinator managing operation lifecycle
//   - AsyncOperation: State container for a single tracked operation
//   - Subscription: SSE channel for real-time UI updates
//   - CompletionCallback: Channel for AI conversation loop notification
//
// ## Data Flow
//
//	┌─────────────────────────────────────────────────────────────────────┐
//	│                    AI Requests Tool                                  │
//	│                          │                                           │
//	│                          ▼                                           │
//	│            ┌─────────────────────────────┐                          │
//	│            │   Tool Execution Returns    │                          │
//	│            │   { run_id: "abc123" }      │                          │
//	│            └─────────────┬───────────────┘                          │
//	│                          │                                           │
//	│                          ▼                                           │
//	│            ┌─────────────────────────────┐                          │
//	│            │    StartTracking() called   │                          │
//	│            │    - Extracts run_id        │                          │
//	│            │    - Starts poll goroutine  │                          │
//	│            └─────────────┬───────────────┘                          │
//	│                          │                                           │
//	│           ┌──────────────┴──────────────┐                           │
//	│           ▼                              ▼                           │
//	│  ┌─────────────────┐          ┌─────────────────┐                   │
//	│  │ SSE Subscribers │          │  Poll Loop      │                   │
//	│  │ (UI updates)    │◀─────────│  (background)   │                   │
//	│  └─────────────────┘  updates └────────┬────────┘                   │
//	│                                        │                             │
//	│                                        ▼                             │
//	│                          ┌─────────────────────────┐                │
//	│                          │ Calls status tool       │                │
//	│                          │ repeatedly until done   │                │
//	│                          └─────────────┬───────────┘                │
//	│                                        │                             │
//	│                                        ▼                             │
//	│                          ┌─────────────────────────┐                │
//	│                          │ On terminal status:     │                │
//	│                          │ - Notify SSE subs       │                │
//	│                          │ - Trigger completion CB │                │
//	│                          │ - AI loop continues     │                │
//	│                          └─────────────────────────┘                │
//	└─────────────────────────────────────────────────────────────────────┘
//
// ## Subscription Systems
//
// There are two notification systems serving different consumers:
//
// 1. SSE Subscribers (SubscribeWithID/UnsubscribeByID):
//   - Used by UI clients for real-time progress display
//   - Buffered channels prevent blocking on slow consumers
//   - ID-based tracking for safe unsubscription
//
// 2. Completion Callbacks (RegisterCompletionCallback/UnregisterCompletionCallback):
//   - Used by AI conversation loop to wait for results
//   - One callback per chat (multiple operations fan into same channel)
//   - Enables auto-continuation after async tools complete
//
// Note: The deprecated Subscribe/Unsubscribe methods use pointer comparison
// which is fragile. Use SubscribeWithID/UnsubscribeByID for new code.
//
// ## Concurrency Model
//
// Operations are accessed from multiple goroutines:
//   - HTTP handlers (read operations, start tracking)
//   - Poll goroutines (update status, trigger callbacks)
//   - Cleanup routine (remove stale operations)
//
// Thread safety is ensured via:
//   - sync.RWMutex for operation map access
//   - OperationSnapshot for lock-free polling config access
//   - Non-blocking channel sends (with logging on full channels)
//
// ## Configuration
//
// Tunable parameters are defined in async_config.go:
//   - Poll intervals and timeouts
//   - Channel buffer sizes
//   - Cleanup intervals and retention
//
// ## Testing
//
// The AsyncTrackerInterface in interfaces.go enables mocking for unit tests.
// Integration tests can use AddTestOperation() to inject test data.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"agent-inbox/domain"
	"agent-inbox/integrations"

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

// snapshotOperation creates a read-only snapshot of immutable operation fields.
// Call this at the start of pollLoop to avoid repeated lock acquisitions.
func (s *AsyncTrackerService) snapshotOperation(toolCallID string) (*OperationSnapshot, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	op, ok := s.operations[toolCallID]
	if !ok || op == nil {
		return nil, false
	}

	snap := &OperationSnapshot{
		ToolCallID:    op.ToolCallID,
		ChatID:        op.ChatID,
		ToolName:      op.ToolName,
		Scenario:      op.Scenario,
		ExternalRunID: op.ExternalRunID,
		AsyncBehavior: op.AsyncBehavior, // Pointer to immutable proto struct
		StartedAt:     op.StartedAt,
	}

	// Extract backoff configuration from proto (use defaults if not configured)
	snap.BackoffInitial = DefaultPollInterval
	snap.BackoffMax = DefaultPollInterval // No backoff by default
	snap.BackoffMultiplier = 1.0          // No backoff by default

	if op.AsyncBehavior != nil && op.AsyncBehavior.StatusPolling != nil {
		polling := op.AsyncBehavior.StatusPolling

		// Use configured base interval if valid
		if polling.PollIntervalSeconds > 0 {
			interval := time.Duration(polling.PollIntervalSeconds) * time.Second
			if interval >= MinPollInterval {
				snap.BackoffInitial = interval
				snap.BackoffMax = interval // Default max to initial if no backoff config
			}
		}

		// Apply backoff config if present
		if backoff := polling.GetBackoff(); backoff != nil {
			if backoff.InitialIntervalSeconds > 0 {
				initial := time.Duration(backoff.InitialIntervalSeconds) * time.Second
				if initial >= MinPollInterval {
					snap.BackoffInitial = initial
				}
			}
			if backoff.MaxIntervalSeconds > 0 {
				snap.BackoffMax = time.Duration(backoff.MaxIntervalSeconds) * time.Second
			}
			if backoff.Multiplier >= 1.0 {
				snap.BackoffMultiplier = float64(backoff.Multiplier)
			}
		}
	}

	return snap, true
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

// RecoverOperations loads active operations from the database and resumes polling.
// Called during service initialization for crash recovery.
// Uses fresh status check approach - queries current status before resuming polling.
func (s *AsyncTrackerService) RecoverOperations(ctx context.Context) error {
	s.mu.RLock()
	repo := s.repo
	s.mu.RUnlock()

	if repo == nil {
		return nil // No repository configured, nothing to recover
	}

	records, err := repo.GetAllActiveAsyncOperations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get active operations for recovery: %w", err)
	}

	if len(records) == 0 {
		log.Printf("[INFO] No async operations to recover")
		return nil
	}

	log.Printf("[INFO] Recovering %d async operations", len(records))

	for _, record := range records {
		// Reconstruct the AsyncOperation from the record
		var asyncBehavior toolspb.AsyncBehavior
		if err := json.Unmarshal(record.AsyncBehavior, &asyncBehavior); err != nil {
			log.Printf("[WARN] Failed to unmarshal async behavior for %s: %v", record.ToolCallID, err)
			continue
		}

		op := &AsyncOperation{
			ToolCallID:    record.ToolCallID,
			ChatID:        record.ChatID,
			ToolName:      record.ToolName,
			Scenario:      record.ScenarioName,
			ExternalRunID: record.OperationID,
			AsyncBehavior: &asyncBehavior,
			Status:        record.Status,
			Progress:      record.Progress,
			Message:       record.Message,
			Phase:         record.Phase,
			Error:         record.Error,
			StartedAt:     record.StartedAt,
			UpdatedAt:     record.UpdatedAt,
			CompletedAt:   record.CompletedAt,
		}

		// Parse result if present
		if len(record.Result) > 0 {
			var result interface{}
			if err := json.Unmarshal(record.Result, &result); err == nil {
				op.Result = result
			}
		}

		// Store in memory
		s.mu.Lock()
		s.operations[record.ToolCallID] = op
		s.mu.Unlock()

		// Start recovery goroutine with fresh status check
		go s.recoverOperation(ctx, op)
	}

	return nil
}

// recoverOperation performs a fresh status check and resumes polling if still active.
// This is safer than resuming mid-poll as it gets the current state from the external service.
func (s *AsyncTrackerService) recoverOperation(ctx context.Context, op *AsyncOperation) {
	log.Printf("[INFO] Recovering operation %s (status=%s)", op.ToolCallID, op.Status)

	// Create snapshot for the status check
	snap, ok := s.snapshotOperation(op.ToolCallID)
	if !ok {
		log.Printf("[WARN] Recovery: operation %s disappeared before recovery", op.ToolCallID)
		return
	}

	// Perform immediate fresh status check
	statusResult, err := s.callStatusToolWithSnapshot(ctx, snap)
	if err != nil {
		log.Printf("[WARN] Recovery status check failed for %s: %v", op.ToolCallID, err)
		// Continue to poll loop anyway - it will handle retries
	} else {
		// Process the result - this may mark as complete if operation finished while we were down
		conditions := snap.AsyncBehavior.CompletionConditions
		isTerminal, status := s.processStatusResult(op, statusResult, conditions)
		if isTerminal {
			log.Printf("[INFO] Recovery: operation %s already completed (status=%s)", op.ToolCallID, status)
			return
		}
	}

	// Create cancellable context for polling
	pollCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancelFuncs[op.ToolCallID] = cancel
	s.mu.Unlock()

	// Start polling loop
	s.pollLoop(pollCtx, op)
}

// Shutdown gracefully stops all active polling and persists state.
// Should be called before server shutdown.
func (s *AsyncTrackerService) Shutdown(ctx context.Context) error {
	log.Printf("[INFO] AsyncTrackerService shutting down...")

	s.mu.Lock()
	repo := s.repo
	toolCallIDs := make([]string, 0, len(s.cancelFuncs))

	// Cancel all polling goroutines
	for toolCallID, cancel := range s.cancelFuncs {
		toolCallIDs = append(toolCallIDs, toolCallID)
		cancel()
	}
	s.mu.Unlock()

	// Mark interrupted operations in database (outside lock to avoid blocking)
	if repo != nil {
		for _, toolCallID := range toolCallIDs {
			if err := repo.UpdateAsyncOperationStatus(ctx, toolCallID, "polling"); err != nil {
				log.Printf("[WARN] Failed to update status for %s during shutdown: %v", toolCallID, err)
			}
		}
	}

	log.Printf("[INFO] AsyncTrackerService shutdown complete (%d operations paused)", len(toolCallIDs))
	return nil
}

// StartTracking begins tracking an async tool operation.
// This should be called after executing a tool that has AsyncBehavior defined.
func (s *AsyncTrackerService) StartTracking(
	ctx context.Context,
	toolCallID string,
	chatID string,
	toolName string,
	scenario string,
	toolResult interface{},
	asyncBehavior *toolspb.AsyncBehavior,
) error {
	if asyncBehavior == nil || asyncBehavior.StatusPolling == nil {
		return fmt.Errorf("no async behavior configuration provided")
	}

	// Extract the operation ID from the tool result
	externalRunID, err := s.extractOperationID(toolResult, asyncBehavior.StatusPolling.OperationIdField)
	if err != nil {
		return fmt.Errorf("failed to extract operation ID: %w", err)
	}

	// Create the operation record
	op := &AsyncOperation{
		ToolCallID:    toolCallID,
		ChatID:        chatID,
		ToolName:      toolName,
		Scenario:      scenario,
		ExternalRunID: externalRunID,
		AsyncBehavior: asyncBehavior,
		Status:        AsyncStatusPending,
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Store the operation and build initial update while holding lock
	s.mu.Lock()
	s.operations[toolCallID] = op
	initialUpdate := buildUpdateFromOp(op, false)
	repo := s.repo // Capture repo reference while holding lock
	s.mu.Unlock()

	// Persist to database if repository is configured (graceful degradation if fails)
	if repo != nil {
		asyncBehaviorJSON, err := json.Marshal(asyncBehavior)
		if err != nil {
			log.Printf("[WARN] Failed to marshal async behavior for %s: %v", toolCallID, err)
		} else {
			record := &AsyncOperationRecord{
				ToolCallID:    toolCallID,
				ChatID:        chatID,
				ToolName:      toolName,
				ScenarioName:  scenario,
				OperationID:   externalRunID,
				Status:        AsyncStatusPending,
				AsyncBehavior: asyncBehaviorJSON,
				StartedAt:     op.StartedAt,
				UpdatedAt:     op.UpdatedAt,
			}
			if err := repo.CreateAsyncOperation(ctx, record); err != nil {
				log.Printf("[WARN] Failed to persist async operation %s: %v", toolCallID, err)
				// Continue with in-memory tracking
			}
		}
	}

	// Push initial update to subscribers so UI shows the operation immediately
	s.pushUpdateData(chatID, initialUpdate)

	// Create cancellable context for polling
	pollCtx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancelFuncs[toolCallID] = cancel
	s.mu.Unlock()

	// Start background polling
	go s.pollLoop(pollCtx, op)

	log.Printf("Started async tracking for %s/%s (toolCallID=%s, runID=%s)",
		scenario, toolName, toolCallID, externalRunID)

	return nil
}

// StopTracking cancels tracking for an operation and marks it as cancelled.
// The operation is kept in memory for a grace period to allow clients to query its status.
func (s *AsyncTrackerService) StopTracking(toolCallID string) {
	s.mu.Lock()
	// Cancel the polling goroutine
	if cancel, ok := s.cancelFuncs[toolCallID]; ok {
		cancel()
		delete(s.cancelFuncs, toolCallID)
	}

	// Mark the operation as cancelled if it exists and hasn't completed yet
	op := s.operations[toolCallID]
	if op != nil && op.CompletedAt == nil {
		op.Status = AsyncStatusCancelled
		op.Error = "Operation cancelled"
		now := time.Now()
		op.CompletedAt = &now
		op.UpdatedAt = now
	}
	repo := s.repo // Capture repo reference while holding lock
	s.mu.Unlock()

	// Persist the cancellation to database
	if repo != nil && op != nil {
		record := &AsyncOperationRecord{
			ToolCallID:  toolCallID,
			Status:      op.Status,
			Error:       op.Error,
			UpdatedAt:   op.UpdatedAt,
			CompletedAt: op.CompletedAt,
		}
		if err := repo.UpdateAsyncOperation(context.Background(), record); err != nil {
			log.Printf("[WARN] Failed to persist cancelled operation %s: %v", toolCallID, err)
		}
	}

	// Trigger completion callback outside of lock
	if op != nil {
		s.triggerCompletionCallback(op, AsyncStatusCancelled)
	}
}

// RemoveOperation removes an operation from tracking.
// This should only be called after the operation has completed and results have been processed.
func (s *AsyncTrackerService) RemoveOperation(toolCallID string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.operations, toolCallID)
	delete(s.cancelFuncs, toolCallID)
}

// CleanupStaleOperations removes completed operations older than the retention duration.
// Returns the number of operations removed.
func (s *AsyncTrackerService) CleanupStaleOperations(retention time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-retention)
	var removed int

	for id, op := range s.operations {
		// Only remove completed operations that are older than the retention period
		if op.CompletedAt != nil && op.CompletedAt.Before(cutoff) {
			delete(s.operations, id)
			delete(s.cancelFuncs, id)
			removed++
		}
	}

	if removed > 0 {
		log.Printf("[INFO] Cleaned up %d stale async operations", removed)
	}
	return removed
}

// StartCleanupRoutine starts a background routine that periodically cleans up stale operations.
// Call this once during service initialization.
func (s *AsyncTrackerService) StartCleanupRoutine(ctx context.Context, interval, retention time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				log.Printf("[INFO] Async tracker cleanup routine stopped")
				return
			case <-ticker.C:
				s.CleanupStaleOperations(retention)
			}
		}
	}()
	log.Printf("[INFO] Started async tracker cleanup routine (interval=%v, retention=%v)", interval, retention)
}

// GetOperationCount returns the number of tracked operations (for monitoring/testing).
func (s *AsyncTrackerService) GetOperationCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.operations)
}

// GetOperation returns an operation by ID.
func (s *AsyncTrackerService) GetOperation(toolCallID string) *AsyncOperation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.operations[toolCallID]
}

// GetActiveOperations returns all active operations for a chat.
func (s *AsyncTrackerService) GetActiveOperations(chatID string) []*AsyncOperation {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*AsyncOperation
	for _, op := range s.operations {
		if op.ChatID == chatID && op.CompletedAt == nil {
			result = append(result, op)
		}
	}
	return result
}

// SubscribeWithID creates a subscription with a unique ID for safe tracking.
// Returns a Subscription that can be passed to UnsubscribeByID.
//
// This is the preferred method over Subscribe as it uses explicit IDs instead
// of fragile pointer comparison for unsubscription.
//
// The returned channel is buffered (see SubscriberChannelBufferSize in async_config.go).
// If the buffer fills, updates are dropped with a warning log.
func (s *AsyncTrackerService) SubscribeWithID(chatID string) *Subscription {
	ch := make(chan AsyncStatusUpdate, SubscriberChannelBufferSize)
	subID := fmt.Sprintf("%s_%d", chatID, time.Now().UnixNano())

	sub := &Subscription{
		ID:      subID,
		ChatID:  chatID,
		Channel: ch,
	}

	s.mu.Lock()
	s.subscriptions[subID] = sub
	s.chatSubs[chatID] = append(s.chatSubs[chatID], subID)
	s.mu.Unlock()

	return sub
}

// UnsubscribeByID removes a subscription by its ID.
// This is safer than Unsubscribe as it uses explicit IDs instead of pointer comparison.
func (s *AsyncTrackerService) UnsubscribeByID(sub *Subscription) {
	if sub == nil {
		return
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Remove from subscriptions map
	delete(s.subscriptions, sub.ID)

	// Remove from chatSubs list
	subs := s.chatSubs[sub.ChatID]
	for i, id := range subs {
		if id == sub.ID {
			s.chatSubs[sub.ChatID] = append(subs[:i], subs[i+1:]...)
			break
		}
	}

	// Clean up empty lists
	if len(s.chatSubs[sub.ChatID]) == 0 {
		delete(s.chatSubs, sub.ChatID)
	}

	// Close the channel
	close(sub.Channel)
}

// RegisterCompletionCallback registers a channel to receive completion events for a chat.
//
// The AI conversation loop uses this to wait for async operations to complete.
// When an async operation reaches a terminal state, an AsyncCompletionEvent is
// sent to this channel, allowing the AI to continue with the results.
//
// Returns a receive-only channel. Call UnregisterCompletionCallback when done.
// The channel is buffered (see CompletionCallbackBufferSize in async_config.go)
// to handle multiple concurrent async operations completing.
//
// Note: Only one callback can be registered per chat. Registering a new callback
// replaces any existing callback (the old channel is NOT closed).
func (s *AsyncTrackerService) RegisterCompletionCallback(chatID string) <-chan AsyncCompletionEvent {
	ch := make(chan AsyncCompletionEvent, CompletionCallbackBufferSize)

	s.mu.Lock()
	s.completionCallbacks[chatID] = ch
	s.mu.Unlock()

	log.Printf("[DEBUG] Registered completion callback for chat %s", chatID)
	return ch
}

// UnregisterCompletionCallback removes a completion callback for a chat.
// Should be called when the AI conversation loop stops waiting.
func (s *AsyncTrackerService) UnregisterCompletionCallback(chatID string) {
	s.mu.Lock()
	if ch, ok := s.completionCallbacks[chatID]; ok {
		close(ch)
		delete(s.completionCallbacks, chatID)
	}
	s.mu.Unlock()

	log.Printf("[DEBUG] Unregistered completion callback for chat %s", chatID)
}

// triggerCompletionCallback sends a completion event to the registered callback.
// Called when an operation reaches a terminal state (completed, failed, timeout, cancelled).
// Also persists the event to the database for multi-consumer support.
// MUST be called while NOT holding the mutex (to avoid deadlock).
func (s *AsyncTrackerService) triggerCompletionCallback(op *AsyncOperation, status string) {
	s.mu.RLock()
	ch, ok := s.completionCallbacks[op.ChatID]
	repo := s.repo
	s.mu.RUnlock()

	event := AsyncCompletionEvent{
		ToolCallID: op.ToolCallID,
		ChatID:     op.ChatID,
		ToolName:   op.ToolName,
		Scenario:   op.Scenario,
		Status:     status,
		Result:     op.Result,
		Error:      op.Error,
	}

	// Persist completion event for multi-consumer support
	if repo != nil {
		var resultJSON json.RawMessage
		if op.Result != nil {
			if data, err := json.Marshal(op.Result); err == nil {
				resultJSON = data
			}
		}
		eventRecord := &AsyncCompletionEventRecord{
			ChatID:     op.ChatID,
			ToolCallID: op.ToolCallID,
			ToolName:   op.ToolName,
			Status:     status,
			Result:     resultJSON,
			Error:      op.Error,
		}
		if err := repo.CreateCompletionEvent(context.Background(), eventRecord); err != nil {
			log.Printf("[WARN] Failed to persist completion event for %s: %v", op.ToolCallID, err)
		}
	}

	// Send to in-memory callback channel if registered
	if !ok {
		return
	}

	select {
	case ch <- event:
		log.Printf("[DEBUG] Sent completion event for %s (status=%s)", op.ToolCallID, status)
	default:
		log.Printf("[WARN] Completion callback channel full for chat %s", op.ChatID)
	}
}

// GetCompletionEvents retrieves completion events for a chat since a given time.
// This enables multi-consumer callbacks - any handler can query for events
// that occurred since their last check, rather than relying on a single channel.
//
// Returns nil (not an error) if the repository is not configured.
// This allows graceful degradation to the in-memory callback system.
func (s *AsyncTrackerService) GetCompletionEvents(ctx context.Context, chatID string, since time.Time) ([]AsyncCompletionEvent, error) {
	s.mu.RLock()
	repo := s.repo
	s.mu.RUnlock()

	if repo == nil {
		return nil, nil
	}

	records, err := repo.GetCompletionEventsSince(ctx, chatID, since)
	if err != nil {
		return nil, fmt.Errorf("failed to get completion events: %w", err)
	}

	events := make([]AsyncCompletionEvent, 0, len(records))
	for _, r := range records {
		var result interface{}
		if len(r.Result) > 0 {
			_ = json.Unmarshal(r.Result, &result)
		}
		events = append(events, AsyncCompletionEvent{
			ToolCallID: r.ToolCallID,
			ChatID:     r.ChatID,
			ToolName:   r.ToolName,
			Status:     r.Status,
			Result:     result,
			Error:      r.Error,
		})
	}

	return events, nil
}

// pollLoop runs the background polling for an operation.
// Uses OperationSnapshot for immutable config to avoid race conditions.
// Implements exponential backoff when configured via AsyncBehavior.StatusPolling.Backoff.
func (s *AsyncTrackerService) pollLoop(ctx context.Context, op *AsyncOperation) {
	// Snapshot immutable fields at the start to avoid repeated lock acquisitions
	// and potential race conditions when reading config.
	snap, ok := s.snapshotOperation(op.ToolCallID)
	if !ok {
		log.Printf("[ERROR] pollLoop: operation not found for %s", op.ToolCallID)
		return
	}

	polling := snap.AsyncBehavior.StatusPolling
	conditions := snap.AsyncBehavior.CompletionConditions

	// Use configured max duration, with reasonable default
	maxDuration := time.Duration(polling.MaxPollDurationSeconds) * time.Second
	if maxDuration <= 0 {
		maxDuration = DefaultMaxPollDuration
	}

	deadline := snap.StartedAt.Add(maxDuration)

	// Initialize dynamic interval for exponential backoff
	// Backoff config is pre-extracted in snapshotOperation for thread-safe access
	currentInterval := snap.BackoffInitial

	// Log backoff configuration if enabled
	if snap.BackoffMultiplier > 1.0 {
		log.Printf("[DEBUG] pollLoop: starting with backoff for %s (initial=%v, max=%v, multiplier=%.2f)",
			snap.ToolCallID, snap.BackoffInitial, snap.BackoffMax, snap.BackoffMultiplier)
	}

	// Use timer instead of ticker for dynamic intervals
	timer := time.NewTimer(currentInterval)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Polling cancelled for %s", snap.ToolCallID)
			return
		case <-timer.C:
			if time.Now().After(deadline) {
				s.handleTimeout(op)
				return
			}

			// Call the status tool using snapshot for immutable config
			statusResult, err := s.callStatusToolWithSnapshot(ctx, snap)
			if err != nil {
				log.Printf("Error polling status for %s: %v", snap.ToolCallID, err)

				// Track consecutive errors
				s.mu.Lock()
				op.ConsecutiveErrors++
				op.LastPollError = err.Error()
				errorCount := op.ConsecutiveErrors
				s.mu.Unlock()

				// Push error update to UI after 2+ consecutive failures
				if errorCount >= 2 {
					s.pushPollErrorUpdate(snap.ChatID, snap.ToolCallID, err.Error(), errorCount)
				}

				// Continue polling despite error - reset timer with current interval
				timer.Reset(currentInterval)
				continue
			}

			// Reset error count on successful poll
			s.mu.Lock()
			op.ConsecutiveErrors = 0
			op.LastPollError = ""
			s.mu.Unlock()

			// Process the status result
			terminal, status := s.processStatusResult(op, statusResult, conditions)
			if terminal {
				log.Printf("Operation %s reached terminal status: %s", snap.ToolCallID, status)
				return
			}

			// Calculate next interval with exponential backoff
			if snap.BackoffMultiplier > 1.0 {
				nextInterval := time.Duration(float64(currentInterval) * snap.BackoffMultiplier)
				if nextInterval > snap.BackoffMax {
					nextInterval = snap.BackoffMax
				}
				if nextInterval != currentInterval {
					log.Printf("[DEBUG] pollLoop: backoff %s interval %v -> %v",
						snap.ToolCallID, currentInterval, nextInterval)
				}
				currentInterval = nextInterval
			}

			// Reset timer with (potentially new) interval
			timer.Reset(currentInterval)
		}
	}
}

// callStatusToolWithSnapshot invokes the status tool using immutable snapshot data.
// This avoids race conditions by not reading from the mutable AsyncOperation.
func (s *AsyncTrackerService) callStatusToolWithSnapshot(ctx context.Context, snap *OperationSnapshot) (interface{}, error) {
	polling := snap.AsyncBehavior.StatusPolling

	// Build arguments for status tool
	args := map[string]interface{}{
		polling.StatusToolIdParam: snap.ExternalRunID,
	}
	argsJSON, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal status tool args: %w", err)
	}

	// Execute the status tool
	record, err := s.toolExecutor.ExecuteTool(ctx, snap.ChatID, "", polling.StatusTool, string(argsJSON))
	if err != nil {
		return nil, err
	}
	if record.Status == domain.StatusFailed {
		return nil, fmt.Errorf("status tool failed: %s", record.ErrorMessage)
	}

	// Parse the result
	var result interface{}
	if err := json.Unmarshal([]byte(record.Result), &result); err != nil {
		return nil, fmt.Errorf("failed to parse status result: %w", err)
	}

	return result, nil
}

// processStatusResult updates the operation based on status tool response.
// Returns (isTerminal, status) to avoid reading from op after lock release.
//
// The function extracts values from the result using dot-notation paths configured
// in the tool's AsyncBehavior.CompletionConditions:
//   - StatusField: Required. Path to the status string (e.g., "data.run.status")
//   - ErrorField: Optional. Path to error message if status indicates failure
//   - ResultField: Optional. Path to final result data on success
//
// Terminal status is determined by matching the extracted status against
// SuccessValues or FailureValues from the completion conditions.
func (s *AsyncTrackerService) processStatusResult(op *AsyncOperation, result interface{}, conditions *toolspb.CompletionConditions) (bool, string) {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		log.Printf("[WARN] processStatusResult: result is not a map for operation %s: %T", op.ToolCallID, result)
		return false, ""
	}

	// Extract status using the configured field path
	status := ExtractStringField(resultMap, conditions.StatusField)
	if status == "" {
		log.Printf("[WARN] processStatusResult: failed to extract status using path %q for operation %s, result: %+v",
			conditions.StatusField, op.ToolCallID, resultMap)
		return false, ""
	}

	log.Printf("[DEBUG] processStatusResult: operation %s status updated to %q", op.ToolCallID, status)

	// Update operation and build update struct inside lock to avoid race
	s.mu.Lock()
	op.Status = status
	op.UpdatedAt = time.Now()

	// Extract progress if configured
	if op.AsyncBehavior.ProgressTracking != nil {
		if progress := ExtractIntField(resultMap, op.AsyncBehavior.ProgressTracking.ProgressField); progress != nil {
			op.Progress = progress
		}
		if message := ExtractStringField(resultMap, op.AsyncBehavior.ProgressTracking.MessageField); message != "" {
			op.Message = message
		}
		if phase := ExtractStringField(resultMap, op.AsyncBehavior.ProgressTracking.PhaseField); phase != "" {
			op.Phase = phase
		}
	}

	// Check for error
	if conditions.ErrorField != "" {
		if errMsg := ExtractStringField(resultMap, conditions.ErrorField); errMsg != "" {
			op.Error = errMsg
		}
	}

	// Check for result
	if conditions.ResultField != "" {
		if resultVal := ExtractField(resultMap, conditions.ResultField); resultVal != nil {
			op.Result = resultVal
		}
	}

	// Check terminal conditions - compare status against configured success/failure values
	isSuccess := ContainsString(conditions.SuccessValues, status)
	isFailure := ContainsString(conditions.FailureValues, status)
	isTerminal := isSuccess || isFailure

	if isTerminal {
		now := time.Now()
		op.CompletedAt = &now
	}

	// Build update struct while holding the lock to avoid race condition
	update := buildUpdateFromOp(op, isTerminal)
	repo := s.repo // Capture repo reference while holding lock
	toolCallID := op.ToolCallID
	chatID := op.ChatID
	s.mu.Unlock()

	// Persist update to database
	if repo != nil {
		var resultJSON json.RawMessage
		if op.Result != nil {
			if data, err := json.Marshal(op.Result); err == nil {
				resultJSON = data
			}
		}
		record := &AsyncOperationRecord{
			ToolCallID:  toolCallID,
			Status:      op.Status,
			Progress:    op.Progress,
			Message:     op.Message,
			Phase:       op.Phase,
			Result:      resultJSON,
			Error:       op.Error,
			UpdatedAt:   op.UpdatedAt,
			CompletedAt: op.CompletedAt,
		}
		if err := repo.UpdateAsyncOperation(context.Background(), record); err != nil {
			log.Printf("[WARN] Failed to persist operation update for %s: %v", toolCallID, err)
		}
	}

	// Push update to subscribers (using pre-built update)
	s.pushUpdateData(chatID, update)

	// If terminal, trigger completion callback for AI conversation resumption
	if isTerminal {
		s.triggerCompletionCallback(op, status)
	}

	return isTerminal, status
}

// handleTimeout marks an operation as timed out.
// Called when polling exceeds the configured MaxPollDurationSeconds.
func (s *AsyncTrackerService) handleTimeout(op *AsyncOperation) {
	s.mu.Lock()
	op.Status = AsyncStatusTimeout
	op.Error = "Operation timed out"
	now := time.Now()
	op.CompletedAt = &now
	op.UpdatedAt = now
	// Build update while holding lock to avoid race
	update := buildUpdateFromOp(op, true)
	chatID := op.ChatID
	toolCallID := op.ToolCallID
	repo := s.repo
	s.mu.Unlock()

	// Persist timeout to database
	if repo != nil {
		record := &AsyncOperationRecord{
			ToolCallID:  toolCallID,
			Status:      op.Status,
			Error:       op.Error,
			UpdatedAt:   op.UpdatedAt,
			CompletedAt: op.CompletedAt,
		}
		if err := repo.UpdateAsyncOperation(context.Background(), record); err != nil {
			log.Printf("[WARN] Failed to persist timeout for %s: %v", toolCallID, err)
		}
	}

	s.pushUpdateData(chatID, update)

	// Trigger completion callback for AI conversation resumption
	s.triggerCompletionCallback(op, AsyncStatusTimeout)

	log.Printf("Operation %s timed out", toolCallID)
}

// buildUpdateFromOp creates an AsyncStatusUpdate from an operation.
// MUST be called while holding s.mu lock.
func buildUpdateFromOp(op *AsyncOperation, isTerminal bool) AsyncStatusUpdate {
	return AsyncStatusUpdate{
		ToolCallID: op.ToolCallID,
		ChatID:     op.ChatID,
		ToolName:   op.ToolName,
		Status:     op.Status,
		Progress:   op.Progress,
		Message:    op.Message,
		Phase:      op.Phase,
		Result:     op.Result,
		Error:      op.Error,
		IsTerminal: isTerminal,
		UpdatedAt:  op.UpdatedAt,
	}
}

// BuildUpdateFromOperation creates an AsyncStatusUpdate from an operation.
// This is the public API for converting operations to updates (e.g., for HTTP handlers).
// Terminal status is determined by whether CompletedAt is set.
func BuildUpdateFromOperation(op *AsyncOperation) AsyncStatusUpdate {
	return AsyncStatusUpdate{
		ToolCallID: op.ToolCallID,
		ChatID:     op.ChatID,
		ToolName:   op.ToolName,
		Status:     op.Status,
		Progress:   op.Progress,
		Message:    op.Message,
		Phase:      op.Phase,
		Result:     op.Result,
		Error:      op.Error,
		IsTerminal: op.CompletedAt != nil,
		UpdatedAt:  op.UpdatedAt,
	}
}

// pushUpdateData sends a pre-built update to all subscribers for the chat.
// Always build the update while holding the mutex, then pass it here.
func (s *AsyncTrackerService) pushUpdateData(chatID string, update AsyncStatusUpdate) {
	s.mu.RLock()
	subIDs := s.chatSubs[chatID]
	// Copy subscription pointers to avoid holding lock during send
	subs := make([]*Subscription, 0, len(subIDs))
	for _, id := range subIDs {
		if sub := s.subscriptions[id]; sub != nil {
			subs = append(subs, sub)
		}
	}
	s.mu.RUnlock()

	// Send to subscribers
	for _, sub := range subs {
		select {
		case sub.Channel <- update:
		default:
			// Channel full, skip this update
			log.Printf("Warning: subscriber channel full for chat %s (sub=%s)", chatID, sub.ID)
		}
	}
}

// pushPollErrorUpdate sends a poll error notification to subscribers.
// Called when consecutive poll failures occur to surface errors to the UI.
func (s *AsyncTrackerService) pushPollErrorUpdate(chatID, toolCallID, errMsg string, errorCount int) {
	s.mu.RLock()
	op := s.operations[toolCallID]
	s.mu.RUnlock()

	if op == nil {
		return
	}

	update := AsyncStatusUpdate{
		ToolCallID: toolCallID,
		ChatID:     chatID,
		ToolName:   op.ToolName,
		Status:     op.Status,
		Progress:   op.Progress,
		Message:    op.Message,
		Phase:      op.Phase,
		Error:      fmt.Sprintf("Status check failed (%d attempts): %s", errorCount, errMsg),
		IsTerminal: false,
		UpdatedAt:  time.Now(),
	}
	s.pushUpdateData(chatID, update)
}

// extractOperationID extracts the operation ID from the tool result.
//
// The fieldPath uses dot notation to navigate nested structures.
// For example, if the tool returns {"data": {"run": {"id": "abc123"}}},
// the fieldPath "data.run.id" would extract "abc123".
func (s *AsyncTrackerService) extractOperationID(result interface{}, fieldPath string) (string, error) {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("result is not a map")
	}

	value := ExtractStringField(resultMap, fieldPath)
	if value == "" {
		return "", fmt.Errorf("field %s not found or empty", fieldPath)
	}

	return value, nil
}

// CancelOperation cancels a running async operation using the configured cancel tool.
func (s *AsyncTrackerService) CancelOperation(ctx context.Context, toolCallID string) error {
	s.mu.RLock()
	op := s.operations[toolCallID]
	s.mu.RUnlock()

	if op == nil {
		return fmt.Errorf("operation not found: %s", toolCallID)
	}

	// Check if cancellation is configured
	if op.AsyncBehavior == nil || op.AsyncBehavior.Cancellation == nil {
		// No cancel tool configured - just stop tracking
		s.StopTracking(toolCallID)
		return nil
	}

	cancel := op.AsyncBehavior.Cancellation

	// Build arguments for cancel tool
	args := map[string]interface{}{
		cancel.CancelToolIdParam: op.ExternalRunID,
	}
	argsJSON, _ := json.Marshal(args)

	// Execute the cancel tool
	_, err := s.toolExecutor.ExecuteTool(ctx, op.ChatID, "", cancel.CancelTool, string(argsJSON))
	if err != nil {
		return fmt.Errorf("failed to cancel operation: %w", err)
	}

	// Stop tracking
	s.StopTracking(toolCallID)

	return nil
}

// ForceRefresh performs an immediate status poll for an operation, bypassing the normal interval.
// This is useful for manual refresh requests from the UI.
// Returns the updated AsyncStatusUpdate and any error encountered.
func (s *AsyncTrackerService) ForceRefresh(ctx context.Context, toolCallID string) (*AsyncStatusUpdate, error) {
	// Get the operation
	s.mu.RLock()
	op := s.operations[toolCallID]
	s.mu.RUnlock()

	if op == nil {
		return nil, fmt.Errorf("operation not found: %s", toolCallID)
	}

	// Check if operation is already terminal
	if op.CompletedAt != nil {
		// Return current status without polling
		s.mu.RLock()
		update := buildUpdateFromOp(op, true)
		s.mu.RUnlock()
		return &update, nil
	}

	// Get snapshot for polling
	snap, ok := s.snapshotOperation(toolCallID)
	if !ok {
		return nil, fmt.Errorf("failed to snapshot operation: %s", toolCallID)
	}

	// Verify we have the necessary configuration
	if snap.AsyncBehavior == nil || snap.AsyncBehavior.StatusPolling == nil {
		return nil, fmt.Errorf("operation %s does not have status polling configured", toolCallID)
	}

	// Call the status tool immediately
	statusResult, err := s.callStatusToolWithSnapshot(ctx, snap)
	if err != nil {
		// Still return current status with the error
		s.mu.RLock()
		update := buildUpdateFromOp(op, false)
		update.Error = fmt.Sprintf("Refresh failed: %v", err)
		s.mu.RUnlock()
		return &update, nil // Return update with error, not an error return
	}

	// Process the result (this updates the operation and pushes to subscribers)
	conditions := snap.AsyncBehavior.CompletionConditions
	isTerminal, _ := s.processStatusResult(op, statusResult, conditions)

	// Return the updated status
	s.mu.RLock()
	update := buildUpdateFromOp(op, isTerminal)
	s.mu.RUnlock()

	return &update, nil
}

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------
//
// Generic field extraction utilities are in field_extractor.go:
//   - ExtractField: Extract any value from nested map using dot notation
//   - ExtractStringField: Extract string value
//   - ExtractIntField: Extract int value
//   - ContainsString: Check if slice contains string

// -----------------------------------------------------------------------------
// Test Helpers
// -----------------------------------------------------------------------------

// AddTestOperation adds an operation directly to the tracker for testing.
// This bypasses the normal StartTracking flow which requires external dependencies.
func (s *AsyncTrackerService) AddTestOperation(op *AsyncOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.operations[op.ToolCallID] = op
}
