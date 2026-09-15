package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestRunStatusParked_Classification pins the parked status's place in the
// status taxonomy: valid, non-terminal, non-active. Getting any of these wrong
// would either let the reconciler reap it (active/terminal confusion) or make
// it un-persistable (invalid).
func TestRunStatusParked_Classification(t *testing.T) {
	if !RunStatusParked.IsValid() {
		t.Error("parked must be a valid RunStatus")
	}
	if RunStatusParked.IsTerminal() {
		t.Error("parked must NOT be terminal (it resumes via wake or cancels)")
	}
	if RunStatusParked.IsActive() {
		t.Error("parked must NOT be active (the process has exited, zero tokens)")
	}
}

// TestRunStatusParked_Transitions pins the parked edges of the run state
// machine: running→parked (park), and parked→{running,failed,cancelled}
// (wake / waiter-error / abort). Everything else out of parked is rejected.
func TestRunStatusParked_Transitions(t *testing.T) {
	allowed := []struct {
		from, to RunStatus
	}{
		{RunStatusRunning, RunStatusParked},
		{RunStatusParked, RunStatusRunning},
		{RunStatusParked, RunStatusFailed},
		{RunStatusParked, RunStatusCancelled},
	}
	for _, tc := range allowed {
		if ok, reason := tc.from.CanTransitionTo(tc.to); !ok {
			t.Errorf("expected %s→%s allowed, got denied: %s", tc.from, tc.to, reason)
		}
	}

	denied := []struct {
		from, to RunStatus
	}{
		{RunStatusParked, RunStatusComplete},    // must wake to running first
		{RunStatusParked, RunStatusNeedsReview}, // not a review path
		{RunStatusStarting, RunStatusParked},    // only a running run parks
		{RunStatusPending, RunStatusParked},
		{RunStatusComplete, RunStatusParked},
	}
	for _, tc := range denied {
		if ok, _ := tc.from.CanTransitionTo(tc.to); ok {
			t.Errorf("expected %s→%s denied, got allowed", tc.from, tc.to)
		}
	}
}

// TestRunStatusParked_LivenessPolicy pins the whole point of Phase 1's split
// fields: parked is SCANNED (so restart recovery / TTL can see it) but expects
// NO heartbeat and NO process, so it is never heartbeat-reaped or orphan-killed.
func TestRunStatusParked_LivenessPolicy(t *testing.T) {
	p := RunStatusParked.LivenessPolicy()
	if !p.Scanned {
		t.Error("parked must be Scanned (restart recovery + optional TTL need to see it)")
	}
	if p.ExpectsHeartbeat {
		t.Error("parked must NOT expect a heartbeat (no live process)")
	}
	if p.ExpectsProcess {
		t.Error("parked must NOT expect a process (it would protect a phantom orphan)")
	}
	if p.StaleAction != StaleRunActionNone {
		t.Errorf("parked StaleAction = %s, want none", p.StaleAction)
	}

	// And it must be in the scanned set.
	found := false
	for _, s := range LivenessScannedStatuses() {
		if s == RunStatusParked {
			found = true
		}
	}
	if !found {
		t.Error("parked must appear in LivenessScannedStatuses()")
	}
}

// TestCanParkRun pins the park precondition: only a live, running run with a
// session ID can be parked, and never twice.
func TestCanParkRun(t *testing.T) {
	sess := "sess-123"
	cases := []struct {
		name string
		run  *Run
		want bool
	}{
		{"running with session", &Run{Status: RunStatusRunning, SessionID: sess}, true},
		{"already parked", &Run{Status: RunStatusParked, SessionID: sess}, false},
		{"running without session", &Run{Status: RunStatusRunning}, false},
		{"starting", &Run{Status: RunStatusStarting, SessionID: sess}, false},
		{"complete", &Run{Status: RunStatusComplete, SessionID: sess}, false},
		{"nil", nil, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got, _ := CanParkRun(tc.run); got != tc.want {
				t.Errorf("CanParkRun = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestCanWakeRun pins that only a parked run is wakeable.
func TestCanWakeRun(t *testing.T) {
	if ok, _ := CanWakeRun(&Run{Status: RunStatusParked}); !ok {
		t.Error("a parked run must be wakeable")
	}
	if ok, _ := CanWakeRun(&Run{Status: RunStatusRunning}); ok {
		t.Error("a running run must not be wakeable")
	}
	if ok, _ := CanWakeRun(nil); ok {
		t.Error("nil run must not be wakeable")
	}
}

// TestCanStopRun_Parked pins that a parked run is stoppable (so abort can move
// it to a terminal state instead of leaving it suspended forever) while the
// non-stoppable states stay non-stoppable.
func TestCanStopRun_Parked(t *testing.T) {
	if ok, _ := CanStopRun(&Run{Status: RunStatusParked}); !ok {
		t.Error("a parked run must be stoppable")
	}
	if ok, _ := CanStopRun(&Run{Status: RunStatusComplete}); ok {
		t.Error("a complete run must not be stoppable")
	}
}

// TestCanContinueRun_ParkedRejected pins that a parked run is NOT operator-
// continuable: it resumes via wake (owned by its waiter), so a manual continue
// would race the waiter.
func TestCanContinueRun_ParkedRejected(t *testing.T) {
	run := &Run{Status: RunStatusParked, SessionID: "sess-1"}
	if ok, _ := CanContinueRun(run); ok {
		t.Error("a parked run must not be operator-continuable (wake owns the resume)")
	}
}

// TestAwaitHandle_RoundTripShape is a lightweight guard that the AwaitHandle
// fields the persistence + waiter layers depend on are present and settable.
func TestAwaitHandle_RoundTripShape(t *testing.T) {
	deadline := time.Now().Add(time.Hour)
	h := &AwaitHandle{
		Producer:     "test-genie",
		Key:          uuid.NewString(),
		Deadline:     &deadline,
		RegisteredAt: time.Now(),
	}
	if h.Producer == "" || h.Key == "" || h.Deadline == nil {
		t.Fatal("await handle fields must be settable")
	}
}
