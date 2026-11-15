package runs

import "testing"

func TestRunStatusTransitionGraph(t *testing.T) {
	valid := []struct {
		from RunStatus
		to   RunStatus
	}{
		{StatusQueued, StatusPushed},
		{StatusPushed, StatusAcked},
		{StatusAcked, StatusRunning},
		{StatusRunning, StatusPassed},
		{StatusRunning, StatusCancelRequested},
		{StatusCancelRequested, StatusUncertain},
		{StatusUncertain, StatusAborted},
	}
	for _, edge := range valid {
		t.Run(edge.from.String()+"->"+edge.to.String(), func(t *testing.T) {
			run := Run{Status: edge.from}
			if err := run.TransitionStatus(edge.to); err != nil {
				t.Fatalf("expected legal transition: %v", err)
			}
			if run.Status != edge.to {
				t.Fatalf("status = %s, want %s", run.Status, edge.to)
			}
		})
	}
}

func TestRunStatusTransitionRejectsUnknownAndTerminalReopen(t *testing.T) {
	cases := []struct {
		name string
		from RunStatus
		to   RunStatus
	}{
		{name: "unknown current", from: RunStatus(99), to: StatusQueued},
		{name: "unknown target", from: StatusQueued, to: RunStatus(99)},
		{name: "terminal reopen", from: StatusPassed, to: StatusRunning},
		{name: "skipped delivery edge", from: StatusQueued, to: StatusUncertain},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			run := Run{Status: tc.from}
			if err := run.TransitionStatus(tc.to); err == nil {
				t.Fatal("expected invalid transition error")
			}
			if run.Status != tc.from {
				t.Fatalf("invalid transition mutated run to %s", run.Status)
			}
		})
	}
}
