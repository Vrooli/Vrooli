package provision

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	provisionv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision"
	provisionconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/provision/provision_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

// fakeProvision is a minimal ProvisionService for the CLI round-trip. The
// waitStatus knob controls the verdict WaitProvisioningOp returns; lastSync
// captures the synced request.
type fakeProvision struct {
	waitStatus provisionv1.ProvisioningStatus
	hasVersion bool
	lastSync   *provisionv1.SyncToRevisionRequest
}

func (f *fakeProvision) SyncToRevision(_ context.Context, req *connect.Request[provisionv1.SyncToRevisionRequest]) (*connect.Response[provisionv1.SyncToRevisionResponse], error) {
	f.lastSync = req.Msg
	return connect.NewResponse(&provisionv1.SyncToRevisionResponse{
		OpId: "op-1", NodeId: req.Msg.NodeId, TargetRevision: req.Msg.TargetRevision, RollbackRevision: req.Msg.RollbackRevision,
	}), nil
}

func (f *fakeProvision) GetProvisioningOp(_ context.Context, req *connect.Request[provisionv1.GetProvisioningOpRequest]) (*connect.Response[provisionv1.GetProvisioningOpResponse], error) {
	return connect.NewResponse(&provisionv1.GetProvisioningOpResponse{
		Op:     &provisionv1.ProvisioningOp{Id: req.Msg.Id, NodeId: "n1", TargetRevision: "rev-B", Status: provisionv1.ProvisioningStatus_PROVISIONING_STATUS_COMPLETED},
		Events: []*provisionv1.ProvisionEvent{{OpId: req.Msg.Id, Kind: provisionv1.ProvisionEventKind_PROVISION_EVENT_KIND_VERSION, Revision: "rev-B"}},
	}), nil
}

func (f *fakeProvision) ListProvisioningOps(context.Context, *connect.Request[provisionv1.ListProvisioningOpsRequest]) (*connect.Response[provisionv1.ListProvisioningOpsResponse], error) {
	return connect.NewResponse(&provisionv1.ListProvisioningOpsResponse{Ops: []*provisionv1.ProvisioningOp{
		{Id: "op-1", NodeId: "n1", TargetRevision: "rev-B", Status: provisionv1.ProvisioningStatus_PROVISIONING_STATUS_COMPLETED},
	}}), nil
}

func (f *fakeProvision) WaitProvisioningOp(_ context.Context, req *connect.Request[provisionv1.WaitProvisioningOpRequest]) (*connect.Response[provisionv1.WaitProvisioningOpResponse], error) {
	return connect.NewResponse(&provisionv1.WaitProvisioningOpResponse{
		Op: &provisionv1.ProvisioningOp{Id: req.Msg.Id, NodeId: "n1", TargetRevision: "rev-B", ResultingRevision: "rev-B", Status: f.waitStatus},
	}), nil
}

func (f *fakeProvision) GetNodeVersion(_ context.Context, req *connect.Request[provisionv1.GetNodeVersionRequest]) (*connect.Response[provisionv1.GetNodeVersionResponse], error) {
	if !f.hasVersion {
		return connect.NewResponse(&provisionv1.GetNodeVersionResponse{HasVersion: false}), nil
	}
	return connect.NewResponse(&provisionv1.GetNodeVersionResponse{
		HasVersion: true,
		Version:    &provisionv1.NodeVersion{NodeId: req.Msg.NodeId, Revision: "rev-B", OpId: "op-1"},
	}), nil
}

func (f *fakeProvision) ReportProvisionEvent(context.Context, *connect.Request[provisionv1.ReportProvisionEventRequest]) (*connect.Response[provisionv1.ReportProvisionEventResponse], error) {
	return connect.NewResponse(&provisionv1.ReportProvisionEventResponse{Accepted: true}), nil
}

func connectAPI(svc provisionconnect.ProvisionServiceHandler) http.Handler {
	path, handler := provisionconnect.NewProvisionServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

// [REQ:BRG-P0-006] sync round-trips the privileged request through the generated
// client and reports the created op id.
func TestProvision_SyncRoundTrip(t *testing.T) {
	svc := &fakeProvision{}
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	schema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "node-id"}},
		Flags:       []cliapp.Flag{{Name: "revision"}, {Name: "rollback"}, {Name: "timeout"}},
	}
	ctx, out := cliapptest.NewCapturedRunContext(core, schema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"node-id": "n1"},
		Flags:       map[string]string{"revision": "rev-B", "rollback": "rev-A"},
	})
	require.NoError(t, h.sync(ctx))
	require.Equal(t, "n1", svc.lastSync.NodeId)
	require.Equal(t, "rev-B", svc.lastSync.TargetRevision)
	require.Equal(t, "rev-A", svc.lastSync.RollbackRevision)
	require.Contains(t, out.String(), "op-1")
}

// [REQ:BRG-P0-006] get / list round-trip through the generated client.
func TestProvision_GetAndList(t *testing.T) {
	core := clitest.NewTestApp(t, connectAPI(&fakeProvision{}))
	h := newHandlers(core)

	idSchema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id"}}}
	getCtx, getOut := cliapptest.NewCapturedRunContext(core, idSchema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"id": "op-1"}})
	require.NoError(t, h.get(getCtx))
	require.Contains(t, getOut.String(), "op-1")

	listCtx, listOut := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{Flags: []cliapp.Flag{{Name: "node"}, {Name: "limit"}}}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(listCtx))
	require.Contains(t, listOut.String(), "op-1")
}

// [REQ:BRG-P0-006] wait returns nil on a completed op and an error on a failed
// one (so the process exits non-zero).
func TestProvision_WaitExitsByVerdict(t *testing.T) {
	waitSchema := cliapp.ArgSchema{
		Positionals: []cliapp.Positional{{Name: "id"}},
		Flags:       []cliapp.Flag{{Name: "timeout"}},
	}

	okCore := clitest.NewTestApp(t, connectAPI(&fakeProvision{waitStatus: provisionv1.ProvisioningStatus_PROVISIONING_STATUS_COMPLETED}))
	ho := newHandlers(okCore)
	okCtx, _ := cliapptest.NewCapturedRunContext(okCore, waitSchema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"id": "op-1"}})
	require.NoError(t, ho.wait(okCtx), "a completed op exits zero")

	badCore := clitest.NewTestApp(t, connectAPI(&fakeProvision{waitStatus: provisionv1.ProvisioningStatus_PROVISIONING_STATUS_FAILED}))
	hb := newHandlers(badCore)
	badCtx, _ := cliapptest.NewCapturedRunContext(badCore, waitSchema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"id": "op-1"}})
	require.Error(t, hb.wait(badCtx), "a failed op exits non-zero")
}

// [REQ:BRG-P0-006] version reports the node's recorded revision, and the
// never-provisioned case is not an error.
func TestProvision_Version(t *testing.T) {
	schema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "node-id"}}}

	withVer := clitest.NewTestApp(t, connectAPI(&fakeProvision{hasVersion: true}))
	hv := newHandlers(withVer)
	verCtx, verOut := cliapptest.NewCapturedRunContext(withVer, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"node-id": "n1"}})
	require.NoError(t, hv.version(verCtx))
	require.Contains(t, verOut.String(), "rev-B")

	noVer := clitest.NewTestApp(t, connectAPI(&fakeProvision{hasVersion: false}))
	hn := newHandlers(noVer)
	noCtx, noOut := cliapptest.NewCapturedRunContext(noVer, schema, cliapptest.TestRunContextOptions{Positionals: map[string]string{"node-id": "n1"}})
	require.NoError(t, hn.version(noCtx))
	require.Contains(t, noOut.String(), "never been provisioned")
}
