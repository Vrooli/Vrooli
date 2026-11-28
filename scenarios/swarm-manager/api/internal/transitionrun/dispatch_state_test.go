package transitionrun

import (
	"errors"
	"testing"
)

func TestDispatchIntentAdvanceTransitionTable(t *testing.T) {
	valid := []struct {
		from, to DispatchState
	}{
		{DispatchSubmitting, DispatchUnknown},
		{DispatchSubmitting, DispatchAcknowledged},
		{DispatchUnknown, DispatchUnknown},
		{DispatchUnknown, DispatchAcknowledged},
	}
	for _, edge := range valid {
		t.Run(string(edge.from)+"_to_"+string(edge.to), func(t *testing.T) {
			intent := DispatchIntent{State: edge.from, LastError: "previous"}
			if err := intent.Advance(edge.to, "next"); err != nil {
				t.Fatalf("valid edge rejected: %v", err)
			}
			if intent.State != edge.to || intent.LastError != "next" {
				t.Fatalf("advanced intent = %+v", intent)
			}
		})
	}
	invalid := []struct {
		from, to DispatchState
	}{
		{"", DispatchSubmitting},
		{DispatchUnknown, DispatchSubmitting},
		{DispatchAcknowledged, DispatchUnknown},
		{DispatchAcknowledged, DispatchSubmitting},
		{DispatchAcknowledged, DispatchAcknowledged},
		{DispatchSubmitting, "corrupt"},
		{"corrupt", DispatchUnknown},
	}
	for _, edge := range invalid {
		t.Run(string(edge.from)+"_rejects_"+string(edge.to), func(t *testing.T) {
			intent := DispatchIntent{State: edge.from, LastError: "preserve"}
			if err := intent.Advance(edge.to, "changed"); !errors.Is(err, ErrInvalidDispatchTransition) {
				t.Fatalf("invalid edge error = %v", err)
			}
			if intent.State != edge.from || intent.LastError != "preserve" {
				t.Fatalf("rejected edge mutated intent: %+v", intent)
			}
		})
	}
}

func TestFileStoreRejectsDispatchStateBypass(t *testing.T) {
	store := NewFileStore(t.TempDir())
	intent := DispatchIntent{
		State:       DispatchSubmitting,
		Correlation: Correlation{TransitionKey: "plan.execute", SubjectRef: "subject"},
		Owner:       "swarm-manager", IdempotencyKey: "dispatch-key", InputDigest: "sha256:input", CreatedAt: "2026-11-14T00:00:00Z",
	}
	if err := store.PutDispatch(intent); err != nil {
		t.Fatal(err)
	}
	if err := intent.Advance(DispatchUnknown, "owner lookup unavailable"); err != nil {
		t.Fatal(err)
	}
	intent.State = DispatchSubmitting // bypasses Advance and recreates a new submission
	if err := store.PutDispatch(intent); !errors.Is(err, ErrInvalidDispatchTransition) {
		t.Fatalf("store accepted bypassed edge: %v", err)
	}
	stored, err := store.GetDispatch("plan.execute", "subject")
	if err != nil || stored.State != DispatchSubmitting {
		t.Fatalf("rejected write changed durable state: %+v %v", stored, err)
	}
}

func TestFileStoreRejectsUnknownDispatchStateAndChangedIdentity(t *testing.T) {
	store := NewFileStore(t.TempDir())
	intent := DispatchIntent{
		State:       DispatchSubmitting,
		Correlation: Correlation{TransitionKey: "plan.execute", SubjectRef: "subject"},
		Owner:       "swarm-manager", IdempotencyKey: "dispatch-key", InputDigest: "sha256:input", CreatedAt: "2026-11-14T00:00:00Z",
	}
	unknown := intent
	unknown.State = "invented"
	if err := store.PutDispatch(unknown); !errors.Is(err, ErrInvalidDispatchTransition) {
		t.Fatalf("store accepted unknown initial state: %v", err)
	}
	if err := store.PutDispatch(intent); err != nil {
		t.Fatal(err)
	}
	next := intent
	if err := next.Advance(DispatchUnknown, "owner lookup unavailable"); err != nil {
		t.Fatal(err)
	}
	next.Owner = "different-owner"
	if err := store.PutDispatch(next); err == nil {
		t.Fatal("store accepted changed immutable owner")
	}
	stored, err := store.GetDispatch("plan.execute", "subject")
	if err != nil || stored.State != DispatchSubmitting {
		t.Fatalf("rejected write changed durable state: %+v %v", stored, err)
	}
}

func TestFileStoreAcknowledgementAddsOriginalOwnerExecutionAndAttempt(t *testing.T) {
	store := NewFileStore(t.TempDir())
	intent := DispatchIntent{
		State:       DispatchSubmitting,
		Correlation: Correlation{TransitionKey: "plan.execute", SubjectKind: "backlog-item", SubjectRef: "subject", WorkflowKey: "workflow", EntityVersion: "v1", FrontierDigest: "frontier", ApplyState: ApplyStateClaimed, DeclaredOutcomes: []string{"complete"}},
		Owner:       "swarm-manager", IdempotencyKey: "dispatch-key", InputDigest: "sha256:input", CreatedAt: "2026-11-14T00:00:00Z",
	}
	if err := store.PutDispatch(intent); err != nil {
		t.Fatal(err)
	}
	intent.Correlation.ExecutionID = "owner-execution"
	intent.Correlation.DefinitionDigest = "sha256:definition"
	intent.Correlation.Attempts = []Attempt{{RunID: "owner-run"}}
	if err := intent.Advance(DispatchAcknowledged, ""); err != nil {
		t.Fatal(err)
	}
	if err := store.PutDispatch(intent); err != nil {
		t.Fatalf("owner acknowledgement could not bind returned execution: %v", err)
	}
}
