package main

import "testing"

func TestAppConstants(t *testing.T) {
	if appName != "workspace-sandbox" {
		t.Fatalf("appName = %q, want workspace-sandbox", appName)
	}
	if appVersion != "0.1.0" {
		t.Fatalf("appVersion = %q, want 0.1.0", appVersion)
	}
	if defaultAPIBase != "" {
		t.Fatalf("defaultAPIBase = %q, want empty", defaultAPIBase)
	}
}

func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Skipf("NewApp requires environment setup: %v", err)
	}
	if app == nil || app.core == nil {
		t.Fatalf("NewApp returned nil wiring")
	}
}

func TestWorkspaceSandboxCLIUsesDomainArchitecture(t *testing.T) {
	app := &App{}
	if groups := app.commandGroups(); len(groups) != 1 {
		t.Fatalf("commandGroups len = %d, want 1", len(groups))
	}
	if got := app.commandGroups()[0].Title; got != "Health" {
		t.Fatalf("commandGroups[0].Title = %q, want Health", got)
	}

	subcommands := map[string]bool{}
	for _, group := range app.subcommandGroups() {
		subcommands[group.Name] = true
	}

	for _, expected := range []string{"sandbox", "process", "change", "maintenance", "provenance"} {
		if !subcommands[expected] {
			t.Fatalf("missing subcommand group %q", expected)
		}
	}
}
