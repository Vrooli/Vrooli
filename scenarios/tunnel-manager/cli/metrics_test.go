package main

import (
	"testing"
)

// [REQ:CLI-METRICS-001] Metrics commands are registered
func TestMetricsCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.registerCommands()
	wantCmds := map[string]bool{
		"metrics latest":  false,
		"metrics history": false,
	}

	for _, g := range groups {
		for _, cmd := range g.Commands {
			if _, ok := wantCmds[cmd.Name]; ok {
				wantCmds[cmd.Name] = true
				if !cmd.NeedsAPI {
					t.Errorf("%s command should require API", cmd.Name)
				}
			}
		}
	}

	for name, found := range wantCmds {
		if !found {
			t.Errorf("command %q not registered", name)
		}
	}
}

// [REQ:CLI-METRICS-002] Metrics API paths
func TestMetricsAPIPaths(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	tests := []struct {
		input string
		want  string
	}{
		{"/metrics/latest", "/api/v1/metrics/latest"},
		{"/metrics/history?hours=24", "/api/v1/metrics/history?hours=24"},
	}

	for _, tc := range tests {
		got := app.apiPath(tc.input)
		if got != tc.want {
			t.Errorf("apiPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// [REQ:CLI-METRICS-003] Metrics commands support JSON output
func TestMetricsJSONFlag(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	if !app.useJSON([]string{"--json"}) {
		t.Error("metrics --json should be detected")
	}
	if !app.useJSON([]string{"-j"}) {
		t.Error("metrics -j should be detected")
	}
	if app.useJSON([]string{}) {
		t.Error("metrics without --json should not detect JSON mode")
	}
}
