package assetstudio

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	studiov1 "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio"
	studioconnect "github.com/vrooli/vrooli/packages/proto/gen/go/asset-studio/v1/studio/studio_v1connect"
)

type staticResolver struct{ url string }

func (r staticResolver) ResolveScenarioURLDefault(context.Context, string) (string, error) {
	return r.url, nil
}

type studioHandler struct {
	studioconnect.UnimplementedStudioServiceHandler
}

func (studioHandler) GetReleasedAssetReference(_ context.Context, request *connect.Request[studiov1.GetReleasedAssetReferenceRequest]) (*connect.Response[studiov1.GetReleasedAssetReferenceResponse], error) {
	if request.Msg.AssetId != "asset-1" {
		return nil, connect.NewError(connect.CodeNotFound, nil)
	}
	return connect.NewResponse(&studiov1.GetReleasedAssetReferenceResponse{Asset: &studiov1.AssetReference{Id: "asset-1", Status: "released"}}), nil
}

// [REQ:CHANMGR-P1-003] The client resolves only a released metadata reference;
// it neither requests nor has a route for artifact bytes.
func TestClientResolvesReleasedAssetReference(t *testing.T) {
	path, handler := studioconnect.NewStudioServiceHandler(studioHandler{})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)
	client := &Client{resolver: staticResolver{url: server.URL}, http: server.Client()}
	reference, err := client.ResolveReleasedAsset(t.Context(), "asset-1")
	if err != nil || reference.ID != "asset-1" {
		t.Fatalf("reference=%+v err=%v", reference, err)
	}
}
