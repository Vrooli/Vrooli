package provision

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/clock"
	internalprovision "vrooli-bridge/internal/provision"
	provmocks "vrooli-bridge/internal/provision/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
)

func ownerCtx() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

func newHarness(t *testing.T) (*connectHandler, internalprovision.Service) {
	t.Helper()
	repo := provmocks.NewFakeRepository()
	nodes := &provmocks.FakeNodeReader{Nodes: map[string]internalprovision.TargetNode{"n1": {ID: "n1"}}}
	pres := &provmocks.FakePresence{Online: map[string]bool{"n1": true}}
	pusher := &provmocks.FakeCommandPusher{Delivered: 1}
	svc := internalprovision.NewService(repo, nodes, pres, &provmocks.FakeAuditSink{}, pusher, clock.System{})
	// Verifier nil: ReportProvisionEvent skips node-auth (pre-pairing stub path),
	// exercising the lifecycle without crypto.
	return NewConnectHandler(Deps{Service: svc}), svc
}

// [REQ:BRG-P0-006] The operator verbs are owner-gated: no identity →
// Unauthenticated.
func TestProvisionHandler_OperatorVerbsRequireOwner(t *testing.T) {
	h, _ := newHarness(t)
	ctx := context.Background()

	_, err := h.SyncToRevision(ctx, connect.NewRequest(&provisionv1.SyncToRevisionRequest{NodeId: "n1", TargetRevision: "r"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.GetProvisioningOp(ctx, connect.NewRequest(&provisionv1.GetProvisioningOpRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.ListProvisioningOps(ctx, connect.NewRequest(&provisionv1.ListProvisioningOpsRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.WaitProvisioningOp(ctx, connect.NewRequest(&provisionv1.WaitProvisioningOpRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
	_, err = h.GetNodeVersion(ctx, connect.NewRequest(&provisionv1.GetNodeVersionRequest{NodeId: "n1"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

// [REQ:BRG-P0-006] SyncToRevision honours the X-Dry-Run header: it validates and
// short-circuits with dry_run=true and an empty op id.
func TestProvisionHandler_SyncDryRunHeader(t *testing.T) {
	h, _ := newHarness(t)
	req := connect.NewRequest(&provisionv1.SyncToRevisionRequest{NodeId: "n1", TargetRevision: "rev-B"})
	req.Header().Set(dryRunHeader, "true")

	resp, err := h.SyncToRevision(ownerCtx(), req)
	require.NoError(t, err)
	require.True(t, resp.Msg.DryRun)
	require.Empty(t, resp.Msg.OpId)
	require.Equal(t, "rev-B", resp.Msg.TargetRevision)
}

// [REQ:BRG-P0-006] ReportProvisionEvent is node-facing (NOT owner-gated): the
// agent reports without an owner identity. An event for an unknown op is acked
// (accepted=false) without error; a missing op_id is an invalid argument.
func TestProvisionHandler_ReportEvent_NodeFacing(t *testing.T) {
	h, _ := newHarness(t)
	resp, err := h.ReportProvisionEvent(context.Background(), connect.NewRequest(&provisionv1.ReportProvisionEventRequest{
		Event: &provisionv1.ProvisionEvent{OpId: "ghost", Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_LOG},
	}))
	require.NoError(t, err)
	require.False(t, resp.Msg.Accepted)

	_, err = h.ReportProvisionEvent(context.Background(), connect.NewRequest(&provisionv1.ReportProvisionEventRequest{Event: &provisionv1.ProvisionEvent{}}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}

// [REQ:BRG-P0-006] End-to-end durable provisioning op through the handler:
// SyncToRevision creates the op, the node streams STATUS/VERSION/EXIT back via
// ReportProvisionEvent, the owner's block-once WaitProvisioningOp returns the
// terminal verdict, and GetNodeVersion reflects the new revision.
func TestProvisionHandler_DurableOpE2E(t *testing.T) {
	h, _ := newHarness(t)
	syncResp, err := h.SyncToRevision(ownerCtx(), connect.NewRequest(&provisionv1.SyncToRevisionRequest{NodeId: "n1", TargetRevision: "rev-B"}))
	require.NoError(t, err)
	id := syncResp.Msg.OpId
	require.NotEmpty(t, id)

	type waitResult struct {
		resp *connect.Response[provisionv1.WaitProvisioningOpResponse]
		err  error
	}
	done := make(chan waitResult, 1)
	go func() {
		resp, werr := h.WaitProvisioningOp(ownerCtx(), connect.NewRequest(&provisionv1.WaitProvisioningOpRequest{Id: id, TimeoutSeconds: 5}))
		done <- waitResult{resp, werr}
	}()
	time.Sleep(20 * time.Millisecond)

	report := func(ev *provisionv1.ProvisionEvent) {
		_, rerr := h.ReportProvisionEvent(context.Background(), connect.NewRequest(&provisionv1.ReportProvisionEventRequest{Event: ev}))
		require.NoError(t, rerr)
	}
	report(&provisionv1.ProvisionEvent{OpId: id, Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_STATUS, Sequence: 1, Status: "fetching"})
	report(&provisionv1.ProvisionEvent{OpId: id, Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_VERSION, Sequence: 2, Revision: "rev-B"})
	report(&provisionv1.ProvisionEvent{OpId: id, Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_EXIT, Sequence: 3, ExitCode: 0})

	select {
	case res := <-done:
		require.NoError(t, res.err)
		require.False(t, res.resp.Msg.TimedOut)
		require.Equal(t, provisionv1.ProvisioningStatus_PROVISIONING_STATUS_COMPLETED, res.resp.Msg.Op.Status)
	case <-time.After(2 * time.Second):
		t.Fatal("WaitProvisioningOp did not return after the terminal event")
	}

	// GetNodeVersion now reports rev-B.
	verResp, err := h.GetNodeVersion(ownerCtx(), connect.NewRequest(&provisionv1.GetNodeVersionRequest{NodeId: "n1"}))
	require.NoError(t, err)
	require.True(t, verResp.Msg.HasVersion)
	require.Equal(t, "rev-B", verResp.Msg.Version.Revision)

	// GetProvisioningOp re-attaches with the full event history.
	getResp, err := h.GetProvisioningOp(ownerCtx(), connect.NewRequest(&provisionv1.GetProvisioningOpRequest{Id: id}))
	require.NoError(t, err)
	require.Len(t, getResp.Msg.Events, 3)
}

// [REQ:BRG-P0-006] GetNodeVersion for a never-provisioned node is not an error —
// it reports has_version=false.
func TestProvisionHandler_GetNodeVersion_NeverProvisioned(t *testing.T) {
	h, _ := newHarness(t)
	resp, err := h.GetNodeVersion(ownerCtx(), connect.NewRequest(&provisionv1.GetNodeVersionRequest{NodeId: "n1"}))
	require.NoError(t, err)
	require.False(t, resp.Msg.HasVersion)
}
