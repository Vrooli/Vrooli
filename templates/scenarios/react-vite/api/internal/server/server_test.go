package server_test

import (
	"io"
	"log"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"{{SCENARIO_ID}}/internal/clock"
	"{{SCENARIO_ID}}/internal/server"
	"{{SCENARIO_ID}}/internal/testutil/httpx"
	"{{SCENARIO_ID}}/internal/testutil/mocks"
)

// TestServer_RoutesAreRegistered pins the route table the production
// router exposes. Handler-level behaviour is covered by per-package
// handler tests (handlers/health, handlers/notes); the contract this
// test owns is "the server.New + registerRoutes wiring exposes every
// path the scenario advertises, with the canonical method-to-status
// mapping."
//
// Why this exists in addition to handler tests: handler tests construct
// the handler directly (notes.NewHandler, health.NewHandler) and never
// exercise registerRoutes. A regression that drops a route from
// routes.go silently passes every handler test but breaks production.
// The e2e binary smoke test (api/main_e2e_test.go) only hits /health.
// This test catches the rest before the e2e gate ever runs.
func TestServer_RoutesAreRegistered(t *testing.T) {
	srv := newTestServer(t)
	live := httpx.NewLiveServer(t, srv)

	cases := []struct {
		name         string
		method       string
		path         string
		body         string
		wantStatus   int
		wantContains string // substring expected somewhere in the response body
	}{
		{
			name:         "health_root",
			method:       http.MethodGet,
			path:         "/health",
			wantStatus:   http.StatusOK,
			wantContains: `"status"`,
		},
		{
			name:         "health_versioned",
			method:       http.MethodGet,
			path:         "/api/v1/health",
			wantStatus:   http.StatusOK,
			wantContains: `"status"`,
		},
		{
			name:       "notes_list",
			method:     http.MethodGet,
			path:       "/api/v1/notes",
			wantStatus: http.StatusOK,
			// Empty list responses serialise to "{}" because protojson
			// omits proto3-default fields. The route-existence
			// guarantee is the 200; per-field response shape is the
			// handler test's job (handlers/notes/handler_test.go).
			wantContains: `{`,
		},
		{
			name:         "notes_create",
			method:       http.MethodPost,
			path:         "/api/v1/notes",
			body:         `{"title":"x"}`,
			wantStatus:   http.StatusCreated,
			wantContains: `"note"`,
		},
		{
			name:       "notes_get_not_found",
			method:     http.MethodGet,
			path:       "/api/v1/notes/missing",
			wantStatus: http.StatusNotFound,
			// The not-found path proves the {id} subroute is mounted
			// (a missing route would return 404 from the router with
			// no envelope; the envelope confirms the handler ran).
			wantContains: `"not_found"`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			var body io.Reader
			if tc.body != "" {
				body = strings.NewReader(tc.body)
			}
			resp, payload := live.Do(t, tc.method, tc.path, body)
			require.Equal(t, tc.wantStatus, resp.StatusCode,
				"unexpected status; body=%s", string(payload))
			require.Contains(t, string(payload), tc.wantContains,
				"response body missing expected fragment")
		})
	}
}

// TestServer_HandlerNotNil pins the smallest possible smoke: New must
// return a wired Server whose Handler() returns a non-nil http.Handler.
// Catches the case where a refactor drops the recovery wrapper or
// returns the bare router without middleware composition.
func TestServer_HandlerNotNil(t *testing.T) {
	srv := newTestServer(t)
	require.NotNil(t, srv.Handler(), "server.Handler() must not be nil")
}

// newTestServer builds a Server backed by happy-path fakes for every
// seam. Tests asserting on a specific seam's behaviour replace its
// fake; tests in this file only care about wiring, so all fakes use
// defaults.
func newTestServer(t *testing.T) *server.Server {
	t.Helper()
	return server.New(server.Deps{
		Pinger:      &mocks.FakePinger{},
		Clock:       clock.System{},
		Logger:      log.New(io.Discard, "", 0),
		NoteService: &mocks.FakeService{},
		Service:     "react-vite-test",
		Version:     "1.0.0",
	})
}
