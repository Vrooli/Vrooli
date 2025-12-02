package main

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliutil"
	"scenario-completeness-scoring/cli/format"
)

func TestHelpCommand(t *testing.T) {
	app := newTestApp(t)
	output := captureStdout(t, func() {
		if err := app.Run([]string{"help"}); err != nil {
			t.Fatalf("help command failed: %v", err)
		}
	})
	if !strings.Contains(output, "Usage:") {
		t.Fatalf("expected help output to contain Usage, got: %s", output)
	}
	if !strings.Contains(output, "Commands:") {
		t.Fatalf("expected help output to list commands, got: %s", output)
	}
}

func TestVersionCommand(t *testing.T) {
	app := newTestApp(t)
	output := captureStdout(t, func() {
		if err := app.Run([]string{"version"}); err != nil {
			t.Fatalf("version command failed: %v", err)
		}
	})
	if !strings.Contains(strings.ToLower(output), "version") {
		t.Fatalf("expected version output, got: %s", output)
	}
}

func TestConfigureCommand(t *testing.T) {
	app := newTestApp(t)
	apiBase := "http://test.example.com"

	if err := app.Run([]string{"configure", "api_base", apiBase}); err != nil {
		t.Fatalf("configure set failed: %v", err)
	}

	output := captureStdout(t, func() {
		if err := app.Run([]string{"configure"}); err != nil {
			t.Fatalf("configure get failed: %v", err)
		}
	})
	if !strings.Contains(output, apiBase) {
		t.Fatalf("expected configured api_base to be printed, got: %s", output)
	}
}

func TestUnknownCommand(t *testing.T) {
	app := newTestApp(t)
	err := app.Run([]string{"invalid_command"})
	if err == nil {
		t.Fatalf("expected error for unknown command")
	}
	if !strings.Contains(err.Error(), "Unknown command") {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestConfigDirectoryCreated(t *testing.T) {
	app := newTestApp(t)
	dir := filepath.Dir(app.core.ConfigFile.Path)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("expected config directory to exist: %v", err)
	}
}

func TestGlobalFlagApiBaseAcceptedAnywhere(t *testing.T) {
	app := newTestApp(t)
	if err := app.Run([]string{"--api-base", "http://example.com", "help"}); err != nil {
		t.Fatalf("run with api-base: %v", err)
	}
	if app.core.APIOverride != "http://example.com" {
		t.Fatalf("expected apiOverride to be set, got %q", app.core.APIOverride)
	}
}

func TestGlobalFlagApiBaseMissingValue(t *testing.T) {
	app := newTestApp(t)
	err := app.Run([]string{"--api-base"})
	if err == nil || !strings.Contains(err.Error(), "missing value") {
		t.Fatalf("expected missing value error, got %v", err)
	}
}

func TestBuildAPIBaseOptionsUsesPortEnv(t *testing.T) {
	app := newTestApp(t)
	t.Setenv("API_PORT", "4321")
	base := cliutil.DetermineAPIBase(app.core.APIBaseOptions())
	if base != "http://localhost:4321" {
		t.Fatalf("expected API base from port env, got %s", base)
	}
}

func TestNoColorFlagDisablesColor(t *testing.T) {
	format.SetColorEnabled(true)
	t.Cleanup(func() { format.SetColorEnabled(true) })
	app := newTestApp(t)
	if err := app.Run([]string{"--no-color", "help"}); err != nil {
		t.Fatalf("help with no-color failed: %v", err)
	}
	if format.ColorEnabled() {
		t.Fatalf("expected colors to be disabled after --no-color")
	}
}

func TestScoreCommandRunsStaleCheckViaDispatcher(t *testing.T) {
	format.SetColorEnabled(false)
	t.Cleanup(func() { format.SetColorEnabled(true) })
	app := newTestApp(t)
	called := false
	app.core.CLI.SetStaleChecker(&cliutil.StaleChecker{
		BuildFingerprint: "fp",
		BuildSourceRoot:  t.TempDir(),
		FingerprintFunc: func(root string, skip ...string) (string, error) {
			called = true
			return "fp", nil
		},
		LookPathFunc: func(file string) (string, error) {
			return "/usr/bin/go", nil
		},
	})

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"scenario":"demo","category":"","score":0,"base_score":0,"validation_penalty":0,"classification":"","breakdown":{},"metrics":{},"validation_analysis":{"has_issues":false,"issue_count":0,"issues":[],"total_penalty":0},"recommendations":[],"partial_result":{},"calculated_at":""}`)
	}))
	defer server.Close()
	t.Setenv("SCENARIO_COMPLETENESS_SCORING_API_BASE", server.URL)

	if err := app.Run([]string{"score", "demo"}); err != nil {
		t.Fatalf("score command failed: %v", err)
	}
	if !called {
		t.Fatalf("expected stale checker to run for score command")
	}
}

func TestHistoryCommandSendsQuery(t *testing.T) {
	app := newTestApp(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/scores/demo/history" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "5" {
			t.Fatalf("expected limit query param, got %s", r.URL.RawQuery)
		}
		fmt.Fprint(w, `{"items":[]}`)
	}))
	defer server.Close()
	t.Setenv("SCENARIO_COMPLETENESS_SCORING_API_BASE", server.URL)

	if err := app.Run([]string{"history", "--limit", "5", "demo"}); err != nil {
		t.Fatalf("history command failed: %v", err)
	}
}

func TestWhatIfCommandSendsPayload(t *testing.T) {
	app := newTestApp(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		defer r.Body.Close()
		if r.URL.Path != "/api/v1/scores/demo/what-if" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if !bytes.Contains(body, []byte(`"changes"`)) {
			t.Fatalf("expected changes payload, got %s", string(body))
		}
		fmt.Fprint(w, `{"ok":true}`)
	}))
	defer server.Close()
	t.Setenv("SCENARIO_COMPLETENESS_SCORING_API_BASE", server.URL)

	if err := app.Run([]string{"what-if", "demo"}); err != nil {
		t.Fatalf("what-if command failed: %v", err)
	}
}

func TestPresetApplyCommand(t *testing.T) {
	app := newTestApp(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/config/presets/default/apply" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"applied":"default"}`)
	}))
	defer server.Close()
	t.Setenv("SCENARIO_COMPLETENESS_SCORING_API_BASE", server.URL)

	if err := app.Run([]string{"preset", "apply", "default"}); err != nil {
		t.Fatalf("preset apply failed: %v", err)
	}
}

func TestRecommendCommand(t *testing.T) {
	app := newTestApp(t)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/recommendations/demo" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		fmt.Fprint(w, `{"recommendations":[]}`)
	}))
	defer server.Close()
	t.Setenv("SCENARIO_COMPLETENESS_SCORING_API_BASE", server.URL)

	if err := app.Run([]string{"recommend", "demo"}); err != nil {
		t.Fatalf("recommend failed: %v", err)
	}
}

func newTestApp(t *testing.T) *App {
	t.Helper()
	tempHome := t.TempDir()
	t.Setenv("HOME", tempHome)
	t.Setenv("SCENARIO_COMPLETENESS_SCORING_API_TOKEN", "test-token")
	app, err := NewApp()
	if err != nil {
		t.Fatalf("new app: %v", err)
	}
	return app
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	original := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w

	fn()

	w.Close()
	os.Stdout = original

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	return buf.String()
}
