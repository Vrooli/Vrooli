package main

import (
	"context"
	"testing"
	"time"

	"agent-inbox/domain"
)

func TestToolCallQueriesPreserveHistoryAndOrphanContracts(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)
	chat := createTestChat(t, ts)
	base := time.Date(2026, 10, 28, 12, 0, 0, 123456789, time.UTC)
	for i, status := range []domain.ToolCallStatus{domain.StatusPending, domain.StatusRunning, domain.StatusCompleted} {
		message := addTestMessage(t, ts, chat.ID, "assistant", "tool call")
		record := &domain.ToolCallRecord{
			ID: []string{"older", "newer", "done"}[i], ChatID: chat.ID,
			ToolName: "search", Arguments: `{"query":"x"}`, Result: `{"ok":true}`,
			Status: status, ScenarioName: "web-console", ExternalRunID: "run-" + string(status),
			StartedAt: base.Add(time.Duration(i) * time.Minute),
		}
		if status == domain.StatusCompleted {
			record.CompletedAt = base.Add(4 * time.Minute)
		}
		if err := ts.repo.SaveToolCallRecord(context.Background(), message.ID, record); err != nil {
			t.Fatalf("SaveToolCallRecord(%s): %v", status, err)
		}
	}
	var saved int
	if err := ts.repo.DB().QueryRowContext(context.Background(), "SELECT COUNT(*) FROM tool_calls WHERE chat_id = ?", chat.ID).Scan(&saved); err != nil || saved != 3 {
		t.Fatalf("saved tool-call rows = %d, %v", saved, err)
	}

	history, err := ts.repo.ListToolCallsForChat(context.Background(), chat.ID)
	if err != nil {
		t.Fatalf("ListToolCallsForChat: %v", err)
	}
	if len(history) != 3 || history[0].ID != "done" || history[1].ID != "newer" || history[2].ID != "older" {
		t.Fatalf("chat history order = %+v", history)
	}
	if history[0].ScenarioName != "web-console" || history[0].ExternalRunID != "run-completed" || history[0].Result != `{"ok":true}` || !history[0].CompletedAt.Equal(base.Add(4*time.Minute)) {
		t.Fatalf("chat history fields = %+v", history[0])
	}

	orphans, err := ts.repo.ListOrphanedToolCalls(context.Background())
	if err != nil {
		t.Fatalf("ListOrphanedToolCalls: %v", err)
	}
	if len(orphans) != 2 || orphans[0].ID != "older" || orphans[1].ID != "newer" {
		t.Fatalf("orphan order/filter = %+v", orphans)
	}
	if !orphans[0].CompletedAt.IsZero() {
		t.Fatalf("pending call completed_at = %v, want zero", orphans[0].CompletedAt)
	}

	empty, err := ts.repo.ListToolCallsForChat(context.Background(), "no-tool-calls")
	if err != nil || empty == nil || len(empty) != 0 {
		t.Fatalf("empty chat query = %#v, %v; want non-nil empty slice", empty, err)
	}
}

func TestToolCallRepositoryEnforcesLifecycleAcrossUpsertsAndRecovery(t *testing.T) {
	ts := setupTestServer(t)
	defer ts.cleanup(t)
	chat := createTestChat(t, ts)
	message := addTestMessage(t, ts, chat.ID, "assistant", "tool call")
	ctx := context.Background()
	record := &domain.ToolCallRecord{
		ID: "approval-lifecycle", ChatID: chat.ID, ToolName: "search", Arguments: "{}",
		Status: domain.StatusPendingApproval, StartedAt: time.Now(),
	}
	if err := ts.repo.SaveToolCallRecord(ctx, message.ID, record); err != nil {
		t.Fatalf("insert pending approval: %v", err)
	}
	if err := ts.repo.UpdateToolCallStatus(ctx, record.ID, domain.StatusApproved, ""); err != nil {
		t.Fatalf("approve: %v", err)
	}
	record.Status = domain.StatusCompleted
	record.Result = `{"ok":true}`
	record.CompletedAt = time.Now()
	if err := ts.repo.SaveToolCallRecord(ctx, message.ID, record); err != nil {
		t.Fatalf("persist completion: %v", err)
	}
	if err := ts.repo.UpdateToolCallStatus(ctx, record.ID, domain.StatusRunning, ""); err == nil {
		t.Fatal("terminal record was resurrected by status update")
	}
	if err := ts.repo.SaveToolCallRecord(ctx, message.ID, &domain.ToolCallRecord{
		ID: record.ID, ChatID: chat.ID, ToolName: "search", Arguments: "{}", Status: domain.StatusFailed,
	}); err == nil {
		t.Fatal("terminal record was overwritten by an invalid upsert")
	}
	tx, err := ts.repo.BeginTx(ctx, nil)
	if err != nil {
		t.Fatalf("begin guarded transaction: %v", err)
	}
	if err := ts.repo.SaveToolCallRecordTx(ctx, tx, message.ID, &domain.ToolCallRecord{
		ID: record.ID, ChatID: chat.ID, ToolName: "search", Arguments: "{}", Status: domain.StatusFailed,
	}); err == nil {
		_ = tx.Rollback()
		t.Fatal("transactional upsert overwrote a terminal record")
	}
	_ = tx.Rollback()

	orphan := &domain.ToolCallRecord{ID: "recovery-lifecycle", ChatID: chat.ID, ToolName: "agent", Status: domain.StatusRunning, StartedAt: time.Now()}
	if err := ts.repo.SaveToolCallRecord(ctx, message.ID, orphan); err != nil {
		t.Fatalf("insert recoverable execution: %v", err)
	}
	if err := ts.repo.UpdateToolCallStatus(ctx, orphan.ID, domain.StatusCompleted, "recovered result"); err != nil {
		t.Fatalf("reconcile active persisted execution: %v", err)
	}
	if err := ts.repo.UpdateToolCallStatus(ctx, orphan.ID, domain.StatusCancelled, "late recovery"); err == nil {
		t.Fatal("terminal recovery record accepted a second outcome")
	}
}
