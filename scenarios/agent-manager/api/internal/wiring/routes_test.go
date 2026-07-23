package wiring

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
)

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
