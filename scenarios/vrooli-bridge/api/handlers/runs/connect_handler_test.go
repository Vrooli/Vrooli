package runs

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/auth"
	internalruns "vrooli-bridge/internal/runs"
	runsmocks "vrooli-bridge/internal/runs/mocks"

	"github.com/vrooli/api-core/scheduletest"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
)

func ownerCtx() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

func newHarness(t *testing.T) (*connectHandler, internalruns.Service) {
	t.Helper()
	svc := internalruns.NewService(runsmocks.NewFakeRepository(), scheduletest.New(time.Now()))
	// Verifier nil: ReportRunEvent skips node-auth (the pre-pairing stub path),
	// so this test exercises the lifecycle without crypto.
	return NewConnectHandler(Deps{Service: svc}), svc
}

func reportEvent(h *connectHandler, ev *sharedv1.RunEvent) (bool, error) {
	resp, err := h.ReportRunEvent(context.Background(), connect.NewRequest(&runsv1.ReportRunEventRequest{Event: ev}))
	if err != nil {
		return false, err
	}
	return resp.Msg.Accepted, nil
}

// [REQ:BRG-P0-005] The operator verbs are owner-gated: no identity →
// Unauthenticated.
func TestRunsHandler_OperatorVerbsRequireOwner(t *testing.T) {
	h, _ := newHarness(t)
	ctx := context.Background()

	_, err := h.GetRun(ctx, connect.NewRequest(&runsv1.GetRunRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.ListRuns(ctx, connect.NewRequest(&runsv1.ListRunsRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.WaitRun(ctx, connect.NewRequest(&runsv1.WaitRunRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.AbortRun(ctx, connect.NewRequest(&runsv1.AbortRunRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P0-005] ReportRunEvent is node-facing (NOT owner-gated): the agent
// reports without an owner identity. An event for an unknown run is acked
// (accepted=false) without error so a confused node stops re-sending.
func TestRunsHandler_ReportRunEvent_NodeFacing(t *testing.T) {
	h, _ := newHarness(t)
	accepted, err := reportEvent(h, &sharedv1.RunEvent{RunId: "ghost", Kind: sharedv1.RunEventKind_RUN_EVENT_KIND_LOG})
	require.NoError(t, err)
	require.False(t, accepted)

	// A missing run_id is an invalid argument.
	_, err = h.ReportRunEvent(context.Background(), connect.NewRequest(&runsv1.ReportRunEventRequest{Event: &sharedv1.RunEvent{}}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// [REQ:BRG-P0-005] End-to-end durable run through the handler: create a run, the
// node streams STATUS/LOG/EXIT back via ReportRunEvent, and the owner's
// block-once WaitRun returns the terminal verdict; GetRun re-attaches with the
// full event history.
func TestRunsHandler_DurableRunE2E(t *testing.T) {
	h, svc := newHarness(t)
	run, err := svc.Create(context.Background(), internalruns.CreateInput{NodeID: "n1", Scenario: "web-search", Verb: "scenario test"})
	require.NoError(t, err)

	// A blocked waiter wakes exactly once on the terminal transition.
	type waitResult struct {
		resp *connect.Response[runsv1.WaitRunResponse]
		err  error
	}
	done := make(chan waitResult, 1)
	go func() {
		resp, err := h.WaitRun(ownerCtx(), connect.NewRequest(&runsv1.WaitRunRequest{Id: run.ID, TimeoutSeconds: 5}))
		done <- waitResult{resp, err}
	}()
	time.Sleep(20 * time.Millisecond)

	// The node reports progress (no owner identity — node-facing).
	for _, ev := range []*sharedv1.RunEvent{
		{RunId: run.ID, Kind: sharedv1.RunEventKind_RUN_EVENT_KIND_STATUS, Sequence: 1, Status: "running"},
		{RunId: run.ID, Kind: sharedv1.RunEventKind_RUN_EVENT_KIND_LOG, Sequence: 2, LogChunk: "PASS web-search\n"},
		{RunId: run.ID, Kind: sharedv1.RunEventKind_RUN_EVENT_KIND_EXIT, Sequence: 3, ExitCode: 0},
	} {
		accepted, err := reportEvent(h, ev)
		require.NoError(t, err)
		require.True(t, accepted)
	}

	select {
	case got := <-done:
		require.NoError(t, got.err)
		require.False(t, got.resp.Msg.TimedOut)
		require.Equal(t, runsv1.RunStatus_RUN_STATUS_PASSED, got.resp.Msg.Run.Status)
	case <-time.After(2 * time.Second):
		t.Fatal("WaitRun did not return on the terminal transition")
	}

	// GetRun re-attaches with the full history.
	getResp, err := h.GetRun(ownerCtx(), connect.NewRequest(&runsv1.GetRunRequest{Id: run.ID}))
	require.NoError(t, err)
	require.Equal(t, runsv1.RunStatus_RUN_STATUS_PASSED, getResp.Msg.Run.Status)
	require.Len(t, getResp.Msg.Events, 3)
}

// [REQ:BRG-P0-005] GetRun on an unknown id is a NotFound.
func TestRunsHandler_GetRunNotFound(t *testing.T) {
	h, _ := newHarness(t)
	_, err := h.GetRun(ownerCtx(), connect.NewRequest(&runsv1.GetRunRequest{Id: "ghost"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}
