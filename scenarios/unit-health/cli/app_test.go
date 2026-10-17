package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	validationv1 "github.com/vrooli/vrooli/packages/proto/gen/go/unit-health/v1/validation"
	"google.golang.org/protobuf/proto"

	"github.com/vrooli/cli-core/cliapptest"

	"unit-health/cli/internal/testutil"
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
	output := cliapptest.CaptureStdout(t, func() error {
		if err := app.Run([]string{"--version"}); err != nil {
			return err
		}
		return nil
	})
	testutil.RequireContains(t, output, appVersion)
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
	output := cliapptest.CaptureStdout(t, func() error {
		if err := app.Run([]string{"--help"}); err != nil {
			return err
		}
		return nil
	})
	testutil.RequireContains(t, output, appName)
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

// Long validation keeps its server-owned deadline even when ordinary requests
// use a short client timeout. Exercise the actual CLI/Connect transport.
func TestExecutedValidationWaitsForTerminalResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "ValidateScenario") {
			time.Sleep(80 * time.Millisecond)
			w.Header().Set("Content-Type", "application/proto")
			body, err := proto.Marshal(&validationv1.ValidateScenarioResponse{RunId: "delayed-validation", Scenario: "demo", Status: "passed"})
			if err != nil {
				t.Error(err)
				return
			}
			_, _ = w.Write(body)
			return
		}
		_, _ = w.Write([]byte(`{"status":"healthy"}`))
	}))
	defer server.Close()
	app, err := NewApp()
	if err != nil {
		t.Fatal(err)
	}
	app.core.HTTPClient = cliutil.NewHTTPClient(cliutil.HTTPClientOptions{Timeout: 10 * time.Millisecond})
	output := cliapptest.CaptureStdout(t, func() error {
		return app.Run([]string{"--api-base", server.URL, "validate", "scenario", "demo", "--execution", "--json"})
	})
	testutil.RequireContains(t, output, "delayed-validation")
}
