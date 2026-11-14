package services

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"agent-inbox/domain"
	"agent-inbox/persistence"

	_ "modernc.org/sqlite"
)

func TestStartupReconciliationCancelsUntrackedExecutionsAndPreservesTerminalRows(t *testing.T) {
	ctx := context.Background()
	db, err := sql.Open("sqlite", ":memory:?_pragma=foreign_keys(ON)")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	repo := persistence.NewRepository(db)
	if err := repo.InitSchema(ctx); err != nil {
		t.Fatal(err)
	}
	chat, err := repo.CreateChat(ctx, "recovery", "model", domain.ViewModeBubble, domain.ChatModeLLM)
	if err != nil {
		t.Fatal(err)
	}
	message, err := repo.SaveAssistantMessage(ctx, chat.ID, "model", "interrupted tools", 0, "")
	if err != nil {
		t.Fatal(err)
	}
	for _, status := range []domain.ToolCallStatus{domain.StatusPending, domain.StatusRunning} {
		record := &domain.ToolCallRecord{ID: "orphan-" + string(status), ChatID: chat.ID, ToolName: "local-tool", Status: status, StartedAt: time.Now()}
		if err := repo.SaveToolCallRecord(ctx, message.ID, record); err != nil {
			t.Fatalf("insert %s orphan: %v", status, err)
		}
	}
	terminal := &domain.ToolCallRecord{ID: "finished", ChatID: chat.ID, ToolName: "local-tool", Status: domain.StatusCompleted, StartedAt: time.Now(), CompletedAt: time.Now()}
	if err := repo.SaveToolCallRecord(ctx, message.ID, terminal); err != nil {
		t.Fatal(err)
	}

	reconciler := NewReconciliationService(repo)
	count, err := reconciler.ReconcileOrphanedToolCalls(ctx)
	if err != nil || count != 2 {
		t.Fatalf("reconcile returned count=%d err=%v, want two resolved rows", count, err)
	}
	for _, id := range []string{"orphan-pending", "orphan-running"} {
		record, err := repo.GetToolCallByID(ctx, id)
		if err != nil {
			t.Fatal(err)
		}
		if record.Status != domain.StatusCancelled || record.CompletedAt.IsZero() {
			t.Errorf("recovered %s = status %q completed_at %v", id, record.Status, record.CompletedAt)
		}
	}
	finished, err := repo.GetToolCallByID(ctx, terminal.ID)
	if err != nil || finished.Status != domain.StatusCompleted {
		t.Fatalf("terminal row changed after recovery: status=%q err=%v", finished.Status, err)
	}
}
