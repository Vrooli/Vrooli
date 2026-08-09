package eventlog_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"reflect"
	"testing"

	"swarm-manager/internal/eventlog"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/provenance"
	_ "modernc.org/sqlite"
)

func setupEmitter(t *testing.T) (*eventlog.Emitter, *eventlog.SQLiteRepository) {
	t.Helper()
	sqldb, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	t.Cleanup(func() { sqldb.Close() })

	repo := eventlog.NewSQLiteRepository(database.NewFromPrimary(sqldb))
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
	if p.Kind != "execute" || p.Priority != 5 || p.Milestone != "init-a" || p.Effort != "M" {
		t.Errorf("payload: %+v", p)
	}
}

func TestEmitBacklogCreatedFromContextPersistsVerifiedRunAttribution(t *testing.T) {
	emitter, repo := setupEmitter(t)
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Actor: provenance.ActorAgent, VerificationStatus: provenance.VerificationVerified, RunID: "run-event", ProfileKey: "team/member"})
	emitter.EmitBacklogCreatedFromContext(ctx, "execute/attributed", "execute", "backlog", 5, "", "", "user", "")

	e := lastEvent(t, repo)
	if e.ActorType != provenance.ActorAgent || e.ActorID != "team/member" || e.RunID != "run-event" || e.VerificationStatus != provenance.VerificationVerified {
		t.Fatalf("verified event attribution = %q/%q/%q/%q", e.ActorType, e.ActorID, e.RunID, e.VerificationStatus)
	}
}

func TestEmitBacklogCreatedFromContextPersistsHarnessObservationWithoutRun(t *testing.T) {
	emitter, repo := setupEmitter(t)
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{Invocation: provenance.Invocation{HarnessSessionID: "codex-thread-1", HarnessKind: "codex"}})
	emitter.EmitBacklogCreatedFromContext(ctx, "execute/observed", "execute", "backlog", 5, "", "", "user", "")

	e := lastEvent(t, repo)
	if e.HarnessSessionID != "codex-thread-1" || e.HarnessKind != "codex" || e.VerificationStatus != provenance.VerificationAbsent || e.ActorID != "" {
		t.Fatalf("harness observation = %+v", e)
	}
}

func TestEmitBacklogStatusChanged(t *testing.T) {
	emitter, repo := setupEmitter(t)

	emitter.EmitBacklogStatusChanged(context.Background(), "fix/bug-1", "backlog", "in_progress")

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

	emitter.EmitBacklogStatusChangedFromSource(context.Background(), "execute/do-thing", "ready", "completed", eventlog.BacklogMutationSourcePayload{
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
	if e.ActorType != "milestone_review" {
		t.Errorf("review must dominate feedback: actor_type=%q", e.ActorType)
	}
	if e.ActorID != "ui-rewrite/review-002" {
		t.Errorf("actor_id: got %q", e.ActorID)
	}
}

// TestDurabilityEventSeamsCarryProvenance is the guard for the durability read
// lane. The three event types swarm-manager's durability handler reads must all
// persist verified attribution; if any of them regresses to the context-free
// emit helper the whole lane silently reports verification_status=absent and
// every verdict becomes unlinked. Add a case here whenever the durability
// handler starts reading a new event type.
func TestDurabilityEventSeamsCarryProvenance(t *testing.T) {
	ctx := provenance.NewContext(context.Background(), provenance.Provenance{
		Actor:              provenance.ActorAgent,
		VerificationStatus: provenance.VerificationVerified,
		RunID:              "run-durability",
		ProfileKey:         "team/member",
	})

	cases := []struct {
		name  string
		emit  func(*eventlog.Emitter)
		event eventlog.EventType
	}{
		{
			name:  "backlog status changed",
			emit:  func(e *eventlog.Emitter) { e.EmitBacklogStatusChanged(ctx, "fix/bug-1", "failed", "completed") },
			event: eventlog.EventBacklogStatusChanged,
		},
		{
			name:  "record superseded",
			emit:  func(e *eventlog.Emitter) { e.EmitRecordSuperseded(ctx, "rec-new", "rec-old", "rework") },
			event: eventlog.EventRecordSuperseded,
		},
		{
			name:  "review failed",
			emit:  func(e *eventlog.Emitter) { e.EmitReviewFailed(ctx, "exec-1", "insufficient evidence", 12) },
			event: eventlog.EventReviewFailed,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			emitter, repo := setupEmitter(t)
			tc.emit(emitter)

			e := lastEvent(t, repo)
			if e.EventType != tc.event {
				t.Fatalf("event_type = %q, want %q", e.EventType, tc.event)
			}
			if e.VerificationStatus != provenance.VerificationVerified {
				t.Errorf("verification_status = %q, want %q", e.VerificationStatus, provenance.VerificationVerified)
			}
			if e.ActorType != provenance.ActorAgent || e.ActorID != "team/member" || e.RunID != "run-durability" {
				t.Errorf("actor = %q/%q run=%q, want %q/%q run=%q", e.ActorType, e.ActorID, e.RunID, provenance.ActorAgent, "team/member", "run-durability")
			}
		})
	}
}
