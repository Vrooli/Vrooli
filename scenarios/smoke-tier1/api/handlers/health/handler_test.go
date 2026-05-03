package health_test

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"

	"smoke-tier1/internal/clock"
	"smoke-tier1/internal/server"
	"smoke-tier1/internal/testutil/assertx"
	"smoke-tier1/internal/testutil/fixtures"
	"smoke-tier1/internal/testutil/httpx"
	"smoke-tier1/internal/testutil/mocks"
)

// TestHealthHandler exercises the production /health endpoint through
// the full middleware stack — exactly as a real HTTP client would hit
// it. Failure modes the test covers:
//
//   - happy path: ping succeeds → status="healthy", readiness=true,
//     HTTP 200, dependencies.database.connected=true
//   - critical dependency failure: ping errors → status="unhealthy",
//     readiness=false, HTTP 503, dependencies.database.connected=false,
//     error message surfaced verbatim
//   - the handler invokes the Pinger exactly once per request (no
//     accidental double-checks, no zero-checks)
//
// The pattern: spin up a *server.Server with mocked deps, wrap it in
// httpx.NewLiveServer (real httptest.Server over real socket), issue a
// real HTTP request, decode JSON straight into the typed
// fixtures.HealthResponse mirror (which carries the api-core/health
// JSON tags), assert on typed fields. Same shape every future handler
// test in this scenario should follow.
func TestHealthHandler(t *testing.T) {
	cases := []struct {
		name           string
		pingErr        error
		wantStatusCode int
		wantStatus     string
		wantReadiness  bool
		wantConnected  bool
		wantErrSubstr  string
	}{
		{
			name:           "ok",
			pingErr:        nil,
			wantStatusCode: http.StatusOK,
			wantStatus:     "healthy",
			wantReadiness:  true,
			wantConnected:  true,
		},
		{
			name:           "db_unreachable",
			pingErr:        errors.New("connection refused"),
			wantStatusCode: http.StatusServiceUnavailable,
			wantStatus:     "unhealthy",
			wantReadiness:  false,
			wantConnected:  false,
			wantErrSubstr:  "connection refused",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pinger := &mocks.FakePinger{PingErr: tc.pingErr}
			srv := server.New(server.Deps{
				Pinger:  pinger,
				Clock:   clock.System{},
				Logger:  log.New(discardWriter{}, "", 0),
				Service: "react-vite-test",
				Version: "1.0.0",
			})
			live := httpx.NewLiveServer(t, srv)

			resp, body := live.Do(t, http.MethodGet, "/health", nil)
			assertx.AssertStatus(t, resp, tc.wantStatusCode)

			got := assertx.MustDecodeJSON[fixtures.HealthResponse](t, body)
			require.Equal(t, tc.wantStatus, got.Status, "response.status")
			require.Equal(t, "react-vite-test", got.Service, "response.service")
			require.Equal(t, "1.0.0", got.Version, "response.version")
			require.Equal(t, tc.wantReadiness, got.Readiness, "response.readiness")

			dep, ok := got.Dependencies["database"]
			require.True(t, ok, "response.dependencies must include 'database'; got %v", got.Dependencies)
			require.Equal(t, tc.wantConnected, dep.Connected, "dependencies.database.connected")
			if tc.wantErrSubstr != "" {
				require.Contains(t, fmt.Sprint(dep.Error), tc.wantErrSubstr, "dependencies.database.error")
			}

			require.Equal(t, int64(1), pinger.Calls.Load(), "Pinger.PingContext call count")
		})
	}
}

// discardWriter swallows middleware log output so the test's stderr
// stays clean. The real log output is exercised by the middleware's
// own test (internal/middleware/logging_test.go).
type discardWriter struct{}

func (discardWriter) Write(p []byte) (int, error) { return len(p), nil }
