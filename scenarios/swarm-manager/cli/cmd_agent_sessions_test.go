package main

import (
	"net/http"
	"strings"
	"testing"

	clitest "swarm-manager/cli/internal/testutil"
)

func TestCmdSessionsDeleteRequiresConfirmation(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	if err := app.cmdSessionsDelete([]string{"--id", "sess_1"}); err == nil {
		t.Fatal("cmdSessionsDelete without --yes error = nil, want confirmation error")
	}
}

func TestCmdSessionsDeleteCallsDeleteEndpoint(t *testing.T) {
	var method, path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"session_id":"sess_1"}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	out := clitest.CaptureStdout(t, func() error {
		return app.cmdSessionsDelete([]string{"--id", "sess_1", "--yes"})
	})

	if method != http.MethodDelete || path != "/api/v1/agent-sessions/sess_1" {
		t.Fatalf("request = %s %s, want DELETE /api/v1/agent-sessions/sess_1", method, path)
	}
	if !containsAll(out, "Session sess_1 deleted.", "were preserved") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestCmdSessionsGetRefreshCallsRefreshEndpoint(t *testing.T) {
	var method, path string
	clitest.NewAPIServer(t, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		_, _ = w.Write([]byte(`{"session":{"id":"sess_1","title":"Plan","kind":"meta_orchestration","status":"failed","run_id":"run-1","created_at":"2026-05-06T00:00:00Z","updated_at":"2026-05-06T00:01:00Z"}}`))
	}))
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	out := clitest.CaptureStdout(t, func() error {
		return app.cmdSessionsGet([]string{"--id", "sess_1", "--refresh"})
	})

	if method != http.MethodPost || path != "/api/v1/agent-sessions/sess_1/refresh" {
		t.Fatalf("request = %s %s, want POST /api/v1/agent-sessions/sess_1/refresh", method, path)
	}
	if !containsAll(out, "Plan (failed)", "Run ID: run-1") {
		t.Fatalf("unexpected output: %s", out)
	}
}

func TestSessionsCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}
	for _, args := range [][]string{
		{"sessions", "list"},
		{"sessions", "get"},
		{"sessions", "delete"},
	} {
		_ = app.Run(args)
	}
}

func containsAll(value string, needles ...string) bool {
	for _, needle := range needles {
		if !strings.Contains(value, needle) {
			return false
		}
	}
	return true
}
