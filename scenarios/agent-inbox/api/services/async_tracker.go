// Package services provides application services for the Agent Inbox scenario.
//
// This file implements the AsyncTrackerService for tracking long-running tool
// operations and providing status updates via Server-Sent Events (SSE).
//
// ARCHITECTURE:
// - AsyncTrackerService: Manages background polling for async tools
// - AsyncOperation: Represents a tracked async tool execution
// - Subscribers receive updates via channels (for SSE endpoints)
//
// POLLING FLOW:
// 1. Tool executor calls StartTracking() after executing an async tool
// 2. Tracker extracts operation ID from result using AsyncBehavior config
// 3. Background goroutine polls status tool at configured intervals
// 4. Updates are pushed to all subscribers for the chat
// 5. Polling stops on terminal status or timeout
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

	// Subscribers for status updates (chatID -> channels)
	// Deprecated: Use subscriptions/chatSubs instead for ID-based tracking
	subscribers map[string][]chan<- AsyncStatusUpdate

	// ID-based subscription tracking (replaces fragile pointer comparison)
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
	ToolCallID    string                   `json:"tool_call_id"`
	ChatID        string                   `json:"chat_id"`
	ToolName      string                   `json:"tool_name"`
	Scenario      string                   `json:"scenario"`
	ExternalRunID string                   `json:"external_run_id"`
	AsyncBehavior *toolspb.AsyncBehavior   `json:"-"`
	Status        string                   `json:"status"`
	Progress      *int                     `json:"progress,omitempty"`
	Message       string                   `json:"message,omitempty"`
	Phase         string                   `json:"phase,omitempty"`
	Result        interface{}              `json:"result,omitempty"`
	Error         string                   `json:"error,omitempty"`
	StartedAt     time.Time                `json:"started_at"`
	UpdatedAt     time.Time                `json:"updated_at"`
	CompletedAt   *time.Time               `json:"completed_at,omitempty"`
}

// AsyncStatusUpdate represents a status update pushed to subscribers.
type AsyncStatusUpdate struct {
	ToolCallID  string      `json:"tool_call_id"`
	ChatID      string      `json:"chat_id"`
	ToolName    string      `json:"tool_name"`
	Status      string      `json:"status"`
	Progress    *int        `json:"progress,omitempty"`
	Message     string      `json:"message,omitempty"`
	Phase       string      `json:"phase,omitempty"`
	Result      interface{} `json:"result,omitempty"`
	Error       string      `json:"error,omitempty"`
	IsTerminal  bool        `json:"is_terminal"`
	UpdatedAt   time.Time   `json:"updated_at"`
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

	return &OperationSnapshot{
		ToolCallID:    op.ToolCallID,
		ChatID:        op.ChatID,
		ToolName:      op.ToolName,
		Scenario:      op.Scenario,
		ExternalRunID: op.ExternalRunID,
		AsyncBehavior: op.AsyncBehavior, // Pointer to immutable proto struct
		StartedAt:     op.StartedAt,
	}, true
}

// NewAsyncTrackerService creates a new AsyncTrackerService.
func NewAsyncTrackerService(registry *ToolRegistry, executor *integrations.ToolExecutor) *AsyncTrackerService {
	return &AsyncTrackerService{
		operations:          make(map[string]*AsyncOperation),
		subscribers:         make(map[string][]chan<- AsyncStatusUpdate),
		subscriptions:       make(map[string]*Subscription),
		chatSubs:            make(map[string][]string),
		completionCallbacks: make(map[string]chan<- AsyncCompletionEvent),
		cancelFuncs:         make(map[string]context.CancelFunc),
		toolRegistry:        registry,
		toolExecutor:        executor,
	}
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
		Status:        "pending",
		StartedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	// Store the operation
	s.mu.Lock()
	s.operations[toolCallID] = op
	s.mu.Unlock()

	// Push initial update to subscribers so UI shows the operation immediately
	s.pushUpdate(op, false)

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

	// Mark the operation as cancelled if it exists
	op := s.operations[toolCallID]
	if op != nil && op.CompletedAt == nil {
		op.Status = "cancelled"
		op.Error = "Operation cancelled"
		now := time.Now()
		op.CompletedAt = &now
		op.UpdatedAt = now
	}
	s.mu.Unlock()

	// Trigger completion callback outside of lock
	if op != nil {
		s.triggerCompletionCallback(op, "cancelled")
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

// Subscribe creates a channel for receiving updates for a chat.
// The caller is responsible for calling Unsubscribe when done.
func (s *AsyncTrackerService) Subscribe(chatID string) <-chan AsyncStatusUpdate {
	ch := make(chan AsyncStatusUpdate, 100) // Buffer to prevent blocking

	s.mu.Lock()
	s.subscribers[chatID] = append(s.subscribers[chatID], ch)
	s.mu.Unlock()

	return ch
}

// Unsubscribe removes a subscriber channel.
// Deprecated: Use UnsubscribeByID instead for safer subscription management.
func (s *AsyncTrackerService) Unsubscribe(chatID string, ch <-chan AsyncStatusUpdate) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subs := s.subscribers[chatID]
	for i, sub := range subs {
		// Compare channel addresses by converting to bidirectional channel type
		// Note: This works because we know the channel was created as bidirectional
		if fmt.Sprintf("%p", sub) == fmt.Sprintf("%p", ch) {
			s.subscribers[chatID] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}

	// Clean up empty subscriber lists
	if len(s.subscribers[chatID]) == 0 {
		delete(s.subscribers, chatID)
	}
}

// SubscribeWithID creates a subscription with a unique ID for safe tracking.
// Returns a Subscription that can be passed to UnsubscribeByID.
// This is the preferred method over Subscribe as it avoids fragile pointer comparison.
func (s *AsyncTrackerService) SubscribeWithID(chatID string) *Subscription {
	ch := make(chan AsyncStatusUpdate, 100) // Buffer to prevent blocking
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
// The AI conversation loop uses this to wait for async operations to complete.
// Returns a receive-only channel that will receive AsyncCompletionEvent when operations complete.
// Call UnregisterCompletionCallback when done waiting.
func (s *AsyncTrackerService) RegisterCompletionCallback(chatID string) <-chan AsyncCompletionEvent {
	ch := make(chan AsyncCompletionEvent, 10) // Buffer for multiple operations

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
// MUST be called while NOT holding the mutex (to avoid deadlock).
func (s *AsyncTrackerService) triggerCompletionCallback(op *AsyncOperation, status string) {
	s.mu.RLock()
	ch, ok := s.completionCallbacks[op.ChatID]
	s.mu.RUnlock()

	if !ok {
		return
	}

	event := AsyncCompletionEvent{
		ToolCallID: op.ToolCallID,
		ChatID:     op.ChatID,
		ToolName:   op.ToolName,
		Scenario:   op.Scenario,
		Status:     status,
		Result:     op.Result,
		Error:      op.Error,
	}

	select {
	case ch <- event:
		log.Printf("[DEBUG] Sent completion event for %s (status=%s)", op.ToolCallID, status)
	default:
		log.Printf("[WARN] Completion callback channel full for chat %s", op.ChatID)
	}
}

// pollLoop runs the background polling for an operation.
// Uses OperationSnapshot for immutable config to avoid race conditions.
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

	interval := time.Duration(polling.PollIntervalSeconds) * time.Second
	if interval < time.Second {
		interval = 5 * time.Second // Default to 5 seconds
	}

	maxDuration := time.Duration(polling.MaxPollDurationSeconds) * time.Second
	if maxDuration <= 0 {
		maxDuration = time.Hour // Default to 1 hour
	}

	deadline := snap.StartedAt.Add(maxDuration)
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("Polling cancelled for %s", snap.ToolCallID)
			return
		case <-ticker.C:
			if time.Now().After(deadline) {
				s.handleTimeout(op)
				return
			}

			// Call the status tool using snapshot for immutable config
			statusResult, err := s.callStatusToolWithSnapshot(ctx, snap)
			if err != nil {
				log.Printf("Error polling status for %s: %v", snap.ToolCallID, err)
				continue // Keep trying
			}

			// Process the status result
			terminal, status := s.processStatusResult(op, statusResult, conditions)
			if terminal {
				log.Printf("Operation %s reached terminal status: %s", snap.ToolCallID, status)
				return
			}
		}
	}
}

// callStatusTool invokes the status tool to check operation progress.
// Deprecated: Use callStatusToolWithSnapshot instead to avoid race conditions.
func (s *AsyncTrackerService) callStatusTool(ctx context.Context, op *AsyncOperation) (interface{}, error) {
	polling := op.AsyncBehavior.StatusPolling

	// Build arguments for status tool
	args := map[string]interface{}{
		polling.StatusToolIdParam: op.ExternalRunID,
	}
	argsJSON, _ := json.Marshal(args)

	// Execute the status tool
	record, err := s.toolExecutor.ExecuteTool(ctx, op.ChatID, "", polling.StatusTool, string(argsJSON))
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
func (s *AsyncTrackerService) processStatusResult(op *AsyncOperation, result interface{}, conditions *toolspb.CompletionConditions) (bool, string) {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return false, ""
	}

	// Extract status
	status := extractStringField(resultMap, conditions.StatusField)
	if status == "" {
		return false, ""
	}

	// Update operation and build update struct inside lock to avoid race
	s.mu.Lock()
	op.Status = status
	op.UpdatedAt = time.Now()

	// Extract progress if configured
	if op.AsyncBehavior.ProgressTracking != nil {
		if progress := extractIntField(resultMap, op.AsyncBehavior.ProgressTracking.ProgressField); progress != nil {
			op.Progress = progress
		}
		if message := extractStringField(resultMap, op.AsyncBehavior.ProgressTracking.MessageField); message != "" {
			op.Message = message
		}
		if phase := extractStringField(resultMap, op.AsyncBehavior.ProgressTracking.PhaseField); phase != "" {
			op.Phase = phase
		}
	}

	// Check for error
	if conditions.ErrorField != "" {
		if errMsg := extractStringField(resultMap, conditions.ErrorField); errMsg != "" {
			op.Error = errMsg
		}
	}

	// Check for result
	if conditions.ResultField != "" {
		if resultVal := extractField(resultMap, conditions.ResultField); resultVal != nil {
			op.Result = resultVal
		}
	}

	// Check terminal conditions
	isSuccess := contains(conditions.SuccessValues, status)
	isFailure := contains(conditions.FailureValues, status)
	isTerminal := isSuccess || isFailure

	if isTerminal {
		now := time.Now()
		op.CompletedAt = &now
	}

	// Build update struct while holding the lock to avoid race condition
	update := buildUpdateFromOp(op, isTerminal)
	s.mu.Unlock()

	// Push update to subscribers (using pre-built update)
	s.pushUpdateData(op.ChatID, update)

	// If terminal, trigger completion callback for AI conversation resumption
	if isTerminal {
		s.triggerCompletionCallback(op, status)
	}

	return isTerminal, status
}

// handleTimeout marks an operation as timed out.
func (s *AsyncTrackerService) handleTimeout(op *AsyncOperation) {
	s.mu.Lock()
	op.Status = "timeout"
	op.Error = "Operation timed out"
	now := time.Now()
	op.CompletedAt = &now
	op.UpdatedAt = now
	// Build update while holding lock to avoid race
	update := buildUpdateFromOp(op, true)
	chatID := op.ChatID
	toolCallID := op.ToolCallID
	s.mu.Unlock()

	s.pushUpdateData(chatID, update)

	// Trigger completion callback for AI conversation resumption
	s.triggerCompletionCallback(op, "timeout")

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

// pushUpdate sends an update to all subscribers for the chat.
// WARNING: This reads from op without holding the lock, which is a race condition.
// Prefer using pushUpdateData with a pre-built update instead.
func (s *AsyncTrackerService) pushUpdate(op *AsyncOperation, isTerminal bool) {
	// Build update - this is racy but kept for backwards compatibility
	update := AsyncStatusUpdate{
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

	s.pushUpdateData(op.ChatID, update)
}

// pushUpdateData sends a pre-built update to all subscribers for the chat.
// This is the race-safe version that should be preferred over pushUpdate.
func (s *AsyncTrackerService) pushUpdateData(chatID string, update AsyncStatusUpdate) {
	s.mu.RLock()
	// Get old-style subscribers
	oldSubs := s.subscribers[chatID]
	// Get new ID-based subscription IDs
	subIDs := s.chatSubs[chatID]
	// Copy subscription pointers to avoid holding lock during send
	newSubs := make([]*Subscription, 0, len(subIDs))
	for _, id := range subIDs {
		if sub := s.subscriptions[id]; sub != nil {
			newSubs = append(newSubs, sub)
		}
	}
	s.mu.RUnlock()

	// Send to old-style subscribers
	for _, ch := range oldSubs {
		select {
		case ch <- update:
		default:
			// Channel full, skip this update
			log.Printf("Warning: subscriber channel full for chat %s", chatID)
		}
	}

	// Send to new ID-based subscribers
	for _, sub := range newSubs {
		select {
		case sub.Channel <- update:
		default:
			// Channel full, skip this update
			log.Printf("Warning: subscriber channel full for chat %s (sub=%s)", chatID, sub.ID)
		}
	}
}

// extractOperationID extracts the operation ID from the tool result.
func (s *AsyncTrackerService) extractOperationID(result interface{}, fieldPath string) (string, error) {
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("result is not a map")
	}

	value := extractStringField(resultMap, fieldPath)
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

// -----------------------------------------------------------------------------
// Helper Functions
// -----------------------------------------------------------------------------

// extractStringField extracts a string value from a nested map using dot notation.
func extractStringField(m map[string]interface{}, path string) string {
	val := extractField(m, path)
	if val == nil {
		return ""
	}
	if s, ok := val.(string); ok {
		return s
	}
	return ""
}

// extractIntField extracts an int value from a nested map using dot notation.
func extractIntField(m map[string]interface{}, path string) *int {
	val := extractField(m, path)
	if val == nil {
		return nil
	}
	switch v := val.(type) {
	case float64:
		i := int(v)
		return &i
	case int:
		return &v
	case int64:
		i := int(v)
		return &i
	}
	return nil
}

// extractField extracts a value from a nested map using dot notation.
func extractField(m map[string]interface{}, path string) interface{} {
	parts := splitPath(path)
	current := interface{}(m)

	for _, part := range parts {
		if currentMap, ok := current.(map[string]interface{}); ok {
			current = currentMap[part]
			if current == nil {
				return nil
			}
		} else {
			return nil
		}
	}

	return current
}

// splitPath splits a dot-notation path into parts.
func splitPath(path string) []string {
	var parts []string
	var current string
	for _, c := range path {
		if c == '.' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

// contains checks if a slice contains a string.
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

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
