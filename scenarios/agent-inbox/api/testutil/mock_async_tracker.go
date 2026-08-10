// Package testutil provides test utilities for the Agent Inbox API.
package testutil

import (
	"sync"

	"agent-inbox/services"
)

// FakeAsyncTracker provides a controllable async tracker for testing.
type FakeAsyncTracker struct {
	mu                 sync.Mutex
	Operations         map[string]*services.AsyncOperation
	StatusUpdates      chan services.AsyncStatusUpdate
	CompletionEvents   chan services.AsyncCompletionEvent
	StartTrackingCalls []StartTrackingCall
	StartTrackingError error
	CancelCalls        []string
	CancelError        error
}

// StartTrackingCall records a call to StartTracking.
type StartTrackingCall struct {
	ToolCallID string
	ChatID     string
	ToolName   string
	Scenario   string
	ToolResult interface{}
}

// NewFakeAsyncTracker creates a new fake tracker for testing.
func NewFakeAsyncTracker() *FakeAsyncTracker {
	return &FakeAsyncTracker{
		Operations:       make(map[string]*services.AsyncOperation),
		StatusUpdates:    make(chan services.AsyncStatusUpdate, 100),
		CompletionEvents: make(chan services.AsyncCompletionEvent, 10),
	}
}

// SimulateUpdate sends a status update to all channels.
func (f *FakeAsyncTracker) SimulateUpdate(update services.AsyncStatusUpdate) {
	f.mu.Lock()
	defer f.mu.Unlock()
	select {
	case f.StatusUpdates <- update:
	default:
	}
}

// SimulateCompletion sends a completion event.
func (f *FakeAsyncTracker) SimulateCompletion(event services.AsyncCompletionEvent) {
	f.mu.Lock()
	defer f.mu.Unlock()
	select {
	case f.CompletionEvents <- event:
	default:
	}
}

// GetOperation returns a tracked operation by ID.
func (f *FakeAsyncTracker) GetOperation(toolCallID string) *services.AsyncOperation {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.Operations[toolCallID]
}

// Reset clears all state.
func (f *FakeAsyncTracker) Reset() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.Operations = make(map[string]*services.AsyncOperation)
	f.StartTrackingCalls = nil
	f.CancelCalls = nil
	// Drain channels
	for len(f.StatusUpdates) > 0 {
		<-f.StatusUpdates
	}
	for len(f.CompletionEvents) > 0 {
		<-f.CompletionEvents
	}
}
