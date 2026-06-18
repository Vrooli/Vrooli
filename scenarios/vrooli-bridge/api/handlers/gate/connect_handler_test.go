package gate

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/auth"
	internalgate "vrooli-bridge/internal/gate"
	gatemocks "vrooli-bridge/internal/gate/mocks"
	testmocks "vrooli-bridge/internal/testutil/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	gatev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/gate"
)

func ownerCtx() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

func newHarness(t *testing.T) (*connectHandler, *gatemocks.FakeRunner) {
	t.Helper()
	nodes := &gatemocks.FakeNodeLister{Nodes: []internalgate.NodeRef{
		{ID: "ubuntu-1", OS: "linux"},
		{ID: "mac-1", OS: "darwin"},
		{ID: "win-1", OS: "windows"},
	}}
	presence := &gatemocks.FakePresence{Online: map[string]bool{"ubuntu-1": true, "mac-1": true, "win-1": true}}
	runner := gatemocks.NewFakeRunner()
	clk := testmocks.NewFakeClock(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	svc := internalgate.NewService(gatemocks.NewFakeRepository(), nodes, presence, runner, clk)
	return NewConnectHandler(Deps{Service: svc}), runner
}

func runReq() *connect.Request[gatev1.RunGateRequest] {
	return connect.NewRequest(&gatev1.RunGateRequest{
		Scenario:       "web-search",
		TargetRevision: "a1b2c3d",
		TargetOses:     []string{"linux", "darwin", "windows"},
	})
}

// [REQ:BRG-P1-002] The gate verbs are owner-gated: no identity → Unauthenticated.
func TestGateHandler_RequiresOwner(t *testing.T) {
	h, _ := newHarness(t)
	ctx := context.Background()

	_, err := h.RunGate(ctx, runReq())
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.GetGate(ctx, connect.NewRequest(&gatev1.GetGateRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.WaitGate(ctx, connect.NewRequest(&gatev1.WaitGateRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.ListGates(ctx, connect.NewRequest(&gatev1.ListGatesRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P1-002] RunGate fans out one validation per OS and the gate is
// retrievable by id with its per-OS ledger.
func TestGateHandler_RunAndGet(t *testing.T) {
	h, _ := newHarness(t)

	resp, err := h.RunGate(ownerCtx(), runReq())
	require.NoError(t, err)
	require.False(t, resp.Msg.DryRun)
	require.NotEmpty(t, resp.Msg.GateId)
	require.Equal(t, gatev1.GateVerdict_GATE_VERDICT_PENDING, resp.Msg.Verdict)
	require.Len(t, resp.Msg.Results, 3)

	got, err := h.GetGate(ownerCtx(), connect.NewRequest(&gatev1.GetGateRequest{Id: resp.Msg.GateId}))
	require.NoError(t, err)
	require.Equal(t, "web-search", got.Msg.Gate.Scenario)
	require.Equal(t, "a1b2c3d", got.Msg.Gate.TargetRevision)
	require.Len(t, got.Msg.Results, 3)
}

// [REQ:BRG-P1-002] RunGate honours the X-Dry-Run header: it selects + classifies
// each OS and short-circuits with dry_run=true and an empty gate id.
func TestGateHandler_RunDryRunHeader(t *testing.T) {
	h, runner := newHarness(t)
	req := runReq()
	req.Header().Set(dryRunHeader, "true")

	resp, err := h.RunGate(ownerCtx(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.DryRun)
	require.Empty(t, resp.Msg.GateId)
	require.Len(t, resp.Msg.Results, 3)
	require.Empty(t, runner.DispatchedNodes(), "nothing dispatched on a dry-run")
}

// [REQ:BRG-P1-002] WaitGate returns the final PASSED verdict once every OS's
// validation run finishes green.
func TestGateHandler_WaitToVerdict(t *testing.T) {
	h, runner := newHarness(t)

	resp, err := h.RunGate(ownerCtx(), runReq())
	require.NoError(t, err)
	for _, r := range resp.Msg.Results {
		runner.SetVerdict(r.RunId, internalgate.RunVerdict{Terminal: true, Passed: true})
	}

	got, err := h.WaitGate(ownerCtx(), connect.NewRequest(&gatev1.WaitGateRequest{Id: resp.Msg.GateId}))
	require.NoError(t, err)
	require.False(t, got.Msg.TimedOut)
	require.Equal(t, gatev1.GateVerdict_GATE_VERDICT_PASSED, got.Msg.Gate.Verdict)
}

// A missing gate surfaces a NotFound code (typed sentinel mapping).
func TestGateHandler_GetNotFound(t *testing.T) {
	h, _ := newHarness(t)
	_, err := h.GetGate(ownerCtx(), connect.NewRequest(&gatev1.GetGateRequest{Id: "nope"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

// A structural validation failure surfaces InvalidArgument.
func TestGateHandler_InvalidArgument(t *testing.T) {
	h, _ := newHarness(t)
	_, err := h.RunGate(ownerCtx(), connect.NewRequest(&gatev1.RunGateRequest{Scenario: "", TargetRevision: "r", TargetOses: []string{"linux"}}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
