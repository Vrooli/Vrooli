package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
)

func TestExecuteAcceptsPositionalPhases(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/executions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		if !bytes.Contains(body, []byte(`"phases":["unit","integration"]`)) {
			t.Fatalf("expected phases in payload, got: %s", string(body))
		}
		fmt.Fprintf(w, `{"success":true,"phases":[{"name":"unit","status":"passed","durationSeconds":1}],"executionId":"abc"}`)
	}))
	defer server.Close()

	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)

	if err := app.Run([]string{"execute", "demo", "unit", "integration"}); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
}

func TestExecuteAllPhaseSkipsExplicitList(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/executions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		if bytes.Contains(body, []byte(`"phases"`)) {
			t.Fatalf("expected phases to be omitted when 'all' requested, got: %s", string(body))
		}
		fmt.Fprintf(w, `{"success":true,"phases":[],"executionId":"abc"}`)
	}))
	defer server.Close()

	t.Setenv("TEST_GENIE_API_BASE", server.URL)
	app := newTestApp(t)

	if err := app.Run([]string{"execute", "demo", "all"}); err != nil {
		t.Fatalf("execute failed: %v", err)
	}
}

func TestConfigureSetsValues(t *testing.T) {
	app := newTestApp(t)
	if err := app.Run([]string{"configure", "api_base", "http://example.com"}); err != nil {
		t.Fatalf("configure api_base: %v", err)
	}
	if app.core.Config.APIBase != "http://example.com" {
		t.Fatalf("expected api base saved")
	}
	if err := app.Run([]string{"configure", "token", "secret"}); err != nil {
		t.Fatalf("configure token: %v", err)
	}
	if app.core.Config.Token != "secret" {
		t.Fatalf("expected token saved")
	}
}

func TestBuildAPIBaseOptionsUsesPortEnv(t *testing.T) {
	app := newTestApp(t)
	t.Setenv("API_PORT", "4567")
	base := cliutil.DetermineAPIBase(app.core.APIBaseOptions())
	if base != "http://localhost:4567" {
		t.Fatalf("expected base from port env, got %s", base)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	temp := t.TempDir()
	t.Setenv("HOME", temp)
	t.Setenv("TEST_GENIE_CONFIG_DIR", temp)
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}
