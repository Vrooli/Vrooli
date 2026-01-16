package integrations

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"agent-inbox/domain"
)

// testHandler implements ScenarioHandler for testing.
type testHandler struct {
	mu        sync.Mutex
	scenario  string
	tools     map[string]bool
	execFunc  func(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error)
	execCalls []testExecCall
}

type testExecCall struct {
	ToolName string
	Args     map[string]interface{}
}

func newTestHandler(scenario string, tools ...string) *testHandler {
	h := &testHandler{
		scenario: scenario,
		tools:    make(map[string]bool),
	}
	for _, t := range tools {
		h.tools[t] = true
	}
	return h
}

func (h *testHandler) Scenario() string {
	return h.scenario
}

func (h *testHandler) CanHandle(toolName string) bool {
	return h.tools[toolName]
}

func (h *testHandler) Execute(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
	h.mu.Lock()
	h.execCalls = append(h.execCalls, testExecCall{ToolName: toolName, Args: args})
	h.mu.Unlock()
	if h.execFunc != nil {
		return h.execFunc(ctx, toolName, args)
	}
	return map[string]string{"status": "ok"}, nil
}

func (h *testHandler) callCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.execCalls)
}

// TestNewToolExecutor verifies basic initialization.
func TestNewToolExecutor(t *testing.T) {
	exec := NewToolExecutor()
	if exec == nil {
		t.Fatal("expected non-nil executor")
	}
	if exec.handlers == nil {
		t.Error("expected handlers map to be initialized")
	}
	if exec.legacyHandlers == nil {
		t.Error("expected legacyHandlers map to be initialized")
	}
}

// TestNewToolExecutorWithHandlers verifies initialization with handlers.
func TestNewToolExecutorWithHandlers(t *testing.T) {
	h1 := newTestHandler("scenario-a", "tool-a")
	h2 := newTestHandler("scenario-b", "tool-b")

	exec := NewToolExecutorWithHandlers(h1, h2)

	if !exec.IsKnownTool("tool-a") {
		t.Error("expected tool-a to be known")
	}
	if !exec.IsKnownTool("tool-b") {
		t.Error("expected tool-b to be known")
	}
	if exec.IsKnownTool("unknown-tool") {
		t.Error("expected unknown-tool to not be known")
	}
}

// TestRegisterHandler verifies handler registration.
func TestRegisterHandler(t *testing.T) {
	exec := NewToolExecutor()
	h := newTestHandler("test-scenario", "test-tool")

	exec.RegisterHandler(h)

	if !exec.IsKnownTool("test-tool") {
		t.Error("expected tool to be known after registration")
	}
	if exec.GetToolScenario("test-tool") != "test-scenario" {
		t.Errorf("expected scenario 'test-scenario', got '%s'", exec.GetToolScenario("test-tool"))
	}
}

// TestUnregisterHandler verifies handler removal.
func TestUnregisterHandler(t *testing.T) {
	exec := NewToolExecutor()
	h := newTestHandler("test-scenario", "test-tool")
	exec.RegisterHandler(h)

	exec.UnregisterHandler("test-scenario")

	if exec.IsKnownTool("test-tool") {
		t.Error("expected tool to be unknown after unregistration")
	}
}

// TestExecuteTool_Success verifies successful tool execution.
func TestExecuteTool_Success(t *testing.T) {
	h := newTestHandler("scenario", "my-tool")
	h.execFunc = func(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{
			"result": "success",
			"value":  42,
		}, nil
	}
	exec := NewToolExecutorWithHandlers(h)

	record, err := exec.ExecuteTool(
		context.Background(),
		"chat-123",
		"tc-456",
		"my-tool",
		`{"param": "value"}`,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record == nil {
		t.Fatal("expected non-nil record")
	}
	if record.Status != domain.StatusCompleted {
		t.Errorf("expected status '%s', got '%s'", domain.StatusCompleted, record.Status)
	}
	if record.ID != "tc-456" {
		t.Errorf("expected ID 'tc-456', got '%s'", record.ID)
	}
	if record.ChatID != "chat-123" {
		t.Errorf("expected ChatID 'chat-123', got '%s'", record.ChatID)
	}
	if record.ToolName != "my-tool" {
		t.Errorf("expected ToolName 'my-tool', got '%s'", record.ToolName)
	}
	if record.ScenarioName != "scenario" {
		t.Errorf("expected ScenarioName 'scenario', got '%s'", record.ScenarioName)
	}
	if record.Result == "" {
		t.Error("expected non-empty result")
	}

	// Verify handler was called with correct args
	if len(h.execCalls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(h.execCalls))
	}
	if h.execCalls[0].ToolName != "my-tool" {
		t.Errorf("expected tool name 'my-tool', got '%s'", h.execCalls[0].ToolName)
	}
	if h.execCalls[0].Args["param"] != "value" {
		t.Errorf("expected param='value', got '%v'", h.execCalls[0].Args["param"])
	}
}

// TestExecuteTool_UnknownTool verifies error for unregistered tools.
func TestExecuteTool_UnknownTool(t *testing.T) {
	exec := NewToolExecutor()

	record, err := exec.ExecuteTool(
		context.Background(),
		"chat-123",
		"tc-456",
		"unknown-tool",
		`{}`,
	)

	if err == nil {
		t.Fatal("expected error for unknown tool")
	}

	var unknownErr *UnknownToolError
	if !errors.As(err, &unknownErr) {
		t.Errorf("expected UnknownToolError, got %T: %v", err, err)
	}
	if unknownErr.ToolName != "unknown-tool" {
		t.Errorf("expected tool name 'unknown-tool', got '%s'", unknownErr.ToolName)
	}

	if record == nil {
		t.Fatal("expected non-nil record even on error")
	}
	if record.Status != domain.StatusFailed {
		t.Errorf("expected status '%s', got '%s'", domain.StatusFailed, record.Status)
	}
}

// TestExecuteTool_InvalidJSON verifies error for malformed arguments.
func TestExecuteTool_InvalidJSON(t *testing.T) {
	h := newTestHandler("scenario", "my-tool")
	exec := NewToolExecutorWithHandlers(h)

	record, err := exec.ExecuteTool(
		context.Background(),
		"chat-123",
		"tc-456",
		"my-tool",
		`not valid json`,
	)

	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if record == nil {
		t.Fatal("expected non-nil record even on error")
	}
	if record.Status != domain.StatusFailed {
		t.Errorf("expected status '%s', got '%s'", domain.StatusFailed, record.Status)
	}
	if record.ErrorMessage == "" {
		t.Error("expected non-empty error message")
	}

	// Handler should not have been called
	if len(h.execCalls) != 0 {
		t.Errorf("expected 0 calls, got %d", len(h.execCalls))
	}
}

// TestExecuteTool_HandlerError verifies error handling from handlers.
func TestExecuteTool_HandlerError(t *testing.T) {
	h := newTestHandler("scenario", "my-tool")
	expectedErr := errors.New("handler failed")
	h.execFunc = func(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
		return nil, expectedErr
	}
	exec := NewToolExecutorWithHandlers(h)

	record, err := exec.ExecuteTool(
		context.Background(),
		"chat-123",
		"tc-456",
		"my-tool",
		`{}`,
	)

	if err == nil {
		t.Fatal("expected error from handler")
	}
	if !errors.Is(err, expectedErr) {
		t.Errorf("expected error '%v', got '%v'", expectedErr, err)
	}
	if record == nil {
		t.Fatal("expected non-nil record even on error")
	}
	if record.Status != domain.StatusFailed {
		t.Errorf("expected status '%s', got '%s'", domain.StatusFailed, record.Status)
	}
}

// TestExecuteTool_ExtractsRunID verifies run_id extraction for async operations.
func TestExecuteTool_ExtractsRunID(t *testing.T) {
	h := newTestHandler("async-scenario", "async-tool")
	h.execFunc = func(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
		return map[string]interface{}{
			"status": "pending",
			"run_id": "run-abc-123",
		}, nil
	}
	exec := NewToolExecutorWithHandlers(h)

	record, err := exec.ExecuteTool(
		context.Background(),
		"chat-123",
		"tc-456",
		"async-tool",
		`{}`,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.ExternalRunID != "run-abc-123" {
		t.Errorf("expected ExternalRunID 'run-abc-123', got '%s'", record.ExternalRunID)
	}
}

// TestExecuteTool_NoRunIDInResult verifies no ExternalRunID when absent.
func TestExecuteTool_NoRunIDInResult(t *testing.T) {
	h := newTestHandler("scenario", "my-tool")
	h.execFunc = func(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
		return map[string]string{"status": "ok"}, nil
	}
	exec := NewToolExecutorWithHandlers(h)

	record, err := exec.ExecuteTool(
		context.Background(),
		"chat-123",
		"tc-456",
		"my-tool",
		`{}`,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.ExternalRunID != "" {
		t.Errorf("expected empty ExternalRunID, got '%s'", record.ExternalRunID)
	}
}

// TestExecuteTool_MultipleHandlers verifies routing to correct handler.
func TestExecuteTool_MultipleHandlers(t *testing.T) {
	h1 := newTestHandler("scenario-a", "tool-a")
	h1.execFunc = func(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
		return map[string]string{"from": "handler-a"}, nil
	}
	h2 := newTestHandler("scenario-b", "tool-b")
	h2.execFunc = func(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
		return map[string]string{"from": "handler-b"}, nil
	}

	exec := NewToolExecutorWithHandlers(h1, h2)

	// Call tool-b
	record, err := exec.ExecuteTool(
		context.Background(),
		"chat-123",
		"tc-456",
		"tool-b",
		`{}`,
	)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if record.ScenarioName != "scenario-b" {
		t.Errorf("expected ScenarioName 'scenario-b', got '%s'", record.ScenarioName)
	}

	// Verify h1 was not called
	if len(h1.execCalls) != 0 {
		t.Errorf("expected h1 to have 0 calls, got %d", len(h1.execCalls))
	}
	// Verify h2 was called
	if len(h2.execCalls) != 1 {
		t.Errorf("expected h2 to have 1 call, got %d", len(h2.execCalls))
	}
}

// TestIsKnownTool_ScenarioHandlers verifies tool lookup with scenario handlers.
func TestIsKnownTool_ScenarioHandlers(t *testing.T) {
	h := newTestHandler("scenario", "tool-a", "tool-b")
	exec := NewToolExecutorWithHandlers(h)

	tests := []struct {
		toolName string
		expected bool
	}{
		{"tool-a", true},
		{"tool-b", true},
		{"tool-c", false},
		{"", false},
	}

	for _, tc := range tests {
		if got := exec.IsKnownTool(tc.toolName); got != tc.expected {
			t.Errorf("IsKnownTool(%q) = %v, want %v", tc.toolName, got, tc.expected)
		}
	}
}

// TestGetToolScenario verifies scenario lookup.
func TestGetToolScenario(t *testing.T) {
	h1 := newTestHandler("scenario-a", "tool-a")
	h2 := newTestHandler("scenario-b", "tool-b")
	exec := NewToolExecutorWithHandlers(h1, h2)

	tests := []struct {
		toolName string
		expected string
	}{
		{"tool-a", "scenario-a"},
		{"tool-b", "scenario-b"},
		{"unknown", ""},
	}

	for _, tc := range tests {
		if got := exec.GetToolScenario(tc.toolName); got != tc.expected {
			t.Errorf("GetToolScenario(%q) = %q, want %q", tc.toolName, got, tc.expected)
		}
	}
}

// TestExecuteTool_RecordTimestamps verifies timestamps are set correctly.
func TestExecuteTool_RecordTimestamps(t *testing.T) {
	h := newTestHandler("scenario", "my-tool")
	exec := NewToolExecutorWithHandlers(h)

	beforeExec := time.Now()
	record, err := exec.ExecuteTool(
		context.Background(),
		"chat-123",
		"tc-456",
		"my-tool",
		`{}`,
	)
	afterExec := time.Now()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if record.StartedAt.Before(beforeExec) || record.StartedAt.After(afterExec) {
		t.Errorf("StartedAt %v not in expected range [%v, %v]", record.StartedAt, beforeExec, afterExec)
	}

	if record.CompletedAt.Before(record.StartedAt) || record.CompletedAt.After(afterExec) {
		t.Errorf("CompletedAt %v not in expected range [%v, %v]", record.CompletedAt, record.StartedAt, afterExec)
	}
}

// TestUnknownToolError_Error verifies error message format.
func TestUnknownToolError_Error(t *testing.T) {
	err := &UnknownToolError{ToolName: "my-tool"}
	expected := "unknown tool: my-tool"
	if err.Error() != expected {
		t.Errorf("expected error '%s', got '%s'", expected, err.Error())
	}
}

// TestExecuteTool_ContextCancellation verifies context is passed through.
func TestExecuteTool_ContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	h := newTestHandler("scenario", "my-tool")
	var receivedCtx context.Context
	h.execFunc = func(ctx context.Context, toolName string, args map[string]interface{}) (interface{}, error) {
		receivedCtx = ctx
		return map[string]string{"status": "ok"}, nil
	}
	exec := NewToolExecutorWithHandlers(h)

	_, _ = exec.ExecuteTool(ctx, "chat", "tc", "my-tool", `{}`)

	// Verify the cancelled context was passed to handler
	if receivedCtx == nil {
		t.Fatal("expected context to be passed to handler")
	}
	if receivedCtx.Err() == nil {
		t.Error("expected context to be cancelled")
	}
}

// TestExecuteTool_ConcurrentAccess verifies thread safety.
func TestExecuteTool_ConcurrentAccess(t *testing.T) {
	h := newTestHandler("scenario", "my-tool")
	exec := NewToolExecutorWithHandlers(h)

	const goroutines = 10
	done := make(chan bool, goroutines)

	for i := 0; i < goroutines; i++ {
		go func(id int) {
			_, err := exec.ExecuteTool(
				context.Background(),
				"chat",
				"tc",
				"my-tool",
				`{}`,
			)
			if err != nil {
				t.Errorf("goroutine %d: unexpected error: %v", id, err)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < goroutines; i++ {
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("timeout waiting for goroutines")
		}
	}

	// All calls should have succeeded
	if h.callCount() != goroutines {
		t.Errorf("expected %d calls, got %d", goroutines, h.callCount())
	}
}

// TestRegisterProtocolHandler verifies protocol handler registration.
func TestRegisterProtocolHandler(t *testing.T) {
	exec := NewToolExecutor()
	exec.RegisterProtocolHandler("my-scenario", "http://localhost:8080", []string{"tool-1", "tool-2"})

	if !exec.IsKnownTool("tool-1") {
		t.Error("expected tool-1 to be known")
	}
	if !exec.IsKnownTool("tool-2") {
		t.Error("expected tool-2 to be known")
	}
	if exec.GetToolScenario("tool-1") != "my-scenario" {
		t.Errorf("expected scenario 'my-scenario', got '%s'", exec.GetToolScenario("tool-1"))
	}
}
