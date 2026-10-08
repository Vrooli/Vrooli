package fleet

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/auth"
	internalfleet "vrooli-bridge/internal/fleet"
	fleetmocks "vrooli-bridge/internal/fleet/mocks"

	"github.com/vrooli/api-core/scheduletest"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	fleetv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/fleet"
)

func ownerCtx() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

func newHarness(t *testing.T) *connectHandler {
	t.Helper()
	nodes := &fleetmocks.FakeNodeLister{Nodes: []internalfleet.NodeRef{{ID: "n1"}, {ID: "n2"}}}
	presence := &fleetmocks.FakePresence{Online: map[string]bool{"n1": true, "n2": true}}
	prov := &fleetmocks.FakeProvisioner{}
	clk := scheduletest.New(time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC))
	svc := internalfleet.NewService(fleetmocks.NewFakeRepository(), nodes, presence, prov, clk)
	return NewConnectHandler(Deps{Service: svc})
}

// [REQ:BRG-P1-001] The fleet verbs are owner-gated: no identity → Unauthenticated.
func TestFleetHandler_RequiresOwner(t *testing.T) {
	h := newHarness(t)
	ctx := context.Background()

	_, err := h.RollFleet(ctx, connect.NewRequest(&fleetv1.RollFleetRequest{TargetRevision: "r"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.GetRollout(ctx, connect.NewRequest(&fleetv1.GetRolloutRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.ListRollouts(ctx, connect.NewRequest(&fleetv1.ListRolloutsRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P1-001] RollFleet dispatches eligible nodes and returns the per-node
// ledger; the rollout is retrievable by id.
func TestFleetHandler_RollAndGet(t *testing.T) {
	h := newHarness(t)

	resp, err := h.RollFleet(ownerCtx(), connect.NewRequest(&fleetv1.RollFleetRequest{TargetRevision: "rev-B"}))
	require.NoError(t, err)
	require.False(t, resp.Msg.DryRun)
	require.NotEmpty(t, resp.Msg.RolloutId)
	require.Equal(t, fleetv1.RolloutStatus_ROLLOUT_STATUS_DISPATCHED, resp.Msg.Status)
	require.Len(t, resp.Msg.Results, 2)

	got, err := h.GetRollout(ownerCtx(), connect.NewRequest(&fleetv1.GetRolloutRequest{Id: resp.Msg.RolloutId}))
	require.NoError(t, err)
	require.Equal(t, "rev-B", got.Msg.Rollout.TargetRevision)
	require.Len(t, got.Msg.Results, 2)
}

// [REQ:BRG-P1-001] RollFleet honours the X-Dry-Run header: it classifies and
// short-circuits with dry_run=true and an empty rollout id.
func TestFleetHandler_RollDryRunHeader(t *testing.T) {
	h := newHarness(t)
	req := connect.NewRequest(&fleetv1.RollFleetRequest{TargetRevision: "rev-B"})
	req.Header().Set(dryRunHeader, "true")

	resp, err := h.RollFleet(ownerCtx(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.DryRun)
	require.Empty(t, resp.Msg.RolloutId)
	require.Len(t, resp.Msg.Results, 2)
}
