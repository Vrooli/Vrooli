package toolexecution

import (
	"context"
	"testing"
)

func TestNewServerExecutor(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{})
	if executor == nil {
		t.Fatal("expected executor instance")
	}
}

func TestExecuteUnknownTool(t *testing.T) {
	executor := NewServerExecutor(ServerExecutorConfig{})

	result, err := executor.Execute(context.Background(), "__unknown_tool__", map[string]interface{}{})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if result == nil {
		t.Fatal("expected execution result")
	}
	if result.Success {
		t.Fatal("expected unknown tool execution to fail")
	}
	if result.Code != CodeUnknownTool {
		t.Fatalf("expected code %q, got %q", CodeUnknownTool, result.Code)
	}
}
