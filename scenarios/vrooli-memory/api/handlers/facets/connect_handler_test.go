package facets

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	sourcev1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets"
	sourceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets/facets_v1connect"
	memoryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-memory/v1/facets"
)

type recordingFacetsClient struct {
	sourceconnect.UnimplementedFacetsServiceHandler
	assignScope string
}

func (f *recordingFacetsClient) AssignFacet(_ context.Context, req *connect.Request[sourcev1.AssignFacetRequest]) (*connect.Response[sourcev1.AssignFacetResponse], error) {
	f.assignScope = req.Msg.GetScope()
	return connect.NewResponse(&sourcev1.AssignFacetResponse{}), nil
}

func TestConnectHandlerPreservesExplicitFacetScope(t *testing.T) {
	client := &recordingFacetsClient{}
	handler := NewConnectHandler(client, nil)

	_, err := handler.AssignFacet(context.Background(), connect.NewRequest(&memoryv1.AssignFacetRequest{
		EntryId: "entry-1",
		FacetId: "episode",
		Scope:   "team:director-swarm",
	}))
	require.NoError(t, err)
	require.Equal(t, "team:director-swarm", client.assignScope)
}
