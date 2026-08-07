package health_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"vrooli-memory/internal/maintenance"

	"vrooli-memory/handlers/health"
	"vrooli-memory/internal/testutil/mocks"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// TestModule_Shape pins the public contract.
func TestModule_Shape(t *testing.T) {
	m := health.Module(&mocks.FakePinger{}, "react-vite-test", "1.0.0", nil)

	require.Equal(t, "health", m.Name)
	require.NotNil(t, m.Mount)
	require.Same(t, &health.Endpoints[0], &m.Endpoints[0],
		"Module.Endpoints must reference the package-level Endpoints slice")
	require.Len(t, health.Endpoints, 1, "health ships a single endpoint with /api/v1/health as a path alias")
}

// TestModule_BothAliasesReachable proves the /health and /api/v1/health
// paths both route to the same handler. A regression that drops one
// alias (forgot to register, typo) fails here, not in the e2e gate.
func TestModule_BothAliasesReachable(t *testing.T) {
	m := health.Module(&mocks.FakePinger{}, "react-vite-test", "1.0.0", nil)
	r := mux.NewRouter()
	m.Mount(r)

	for _, path := range []string{"/health", "/api/v1/health"} {
		t.Run(path, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, path, nil)
			require.NoError(t, err)
			rw := newRecorder()
			r.ServeHTTP(rw, req)

			require.Equal(t, http.StatusOK, rw.status,
				"unexpected status; body=%s", rw.body.String())
			require.Contains(t, rw.body.String(), `"status"`,
				"response must include the status field")
		})
	}
}

type recorder struct {
	headers http.Header
	body    *strings.Builder
	status  int
}

func newRecorder() *recorder {
	return &recorder{
		headers: http.Header{},
		body:    &strings.Builder{},
		status:  http.StatusOK,
	}
}

func (r *recorder) Header() http.Header         { return r.headers }
func (r *recorder) Write(p []byte) (int, error) { return r.body.Write(p) }
func (r *recorder) WriteHeader(s int)           { r.status = s }

// The canopy check must report the engine's recorded numbers verbatim. An
// earlier version recomputed eligibility in SQL and read 273 against the
// engine's 16, because it omitted the recency, pin, and vector guards.
type stubCanopy struct {
	run maintenance.Run
	err error
}

func (s stubCanopy) Latest(context.Context) (maintenance.Run, error) { return s.run, s.err }

func TestCanopyReportsRecordedFrontierWithoutRecomputing(t *testing.T) {
	cases := []struct {
		name        string
		compaction  maintenance.Compaction
		wantDetail  string
		wantHealthy bool
	}{
		{
			name:        "at target",
			compaction:  maintenance.Compaction{Status: "completed", FrontierAfter: 16, Target: 16},
			wantDetail:  "eligible_frontier=16 target=16 last_pass=completed status=ok",
			wantHealthy: true,
		},
		{
			name:        "inside the backlog factor",
			compaction:  maintenance.Compaction{Status: "completed", FrontierAfter: 160, Target: 16},
			wantDetail:  "eligible_frontier=160 target=16 last_pass=completed status=ok",
			wantHealthy: true,
		},
		{
			name:        "beyond the backlog factor",
			compaction:  maintenance.Compaction{Status: "completed", FrontierAfter: 161, Target: 16},
			wantDetail:  "eligible_frontier=161 target=16 last_pass=completed status=lagging",
			wantHealthy: false,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			h := health.NewHandler(health.Deps{
				Pinger: &mocks.FakePinger{}, Service: "s", Version: "1",
				Canopy: stubCanopy{run: maintenance.Run{Compaction: tc.compaction}},
			})
			rec := httptest.NewRecorder()
			h(rec, httptest.NewRequest(http.MethodGet, "/health", nil))

			var body struct {
				Status       string `json:"status"`
				Dependencies map[string]struct {
					Database string `json:"database"`
				} `json:"dependencies"`
			}
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
			require.Equal(t, tc.wantDetail, body.Dependencies["canopy"].Database)
			require.Equal(t, tc.wantHealthy, body.Status == "healthy")
		})
	}
}
