package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

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
