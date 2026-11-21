package runs

import "testing"

// TestTransition_Matrix covers the legal transitions and rejects the illegal
// ones FLOWS.md names (snapshotting before capturing; any escape from a
// terminal state).
func TestTransition_Matrix(t *testing.T) {
	states := []RunStatus{RunPending, RunCapturing, RunSnapshotting, RunCompleted, RunPartialFailed, RunFailed}
	events := []RunEvent{EventStartCapture, EventStartSnapshot, EventComplete, EventPartialFail, EventFail}
	legal := map[struct {
		from  RunStatus
		event RunEvent
	}]RunStatus{
		{RunPending, EventStartCapture}:     RunCapturing,
		{RunPending, EventFail}:             RunFailed,
		{RunCapturing, EventStartSnapshot}:  RunSnapshotting,
		{RunCapturing, EventPartialFail}:    RunPartialFailed,
		{RunCapturing, EventFail}:           RunFailed,
		{RunSnapshotting, EventComplete}:    RunCompleted,
		{RunSnapshotting, EventPartialFail}: RunPartialFailed,
		{RunSnapshotting, EventFail}:        RunFailed,
	}
	for _, from := range states {
		for _, event := range events {
			key := struct {
				from  RunStatus
				event RunEvent
			}{from, event}
			want, wantOK := legal[key]
			got, gotOK := Transition(from, event)
			if gotOK != wantOK || (gotOK && got != want) || (!gotOK && got != from) {
				t.Errorf("Transition(%s,%s) = %s,%v; want %s,%v", from, event, got, gotOK, want, wantOK)
			}
		}
	}
}

func TestRunTransitionStatusRejectsWithoutMutation(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status RunStatus
		event  RunEvent
	}{
		{name: "unknown state", status: "future-state", event: EventFail},
		{name: "wrong branch", status: RunCapturing, event: EventComplete},
		{name: "terminal state", status: RunCompleted, event: EventFail},
	} {
		t.Run(tc.name, func(t *testing.T) {
			run := Run{Status: tc.status}
			if err := run.TransitionStatus(tc.event); err == nil {
				t.Fatal("expected transition to fail")
			}
			if run.Status != tc.status {
				t.Fatalf("invalid transition mutated status to %q; want %q", run.Status, tc.status)
			}
		})
	}
}

// TestClassifyTerminal pins the partial-failure rule: any success with some
// failure is partial; all-non-success is failed; clean is completed.
func TestClassifyTerminal(t *testing.T) {
	cases := []struct {
		s, f, b int
		want    RunStatus
	}{
		{2, 0, 0, RunCompleted},
		{1, 1, 0, RunPartialFailed},
		{1, 0, 1, RunPartialFailed},
		{0, 2, 0, RunFailed},
		{0, 0, 2, RunFailed},
		{0, 1, 1, RunFailed},
	}
	for _, tc := range cases {
		if got := classifyTerminal(tc.s, tc.f, tc.b); got != tc.want {
			t.Errorf("classifyTerminal(%d,%d,%d) = %s; want %s", tc.s, tc.f, tc.b, got, tc.want)
		}
	}
}

// TestCheckInvariants rejects a run whose recorded status contradicts its
// outcomes (e.g. claiming completed with a failed outcome).
func TestCheckInvariants(t *testing.T) {
	good := Run{Status: RunCompleted, Outcomes: []TargetOutcome{{Status: OutcomeSucceeded}}}
	if err := CheckInvariants(good); err != nil {
		t.Errorf("good run failed invariants: %v", err)
	}
	bad := Run{Status: RunCompleted, Outcomes: []TargetOutcome{{Status: OutcomeSucceeded}, {Status: OutcomeFailed}}}
	if err := CheckInvariants(bad); err == nil {
		t.Error("a completed run with a failed outcome must violate invariants")
	}
}

func TestCheckInvariantsAllowsPartialFailureFromWarningsOrPersistenceErrors(t *testing.T) {
	for _, run := range []Run{
		{Status: RunPartialFailed, Outcomes: []TargetOutcome{{Status: OutcomeSucceeded, Warning: "retention failed"}}},
		{Status: RunPartialFailed, Error: "cleanup_failed", Outcomes: []TargetOutcome{{Status: OutcomeSucceeded}}},
	} {
		if err := CheckInvariants(run); err != nil {
			t.Errorf("valid partial-failure run rejected: %+v: %v", run, err)
		}
	}
}
