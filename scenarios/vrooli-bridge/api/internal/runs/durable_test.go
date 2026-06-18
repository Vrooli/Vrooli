package runs_test

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/runs"
	"vrooli-bridge/internal/runs/mocks"
	tmocks "vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

func newService(t *testing.T) (runs.Service, *mocks.FakeRepository) {
	t.Helper()
	clk := tmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	repo := mocks.NewFakeRepository()
	return runs.NewService(repo, clk), repo
}

func mustCreate(t *testing.T, svc runs.Service) runs.Run {
	t.Helper()
	run, err := svc.Create(context.Background(), runs.CreateInput{NodeID: "n1", Scenario: "web-search", Verb: "scenario test", Args: []string{"web-search"}})
	require.NoError(t, err)
	require.Equal(t, runs.StatusQueued, run.Status)
	return run
}

// [REQ:BRG-P0-005] Wait blocks once and returns exactly when the run reaches a
// terminal status, carrying the real exit code — no polling.
func TestWait_BlockOnceReturnsTerminal(t *testing.T) {
	svc, _ := newService(t)
	run := mustCreate(t, svc)

	go func() {
		time.Sleep(20 * time.Millisecond)
		_, _ = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventStatus, Sequence: 1, Status: "running"})
		_, _ = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventExit, Sequence: 2, ExitCode: 0})
	}()

	got, timedOut, err := svc.Wait(context.Background(), run.ID, 5*time.Second)
	require.NoError(t, err)
	require.False(t, timedOut)
	require.Equal(t, runs.StatusPassed, got.Status)
	require.Equal(t, int32(0), got.ExitCode)
}

// [REQ:BRG-P0-005] A non-zero exit yields FAILED with the code preserved.
func TestWait_FailedExit(t *testing.T) {
	svc, _ := newService(t)
	run := mustCreate(t, svc)
	_, _ = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventExit, Sequence: 1, ExitCode: 2})

	got, timedOut, err := svc.Wait(context.Background(), run.ID, time.Second)
	require.NoError(t, err)
	require.False(t, timedOut)
	require.Equal(t, runs.StatusFailed, got.Status)
	require.Equal(t, int32(2), got.ExitCode)
}

// [REQ:BRG-P0-005] Waiting on an already-terminal run returns immediately.
func TestWait_AlreadyTerminal(t *testing.T) {
	svc, _ := newService(t)
	run := mustCreate(t, svc)
	_, _ = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventExit, Sequence: 1, ExitCode: 0})

	start := time.Now()
	got, timedOut, err := svc.Wait(context.Background(), run.ID, time.Minute)
	require.NoError(t, err)
	require.False(t, timedOut)
	require.Equal(t, runs.StatusPassed, got.Status)
	require.Less(t, time.Since(start), 500*time.Millisecond, "should not block on a terminal run")
}

// [REQ:BRG-P0-005] A wait that elapses before the run finishes returns
// timed_out=true with the latest non-terminal snapshot (CLI → exit 124).
func TestWait_TimeoutReturnsTimedOut(t *testing.T) {
	svc, _ := newService(t)
	run := mustCreate(t, svc)

	got, timedOut, err := svc.Wait(context.Background(), run.ID, 30*time.Millisecond)
	require.NoError(t, err)
	require.True(t, timedOut)
	require.Equal(t, runs.StatusQueued, got.Status)
}

// [REQ:BRG-P0-005] The run is server-owned: it continues after the dispatching
// client disconnects and is re-attachable by id (Get) showing the terminal
// verdict + the full event history.
func TestRun_SurvivesClientDisconnectAndReattach(t *testing.T) {
	svc, _ := newService(t)
	run := mustCreate(t, svc)

	// The "client" is gone; the node keeps reporting against the run id.
	_, _ = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventStatus, Sequence: 1, Status: "running"})
	_, _ = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventLog, Sequence: 2, LogChunk: "PASS web-search\n"})
	_, _ = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventExit, Sequence: 3, ExitCode: 0})

	// A fresh client re-attaches by id and reads the terminal verdict + logs.
	got, events, err := svc.Get(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, runs.StatusPassed, got.Status)
	require.Len(t, events, 3)
	require.Equal(t, "PASS web-search\n", events[1].LogChunk)
}

// [REQ:BRG-P0-005] Stale-completion safety: a late event for an
// already-terminal run is acknowledged but does not change the verdict.
func TestAppendEvent_StaleCompletionIgnored(t *testing.T) {
	svc, _ := newService(t)
	run := mustCreate(t, svc)
	_, _ = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventExit, Sequence: 1, ExitCode: 0})

	accepted, err := svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventLog, Sequence: 2, LogChunk: "late\n"})
	require.NoError(t, err)
	require.False(t, accepted, "a late event for a terminal run has no further effect")

	got, _, _ := svc.Get(context.Background(), run.ID)
	require.Equal(t, runs.StatusPassed, got.Status, "verdict unchanged")
}

// [REQ:BRG-P0-005] An event for an unknown run is acknowledged without error so
// a confused node stops re-sending (no spin).
func TestAppendEvent_UnknownRun(t *testing.T) {
	svc, _ := newService(t)
	accepted, err := svc.AppendEvent(context.Background(), runs.RunEvent{RunID: "ghost", Kind: runs.EventLog, Sequence: 1})
	require.NoError(t, err)
	require.False(t, accepted)
}

// [REQ:BRG-P0-005] The first running STATUS transitions QUEUED→RUNNING and
// stamps started_at.
func TestAppendEvent_RunningTransition(t *testing.T) {
	svc, _ := newService(t)
	run := mustCreate(t, svc)
	accepted, err := svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventStatus, Sequence: 1, Status: "running"})
	require.NoError(t, err)
	require.True(t, accepted)

	got, _, _ := svc.Get(context.Background(), run.ID)
	require.Equal(t, runs.StatusRunning, got.Status)
	require.False(t, got.StartedAt.IsZero())
}

// [REQ:BRG-P0-005] Abort marks a non-terminal run ABORTED, wakes a blocked
// waiter, and is idempotent on an already-terminal run.
func TestAbort_WakesWaiterAndIsIdempotent(t *testing.T) {
	svc, _ := newService(t)
	run := mustCreate(t, svc)

	done := make(chan runs.Run, 1)
	go func() {
		got, _, _ := svc.Wait(context.Background(), run.ID, 5*time.Second)
		done <- got
	}()
	time.Sleep(20 * time.Millisecond)

	aborted, err := svc.Abort(context.Background(), run.ID, "superseded")
	require.NoError(t, err)
	require.Equal(t, runs.StatusAborted, aborted.Status)

	select {
	case got := <-done:
		require.Equal(t, runs.StatusAborted, got.Status, "the waiter woke on the abort")
	case <-time.After(2 * time.Second):
		t.Fatal("waiter did not wake on abort")
	}

	// Idempotent: aborting again returns the terminal run unchanged.
	again, err := svc.Abort(context.Background(), run.ID, "again")
	require.NoError(t, err)
	require.Equal(t, runs.StatusAborted, again.Status)
}
