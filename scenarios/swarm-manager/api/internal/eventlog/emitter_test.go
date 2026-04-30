package eventlog_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"

	"swarm-manager/internal/eventlog"

	_ "modernc.org/sqlite"
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

func TestEmitBacklogStatusChangedFromSource(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitBacklogStatusChangedFromSource("execute/do-thing", "ready", "completed", eventlog.BacklogMutationSourcePayload{
		Entrypoint:     "initiative.operating_mode.complete_items",
		InitiativeName: "init-a",
		Mode:           "holistic-loop",
		Phase:          "execute",
		Round:          3,
		RunID:          "run-123",
		RequestedBy:    "operator",
	}, []string{"execute/do-thing"})

	e := lastEvent(t, repo)
	if e.EventType != eventlog.EventBacklogStatusChanged {
		t.Errorf("event_type: got %q", e.EventType)
	}
	if e.ActorType != "operating_mode" || e.ActorID != "run-123" {
		t.Errorf("actor: got %q/%q", e.ActorType, e.ActorID)
	}

	var p eventlog.StatusChangePayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.From != "ready" || p.To != "completed" {
		t.Fatalf("status payload = %+v", p)
	}
	if p.Source == nil {
		t.Fatalf("source missing from payload: %+v", p)
	}
	if p.Source.Mode != "holistic-loop" || p.Source.Phase != "execute" || p.Source.Round != 3 || p.Source.RunID != "run-123" || p.Source.RequestedBy != "operator" {
		t.Fatalf("source payload = %+v", p.Source)
	}
	if !reflect.DeepEqual(p.ItemRefs, []string{"execute/do-thing"}) {
		t.Fatalf("item refs = %+v", p.ItemRefs)
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

	emitter.EmitBacklogDeleted("idea/gone")

	e := lastEvent(t, repo)
	if e.EventType != eventlog.EventBacklogDeleted {
		t.Errorf("event_type: got %q", e.EventType)
	}
	// Nil payload should result in nil metadata.
	if e.Metadata != nil {
		t.Errorf("expected nil metadata, got %s", e.Metadata)
	}
}

func TestEmitInitiativeArchived(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitInitiativeArchived("init-old", "active", "2026-04-06T00:00:00Z")

	e := lastEvent(t, repo)
	if e.EventType != eventlog.EventInitiativeArchived {
		t.Errorf("event_type: got %q", e.EventType)
	}
	var p eventlog.ArchivePayload
	if err := json.Unmarshal(e.Metadata, &p); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if p.PreviousStatus != "active" || p.ArchivedAt != "2026-04-06T00:00:00Z" {
		t.Errorf("payload: %+v", p)
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

func TestEmitBacklogProposalApplied_FeedbackRoundAttribution(t *testing.T) {
	emitter, repo := setupEmitter(t)

	payload := eventlog.ProposalAppliedPayload{
		InitiativeName:  "ui-rewrite",
		FeedbackRoundID: "ui-rewrite/round-001",
		RoundNumber:     1,
		RoundSlug:       "first-pass",
		Entrypoint:      "initiative.feedback",
		DecidedBy:       "matt",
		MutationID:      "m1",
		Op:              "add_item",
		Target:          "execute/baz",
	}
	emitter.EmitBacklogProposalApplied("execute/baz", payload)

	e := lastEvent(t, repo)
	if e.EntityType != eventlog.EntityBacklogItem {
		t.Errorf("entity_type: got %q", e.EntityType)
	}
	if e.EntityID != "execute/baz" {
		t.Errorf("entity_id: got %q", e.EntityID)
	}
	if e.EventType != eventlog.EventBacklogProposalApplied {
		t.Errorf("event_type: got %q", e.EventType)
	}
	if e.ActorType != "feedback_round" {
		t.Errorf("actor_type: got %q", e.ActorType)
	}
	if e.ActorID != "ui-rewrite/round-001" {
		t.Errorf("actor_id: got %q", e.ActorID)
	}
	var got eventlog.ProposalAppliedPayload
	if err := json.Unmarshal(e.Metadata, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !reflect.DeepEqual(got, payload) {
		t.Errorf("payload roundtrip: got %+v want %+v", got, payload)
	}
}

func TestEmitBacklogProposalApplied_ReviewRoundTakesPrecedence(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitBacklogProposalApplied("execute/foo", eventlog.ProposalAppliedPayload{
		InitiativeName:  "ui-rewrite",
		FeedbackRoundID: "ui-rewrite/round-001",
		ReviewRoundID:   "ui-rewrite/review-002",
		MutationID:      "m1",
		Op:              "change_status",
		Target:          "execute/foo",
	})
	e := lastEvent(t, repo)
	if e.ActorType != "initiative_review" {
		t.Errorf("review must dominate feedback: actor_type=%q", e.ActorType)
	}
	if e.ActorID != "ui-rewrite/review-002" {
		t.Errorf("actor_id: got %q", e.ActorID)
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
