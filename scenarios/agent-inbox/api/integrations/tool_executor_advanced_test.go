package integrations

import (
	"context"
	"testing"
	"time"
)

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
