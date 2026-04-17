package services

import (
	"agent-inbox/domain"
	"context"
	"testing"
)

func TestExecuteToolCalls_MultipleTools_Success(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	registry.addTool("scenario", createSimpleTool("tool_a", "Tool A"))
	registry.addTool("scenario", createSimpleTool("tool_b", "Tool B"))
	registry.addTool("scenario", createSimpleTool("tool_c", "Tool C"))

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
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

func TestExecuteToolCalls_WithApprovalRequired(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	registry.addTool("scenario", createSimpleTool("dangerous_tool", "A dangerous tool"))
	registry.ApprovalRequirements["dangerous_tool"] = true

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "dangerous_tool", `{"target": "important"}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify approval is pending
	if !outcome.HasPendingApprovals {
		t.Error("expected pending approvals")
	}
	if len(outcome.PendingApprovals) != 1 {
		t.Fatalf("expected 1 pending approval, got %d", len(outcome.PendingApprovals))
	}

	// Verify executor was NOT called
	if len(executor.GetExecuteCalls()) != 0 {
		t.Error("executor should not be called for tools requiring approval")
	}

	// Verify result indicates pending
	if outcome.Results[0].Status != domain.StatusPendingApproval {
		t.Errorf("expected status %s, got %s", domain.StatusPendingApproval, outcome.Results[0].Status)
	}
}
