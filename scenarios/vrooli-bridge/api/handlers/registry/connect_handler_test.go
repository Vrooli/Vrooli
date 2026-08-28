package registry

import (
	"context"
	"testing"
	"time"

	"vrooli-bridge/internal/auth"
	"vrooli-bridge/internal/registry"
	"vrooli-bridge/internal/registry/mocks"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
)

// fakePresence is a Presence double keyed by node id. flagged marks an online
// node as protocol-incompatible (online but not dispatchable → NEEDS_UPDATE).
type fakePresence struct {
	online  map[string]bool
	flagged map[string]bool
}

func (f fakePresence) IsOnline(id string) bool { return f.online[id] }

func (f fakePresence) Dispatchable(id string) bool { return f.online[id] && !f.flagged[id] }

func ownerCtx() context.Context {
	return auth.WithIdentity(context.Background(), auth.Identity{OwnerID: "owner-1"})
}

func newHarness(svc registry.Service, presence Presence) *connectHandler {
	return NewConnectHandler(Deps{Service: svc, Presence: presence})
}

// [REQ:BRG-P0-001] Every owner-gated RPC fails closed (Unauthenticated) when no
// owner identity is present in the context.
func TestHandler_RequiresOwner(t *testing.T) {
	h := newHarness(&mocks.FakeService{}, nil)
	ctx := context.Background() // no identity injected

	_, err := h.ListNodes(ctx, connect.NewRequest(&registryv1.ListNodesRequest{}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = h.RegisterNode(ctx, connect.NewRequest(&registryv1.RegisterNodeRequest{Name: "a", Os: "linux", Arch: "amd64"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = h.GetNode(ctx, connect.NewRequest(&registryv1.GetNodeRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = h.UpdateNode(ctx, connect.NewRequest(&registryv1.UpdateNodeRequest{Id: "x", Name: "y"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = h.RevokeNode(ctx, connect.NewRequest(&registryv1.RevokeNodeRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))

	_, err = h.RemoveNode(ctx, connect.NewRequest(&registryv1.RemoveNodeRequest{Id: "x"}))
	require.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}

func TestHandler_RemoveNode_DelegatesForOwner(t *testing.T) {
	svc := &mocks.FakeService{}
	h := newHarness(svc, nil)
	resp, err := h.RemoveNode(ownerCtx(), connect.NewRequest(&registryv1.RemoveNodeRequest{Id: "revoked-node"}))
	require.NoError(t, err)
	require.Equal(t, "revoked-node", resp.Msg.RemovedNodeId)
	require.Equal(t, []string{"revoked-node"}, svc.RemoveIDs)
}

// [REQ:BRG-P1-001] An online node whose agent protocol is flagged
// (online but not dispatchable) reads NEEDS_UPDATE in the overlay, so the
// operator/UI sees it is excluded from work until the agent is updated.
func TestHandler_GetNode_FlaggedNodeReadsNeedsUpdate(t *testing.T) {
	svc := &mocks.FakeService{GetOut: registry.Node{ID: "n1", Name: "stale", OS: "linux", Arch: "amd64", LastSeenAt: time.Now().UTC()}}
	h := newHarness(svc, fakePresence{
		online:  map[string]bool{"n1": true},
		flagged: map[string]bool{"n1": true},
	})

	resp, err := h.GetNode(ownerCtx(), connect.NewRequest(&registryv1.GetNodeRequest{Id: "n1"}))
	require.NoError(t, err)
	require.True(t, resp.Msg.Node.Online)
	require.Equal(t, registryv1.NodeStatus_NODE_STATUS_NEEDS_UPDATE, resp.Msg.Node.Status)
}

func TestHandler_RegisterNode_PassesInputAndOverlaysPresence(t *testing.T) {
	svc := &mocks.FakeService{RegisterOut: registry.Node{ID: "n1", Name: "a", OS: "linux", Arch: "amd64", LastSeenAt: time.Now().UTC()}}
	h := newHarness(svc, fakePresence{online: map[string]bool{"n1": true}})

	resp, err := h.RegisterNode(ownerCtx(), connect.NewRequest(&registryv1.RegisterNodeRequest{
		Name: "a", Os: "linux", Arch: "amd64", Scopes: []string{"scenario test*"},
	}))
	require.NoError(t, err)
	require.Equal(t, "n1", resp.Msg.Node.Id)
	require.True(t, resp.Msg.Node.Online, "presence overlay applied")
	require.Equal(t, registryv1.NodeStatus_NODE_STATUS_ONLINE, resp.Msg.Node.Status)

	require.Len(t, svc.RegisterInputs, 1)
	require.Equal(t, "a", svc.RegisterInputs[0].Name)
	require.Equal(t, []string{"scenario test*"}, svc.RegisterInputs[0].Scopes)
}

func TestHandler_ListNodes_OverlaysPerNodePresence(t *testing.T) {
	svc := &mocks.FakeService{ListOut: []registry.Node{
		{ID: "on", Name: "online-node", OS: "linux", Arch: "amd64", LastSeenAt: time.Now().UTC()},
		{ID: "off", Name: "offline-node", OS: "linux", Arch: "amd64"},
	}}
	h := newHarness(svc, fakePresence{online: map[string]bool{"on": true}})

	resp, err := h.ListNodes(ownerCtx(), connect.NewRequest(&registryv1.ListNodesRequest{}))
	require.NoError(t, err)
	require.Len(t, resp.Msg.Nodes, 2)

	byID := map[string]*registryv1.Node{}
	for _, n := range resp.Msg.Nodes {
		byID[n.Id] = n
	}
	require.True(t, byID["on"].Online)
	require.Equal(t, registryv1.NodeStatus_NODE_STATUS_ONLINE, byID["on"].Status)
	require.False(t, byID["off"].Online)
	require.Equal(t, registryv1.NodeStatus_NODE_STATUS_OFFLINE, byID["off"].Status)
}

func TestHandler_ListNodes_DoesNotTreatStaleDurableHeartbeatAsFresh(t *testing.T) {
	svc := &mocks.FakeService{ListOut: []registry.Node{{
		ID: "old", Name: "old-node", OS: "darwin", Arch: "amd64",
		LastSeenAt: time.Now().UTC().Add(-7 * 24 * time.Hour),
	}}}
	h := newHarness(svc, fakePresence{online: map[string]bool{"old": true}})

	resp, err := h.ListNodes(ownerCtx(), connect.NewRequest(&registryv1.ListNodesRequest{}))
	require.NoError(t, err)
	require.False(t, resp.Msg.Nodes[0].HeartbeatFresh)
	require.False(t, resp.Msg.Nodes[0].Online)
	require.False(t, resp.Msg.Nodes[0].Dispatchable)
	require.Greater(t, resp.Msg.Nodes[0].HeartbeatAgeSeconds, int64(7*24*60*60-60))
}

func TestHandler_ListNodes_DoesNotTreatStaleControlPlaneAsFresh(t *testing.T) {
	svc := &mocks.FakeService{ListOut: []registry.Node{{
		ID: "control-plane", Name: "swarminator", Kind: registry.KindControlPlane,
		OS: "linux", Arch: "amd64", LastSeenAt: time.Now().UTC().Add(-7 * 24 * time.Hour),
	}}}
	h := newHarness(svc, fakePresence{online: map[string]bool{"control-plane": true}})

	resp, err := h.ListNodes(ownerCtx(), connect.NewRequest(&registryv1.ListNodesRequest{}))
	require.NoError(t, err)
	node := resp.Msg.Nodes[0]
	require.False(t, node.HeartbeatFresh)
	require.False(t, node.Online)
	require.False(t, node.Dispatchable)
	require.Greater(t, node.HeartbeatAgeSeconds, int64(7*24*60*60-60))
}

func TestHandler_ListNodes_MissingHeartbeatIsNotFresh(t *testing.T) {
	svc := &mocks.FakeService{ListOut: []registry.Node{{
		ID: "never-seen", Name: "never-seen", Kind: registry.KindControlPlane,
		OS: "linux", Arch: "amd64",
	}}}
	h := newHarness(svc, fakePresence{online: map[string]bool{"never-seen": true}})

	resp, err := h.ListNodes(ownerCtx(), connect.NewRequest(&registryv1.ListNodesRequest{}))
	require.NoError(t, err)
	node := resp.Msg.Nodes[0]
	require.False(t, node.HeartbeatFresh)
	require.False(t, node.Online)
	require.False(t, node.Dispatchable)
	require.Equal(t, int64(0), node.HeartbeatAgeSeconds)
}

// [REQ:BRG-P0-001] A revoked node always reads REVOKED, never online, even if a
// channel lingers (the handler does not consult presence on revoke).
func TestHandler_RevokeNode_StatusIsRevoked(t *testing.T) {
	revokedNode := registry.Node{ID: "n1", Name: "a", OS: "linux", Arch: "amd64"}
	revokedNode.RevokedAt = revokedNode.CreatedAt.Add(1) // non-zero
	svc := &mocks.FakeService{RevokeOut: revokedNode}
	h := newHarness(svc, fakePresence{online: map[string]bool{"n1": true}})

	resp, err := h.RevokeNode(ownerCtx(), connect.NewRequest(&registryv1.RevokeNodeRequest{Id: "n1"}))
	require.NoError(t, err)
	require.Equal(t, registryv1.NodeStatus_NODE_STATUS_REVOKED, resp.Msg.Node.Status)
	require.False(t, resp.Msg.Node.Online)
	require.Equal(t, []string{"n1"}, svc.RevokeIDs)
}

// fakeCredentialRevoker records which nodes had their credentials severed.
type fakeCredentialRevoker struct {
	revoked []string
	err     error
}

func (f *fakeCredentialRevoker) RevokeCredential(_ context.Context, nodeID string) error {
	f.revoked = append(f.revoked, nodeID)
	return f.err
}

// fakeDisconnector records which nodes were force-disconnected.
type fakeDisconnector struct{ disconnected []string }

func (f *fakeDisconnector) Disconnect(nodeID string) int {
	f.disconnected = append(f.disconnected, nodeID)
	return 1
}

// [REQ:BRG-P0-002] Atomic revocation: a single RevokeNode revokes the durable
// record AND severs the mutual-auth credential AND drops the live channel — in
// one operation, so the node loses identity, auth, and presence together.
func TestHandler_RevokeNode_IsAtomic(t *testing.T) {
	revokedNode := registry.Node{ID: "n1", Name: "a", OS: "linux", Arch: "amd64"}
	revokedNode.RevokedAt = revokedNode.CreatedAt.Add(1)
	svc := &mocks.FakeService{RevokeOut: revokedNode}
	creds := &fakeCredentialRevoker{}
	disc := &fakeDisconnector{}
	h := NewConnectHandler(Deps{Service: svc, Credentials: creds, Disconnect: disc})

	_, err := h.RevokeNode(ownerCtx(), connect.NewRequest(&registryv1.RevokeNodeRequest{Id: "n1"}))
	require.NoError(t, err)

	require.Equal(t, []string{"n1"}, svc.RevokeIDs, "durable record revoked")
	require.Equal(t, []string{"n1"}, creds.revoked, "mutual-auth credential severed")
	require.Equal(t, []string{"n1"}, disc.disconnected, "live channel dropped")
}

// [REQ:BRG-P0-002] A credential-revoke failure does not block the durable
// revoke (best-effort-logged) — the node is still marked revoked.
func TestHandler_RevokeNode_CredentialFailureStillRevokes(t *testing.T) {
	revokedNode := registry.Node{ID: "n1"}
	revokedNode.RevokedAt = revokedNode.CreatedAt.Add(1)
	svc := &mocks.FakeService{RevokeOut: revokedNode}
	creds := &fakeCredentialRevoker{err: context.DeadlineExceeded}
	disc := &fakeDisconnector{}
	h := NewConnectHandler(Deps{Service: svc, Credentials: creds, Disconnect: disc})

	resp, err := h.RevokeNode(ownerCtx(), connect.NewRequest(&registryv1.RevokeNodeRequest{Id: "n1"}))
	require.NoError(t, err)
	require.Equal(t, registryv1.NodeStatus_NODE_STATUS_REVOKED, resp.Msg.Node.Status)
	require.Equal(t, []string{"n1"}, disc.disconnected, "channel still dropped despite credential error")
}

func TestHandler_GetNode_NotFoundMapsTo404(t *testing.T) {
	svc := &mocks.FakeService{GetErr: registry.ErrNodeNotFound{ID: "ghost"}}
	h := newHarness(svc, nil)
	_, err := h.GetNode(ownerCtx(), connect.NewRequest(&registryv1.GetNodeRequest{Id: "ghost"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestHandler_RegisterNode_InvalidMapsTo400(t *testing.T) {
	svc := &mocks.FakeService{RegisterErr: registry.ErrInvalidNode{Field: "name", Reason: "required"}}
	h := newHarness(svc, nil)
	_, err := h.RegisterNode(ownerCtx(), connect.NewRequest(&registryv1.RegisterNodeRequest{Os: "linux", Arch: "amd64"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
