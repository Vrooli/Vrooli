package assetstudio

import (
	"context"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	studiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio"
	studioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio/studio_v1connect"
)

type staticResolver struct{ url string }

func (r staticResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, nil
}

type referenceHandler struct {
	studioconnect.UnimplementedStudioServiceHandler
}

func (referenceHandler) GetReleasedAssetReference(_ context.Context, request *connect.Request[studiov1.GetReleasedAssetReferenceRequest]) (*connect.Response[studiov1.GetReleasedAssetReferenceResponse], error) {
	if request.Msg.AssetId != "asset-1" {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&studiov1.GetReleasedAssetReferenceResponse{Asset: &studiov1.AssetReference{Id: "asset-1", Status: "released", AltText: "Asset Studio source alt", MediaType: "image/png", Width: 1600, Height: 900}}), nil
}

// [REQ:CONTENTD-P1-009] Content Desk resolves only the released metadata
// reference; the generated contract has no image byte field to transfer.
func TestClientResolvesReleasedAssetMetadataOnly(t *testing.T) {
	_, handler := studioconnect.NewStudioServiceHandler(referenceHandler{})
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := &Client{resolver: staticResolver{url: server.URL}, http: server.Client()}
	ref, err := client.ResolveReleasedAsset(context.Background(), "asset-1")
	require.NoError(t, err)
	require.Equal(t, Reference{ID: "asset-1", AltText: "Asset Studio source alt", MediaType: "image/png", Width: 1600, Height: 900}, ref)
}
