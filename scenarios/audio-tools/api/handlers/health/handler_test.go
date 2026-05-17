package health_test

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"
	"time"

	"audio-tools/handlers/health"
	"audio-tools/internal/capabilities"
	capmocks "audio-tools/internal/capabilities/mocks"
	"audio-tools/internal/clock"
	"audio-tools/internal/modulekit"
	"audio-tools/internal/server"
	"audio-tools/internal/testutil/assertx"
	"audio-tools/internal/testutil/httpx"
	"audio-tools/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/audio-tools/v1/health"

	"github.com/gorilla/mux"
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
// real HTTP request, decode JSON straight into the generated proto
// type via assertx.MustUnmarshalProto, assert on typed fields. Same
// shape every future handler test in this scenario should follow —
// when the endpoint's wire shape lives in packages/proto/, decode
// through protojson; when it doesn't yet, MustDecodeJSON is the
// fallback (but adding the proto first is the right move).
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
			reg := capabilities.NewRegistry(
				[]capabilities.Def{{ID: "whisper-stt", DependencyKind: capabilities.DependencyResource, DependencySlug: "whisper"}},
				map[string]capabilities.Checker{
					"whisper-stt": capmocks.NewFakeChecker(capabilities.StatusAvailable, "ok"),
				},
				time.Minute,
			)
			h := health.NewHandler(health.Deps{
				Pinger:   pinger,
				Registry: reg,
				Service:  "react-vite-test",
				Version:  "1.0.0",
			})
			mod := modulekit.Module{
				Name: "health",
				Mount: func(r *mux.Router) {
					r.HandleFunc("/health", h).Methods(http.MethodGet)
				},
			}
			srv := server.New(
				server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)},
				mod,
			)
			live := httpx.NewLiveServer(t, srv)

			resp, body := live.Do(t, http.MethodGet, "/health", nil)
			assertx.AssertStatus(t, resp, tc.wantStatusCode)

			got := assertx.MustUnmarshalProto[healthv1.Response](t, body)
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

			// Providers dep must always appear when a Registry is wired,
			// independent of the database state. It is non-Critical: an
			// unavailable provider must NOT flip readiness.
			providers, ok := got.Dependencies["providers"]
			require.True(t, ok, "response.dependencies must include 'providers'; got %v", got.Dependencies)
			require.True(t, providers.Connected, "providers dep should be connected when registry has no unavailable providers")
		})
	}
}

// TestHealthHandler_ProvidersDownDoesNotFlipReadiness asserts that an
// unavailable provider degrades dependencies["providers"] but does NOT
// flip readiness (the providers dep is registered as non-Critical).
func TestHealthHandler_ProvidersDownDoesNotFlipReadiness(t *testing.T) {
	pinger := &mocks.FakePinger{}
	reg := capabilities.NewRegistry(
		[]capabilities.Def{
			{ID: "whisper-stt", DependencyKind: capabilities.DependencyResource, DependencySlug: "whisper"},
			{ID: "audio-tools", DependencyKind: capabilities.DependencyScenario, DependencySlug: "audio-tools"},
		},
		map[string]capabilities.Checker{
			"whisper-stt": capmocks.NewFakeChecker(capabilities.StatusUnavailable, "down"),
		},
		time.Minute,
	)
	h := health.NewHandler(health.Deps{Pinger: pinger, Registry: reg, Service: "react-vite-test", Version: "1.0.0"})
	mod := modulekit.Module{
		Name:  "health",
		Mount: func(r *mux.Router) { r.HandleFunc("/health", h).Methods(http.MethodGet) },
	}
	srv := server.New(
		server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)},
		mod,
	)
	live := httpx.NewLiveServer(t, srv)

	resp, body := live.Do(t, http.MethodGet, "/health", nil)
	assertx.AssertStatus(t, resp, http.StatusOK)

	got := assertx.MustUnmarshalProto[healthv1.Response](t, body)
	require.True(t, got.Readiness, "providers being down must NOT flip readiness (non-Critical)")
	providers, ok := got.Dependencies["providers"]
	require.True(t, ok)
	require.False(t, providers.Connected)
	require.Contains(t, fmt.Sprint(providers.Error), "whisper-stt")
}
