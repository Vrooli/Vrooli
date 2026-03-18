// Package testutil provides test utilities for the Agent Inbox API.
package testutil

import (
	"context"
	"sync"

	"agent-inbox/domain"
)

// MockToolExecutor provides a controllable tool executor for testing.
type MockToolExecutor struct {
	mu            sync.Mutex
	ExecuteCalls  []MockExecuteCall
	ExecuteResult *domain.ToolCallRecord
	ExecuteError  error
	// ExecuteFunc allows custom behavior per call
	ExecuteFunc func(ctx context.Context, chatID, toolCallID, toolName, args string) (*domain.ToolCallRecord, error)
}

// MockExecuteCall records a call to ExecuteTool.
type MockExecuteCall struct {
	ChatID     string
	ToolCallID string
	ToolName   string
	Arguments  string
}

// ExecuteTool implements the tool executor interface for tests.
func (m *MockToolExecutor) ExecuteTool(ctx context.Context, chatID, toolCallID, toolName, args string) (*domain.ToolCallRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.ExecuteCalls = append(m.ExecuteCalls, MockExecuteCall{
		ChatID:     chatID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Arguments:  args,
	})

	if m.ExecuteFunc != nil {
		return m.ExecuteFunc(ctx, chatID, toolCallID, toolName, args)
	}

	return m.ExecuteResult, m.ExecuteError
}

// Reset clears all recorded calls.
func (m *MockToolExecutor) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.ExecuteCalls = nil
}

// CallCount returns the number of ExecuteTool calls.
func (m *MockToolExecutor) CallCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.ExecuteCalls)
}

// LastCall returns the most recent call, or nil if none.
func (m *MockToolExecutor) LastCall() *MockExecuteCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.ExecuteCalls) == 0 {
		return nil
	}
	return &m.ExecuteCalls[len(m.ExecuteCalls)-1]
}
