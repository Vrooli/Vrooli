package main

import (
	"encoding/json"
	"os"
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

// TestMetadata pins the binary identity: appName is the scenario id and the
// version is service.version from .vrooli/service.json, its one source.
func TestMetadata(t *testing.T) {
	if appName != "research-desk" {
		t.Fatalf("appName = %q", appName)
	}
	data, err := os.ReadFile("../.vrooli/service.json")
	if err != nil {
		t.Fatal(err)
	}
	if want := serviceVersion(t, data); version(".") != want || want == "" {
		t.Fatalf("version = %q, service.json says %q", version("."), want)
	}
	if version(t.TempDir()) != "unknown" {
		t.Fatal("an unreadable service.json is unknown")
	}
}

func serviceVersion(t *testing.T, data []byte) string {
	t.Helper()
	var m struct {
		Service struct{ Version string } `json:"service"`
	}
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	return m.Service.Version
}
