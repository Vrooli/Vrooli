// Package testutil provides test utilities for the Agent Inbox API.
package testutil

import (
	"agent-inbox/domain"
	"context"
	"sync"
)

// MockStreamWriter records SSE events for testing.
type MockStreamWriter struct {
	mu     sync.Mutex
	Events []MockSSEEvent
}

// MockSSEEvent represents a recorded SSE event.
type MockSSEEvent struct {
	Type string
	Data interface{}
}

// NewMockStreamWriter creates a new mock stream writer.
func NewMockStreamWriter() *MockStreamWriter {
	return &MockStreamWriter{}
}

// RecordEvent adds an event to the recorded list.
func (m *MockStreamWriter) RecordEvent(eventType string, data interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = append(m.Events, MockSSEEvent{Type: eventType, Data: data})
}

// GetEvents returns a copy of all recorded events.
func (m *MockStreamWriter) GetEvents() []MockSSEEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	events := make([]MockSSEEvent, len(m.Events))
	copy(events, m.Events)
	return events
}

// EventCount returns the number of recorded events.
func (m *MockStreamWriter) EventCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.Events)
}

// FindEvents returns all events of the given type.
func (m *MockStreamWriter) FindEvents(eventType string) []MockSSEEvent {
	m.mu.Lock()
	defer m.mu.Unlock()
	var found []MockSSEEvent
	for _, e := range m.Events {
		if e.Type == eventType {
			found = append(found, e)
		}
	}
	return found
}

// Reset clears all recorded events.
func (m *MockStreamWriter) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.Events = nil
}

// MockScenarioHandler provides a controllable scenario handler for testing.
type MockScenarioHandler struct {
	mu           sync.Mutex
	HandleCalls  []MockHandleCall
	HandleResult *domain.ToolCallRecord
	HandleError  error
	HandleFunc   func(ctx context.Context, toolName, args string) (*domain.ToolCallRecord, error)
}

// MockHandleCall records a call to Handle.
type MockHandleCall struct {
	ToolName  string
	Arguments string
}

// Handle implements the scenario handler interface for tests.
func (m *MockScenarioHandler) Handle(ctx context.Context, toolName, args string) (*domain.ToolCallRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.HandleCalls = append(m.HandleCalls, MockHandleCall{
		ToolName:  toolName,
		Arguments: args,
	})

	if m.HandleFunc != nil {
		return m.HandleFunc(ctx, toolName, args)
	}

	return m.HandleResult, m.HandleError
}

// ResetScenarioHandler clears all recorded calls.
func (m *MockScenarioHandler) ResetScenarioHandler() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.HandleCalls = nil
}
