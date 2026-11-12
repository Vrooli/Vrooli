package healthapi_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/apihttptest"
	"github.com/vrooli/api-core/schedule"
	httpx "github.com/vrooli/api-core/servertest"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/research-desk/v1/shared"

	"research-desk/handlers/healthapi"
	"research-desk/internal/server"
)

// A missing research home degrades the desk but never makes it unready: the
// manual paths still work and the reason is reported (charter I5).
func TestHealthReportsTheResearchHome(t *testing.T) {
	cases := []struct {
		name       string
		homeErr    error
		wantStatus string
	}{
		{"readable", nil, "healthy"},
		{"missing", errors.New("research home /x/.vrooli/research does not exist"), "degraded"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mod := healthapi.Module(healthapi.Deps{
				ResearchHome: func(context.Context) error { return tc.homeErr },
				Service:      "research-desk-api",
				Version:      "test",
			})
			srv := server.New(server.Deps{Clock: schedule.System(), Logger: log.New(io.Discard, "", 0)}, mod)
			live := httpx.NewLiveServer(t, srv)

			for _, path := range []string{"/health", "/api/v1/health"} {
				resp, body := live.Do(t, http.MethodGet, path, nil)
				apihttptest.AssertStatus(t, resp, http.StatusOK)
				got := apihttptest.MustUnmarshalProto[sharedv1.Response](t, body)
				require.Equal(t, tc.wantStatus, got.Status, path)
				require.True(t, got.Readiness, path)
				dep, ok := got.Dependencies["research-home"]
				require.True(t, ok, "dependencies must include research-home; got %v", got.Dependencies)
				require.Equal(t, tc.homeErr == nil, dep.Connected)
			}
		})
	}
}

// F14: health says how automatic launch reaches Web Console.
func TestHealthSaysTheWebConsoleAccess(t *testing.T) {
	said := "Web Console access: local availability policy (no desk credential)"
	mod := healthapi.Module(healthapi.Deps{
		ResearchHome:     func(context.Context) error { return nil },
		WebConsoleAccess: func(context.Context) (bool, string) { return true, said },
		Service:          "research-desk-api",
		Version:          "test",
	})
	srv := server.New(server.Deps{Clock: schedule.System(), Logger: log.New(io.Discard, "", 0)}, mod)
	resp, body := httpx.NewLiveServer(t, srv).Do(t, http.MethodGet, "/health", nil)
	apihttptest.AssertStatus(t, resp, http.StatusOK)
	var got struct {
		Metrics map[string]any `json:"metrics"`
	}
	require.NoError(t, json.Unmarshal(body, &got))
	require.Equal(t, said, got.Metrics["web_console_access"])
}
