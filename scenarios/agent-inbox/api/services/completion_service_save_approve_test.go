package services

import (
	"agent-inbox/domain"
	"context"
	"encoding/json"
	"errors"
	"testing"
)

func TestExecuteToolCalls_MixedApproval(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	registry.addTool("scenario", createSimpleTool("safe_tool", "A safe tool"))
	registry.addTool("scenario", createSimpleTool("dangerous_tool", "A dangerous tool"))
	registry.ApprovalRequirements["safe_tool"] = false
	registry.ApprovalRequirements["dangerous_tool"] = true

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "safe_tool", `{}`),
		makeToolCall("tc-2", "dangerous_tool", `{}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify mixed results
	if len(outcome.Results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(outcome.Results))
	}
	if outcome.Results[0].Status != domain.StatusCompleted {
		t.Errorf("safe tool: expected %s, got %s", domain.StatusCompleted, outcome.Results[0].Status)
	}
	if outcome.Results[1].Status != domain.StatusPendingApproval {
		t.Errorf("dangerous tool: expected %s, got %s", domain.StatusPendingApproval, outcome.Results[1].Status)
	}

	// Verify only safe tool executed
	if len(executor.GetExecuteCalls()) != 1 {
		t.Errorf("expected 1 executor call, got %d", len(executor.GetExecuteCalls()))
	}
}

func TestExecuteToolCalls_ExecutionFailure(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	registry.addTool("scenario", createSimpleTool("failing_tool", "A failing tool"))

	// Make executor return an error
	executor.executeError = errors.New("tool execution failed")

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "failing_tool", `{}`),
	}

	outcome, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")

	// Should return error but also include the result
	if err == nil {
		t.Error("expected error for failed tool execution")
	}
	if outcome == nil {
		t.Fatal("expected outcome even on error")
	}
	if len(outcome.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(outcome.Results))
	}

	// Result should still be saved
	if len(repo.saveToolResponseMessageCalls) != 1 {
		t.Error("tool response should still be saved on failure")
	}
}

func TestExecuteToolCalls_SkillsInjection(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()
	registry := newMockToolRegistry()

	registry.addTool("scenario", createSimpleTool("tool_with_skills", "Tool that receives skills"))

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
		Registry: registry,
	})

	// Set skills
	svc.SetSkills([]SkillPayload{
		{
			Key:     "skill1",
			Label:   "Skill 1",
			Content: "Skill content here",
		},
	})

	toolCalls := []domain.ToolCall{
		makeToolCall("tc-1", "tool_with_skills", `{"input": "test"}`),
	}

	_, err := svc.ExecuteToolCalls(context.Background(), "chat-1", "msg-1", toolCalls, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify skills were injected into arguments
	calls := executor.GetExecuteCalls()
	if len(calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(calls))
	}

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(calls[0].Arguments), &args); err != nil {
		t.Fatalf("failed to parse arguments: %v", err)
	}

	if _, ok := args["_context_attachments"]; !ok {
		t.Error("expected _context_attachments in arguments")
	}
}

// =============================================================================
// Tests for SaveCompletionResult
// =============================================================================

func TestSaveCompletionResult_RegularMessage(t *testing.T) {
	repo := newMockCompletionRepository()
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	result := &domain.CompletionResult{
		Content:      "Hello, how can I help you?",
		FinishReason: "stop",
		TokenCount:   10,
	}

	msg, err := svc.SaveCompletionResult(context.Background(), "chat-1", "gpt-4", result, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg == nil {
		t.Fatal("expected message")
	}

	// Verify regular message save was called (not with tool calls)
	if len(repo.saveAssistantMessageCalls) != 1 {
		t.Fatalf("expected 1 save call, got %d", len(repo.saveAssistantMessageCalls))
	}

	call := repo.saveAssistantMessageCalls[0]
	if call.Content != "Hello, how can I help you?" {
		t.Errorf("unexpected content: %s", call.Content)
	}
	if len(call.ToolCalls) != 0 {
		t.Error("expected no tool calls for regular message")
	}
}

func TestSaveCompletionResult_WithToolCalls(t *testing.T) {
	repo := newMockCompletionRepository()
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	result := &domain.CompletionResult{
		Content:      "",
		FinishReason: "tool_calls",
		ToolCalls: []domain.ToolCall{
			makeToolCall("tc-1", "search", `{"query": "test"}`),
		},
	}

	msg, err := svc.SaveCompletionResult(context.Background(), "chat-1", "gpt-4", result, "parent-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msg == nil {
		t.Fatal("expected message")
	}

	// Verify message with tool calls was saved
	if len(repo.saveAssistantMessageCalls) != 1 {
		t.Fatalf("expected 1 save call, got %d", len(repo.saveAssistantMessageCalls))
	}

	call := repo.saveAssistantMessageCalls[0]
	if len(call.ToolCalls) != 1 {
		t.Errorf("expected 1 tool call, got %d", len(call.ToolCalls))
	}
}

func TestSaveCompletionResult_UpdatesActiveLeaf(t *testing.T) {
	repo := newMockCompletionRepository()
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	result := &domain.CompletionResult{
		Content: "Test message",
	}

	msg, _ := svc.SaveCompletionResult(context.Background(), "chat-1", "gpt-4", result, "parent-1")

	// Verify active leaf was updated
	if len(repo.setActiveLeafCalls) != 1 {
		t.Fatalf("expected 1 SetActiveLeaf call, got %d", len(repo.setActiveLeafCalls))
	}

	if repo.setActiveLeafCalls[0].MessageID != msg.ID {
		t.Error("active leaf should be set to new message ID")
	}
}
