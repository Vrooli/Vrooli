package runs_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"vrooli-bridge/internal/runs"
	"vrooli-bridge/internal/runs/mocks"

	"github.com/vrooli/api-core/scheduletest"

	"github.com/stretchr/testify/require"
)

type fakeCanceller struct {
	mu        sync.Mutex
	cancels   []string
	lastNode  string
	delivered int
	err       error
}

func (f *fakeCanceller) CancelJob(_ context.Context, nodeID, runID, _ string) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.cancels = append(f.cancels, runID)
	f.lastNode = nodeID
	return f.delivered, f.err
}

// [REQ:BRG-P1-004] AbortRun pushes a node-cancel (AbortJob), records
// CANCEL_REQUESTED, and waits for a node EXIT before claiming ABORTED.
func TestAbort_PushesNodeCancelUntilExitConfirmsTermination(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	repo := mocks.NewFakeRepository()
	canceller := &fakeCanceller{delivered: 1}

	var hookRuns []string
	svc := runs.NewService(repo, clk,
		runs.WithCanceller(canceller),
		runs.WithTerminalHook(func(_ context.Context, run runs.Run) { hookRuns = append(hookRuns, run.ID) }),
	)

	run, err := svc.Create(context.Background(), runs.CreateInput{NodeID: "n1", Verb: "scenario test"})
	require.NoError(t, err)
	_, err = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventStatus, Sequence: 1, Status: "running"})
	require.NoError(t, err)

	requested, err := svc.Abort(context.Background(), run.ID, "operator abort")
	require.NoError(t, err)
	require.Equal(t, runs.StatusCancelRequested, requested.Status)
	require.False(t, requested.CancellationConfirmed)
	require.Empty(t, hookRuns, "the queue slot remains occupied until EXIT evidence")

	require.Equal(t, []string{run.ID}, canceller.cancels, "the node is told to stop the run")
	require.Equal(t, "n1", canceller.lastNode)

	_, err = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventExit, Sequence: 3, ExitCode: 143})
	require.NoError(t, err)
	aborted, _, err := svc.Get(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, runs.StatusAborted, aborted.Status)
	require.True(t, aborted.CancellationConfirmed)
	require.Equal(t, []string{run.ID}, hookRuns, "the terminal hook fires to free the queue slot")
}

func TestAbort_UnconfirmedDeliveryBecomesUncertainWithoutReleasingSlot(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	repo := mocks.NewFakeRepository()
	canceller := &fakeCanceller{}
	var hookRuns []string
	svc := runs.NewService(repo, clk,
		runs.WithCanceller(canceller),
		runs.WithTerminalHook(func(_ context.Context, run runs.Run) { hookRuns = append(hookRuns, run.ID) }),
	)
	run, err := svc.Create(context.Background(), runs.CreateInput{NodeID: "n1", Verb: "scenario test"})
	require.NoError(t, err)
	_, err = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventStatus, Sequence: 1, Status: "running"})
	require.NoError(t, err)

	uncertain, err := svc.Abort(context.Background(), run.ID, "operator abort")
	require.NoError(t, err)
	require.Equal(t, runs.StatusUncertain, uncertain.Status)
	require.Contains(t, uncertain.StatusReason, "unconfirmed")
	require.Empty(t, hookRuns)

	_, err = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventExit, Sequence: 3, ExitCode: 143})
	require.NoError(t, err)
	confirmed, _, err := svc.Get(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, runs.StatusAborted, confirmed.Status)
	require.True(t, confirmed.CancellationConfirmed)
	require.Equal(t, []string{run.ID}, hookRuns)
}

func TestDeliveryAckAfterCancellationDoesNotReopenLifecycle(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	repo := mocks.NewFakeRepository()
	svc := runs.NewService(repo, clk, runs.WithCanceller(&fakeCanceller{delivered: 1}))
	run, err := svc.Create(context.Background(), runs.CreateInput{NodeID: "n1", Verb: "scenario test"})
	require.NoError(t, err)
	_, err = svc.AppendEvent(context.Background(), runs.RunEvent{RunID: run.ID, Kind: runs.EventStatus, Sequence: 1, Status: "running"})
	require.NoError(t, err)
	requested, err := svc.Abort(context.Background(), run.ID, "operator abort")
	require.NoError(t, err)

	require.NoError(t, svc.RecordDeliveryAck(context.Background(), runs.DeliveryAck{
		NodeID: "n1", FrameID: "cancel-frame", RunID: run.ID,
	}))
	got, _, err := svc.Get(context.Background(), run.ID)
	require.NoError(t, err)
	require.Equal(t, runs.StatusCancelRequested, requested.Status)
	require.Equal(t, runs.StatusCancelRequested, got.Status)
}

// [REQ:BRG-P1-004] A natural completion (node EXIT) fires the terminal hook but
// does NOT push a node-cancel (the run already exited).
func TestAppendEvent_TerminalFiresHookWithoutCancel(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
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
