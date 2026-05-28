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

func (f *fakeEvents) EmitRecordSuperseded(id, superseded, reason string) {
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
