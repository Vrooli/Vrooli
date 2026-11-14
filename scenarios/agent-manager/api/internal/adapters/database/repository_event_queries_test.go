// Responsibility: retain repository test declarations within their original package.
package database

import (
	"context"
	"errors"
	"testing"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite"
)

func TestSQLiteStoreReadsLegacyTypedPayloadRow(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	repos := NewRepositories(db, logrus.New())
	store := event.NewSQLiteStore(db, logrus.New())
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
	// stay as written and must decode into the current typed event schema.
	if _, err := db.ExecContext(ctx, `INSERT INTO run_events (id, run_id, sequence, event_type, data) VALUES (?, ?, 0, ?, ?)`, uuid.New().String(), run.ID.String(), domain.EventTypeToolResult, `{"toolName":"Read","toolOutput":"file contents","toolError":"permission denied"}`); err != nil {
		t.Fatalf("seed legacy event row: %v", err)
	}
	events, err := store.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1, Limit: 10})
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

func TestInitializeSchemaPreservesLegacyEventBytesAndReadsThem(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	store := event.NewSQLiteStore(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "legacy startup", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	legacy := `{"toolName":"Read","toolOutput":"retained evidence","toolError":"permission denied"}`
	if _, err := db.ExecContext(ctx, `INSERT INTO run_events (id, run_id, sequence, event_type, data) VALUES (?, ?, 0, ?, ?)`, uuid.New().String(), run.ID.String(), domain.EventTypeToolResult, legacy); err != nil {
		t.Fatal(err)
	}
	if err := db.InitializeSchema(); err != nil {
		t.Fatal(err)
	}
	var persisted string
	if err := db.GetContext(ctx, &persisted, `SELECT data FROM run_events WHERE run_id = ?`, run.ID.String()); err != nil {
		t.Fatal(err)
	}
	if persisted != legacy {
		t.Fatal("startup rewrote historical evidence")
	}
	events, err := store.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1, Limit: 10})
	if err != nil || len(events) != 1 {
		t.Fatal("legacy read failed", err)
	}
	payload, ok := events[0].Data.(*domain.ToolResultEventData)
	if !ok || payload.Output != "retained evidence" || payload.Error != "permission denied" {
		t.Fatalf("legacy payload lost meaning: %+v", events[0].Data)
	}
}

// ============================================================================
// Checkpoint Repository Tests
// ============================================================================

func TestSQLiteStoreMalformedPayloadFailsFilteredAndUnfilteredGet(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repos := NewRepositories(db, logrus.New())
	store := event.NewSQLiteStore(db, logrus.New())
	ctx := context.Background()
	task := &domain.Task{ID: uuid.New(), Title: "event decode", ScopePath: "/test", Status: domain.TaskStatusQueued}
	if err := repos.Tasks.Create(ctx, task); err != nil {
		t.Fatal(err)
	}
	run := &domain.Run{ID: uuid.New(), TaskID: task.ID, Status: domain.RunStatusRunning, Phase: domain.RunPhaseExecuting, ApprovalState: domain.ApprovalStateNone}
	if err := repos.Runs.Create(ctx, run); err != nil {
		t.Fatal(err)
	}
	id := uuid.New()
	raw := `{"limitType":"weekly","usedPercent":"bad"}`
	if _, err := db.ExecContext(ctx, `INSERT INTO run_events(id,run_id,sequence,event_type,timestamp,data) VALUES(?,?,?,?,?,?)`, id, run.ID, 0, "quota", time.Now().UTC().Format(time.RFC3339Nano), raw); err != nil {
		t.Fatal(err)
	}
	for _, read := range []func() ([]*domain.RunEvent, error){func() ([]*domain.RunEvent, error) {
		return store.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1, Limit: 100})
	}, func() ([]*domain.RunEvent, error) {
		return store.Get(ctx, run.ID, event.GetOptions{AfterSequence: -1, EventTypes: []domain.RunEventType{domain.EventTypeQuota}, Limit: 100})
	}} {
		got, err := read()
		var databaseError *domain.DatabaseError
		if got != nil || !errors.As(err, &databaseError) || databaseError.EntityID != id.String() || databaseError.Operation != "decode_event" {
			t.Fatalf("events=%+v error=%v", got, err)
		}
	}
	var persisted string
	if err := db.GetContext(ctx, &persisted, `SELECT data FROM run_events WHERE id=?`, id); err != nil || persisted != raw {
		t.Fatalf("persisted=%q error=%v", persisted, err)
	}
}
