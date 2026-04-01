package eventlog_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"

	_ "modernc.org/sqlite"

	"swarm-manager/internal/eventlog"
)

func setupEmitter(t *testing.T) (*eventlog.Emitter, *eventlog.SQLiteRepository) {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	repo := eventlog.NewSQLiteRepository(db)
	if err := repo.InitSchema(context.Background()); err != nil {
		t.Fatalf("init schema: %v", err)
	}
	return eventlog.NewEmitter(repo), repo
}

func lastEvent(t *testing.T, repo *eventlog.SQLiteRepository) eventlog.Event {
	t.Helper()
	events, err := repo.All(context.Background())
	if err != nil {
		t.Fatalf("all: %v", err)
	}
	if len(events) == 0 {
		t.Fatal("no events found")
	}
	return events[len(events)-1]
}

func TestEmitBacklogCreated(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitBacklogCreated("execute/my-item", "execute", "backlog", 5, "init-a", "M")

	e := lastEvent(t, repo)
	if e.EntityType != eventlog.EntityBacklogItem {
		t.Errorf("entity_type: got %q", e.EntityType)
	}
	if e.EntityID != "execute/my-item" {
		t.Errorf("entity_id: got %q", e.EntityID)
	}
	if e.EventType != eventlog.EventBacklogCreated {
		t.Errorf("event_type: got %q", e.EventType)
	}

	var p eventlog.BacklogCreatedPayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Kind != "execute" || p.Priority != 5 || p.Initiative != "init-a" || p.Effort != "M" {
		t.Errorf("payload: %+v", p)
	}
}

func TestEmitBacklogStatusChanged(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitBacklogStatusChanged("fix/bug-1", "backlog", "in_progress")

	e := lastEvent(t, repo)
	if e.EventType != eventlog.EventBacklogStatusChanged {
		t.Errorf("event_type: got %q", e.EventType)
	}

	var p eventlog.StatusChangePayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.From != "backlog" || p.To != "in_progress" {
		t.Errorf("payload: %+v", p)
	}
}

func TestEmitExecutionCompleted(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitExecutionCompleted("exec-123", 45.5, true)

	e := lastEvent(t, repo)
	if e.EntityType != eventlog.EntityExecution {
		t.Errorf("entity_type: got %q", e.EntityType)
	}
	if e.EventType != eventlog.EventExecutionCompleted {
		t.Errorf("event_type: got %q", e.EventType)
	}

	var p eventlog.ExecutionCompletedPayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.DurationSeconds != 45.5 || !p.HadFixups {
		t.Errorf("payload: %+v", p)
	}
}

func TestEmitInitiativeItemAdded(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitInitiativeItemAdded("init-a", "execute/my-item")

	e := lastEvent(t, repo)
	if e.EntityType != eventlog.EntityInitiative {
		t.Errorf("entity_type: got %q", e.EntityType)
	}
	if e.EventType != eventlog.EventInitiativeItemAdded {
		t.Errorf("event_type: got %q", e.EventType)
	}

	var p eventlog.InitiativeItemPayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Item != "execute/my-item" {
		t.Errorf("payload: %+v", p)
	}
}

func TestEmitQueued(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitQueued("execute", "my-item", 3)

	e := lastEvent(t, repo)
	if e.EntityType != eventlog.EntityQueue {
		t.Errorf("entity_type: got %q", e.EntityType)
	}
	if e.EventType != eventlog.EventQueued {
		t.Errorf("event_type: got %q", e.EventType)
	}
	if e.EntityID != "execute/my-item" {
		t.Errorf("entity_id: got %q", e.EntityID)
	}

	var p eventlog.QueuePayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.Position != 3 {
		t.Errorf("payload: %+v", p)
	}
}

func TestEmitNilPayload(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitInitiativeArchived("init-old")

	e := lastEvent(t, repo)
	if e.EventType != eventlog.EventInitiativeArchived {
		t.Errorf("event_type: got %q", e.EventType)
	}
	// Nil payload should result in nil metadata.
	if e.Metadata != nil {
		t.Errorf("expected nil metadata, got %s", e.Metadata)
	}
}

func TestEmitClarificationStarted(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitClarificationStarted("idea/my-item", 2, "d1", true)

	e := lastEvent(t, repo)
	if e.EntityType != eventlog.EntityBacklogItem {
		t.Errorf("entity_type: got %q", e.EntityType)
	}
	if e.EventType != eventlog.EventClarificationStarted {
		t.Errorf("event_type: got %q", e.EventType)
	}

	var p eventlog.ClarificationStartedPayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.RoundNumber != 2 || p.ItemID != "d1" || !p.HasMessage {
		t.Errorf("payload: %+v", p)
	}
}

func TestEmitClarificationResolved(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitClarificationResolved("fix/bug-1", 1, "d3", 4, "decision")

	e := lastEvent(t, repo)
	if e.EventType != eventlog.EventClarificationResolved {
		t.Errorf("event_type: got %q", e.EventType)
	}

	var p eventlog.ClarificationResolvedPayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.RoundNumber != 1 || p.ItemID != "d3" || p.MessageCount != 4 || p.ImpactLevel != "decision" {
		t.Errorf("payload: %+v", p)
	}
}

func TestEmitClarificationAction(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitClarificationAction("chore/cleanup", 3, "d2", "invalidate_round")

	e := lastEvent(t, repo)
	if e.EventType != eventlog.EventClarificationAction {
		t.Errorf("event_type: got %q", e.EventType)
	}

	var p eventlog.ClarificationActionPayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.RoundNumber != 3 || p.ItemID != "d2" || p.Action != "invalidate_round" {
		t.Errorf("payload: %+v", p)
	}
}
