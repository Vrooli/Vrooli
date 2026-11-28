package performance

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/sirupsen/logrus"
)

func TestPerformanceEndpointsShareJSONResponseOwner(t *testing.T) {
	t.Setenv("BAS_PERF_EXPOSE_ENDPOINT", "true")
	log := logrus.New()
	endpoints := NewEndpoints(NewCollectorRegistry(30, 8), log)
	cases := []struct {
		name string
		run  func(http.ResponseWriter, *http.Request)
		req  *http.Request
	}{
		{name: "all sessions", run: endpoints.GetAllPerformance, req: httptest.NewRequest(http.MethodGet, "/debug/performance", nil)},
		{name: "one session", run: endpoints.GetPerformance, req: httptest.NewRequest(http.MethodGet, "/debug/performance/session-a", nil)},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			req := tc.req
			if tc.name == "one session" {
				routeContext := chi.NewRouteContext()
				routeContext.URLParams.Add("sessionId", "session-a")
				req = req.WithContext(context.WithValue(req.Context(), chi.RouteCtxKey, routeContext))
			}
			response := httptest.NewRecorder()
			tc.run(response, req)
			if response.Code != http.StatusOK || response.Header().Get("Content-Type") != "application/json" {
				t.Fatalf("status/content type = %d/%q: %s", response.Code, response.Header().Get("Content-Type"), response.Body.String())
			}
			var body map[string]any
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatalf("decode response: %v", err)
			}
			if body["enabled"] != true {
				t.Fatalf("enabled = %#v, want true in %#v", body["enabled"], body)
			}
			if tc.name == "all sessions" && body["active_sessions"] != float64(0) {
				t.Fatalf("active_sessions = %#v, want 0", body["active_sessions"])
			}
			if tc.name == "one session" && body["session_id"] != "session-a" {
				t.Fatalf("session_id = %#v, want session-a", body["session_id"])
			}
		})
	}
}
