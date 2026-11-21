package heartbeat

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

type retainedOwnerStateRecorder struct {
	captureExecutor
	facts []RetainedOwnerStateFact
}

func (r *retainedOwnerStateRecorder) TeamReviewURLFor(context.Context, string) string {
	return "https://prompt-manager.example/teams/team-a"
}

func (r *retainedOwnerStateRecorder) PublishRetainedOwnerState(_ context.Context, fact RetainedOwnerStateFact) error {
	r.facts = append(r.facts, fact)
	return nil
}

func TestRecoverPublishesRetainedLegacyOwnerStateWithPersistedIdentityAndNoInventedStart(t *testing.T) {
	dir := t.TempDir()
	recorder := &retainedOwnerStateRecorder{}
	ctx := newTeamExecutionContext("team-a", recorder, dir, nil)
	ctx.running["agent-a"] = runningEntry{State: ObligationDispatchUncertain}
	if err := ctx.persistLocked(); err != nil {
		t.Fatalf("persist legacy retained queue row: %v", err)
	}

	ctx.Recover(context.Background())
	if len(recorder.facts) != 1 {
		t.Fatalf("published %d facts, want one", len(recorder.facts))
	}
	first := recorder.facts[0]
	if first.StartedAt != "" {
		t.Fatalf("unknown historical StartedAt was fabricated as %q", first.StartedAt)
	}
	if first.OwnerObligationID == "" || first.ObservedAt == "" {
		t.Fatalf("owner fact lacks persisted identity or actual observation time: %+v", first)
	}
	if first.ReviewURL != "https://prompt-manager.example/teams/team-a" {
		t.Fatalf("unexpected existing review URL %q", first.ReviewURL)
	}

	data, err := os.ReadFile(filepath.Join(dir, "team-queue-team-a.json"))
	if err != nil {
		t.Fatal(err)
	}
	var persisted persistedTeamQueue
	if err := json.Unmarshal(data, &persisted); err != nil {
		t.Fatal(err)
	}
	if len(persisted.Running) != 1 || persisted.Running[0].OwnerObligationID != first.OwnerObligationID {
		t.Fatalf("generated owner identity was not durably stored before publication: %+v", persisted.Running)
	}
	if got := persisted.Running[0].OwnerStateSnapshots[ObligationDispatchUncertain]; got.ObservedAt != first.ObservedAt {
		t.Fatalf("first observation snapshot was not persisted: %+v", got)
	}

	// A new PM process reuses the immutable terms from the queue file.
	restarted := newTeamExecutionContext("team-a", recorder, dir, nil)
	restarted.Recover(context.Background())
	if len(recorder.facts) != 2 || !reflect.DeepEqual(recorder.facts[1], first) {
		t.Fatalf("restart changed the retained owner fact: first=%+v second=%+v", first, recorder.facts)
	}
}

func TestRetainedOwnerStateSelectorExcludesApprovalPendingAndPaused(t *testing.T) {
	for _, state := range []string{ObligationApprovalPending, ObligationPaused, ""} {
		if isRetainedOwnerReviewState(state) {
			t.Errorf("state %q must stay on its existing owner path", state)
		}
	}
	for _, state := range []string{ObligationDispatchUncertain, ObligationOwnerUnreachable, ObligationNeedsReview, ObligationWatcherUnavailable} {
		if !isRetainedOwnerReviewState(state) {
			t.Errorf("retained state %q should be eligible for one owner-state FYI", state)
		}
	}
}
