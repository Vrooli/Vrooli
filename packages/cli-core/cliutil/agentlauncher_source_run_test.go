package cliutil

import (
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const (
	attachedFixtureRunID = "4f1c2a8e-0b6d-4c1e-9a7f-1d2e3f405162"
	governedFixtureRunID = "7a9b8c6d-5e4f-4a3b-8c2d-1e0f9a8b7c6d"
	staleFixtureRunID    = "00000000-1111-4222-8333-444455556666"
)

// The child learns, as a non-secret locator, which Agent Manager run its
// provider usage is accounted to: the run attach opened, or the run of the
// identity a governed launch already holds (which skips attach). A launch that
// cannot name its run drops any inherited locator rather than passing on
// another caller's run.
func TestLaunchExportsTheSourceRunOfAttachedAndGovernedSessions(t *testing.T) {
	launchCodex := func(t *testing.T, request AgentLaunchRequest) string {
		t.Helper()
		var got string
		children := 0
		request.Agent = "codex"
		request.LookPath = func(string) (string, error) { return "/disposable/codex", nil }
		request.RunChild = func(_ context.Context, _ string, _ []string, env []string, _ io.Reader, _, _ io.Writer) error {
			children++
			got = environmentValue(env, SourceRunIDEnv)
			return nil
		}
		if _, err := LaunchCodingAgentResult(context.Background(), request); err != nil || children != 1 {
			t.Fatalf("launch err=%v children=%d", err, children)
		}
		return got
	}
	attachServer := func(t *testing.T, status int, runID string) *httptest.Server {
		t.Helper()
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch filepath.Base(r.URL.Path) {
			case "attach":
				if status != http.StatusCreated {
					w.WriteHeader(status)
					return
				}
				w.WriteHeader(status)
				_ = json.NewEncoder(w).Encode(map[string]any{"run": map[string]string{"id": runID}, "identity_token": credentialRoleTestTokenForRun(runID, nil)})
			case "detach":
				w.WriteHeader(http.StatusOK)
			default:
				http.NotFound(w, r)
			}
		}))
		t.Cleanup(server.Close)
		return server
	}
	stale := []string{"PATH=/safe", SourceRunIDEnv + "=" + staleFixtureRunID}

	t.Run("attached", func(t *testing.T) {
		isolateLauncherModelPolicy(t)
		t.Setenv("TMPDIR", t.TempDir())
		t.Setenv(AgentSessionEnv, "")
		server := attachServer(t, http.StatusCreated, attachedFixtureRunID)
		if got := launchCodex(t, AgentLaunchRequest{APIBase: server.URL, Environment: stale}); got != attachedFixtureRunID {
			t.Fatalf("attached child source run = %q, want %s", got, attachedFixtureRunID)
		}
	})

	t.Run("attach unavailable", func(t *testing.T) {
		isolateLauncherModelPolicy(t)
		t.Setenv("TMPDIR", t.TempDir())
		t.Setenv(AgentSessionEnv, "")
		server := attachServer(t, http.StatusServiceUnavailable, "")
		if got := launchCodex(t, AgentLaunchRequest{APIBase: server.URL, Environment: stale}); got != "" {
			t.Fatalf("unattributed child inherited source run %q", got)
		}
	})

	t.Run("governed", func(t *testing.T) {
		namespace, err := os.MkdirTemp("", "governed-source-run-")
		if err != nil {
			t.Fatal(err)
		}
		defer os.RemoveAll(namespace)
		t.Setenv("TMPDIR", namespace)
		t.Setenv(AgentSessionEnv, "")
		held := credentialRoleTestTokenForRun(governedFixtureRunID, nil)
		t.Setenv(EnvIdentityToken, held)
		socket := AgentCredentialSocket()
		if err := os.MkdirAll(filepath.Dir(socket), 0o700); err != nil {
			t.Fatal(err)
		}
		listener, err := net.Listen("unix", socket)
		if err != nil {
			t.Fatal(err)
		}
		broker := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]string{"token": held})
		})}
		go broker.Serve(listener)
		defer broker.Close()
		attaches := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			attaches++
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()
		got := launchCodex(t, AgentLaunchRequest{RequireNativeIdentity: true, APIBase: server.URL, Environment: stale})
		if attaches != 0 {
			t.Fatalf("governed launch attached %d time(s)", attaches)
		}
		if got != governedFixtureRunID {
			t.Fatalf("governed child source run = %q, want the held identity's run %s", got, governedFixtureRunID)
		}
	})
}
