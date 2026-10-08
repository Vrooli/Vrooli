package runs

import "testing"

// TestTransition_Matrix covers the legal transitions and rejects the illegal
// ones FLOWS.md names (snapshotting before capturing; any escape from a
// terminal state).
func TestTransition_Matrix(t *testing.T) {
	legal := []struct {
		from RunStatus
		ev   RunEvent
		want RunStatus
	}{
		{RunPending, EventStartCapture, RunCapturing},
		{RunCapturing, EventStartSnapshot, RunSnapshotting},
		{RunSnapshotting, EventComplete, RunCompleted},
		{RunCapturing, EventPartialFail, RunPartialFailed},
		{RunSnapshotting, EventPartialFail, RunPartialFailed},
		{RunCapturing, EventFail, RunFailed},
	}
	for _, tc := range legal {
		got, ok := Transition(tc.from, tc.ev)
		if !ok || got != tc.want {
			t.Errorf("Transition(%s,%s) = %s,%v; want %s,true", tc.from, tc.ev, got, ok, tc.want)
		}
	}

	illegal := []struct {
		from RunStatus
		ev   RunEvent
	}{
		{RunPending, EventStartSnapshot},  // snapshot before capture
		{RunPending, EventComplete},       // complete before any work
		{RunCompleted, EventStartCapture}, // escape terminal
		{RunFailed, EventComplete},        // escape terminal
		{RunSnapshotting, EventStartCapture},
	}
	for _, tc := range illegal {
		if _, ok := Transition(tc.from, tc.ev); ok {
			t.Errorf("Transition(%s,%s) should be illegal", tc.from, tc.ev)
		}
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
