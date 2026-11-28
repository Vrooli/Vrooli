package main

import (
	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	validationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// TestNewAppConstructs is the smoke gate: NewApp() must succeed against
// the cli-core wiring declared in app.go. This catches the most common
// regression class — a misconfigured StandardScenarioOptions or a missing
// dependency from cli-core — before any tests touch real commands.
func TestNewAppConstructs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil || app.core == nil || app.core.CLI == nil {
		t.Fatal("NewApp() returned an incomplete app")
	}
}

// TestRunVersion exercises a non-API command path through cli-core.
// --version must succeed and must NOT trigger the NeedsAPI preflight
// (which would try to reach the configured API base and fail in CI).
func TestRunVersion(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--version"}); err != nil {
		t.Fatalf("app.Run(--version) error: %v", err)
	}
}

// TestRunHelp exercises cli-core's help renderer through the scenario
// app's wiring. Help is rendered to stdout by cli-core; we only verify
// Run returns without error. The presence of each registered command in
// the actual help surface is covered by cli-core's own tests.
func TestRunHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("app.Run(--help) error: %v", err)
	}
}

// TestMetadata pins the values app.go declares — appName must match the
// scenario id (post-substitution), appVersion must be non-empty.
// Catches accidental edits that decouple the binary identity from the
// scenario it belongs to.
func TestMetadata(t *testing.T) {
	if strings.TrimSpace(appName) == "" {
		t.Fatal("appName must not be empty")
	}
	if strings.TrimSpace(appVersion) == "" {
		t.Fatal("appVersion must not be empty")
	}
}

func TestSelectedCLIRefusesOldProviderWithoutFullSuiteFallback(t *testing.T) {
	var selectedCalls, ordinaryCalls atomic.Int32
	selectedPath, selectedHandler := validationconnect.NewSelectedValidationRunServiceHandler(validationconnect.UnimplementedSelectedValidationRunServiceHandler{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("request: %s", r.URL.Path)
		if strings.HasPrefix(r.URL.Path, selectedPath) {
			selectedCalls.Add(1)
			selectedHandler.ServeHTTP(w, r)
			return
		}
		if r.URL.Path == validationconnect.DurableValidationRunServiceStartValidationRunProcedure {
			ordinaryCalls.Add(1)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"healthy","readiness":true}`))
	}))
	defer server.Close()
	app, err := NewApp()
	require.NoError(t, err)
	err = app.Run([]string{"--api-base", server.URL, "validate", "start-selected", "personal-planner", "--case-paths", "bas/cases/calendar/duplicate-submit-once.json", "--idempotency-key", "old-provider", "--json"})
	require.Error(t, err)
	t.Logf("refusal: %v", err)
	require.Contains(t, err.Error(), connect.CodeUnimplemented.String())
	require.Equal(t, int32(1), selectedCalls.Load())
	require.Zero(t, ordinaryCalls.Load())
}
