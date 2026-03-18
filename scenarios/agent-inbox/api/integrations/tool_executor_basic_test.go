package integrations

import (
	"context"
	"errors"
	"sync"
	"testing"

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
