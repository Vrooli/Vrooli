package wiring

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

type sandboxHealthStub struct {
	ok     bool
	reason string
}

func (s sandboxHealthStub) IsAvailable(context.Context) (bool, string) {
	return s.ok, s.reason
}

func TestSetupRoutesRegistersBaselineHealthAndMetricsWithoutOptionalServices(t *testing.T) {
	router := mux.NewRouter()
	SetupRoutes(router, RouteDependencies{})
	for _, path := range []string{"/health", "/api/v1/health", "/metrics"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		match := &mux.RouteMatch{}
		if !router.Match(req, match) {
			t.Fatalf("route %s was not registered", path)
		}
	}
}

func TestWorkspaceSandboxHealthCheckerPreservesUnavailableReason(t *testing.T) {
	checker := workspaceSandboxHealthChecker(sandboxHealthStub{reason: "endpoint is not configured"})
	if checker == nil {
		t.Fatal("workspace sandbox health checker is nil")
	}
	result := checker.Check(context.Background())
	if result.Connected || result.Error == nil || result.Error.Error() != "endpoint is not configured" {
		t.Fatalf("health result = %+v", result)
	}
}

func TestWorkspaceSandboxHealthCheckerAcceptsAvailableProvider(t *testing.T) {
	result := workspaceSandboxHealthChecker(sandboxHealthStub{ok: true}).Check(context.Background())
	if !result.Connected || result.Error != nil {
		t.Fatalf("health result = %+v", result)
	}
}

func TestWorkspaceSandboxHealthCheckerSkipsMissingProvider(t *testing.T) {
	if checker := workspaceSandboxHealthChecker(nil); checker != nil {
		t.Fatal("missing provider should not register a health check")
	}
}
