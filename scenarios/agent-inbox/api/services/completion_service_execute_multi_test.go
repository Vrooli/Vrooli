package services

import (
	"context"
	"testing"

	"agent-inbox/domain"
)

func TestExecuteToolCalls_MultipleTools_Success(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "tool_a", `{}`),
		makeToolCall("tc-2", "tool_b", `{}`),
		makeToolCall("tc-3", "tool_c", `{}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(outcome.Results) != 3 {
		t.Fatalf("expected 3 results, got %d", len(outcome.Results))
	}

	// Verify all tools executed
	calls := executor.GetExecuteCalls()
	if len(calls) != 3 {
		t.Fatalf("expected 3 executor calls, got %d", len(calls))
	}

	// Verify order preserved
	expectedNames := []string{"tool_a", "tool_b", "tool_c"}
	for i, call := range calls {
		if call.ToolName != expectedNames[i] {
			t.Errorf("call %d: expected %s, got %s", i, expectedNames[i], call.ToolName)
		}
	}
}
