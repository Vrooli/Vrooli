package records

import (
	"context"
	"testing"
)

func newTestService(t *testing.T) (*Service, *FileStore, *fakeEvents) {
	t.Helper()
	store := newTestStore(t)
	events := &fakeEvents{}
	svc := NewService(store, nil, events)
	return svc, store, events
}

type fakeEvents struct {
	created    []string
	superseded [][2]string
}

func (f *fakeEvents) EmitRecordCreated(id, kind, scenario, backlogRef string, stub bool) {
	tag := id
	if stub {
		tag += "(stub)"
	}
	f.created = append(f.created, tag)
}

func (f *fakeEvents) EmitRecordSuperseded(_ context.Context, id, superseded, reason string) {
	f.superseded = append(f.superseded, [2]string{id, superseded})
}

func TestServiceCreateHappy(t *testing.T) {
	svc, _, events := newTestService(t)
	r, err := svc.Create(context.Background(), CreateInput{
		Kind: KindFix, Scenario: "x",
		Trigger: "t", Approach: "a",
		Outcome: OutcomeShipped,
	})
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if r.Stub {
		t.Errorf("non-stub create should set Stub=false")
	}
	if r.NarrativeAt.IsZero() {
		t.Errorf("NarrativeAt should equal CreatedAt for full record")
	}
	if len(events.created) != 1 {
		t.Errorf("expected one created event, got %d", len(events.created))
	}
}

func TestServiceCreateRequiresNarrative(t *testing.T) {
	svc, _, _ := newTestService(t)
	_, err := svc.Create(context.Background(), CreateInput{
		Kind: KindFix, Scenario: "x", Outcome: OutcomeShipped,
	})
	if err == nil {
		t.Errorf("Create with empty narrative should fail")
	}
}

func TestServiceCreateStub(t *testing.T) {
	svc, _, events := newTestService(t)
	r, err := svc.CreateStub(context.Background(), CreateStubInput{
		Kind: KindFix, Scenario: "x", Outcome: OutcomeShipped,
		BacklogRef: "fix/foo",
	})
	if err != nil {
		t.Fatalf("CreateStub: %v", err)
	}
	if !r.Stub {
		t.Errorf("CreateStub should mark Stub=true")
	}
	if len(events.created) != 1 {
		t.Errorf("expected one created event")
	}
}

func TestServiceUpdateNarrativeFillsStub(t *testing.T) {
	svc, _, _ := newTestService(t)
	stub, _ := svc.CreateStub(context.Background(), CreateStubInput{
		Kind: KindFix, Scenario: "x", Outcome: OutcomeShipped,
	})
	filled, err := svc.UpdateNarrative(context.Background(), stub.ID, Narrative{
		Trigger: "trig", Approach: "app", Outcome: OutcomeShipped,
	})
	if err != nil {
		t.Fatalf("UpdateNarrative: %v", err)
	}
	if filled.Stub {
		t.Errorf("stub should flip to false")
	}
}

func TestServiceSupersedeRejectsCycle(t *testing.T) {
	svc, _, _ := newTestService(t)
	a, _ := svc.Create(context.Background(), CreateInput{
		Kind: KindFix, Scenario: "x", Trigger: "t", Outcome: OutcomeShipped,
	})
	// Create B that supersedes A.
	b, err := svc.Create(context.Background(), CreateInput{
		Kind: KindFix, Scenario: "x", Trigger: "t2", Outcome: OutcomeShipped,
		Supersedes: a.ID,
	})
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}
	// Now ask: supersede B with A — would create A->B->A cycle.
	if _, err := svc.Supersede(context.Background(), b.ID, a.ID, ""); err != ErrSupersedeCycle {
		t.Errorf("expected ErrSupersedeCycle, got %v", err)
	}
}

func TestServiceCreateLinksSupersedes(t *testing.T) {
	svc, _, events := newTestService(t)
	a, _ := svc.Create(context.Background(), CreateInput{
		Kind: KindFix, Scenario: "x", Trigger: "t1", Outcome: OutcomeShipped,
	})
	b, err := svc.Create(context.Background(), CreateInput{
		Kind: KindFix, Scenario: "x", Trigger: "t2", Outcome: OutcomeShipped,
		Supersedes: a.ID,
	})
	if err != nil {
		t.Fatalf("Create B: %v", err)
	}
	got, _ := svc.Get(a.ID)
	if got.SupersededBy != b.ID {
		t.Errorf("A.SupersededBy = %q, want %q", got.SupersededBy, b.ID)
	}
	if len(events.superseded) != 1 {
		t.Errorf("expected one superseded event, got %d", len(events.superseded))
	}
}

func TestCaptureDraftIsPrivateAndRepairPublishesSameID(t *testing.T) {
	svc, _, events := newTestService(t)
	draft, err := svc.Capture(context.Background(), CaptureInput{Kind: "feature", Scenario: "x", Trigger: "goal"})
	if err != nil || draft.Disposition != "draft" || !draft.Record.Draft {
		t.Fatalf("Capture draft = %+v, %v", draft, err)
	}
	if got, _ := svc.List(ListFilter{}); len(got) != 0 {
		t.Fatalf("draft leaked into list: %+v", got)
	}
	if len(events.created) != 0 {
		t.Fatalf("draft emitted publication event: %v", events.created)
	}
	published, err := svc.RepairCapture(context.Background(), draft.Record.ID, CaptureInput{Outcome: "done"})
	if err != nil || published.Disposition != "published" {
		t.Fatalf("RepairCapture = %+v, %v", published, err)
	}
	if published.Record.ID != draft.Record.ID || published.Record.Kind != KindExecute || published.Record.Outcome != OutcomeShipped {
		t.Fatalf("repair did not preserve ID/canonical enums: %+v", published.Record)
	}
	if got, _ := svc.List(ListFilter{}); len(got) != 1 || got[0].Draft {
		t.Fatalf("published record unavailable: %+v", got)
	}
	if len(events.created) != 1 {
		t.Fatalf("want one publication event, got %v", events.created)
	}
}

func TestCaptureRetryReturnsExistingPublishedRecord(t *testing.T) {
	svc, _, events := newTestService(t)
	in := CaptureInput{Kind: "fix", Scenario: "x", Trigger: "goal", Outcome: "shipped", IdempotencyKey: "capture-1"}
	first, err := svc.Capture(context.Background(), in)
	if err != nil || first.Disposition != "published" {
		t.Fatalf("first Capture = %+v, %v", first, err)
	}
	second, err := svc.Capture(context.Background(), in)
	if err != nil || second.Record.ID != first.Record.ID || second.Disposition != "published" {
		t.Fatalf("retry Capture = %+v, %v", second, err)
	}
	if got, _ := svc.List(ListFilter{}); len(got) != 1 {
		t.Fatalf("retry created %d records, want one", len(got))
	}
	if len(events.created) != 1 {
		t.Fatalf("retry emitted %d publication events, want one", len(events.created))
	}
}
