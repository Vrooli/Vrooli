package forest

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/forest/forest_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/forest"
)

type recordingForestClient struct {
	sourceconnect.UnimplementedForestServiceHandler
	compactionScope string
	frontierScope   string
}

func (f *recordingForestClient) RunCompactionPass(_ context.Context, req *connect.Request[sourcev1.RunCompactionPassRequest]) (*connect.Response[sourcev1.RunCompactionPassResponse], error) {
	f.compactionScope = req.Msg.GetScope()
	return connect.NewResponse(&sourcev1.RunCompactionPassResponse{}), nil
}

func (f *recordingForestClient) GetFrontier(_ context.Context, req *connect.Request[sourcev1.GetFrontierRequest]) (*connect.Response[sourcev1.GetFrontierResponse], error) {
	f.frontierScope = req.Msg.GetScope()
	return connect.NewResponse(&sourcev1.GetFrontierResponse{}), nil
}

func (f *recordingForestClient) GetNode(_ context.Context, req *connect.Request[sourcev1.GetNodeRequest]) (*connect.Response[sourcev1.GetNodeResponse], error) {
	return connect.NewResponse(&sourcev1.GetNodeResponse{}), nil
}

func (f *recordingForestClient) RebuildForest(_ context.Context, req *connect.Request[sourcev1.RebuildForestRequest]) (*connect.Response[sourcev1.RebuildForestResponse], error) {
	return connect.NewResponse(&sourcev1.RebuildForestResponse{}), nil
}

func TestConnectHandlerPreservesExplicitForestScope(t *testing.T) {
	client := &recordingForestClient{}
	handler := NewConnectHandler(client, nil)

	_, err := handler.RunCompactionPass(context.Background(), connect.NewRequest(&memoryv1.RunCompactionPassRequest{Scope: "team:director-swarm"}))
	require.NoError(t, err)
	_, err = handler.GetFrontier(context.Background(), connect.NewRequest(&memoryv1.GetFrontierRequest{Scope: "team:director-swarm"}))
	require.NoError(t, err)

	require.Equal(t, "team:director-swarm", client.compactionScope)
	require.Equal(t, "team:director-swarm", client.frontierScope)
}

func TestConnectHandlerUsesDefaultForestScopeWhenOmitted(t *testing.T) {
	client := &recordingForestClient{}
	handler := NewConnectHandler(client, nil)

	_, err := handler.GetFrontier(context.Background(), connect.NewRequest(&memoryv1.GetFrontierRequest{}))
	require.NoError(t, err)
	require.Equal(t, "agent-memory", client.frontierScope)
}
