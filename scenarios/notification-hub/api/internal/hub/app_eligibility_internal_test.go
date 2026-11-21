package hub

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"connectrpc.com/connect"
	exposure "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure"
	exposureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/tunnel-manager/v1/exposure/exposure_v1connect"
)

type fakeExposures struct {
	exposureconnect.UnimplementedExposureServiceHandler
	items []*exposure.Exposure
}

func (f fakeExposures) ListExposures(context.Context, *connect.Request[exposure.ListExposuresRequest]) (*connect.Response[exposure.ListExposuresResponse], error) {
	return connect.NewResponse(&exposure.ListExposuresResponse{Exposures: f.items}), nil
}

func TestTunnelAppEligibilityRequiresAnEnabledCoreRoute(t *testing.T) {
	path, handler := exposureconnect.NewExposureServiceHandler(fakeExposures{items: []*exposure.Exposure{
		{Scenario: "web-console", PublicUrl: "https://web-console.example.test", Tier: "core", Enabled: true},
		{Scenario: "nutrition-planner", PublicUrl: "https://nutrition-planner.example.test", Tier: "leased", Enabled: true},
		{Scenario: "old-app", PublicUrl: "https://old-app.example.test", Tier: "core", Enabled: false},
	}})
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()
	check := TunnelAppEligibility(func(context.Context) (string, error) { return server.URL, nil })

	url, err := check(context.Background(), "web-console")
	if err != nil || url != "https://web-console.example.test" {
		t.Fatalf("core route: got %q, %v", url, err)
	}
	for scenario, want := range map[string]string{"nutrition-planner": "temporary address", "old-app": "no public address", "missing": "no public address"} {
		_, err := check(context.Background(), scenario)
		if !errors.Is(err, ErrAppIneligible) || !strings.Contains(err.Error(), want) {
			t.Errorf("%s: want ineligible %q, got %v", scenario, want, err)
		}
	}
	down := TunnelAppEligibility(func(context.Context) (string, error) { return "", errors.New("not running") })
	if _, err := down(context.Background(), "web-console"); !errors.Is(err, ErrAppIneligible) || !strings.Contains(err.Error(), "not reachable") {
		t.Errorf("tunnel-manager down: got %v", err)
	}
}
