package surfaces

import (
	"connectrpc.com/connect"
	"context"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	surfacesv1 "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces"
	surfacesconnect "github.com/vrooli/vrooli/packages/proto/gen/go/portal/v1/surfaces/surfacesv1connect"
	"net/http/httptest"
	"portal/internal/surfaces"
	"testing"
	"time"
)

func TestCatalogConnectWithoutProvidersAndInvalidReference(t *testing.T) { // [REQ:PORTAL-EVERYWHERE-OPT-01]
	catalog, err := surfaces.NewCatalog(nil, time.Second, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	_, handler := surfacesconnect.NewSurfaceCatalogServiceHandler(&Handler{Catalog: catalog})
	server := httptest.NewServer(handler)
	defer server.Close()
	client := surfacesconnect.NewSurfaceCatalogServiceClient(server.Client(), server.URL)
	response, err := client.List(context.Background(), connect.NewRequest(&surfacesv1.ListRequest{}))
	if err != nil || len(response.Msg.Surfaces) != 0 {
		t.Fatalf("empty catalog unavailable: %v", err)
	}
	_, err = client.Resolve(context.Background(), connect.NewRequest(&surfacesv1.ResolveRequest{ExactRef: &commonv1.SurfaceRef{SurfaceId: "https://attacker.invalid"}}))
	if connect.CodeOf(err) != connect.CodeInvalidArgument {
		t.Fatalf("malformed reference not rejected: %v", err)
	}
}
