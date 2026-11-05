package nodes

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/timestamppb"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	registryconnect "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry/registry_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	cliapptest "github.com/vrooli/cli-core/cliapptest"

	clitest "github.com/vrooli/cli-core/cliapptest"
)

// fakeRegistry is a stateful in-memory NodeRegistryService so the CLI↔API
// round-trip (register → list → get → revoke) exercises the real generated
// client + wire shapes, not just a canned response.
type fakeRegistry struct {
	mu    sync.Mutex
	seq   int
	nodes map[string]*registryv1.Node
}

func newFakeRegistry() *fakeRegistry { return &fakeRegistry{nodes: map[string]*registryv1.Node{}} }

func (f *fakeRegistry) RegisterNode(_ context.Context, req *connect.Request[registryv1.RegisterNodeRequest]) (*connect.Response[registryv1.RegisterNodeResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.seq++
	id := "n" + string(rune('0'+f.seq))
	node := &registryv1.Node{
		Id: id, Name: req.Msg.Name, Os: req.Msg.Os, Arch: req.Msg.Arch,
		Endpoint: req.Msg.Endpoint, Capabilities: req.Msg.Capabilities, Scopes: req.Msg.Scopes,
		Status: registryv1.NodeStatus_NODE_STATUS_OFFLINE, CreatedAt: timestamppb.Now(),
	}
	f.nodes[id] = node
	return connect.NewResponse(&registryv1.RegisterNodeResponse{Node: node}), nil
}

func (f *fakeRegistry) ListNodes(context.Context, *connect.Request[registryv1.ListNodesRequest]) (*connect.Response[registryv1.ListNodesResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	resp := &registryv1.ListNodesResponse{}
	for _, n := range f.nodes {
		resp.Nodes = append(resp.Nodes, n)
	}
	return connect.NewResponse(resp), nil
}

func (f *fakeRegistry) GetNode(_ context.Context, req *connect.Request[registryv1.GetNodeRequest]) (*connect.Response[registryv1.GetNodeResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, registryNotFound{})
	}
	return connect.NewResponse(&registryv1.GetNodeResponse{Node: n}), nil
}

func (f *fakeRegistry) GetNodeReadiness(ctx context.Context, req *connect.Request[registryv1.GetNodeRequest]) (*connect.Response[registryv1.GetNodeReadinessResponse], error) {
	node, err := f.GetNode(ctx, req)
	if err != nil {
		return nil, err
	}
	node.Msg.Node.RegistryRecordPresent = true
	return connect.NewResponse(&registryv1.GetNodeReadinessResponse{Node: node.Msg.Node}), nil
}

func (f *fakeRegistry) UpdateNode(_ context.Context, req *connect.Request[registryv1.UpdateNodeRequest]) (*connect.Response[registryv1.UpdateNodeResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, registryNotFound{})
	}
	n.Name = req.Msg.Name
	n.Endpoint = req.Msg.Endpoint
	return connect.NewResponse(&registryv1.UpdateNodeResponse{Node: n}), nil
}

func (f *fakeRegistry) RevokeNode(_ context.Context, req *connect.Request[registryv1.RevokeNodeRequest]) (*connect.Response[registryv1.RevokeNodeResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, registryNotFound{})
	}
	n.Status = registryv1.NodeStatus_NODE_STATUS_REVOKED
	n.RevokedAt = timestamppb.Now()
	return connect.NewResponse(&registryv1.RevokeNodeResponse{Node: n}), nil
}

func (f *fakeRegistry) RemoveNode(_ context.Context, req *connect.Request[registryv1.RemoveNodeRequest]) (*connect.Response[registryv1.RemoveNodeResponse], error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	n, ok := f.nodes[req.Msg.Id]
	if !ok {
		return nil, connect.NewError(connect.CodeNotFound, registryNotFound{})
	}
	if n.Status != registryv1.NodeStatus_NODE_STATUS_REVOKED {
		return nil, connect.NewError(connect.CodeFailedPrecondition, errors.New("node must be revoked"))
	}
	delete(f.nodes, req.Msg.Id)
	return connect.NewResponse(&registryv1.RemoveNodeResponse{RemovedNodeId: req.Msg.Id}), nil
}

type registryNotFound struct{}

func (registryNotFound) Error() string { return "node not found" }

func connectAPI(svc registryconnect.NodeRegistryServiceHandler) http.Handler {
	path, handler := registryconnect.NewNodeRegistryServiceHandler(svc)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	return mux
}

// [REQ:BRG-P0-001] CLI register/list/get/revoke round-trip against the API
// through the generated Connect client, mutating and reading real wire state.
func TestNodes_RegisterListGetRevokeRoundTrip(t *testing.T) {
	svc := newFakeRegistry()
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	// register
	registerSchema := cliapp.ArgSchema{Flags: []cliapp.Flag{
		{Name: "name"},
		{Name: "os"},
		{Name: "arch"},
		{Name: "kind"},
		{Name: "endpoint"},
		{Name: "capabilities"},
		{Name: "scopes"},
	}}
	regCtx, _ := cliapptest.NewCapturedRunContext(core, registerSchema, cliapptest.TestRunContextOptions{
		Flags: map[string]string{"name": "office-linux", "os": "linux", "arch": "amd64", "scopes": "scenario test*"},
	})
	require.NoError(t, h.register(regCtx))

	// list shows the registered node
	listCtx, listOut := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{}, cliapptest.TestRunContextOptions{})
	require.NoError(t, h.list(listCtx))
	require.Contains(t, listOut.String(), "office-linux")

	// get by id
	idSchema := cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id"}}}
	getCtx, getOut := cliapptest.NewCapturedRunContext(core, idSchema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "n1"},
	})
	require.NoError(t, h.get(getCtx))
	require.Contains(t, getOut.String(), "office-linux")

	// revoke
	revCtx, revOut := cliapptest.NewCapturedRunContext(core, idSchema, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "n1"},
	})
	require.NoError(t, h.revoke(revCtx))
	require.Contains(t, revOut.String(), "Revoked")

	// the fake reflects the revoked status
	require.Equal(t, registryv1.NodeStatus_NODE_STATUS_REVOKED, svc.nodes["n1"].Status)
}

// [REQ:BRG-P0-001] A not-found get surfaces the Connect error to the operator.
func TestNodes_GetNotFoundSurfacesError(t *testing.T) {
	svc := newFakeRegistry()
	core := clitest.NewTestApp(t, connectAPI(svc))
	h := newHandlers(core)

	ctx, _ := cliapptest.NewCapturedRunContext(core, cliapp.ArgSchema{Positionals: []cliapp.Positional{{Name: "id"}}}, cliapptest.TestRunContextOptions{
		Positionals: map[string]string{"id": "ghost"},
	})
	require.Error(t, h.get(ctx))
}
