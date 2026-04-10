package services

import (
	"context"
	"sync"
	"testing"
	"time"

	"agent-inbox/domain"

	toolspb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-inbox/v1/domain"
)

// =============================================================================
// Mock Tool Executor
// =============================================================================

// mockToolExecutor implements ToolExecutorInterface for testing.
type mockToolExecutor struct {
	mu sync.Mutex

	// Execution behavior
	executeFunc   func(ctx context.Context, chatID, toolCallID, toolName, args string) (*domain.ToolCallRecord, error)
	executeResult *domain.ToolCallRecord
	executeError  error

	// Call tracking
	executeCalls []executeToolCall
}

type executeToolCall struct {
	ChatID     string
	ToolCallID string
	ToolName   string
	Arguments  string
}

func newMockToolExecutor() *mockToolExecutor {
	return &mockToolExecutor{}
}

func (m *mockToolExecutor) ExecuteTool(ctx context.Context, chatID, toolCallID, toolName, arguments string) (*domain.ToolCallRecord, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.executeCalls = append(m.executeCalls, executeToolCall{
		ChatID:     chatID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Arguments:  arguments,
	})

	if m.executeFunc != nil {
		return m.executeFunc(ctx, chatID, toolCallID, toolName, arguments)
	}

	if m.executeError != nil {
		return &domain.ToolCallRecord{
			ID:           toolCallID,
			ChatID:       chatID,
			ToolName:     toolName,
			Arguments:    arguments,
			Status:       domain.StatusFailed,
			ErrorMessage: m.executeError.Error(),
			StartedAt:    time.Now(),
			CompletedAt:  time.Now(),
		}, m.executeError
	}

	if m.executeResult != nil {
		result := *m.executeResult
		result.ID = toolCallID
		result.ChatID = chatID
		result.ToolName = toolName
		result.Arguments = arguments
		return &result, nil
	}

	// Default success
	return &domain.ToolCallRecord{
		ID:          toolCallID,
		ChatID:      chatID,
		ToolName:    toolName,
		Arguments:   arguments,
		Status:      domain.StatusCompleted,
		Result:      `{"success": true}`,
		StartedAt:   time.Now(),
		CompletedAt: time.Now(),
	}, nil
}

// SetExecuteFunc sets a custom function for execution.
func (m *mockToolExecutor) SetExecuteFunc(fn func(ctx context.Context, chatID, toolCallID, toolName, args string) (*domain.ToolCallRecord, error)) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.executeFunc = fn
}

// GetExecuteCalls returns all execute calls for verification.
func (m *mockToolExecutor) GetExecuteCalls() []executeToolCall {
	m.mu.Lock()
	defer m.mu.Unlock()
	calls := make([]executeToolCall, len(m.executeCalls))
	copy(calls, m.executeCalls)
	return calls
}

// =============================================================================
// Mock Async Tracker
// =============================================================================

// mockAsyncTrackerForCompletion implements AsyncTrackerInterface for testing.
type mockAsyncTrackerForCompletion struct {
	mu sync.Mutex

	operations map[string]*AsyncOperation

	// Call tracking
	startTrackingCalls []startTrackingCall

	// Error injection
	startTrackingError error
}

type startTrackingCall struct {
	ToolCallID string
	ChatID     string
	ToolName   string
	Scenario   string
}

func newMockAsyncTrackerForCompletion() *mockAsyncTrackerForCompletion {
	return &mockAsyncTrackerForCompletion{
		operations: make(map[string]*AsyncOperation),
	}
}

func (m *mockAsyncTrackerForCompletion) GetActiveOperations(chatID string) []*AsyncOperation {
	m.mu.Lock()
	defer m.mu.Unlock()

	var active []*AsyncOperation
	for _, op := range m.operations {
		if op.ChatID == chatID && op.CompletedAt == nil {
			active = append(active, op)
		}
	}
	return active
}

func (m *mockAsyncTrackerForCompletion) GetOperation(toolCallID string) *AsyncOperation {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.operations[toolCallID]
}

func (m *mockAsyncTrackerForCompletion) StartTracking(ctx context.Context, toolCallID, chatID, toolName, scenario string, toolResult interface{}, asyncBehavior *toolspb.AsyncBehavior) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.startTrackingCalls = append(m.startTrackingCalls, startTrackingCall{
		ToolCallID: toolCallID,
		ChatID:     chatID,
		ToolName:   toolName,
		Scenario:   scenario,
	})

	if m.startTrackingError != nil {
		return m.startTrackingError
	}

	m.operations[toolCallID] = &AsyncOperation{
		ToolCallID: toolCallID,
		ChatID:     chatID,
		ToolName:   toolName,
		Scenario:   scenario,
		Status:     "running",
		StartedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	return nil
}

// AddOperation adds an operation for testing.
func (m *mockAsyncTrackerForCompletion) AddOperation(op *AsyncOperation) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.operations[op.ToolCallID] = op
}

// =============================================================================
// Test Helpers
// =============================================================================

// makeToolCall creates a ToolCall with the given ID, name, and arguments.
// This helper avoids the verbose anonymous struct syntax for Function.
func makeToolCall(id, name, args string) domain.ToolCall {
	tc := domain.ToolCall{
		ID:   id,
		Type: "function",
	}
	tc.Function.Name = name
	tc.Function.Arguments = args
	return tc
}

// =============================================================================
// Tests for ExecuteToolCalls
// =============================================================================

func TestExecuteToolCalls_SingleTool_Success(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	// Add a tool that doesn't require approval
	registry.addTool("test-scenario", createSimpleTool("test_tool", "A test tool"))
	registry.ApprovalRequirements["test_tool"] = false

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "test_tool", `{"input": "value"}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify outcome
	if len(outcome.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(outcome.Results))
	}
	if outcome.Results[0].Status != domain.StatusCompleted {
		t.Errorf("expected status %s, got %s", domain.StatusCompleted, outcome.Results[0].Status)
	}
	if outcome.HasPendingApprovals {
		t.Error("expected no pending approvals")
	}

	// Verify executor was called
	calls := executor.GetExecuteCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 executor call, got %d", len(calls))
	}
	if calls[0].ToolName != "test_tool" {
		t.Errorf("expected tool name 'test_tool', got %q", calls[0].ToolName)
	}

	// Verify tool response message was saved
	if len(repo.saveToolResponseMessageCalls) != 1 {
		t.Fatalf("expected 1 tool response save, got %d", len(repo.saveToolResponseMessageCalls))
	}
}
