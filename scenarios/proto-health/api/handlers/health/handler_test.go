package health_test

import (
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"proto-health/handlers/health"
	"proto-health/internal/clock"
	"proto-health/internal/module"
	"proto-health/internal/server"
	"proto-health/internal/testutil/assertx"
	"proto-health/internal/testutil/httpx"
	"proto-health/internal/testutil/mocks"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/vrooli/packages/proto/descriptorimage"
	healthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/proto-health/v1/health"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/descriptorpb"

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
			h := health.NewHandler(health.Deps{
				Pinger:  pinger,
				Service: "react-vite-test",
				Version: "1.0.0",
			})
			mod := module.Module{
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
		})
	}
}

func TestHealthHandlerRefreshesDescriptorWithoutRestart(t *testing.T) {
	dir := t.TempDir()
	descriptorPath := filepath.Join(dir, "image.binpb")
	writeDescriptorForHealth(t, descriptorPath, "health-refresh/v1/first.proto")
	source, err := descriptorimage.New(descriptorimage.Config{DescriptorPath: descriptorPath})
	require.NoError(t, err)
	pinger := &mocks.FakePinger{}
	h := health.NewHandler(health.Deps{
		Pinger:           pinger,
		Service:          "proto-health-test",
		Version:          "1.0.0",
		DescriptorSource: source,
	})
	apiServer := server.New(server.Deps{Clock: clock.System{}, Logger: log.New(io.Discard, "", 0)}, module.Module{
		Name: "health",
		Mount: func(r *mux.Router) {
			r.HandleFunc("/health", h).Methods(http.MethodGet)
		},
	})
	live := httpx.NewLiveServer(t, apiServer)

	first, _ := live.Do(t, http.MethodGet, "/health", nil)
	firstDigest := first.Header.Get("X-Proto-Descriptor-Digest")
	firstGeneration := first.Header.Get("X-Proto-Descriptor-Generation")
	require.NotEmpty(t, firstDigest)
	require.Equal(t, "1", firstGeneration)

	stage := descriptorPath + ".stage"
	writeDescriptorForHealth(t, stage, "health-refresh/v1/second.proto")
	require.NoError(t, os.Rename(stage, descriptorPath))

	second, _ := live.Do(t, http.MethodGet, "/health", nil)
	require.NotEqual(t, firstDigest, second.Header.Get("X-Proto-Descriptor-Digest"))
	require.Equal(t, "2", second.Header.Get("X-Proto-Descriptor-Generation"))
}

func writeDescriptorForHealth(t *testing.T, path, name string) {
	t.Helper()
	data, err := proto.Marshal(&descriptorpb.FileDescriptorSet{File: []*descriptorpb.FileDescriptorProto{{
		Name: proto.String(name), Syntax: proto.String("proto3"),
	}}})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0o644))
}
