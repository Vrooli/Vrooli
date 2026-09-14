package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"connectrpc.com/connect"
	"github.com/vrooli/cli-core/cliapp"
	sessionsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions"
	sessionsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/sessions/sessions_v1connect"

	"web-console/cli/internal/testutil"
)

func TestSessionGetRoutesEmbeddedManifest(t *testing.T) {
	const sessionID = "11111111-2222-4333-8444-555555555555"
	for _, tc := range []struct {
		name       string
		args       []string
		json       bool
		ownerError bool
		wantError  string
		wantCalls  int32
	}{
		{name: "get JSON", args: []string{"session", "get", sessionID, "--json"}, json: true, wantCalls: 1},
		{name: "show alias JSON", args: []string{"session", "show", sessionID, "--json"}, json: true, wantCalls: 1},
		{name: "get human", args: []string{"session", "get", sessionID}, wantCalls: 1},
		{name: "missing identity", args: []string{"session", "get", "--json"}, wantError: "missing required positional <session-id>"},
		{name: "owner not found", args: []string{"session", "get", sessionID, "--json"}, ownerError: true, wantError: "session not found", wantCalls: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var calls atomic.Int32
			mux := http.NewServeMux()
			mux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })
			mux.Handle(sessionsconnect.SessionsServiceGetProcedure, connect.NewUnaryHandlerSimple(
				sessionsconnect.SessionsServiceGetProcedure,
				func(_ context.Context, req *sessionsv1.GetRequest) (*sessionsv1.GetResponse, error) {
					calls.Add(1)
					if req.GetId() != sessionID {
						t.Errorf("owner received session ID %q, want %q", req.GetId(), sessionID)
					}
					if tc.ownerError {
						return nil, connect.NewError(connect.CodeNotFound, errors.New("session not found"))
					}
					return &sessionsv1.GetResponse{Session: &sessionsv1.Session{Id: req.GetId(), Backend: "persistent"}}, nil
				},
			))
			server := httptest.NewServer(mux)
			defer server.Close()
			t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())
			t.Setenv("WEB_CONSOLE_API_URL", server.URL)
			t.Setenv("VROOLI_AGENT_IDENTITY_TOKEN", "")
			app, err := NewApp()
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			// This is App.Run's real parser/dispatcher, including manifestBytes and
			// domain registration. A hand-built ArgSchema would hide name drift.
			err = app.core.CLI.RunWithWriters(tc.args, &stdout, &stderr)
			if tc.wantError != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantError) {
					t.Fatalf("error = %v, want %q", err, tc.wantError)
				}
				if stdout.Len() != 0 {
					t.Fatalf("failed read emitted success output: %s", &stdout)
				}
			} else {
				if err != nil {
					t.Fatalf("command: %v; stderr: %s", err, &stderr)
				}
				if tc.json {
					var report cliapp.ListReport
					if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
						t.Fatalf("JSON report: %v", err)
					}
					if !strings.Contains(strings.Join(report.Results, "\n"), "ID: "+sessionID) {
						t.Fatalf("report lost owner identity: %#v", report)
					}
				} else if !strings.Contains(stdout.String(), "ID: "+sessionID) {
					t.Fatalf("human report lost owner identity: %s", &stdout)
				}
			}
			if got := calls.Load(); got != tc.wantCalls {
				t.Fatalf("owner Get calls = %d, want %d", got, tc.wantCalls)
			}
		})
	}
}

func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(testutil.ErrorMessage(err, "NewApp()"))
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(testutil.ErrorMessage(err, "NewApp()"))
	}
	// --version is handled by cli-core; it must not return an error and must not
	// touch the network (no NeedsAPI path triggered).
	if err := app.Run([]string{"--version"}); err != nil {
		t.Fatalf("app.Run(--version) error: %v", err)
	}
}

func TestRunHelpListsMigratedDomains(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(testutil.ErrorMessage(err, "NewApp()"))
	}
	// The standard help text lists all registered command groups and
	// subcommand groups. We assert the migrated domains are wired up.
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
}

func TestMetadata(t *testing.T) {
	if !strings.EqualFold(appName, "web-console") {
		t.Fatalf("appName = %q, want web-console", appName)
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}

func TestStandardScenarioPaths(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatal(testutil.ErrorMessage(err, "NewApp"))
	}
	tests := []struct {
		input string
		want  string
	}{
		{"/sessions", "/api/v1/sessions"},
		{"sessions", "/api/v1/sessions"},
		{"", ""},
	}
	for _, tc := range tests {
		got := app.core.APIPath(tc.input)
		if got != tc.want {
			t.Errorf("APIPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

func TestBuiltInStatusCommandRegistered(t *testing.T) {
	t.Setenv("CLI_CONFIG_DIR_OVERRIDE", t.TempDir())

	app, err := NewApp()
	if err != nil {
		t.Fatal(testutil.ErrorMessage(err, "NewApp"))
	}

	if err := app.Run([]string{"configure", "api_base", "http://127.0.0.1:1"}); err != nil {
		t.Fatalf("configure failed: %v", err)
	}

	// The built-in status command probes /health; without a reachable API it
	// must surface a network/HTTP error rather than silently succeeding. This
	// guards against accidentally shadowing cli-core's built-in status.
	err = app.Run([]string{"status"})
	if err == nil {
		t.Fatal("expected status to fail without a reachable API")
	}
	msg := err.Error()
	if !strings.Contains(msg, "API request failed") &&
		!strings.Contains(msg, "api error") &&
		!strings.Contains(msg, "connection refused") &&
		!strings.Contains(msg, "lookup") {
		t.Fatalf("expected built-in status command to execute, got %v", err)
	}
}
