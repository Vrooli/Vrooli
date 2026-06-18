package runs_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"vrooli-bridge/internal/runs"
	"vrooli-bridge/internal/runs/mocks"
	tmocks "vrooli-bridge/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
)

type fakeCanceller struct {
	mu       sync.Mutex
	cancels  []string
	lastNode string
}

func (f *fakeCanceller) CancelJob(_ context.Context, nodeID, runID, _ string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancels = append(f.cancels, runID)
	f.lastNode = nodeID
	return nil
}

// [REQ:BRG-P1-004] AbortRun pushes a node-cancel (AbortJob) so the node STOPS
// the in-flight run rather than running it to completion as an ignored stale
// completion, and fires the terminal hook so the run's queue slot frees.
func TestAbort_PushesNodeCancelAndFiresTerminalHook(t *testing.T) {
	clk := tmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	repo := mocks.NewFakeRepository()
	canceller := &fakeCanceller{}

	var hookRuns []string
	svc := runs.NewService(repo, clk,
		runs.WithCanceller(canceller),
		runs.WithTerminalHook(func(_ context.Context, run runs.Run) { hookRuns = append(hookRuns, run.ID) }),
	)

	run, err := svc.Create(context.Background(), runs.CreateInput{NodeID: "n1", Verb: "scenario test"})
	require.NoError(t, err)

	aborted, err := svc.Abort(context.Background(), run.ID, "operator abort")
	require.NoError(t, err)
	require.Equal(t, runs.StatusAborted, aborted.Status)

	require.Equal(t, []string{run.ID}, canceller.cancels, "the node is told to stop the run")
	require.Equal(t, "n1", canceller.lastNode)
	require.Equal(t, []string{run.ID}, hookRuns, "the terminal hook fires to free the queue slot")
}

// [REQ:BRG-P1-004] A natural completion (node EXIT) fires the terminal hook but
// does NOT push a node-cancel (the run already exited).
func TestAppendEvent_TerminalFiresHookWithoutCancel(t *testing.T) {
	clk := tmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	repo := mocks.NewFakeRepository()
	canceller := &fakeCanceller{}

	var hookRuns []string
	svc := runs.NewService(repo, clk,
		runs.WithCanceller(canceller),
		runs.WithTerminalHook(func(_ context.Context, run runs.Run) { hookRuns = append(hookRuns, run.ID) }),
	)

	run, err := svc.Create(context.Background(), runs.CreateInput{NodeID: "n1", Verb: "scenario test"})
	require.NoError(t, err)
	_, err = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventExit, Sequence: 1, ExitCode: 0})
	require.NoError(t, err)

	require.Equal(t, []string{run.ID}, hookRuns, "the terminal hook fires on natural completion")
	require.Empty(t, canceller.cancels, "no node-cancel push for a run that already exited")
}
