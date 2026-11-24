package readinessprofile

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	readinessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/readiness"
	"github.com/vrooli/vrooli/packages/proto/gen/go/browser-automation-studio/v1/readiness/readinessconnect"
)

type fakeReadinessService struct {
	readinessconnect.UnimplementedReadinessServiceHandler
	request *readinessv1.GetReadinessProfileRequest
}

func (f *fakeReadinessService) GetReadinessProfile(_ context.Context, req *connect.Request[readinessv1.GetReadinessProfileRequest]) (*connect.Response[readinessv1.GetReadinessProfileResponse], error) {
	f.request = req.Msg
	return connect.NewResponse(&readinessv1.GetReadinessProfileResponse{
		Scenario: "image-tools", ProfileVersion: "profile/v3",
		ProfileJson: `{"version":"profile/v3","pages":[{"id":"editor","routes":["/editor"],"runtimeRoutes":["/editor?tab=assets"],"regions":[{"id":"asset-grid","required":true,"binding":{"testId":"asset-grid","note":"visible after load"},"component":{"local":"asset-grid","library":null},"lifecycle":{"kind":"async","states":["loading","ready"]}}]}]}`,
	}), nil
}

func TestClientUsesBASBoundaryAndReturnsSharedProfile(t *testing.T) {
	service := &fakeReadinessService{}
	path, handler := readinessconnect.NewReadinessServiceHandler(service)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := NewClientWithResolver(func(context.Context) (string, error) { return server.URL, nil }, server.Client())
	profile, err := client.GetReadinessProfile(t.Context(), "image-tools", "/editor?tab=assets")
	if err != nil {
		t.Fatal(err)
	}
	if service.request.GetScenario() != "image-tools" || service.request.GetPath() != "/editor?tab=assets" {
		t.Fatalf("BAS request = %+v", service.request)
	}
	if profile.Scenario != "image-tools" || profile.Version != "profile/v3" || len(profile.Pages) != 1 || len(profile.Pages[0].Regions) != 1 {
		t.Fatalf("profile projection = %+v", profile)
	}
	region := profile.Pages[0].Regions[0]
	if region.Binding.TestID != "asset-grid" || region.Binding.Note != "visible after load" || region.Lifecycle.Kind != "async" || len(region.Lifecycle.States) != 2 {
		t.Fatalf("readiness contract fields were lost: %+v", region)
	}
	if string(region.Component) != `{"local":"asset-grid","library":null}` {
		t.Fatalf("component metadata was lost: %s", region.Component)
	}
}

func TestClientRejectsMissingResolver(t *testing.T) {
	if _, err := (*Client)(nil).GetReadinessProfile(t.Context(), "demo", ""); err == nil {
		t.Fatal("nil client must fail closed")
	}
}
