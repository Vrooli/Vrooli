// Responsibility: retain repository test declarations within their original package.
package database

import (
	"agent-manager/internal/domain"
	"context"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
	"strings"
	"testing"
)

func TestEventRepository(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()

	// Create task and run first
	task := &domain.Task{ID: uuid.New(), Title: "Event Task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("Create task: %v", err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("Create run: %v", err)
	}

	// Append events
	events := []*domain.RunEvent{
		{EventType: domain.EventTypeLog, Data: &domain.LogEventData{Message: "line 1", Level: "info"}},
		{EventType: domain.EventTypeLog, Data: &domain.LogEventData{Message: "line 2", Level: "info"}},
		{EventType: domain.EventTypeStatus, Data: &domain.StatusEventData{OldStatus: string(domain.RunStatusPending), NewStatus: string(domain.RunStatusRunning), Reason: "started"}},
	}
	if err := repos.Events.Append(ctx, run.ID, events...); err != nil {
		t.Fatalf("Append: %v", err)
	}

	// Get all events
	gotEvents, err := repos.Events.Get(ctx, run.ID, -1, 100)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if len(gotEvents) != 3 {
		t.Errorf("expected 3 events, got %d", len(gotEvents))
	}

	// Check sequence numbers
	for i, evt := range gotEvents {
		if evt.Sequence != int64(i) {
			t.Errorf("expected sequence %d, got %d", i, evt.Sequence)
		}
	}

	// Get events after sequence
	afterEvents, err := repos.Events.Get(ctx, run.ID, 0, 100)
	if err != nil {
		t.Fatalf("Get after sequence: %v", err)
	}
	if len(afterEvents) != 2 {
		t.Errorf("expected 2 events after sequence 0, got %d", len(afterEvents))
	}

	// Get by type
	logEvents, err := repos.Events.GetByType(ctx, run.ID, []domain.RunEventType{domain.EventTypeLog}, 100)
	if err != nil {
		t.Fatalf("GetByType: %v", err)
	}
	if len(logEvents) != 2 {
		t.Errorf("expected 2 log events, got %d", len(logEvents))
	}

	// Count
	count, err := repos.Events.Count(ctx, run.ID)
	if err != nil {
		t.Fatalf("Count: %v", err)
	}
	if count != 3 {
		t.Errorf("expected count 3, got %d", count)
	}

	// Delete
	if err := repos.Events.Delete(ctx, run.ID); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	count, err = repos.Events.Count(ctx, run.ID)
	if err != nil {
		t.Fatalf("Count after delete: %v", err)
	}
	if count != 0 {
		t.Errorf("expected count 0 after delete, got %d", count)
	}
}

func TestEventRepositoryReadsLegacyTypedPayloadRow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "legacy event task", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatalf("create task: %v", err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatalf("create run: %v", err)
	}

	// This JSON was written by the removed union payload. Existing SQLite rows
	// remain in place and must decode into the current typed event schema.
	if _, err := db.ExecContext(ctx, `INSERT INTO run_events (id, run_id, sequence, event_type, data) VALUES (?, ?, 0, ?, ?)`, uuid.New().String(), run.ID.String(), domain.EventTypeToolResult, `{"toolName":"Read","toolOutput":"file contents","toolError":"permission denied"}`); err != nil {
		t.Fatalf("seed legacy event row: %v", err)
	}
	if err := db.migrateRunEventPayloads(ctx); err != nil {
		t.Fatalf("migrate legacy event row: %v", err)
	}
	var persisted string
	if err := db.GetContext(ctx, &persisted, `SELECT data FROM run_events WHERE run_id = ?`, run.ID.String()); err != nil {
		t.Fatalf("read migrated event JSON: %v", err)
	}
	if strings.Contains(persisted, "toolOutput") || strings.Contains(persisted, "toolError") {
		t.Fatalf("legacy event fields remain after migration: %s", persisted)
	}

	events, err := repos.Events.Get(ctx, run.ID, -1, 10)
	if err != nil {
		t.Fatalf("read legacy event row: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	payload, ok := events[0].Data.(*domain.ToolResultEventData)
	if !ok {
		t.Fatalf("payload = %T, want *domain.ToolResultEventData", events[0].Data)
	}
	if payload.ToolName != "Read" || payload.Output != "file contents" || payload.Error != "permission denied" || payload.Success {
		t.Fatalf("migrated payload = %#v", payload)
	}
}

// ============================================================================
// Checkpoint Repository Tests
// ============================================================================
