package services

import (
	"context"
	"testing"
	"time"

	"agent-inbox/domain"
)

// =============================================================================
// Tests for ApproveToolCall
// =============================================================================

func TestApproveToolCall_Success(t *testing.T) {
	repo := newMockCompletionRepository()
	executor := newMockToolExecutor()

	// Add a pending approval record
	record := &domain.ToolCallRecord{
		ID:        "tc-pending",
		ChatID:    "chat-1",
		MessageID: "msg-1",
		ToolName:  "dangerous_tool",
		Arguments: `{"target": "important"}`,
		Status:    domain.StatusPendingApproval,
		StartedAt: time.Now(),
	}
	repo.AddToolCallRecord(record)

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo:     repo,
		Executor: executor,
	})

	result, err := svc.ApproveToolCall(context.Background(), "chat-1", "tc-pending")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify tool was executed
	if len(executor.GetExecuteCalls()) != 1 {
		t.Error("expected tool to be executed after approval")
	}

	// Verify status was updated to approved first
	if len(repo.updateToolCallStatusCalls) < 1 {
		t.Fatal("expected status update")
	}
	if repo.updateToolCallStatusCalls[0].Status != domain.StatusApproved {
		t.Errorf("expected status %s, got %s", domain.StatusApproved, repo.updateToolCallStatusCalls[0].Status)
	}

	// Verify result
	if result.ToolResult == nil {
		t.Error("expected tool result")
	}
}

func TestApproveToolCall_NotFound(t *testing.T) {
	repo := newMockCompletionRepository()
	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	_, err := svc.ApproveToolCall(context.Background(), "chat-1", "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent tool call")
	}
}

func TestApproveToolCall_WrongChat(t *testing.T) {
	repo := newMockCompletionRepository()
	repo.AddToolCallRecord(&domain.ToolCallRecord{
		ID:     "tc-1",
		ChatID: "chat-other", // Different chat
		Status: domain.StatusPendingApproval,
	})

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	_, err := svc.ApproveToolCall(context.Background(), "chat-1", "tc-1")
	if err == nil {
		t.Error("expected error for wrong chat")
	}
}

func TestApproveToolCall_NotPending(t *testing.T) {
	repo := newMockCompletionRepository()
	repo.AddToolCallRecord(&domain.ToolCallRecord{
		ID:     "tc-1",
		ChatID: "chat-1",
		Status: domain.StatusCompleted, // Already completed
	})

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	_, err := svc.ApproveToolCall(context.Background(), "chat-1", "tc-1")
	if err == nil {
		t.Error("expected error for non-pending tool call")
	}
}

// =============================================================================
// Tests for RejectToolCall
// =============================================================================

func TestRejectToolCall_Success(t *testing.T) {
	repo := newMockCompletionRepository()

	record := &domain.ToolCallRecord{
		ID:        "tc-pending",
		ChatID:    "chat-1",
		MessageID: "msg-1",
		ToolName:  "dangerous_tool",
		Status:    domain.StatusPendingApproval,
	}
	repo.AddToolCallRecord(record)

	svc := NewCompletionServiceWithDeps(CompletionServiceDeps{
		Repo: repo,
	})

	err := svc.RejectToolCall(context.Background(), "chat-1", "tc-pending", "Too risky")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify status was updated to rejected
	if len(repo.updateToolCallStatusCalls) != 1 {
		t.Fatal("expected status update")
	}
	if repo.updateToolCallStatusCalls[0].Status != domain.StatusRejected {
		t.Errorf("expected status %s, got %s", domain.StatusRejected, repo.updateToolCallStatusCalls[0].Status)
	}

	// Verify rejection result was saved
	if len(repo.saveToolResponseMessageCalls) != 1 {
		t.Error("expected tool response message for rejection")
	}
}
