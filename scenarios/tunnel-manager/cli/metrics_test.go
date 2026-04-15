package main

import (
	"testing"

	"tunnel-manager/cli/internal/flags"
)

// [REQ:CLI-METRICS-001] Metrics commands are registered
func TestMetricsCommandsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp: %v", err)
	}

	groups := app.subcommandGroups()
	wantCmds := map[string]bool{"latest": false, "history": false}

	for _, group := range groups {
		if group.Name != "metrics" {
			continue
		}
		for _, cmd := range group.Subcommands {
			if _, ok := wantCmds[cmd.Name]; ok {
				wantCmds[cmd.Name] = true
				if !group.NeedsAPI && !cmd.NeedsAPI {
					t.Errorf("metrics %s should require API", cmd.Name)
				}
			}
		}
	}

	for name, found := range wantCmds {
		if !found {
			t.Errorf("subcommand metrics %q not registered", name)
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
		got := app.core.APIPath(tc.input)
		if got != tc.want {
			t.Errorf("apiPath(%q) = %q, want %q", tc.input, got, tc.want)
		}
	}
}

// [REQ:CLI-METRICS-003] Metrics commands support JSON output
func TestMetricsJSONFlag(t *testing.T) {
	if !flags.HasJSONOutput([]string{"--json"}) {
		t.Error("metrics --json should be detected")
	}
	if !flags.HasJSONOutput([]string{"-j"}) {
		t.Error("metrics -j should be detected")
	}
	if flags.HasJSONOutput([]string{}) {
		t.Error("metrics without --json should not detect JSON mode")
	}
}
