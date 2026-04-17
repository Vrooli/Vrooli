package main

import (
	"testing"
)

func TestNewApp_Initializes(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	if app == nil {
		t.Fatal("NewApp() returned nil")
	}
	if app.core == nil {
		t.Fatal("app.core is nil")
	}
}

func TestNewApp_AllCommandGroupsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	groups := app.registerCommands()
	// Domain groups exclude the standard status and configure commands, which are
	// handled by cli-core directly (status is disabled via IncludeStatusCommand).
	if len(groups) != 4 {
		t.Errorf("registerCommands() returned %d groups, want 4", len(groups))
	}

	expectedTitles := map[string]bool{
		"Templates":       false,
		"Desktop Records": false,
		"Download":        false,
		"Scenarios":       false,
	}
	for _, g := range groups {
		if _, ok := expectedTitles[g.Title]; ok {
			expectedTitles[g.Title] = true
		} else {
			t.Errorf("unexpected command group title: %q", g.Title)
		}
	}
	for title, found := range expectedTitles {
		if !found {
			t.Errorf("missing command group: %q", title)
		}
	}
}

func TestNewApp_AllSubcommandGroupsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	subgroups := app.registerSubcommandGroups()
	// Expect 6 subcommand groups: pipeline, bundle, telemetry, signing, deploy-target, wine
	if len(subgroups) != 6 {
		t.Errorf("registerSubcommandGroups() returned %d groups, want 6", len(subgroups))
	}

	expectedNames := map[string]int{
		"pipeline":      11,
		"bundle":        1,
		"telemetry":     6,
		"signing":       8,
		"deploy-target": 5,
		"wine":          3,
	}
	for _, sg := range subgroups {
		expectedCount, ok := expectedNames[sg.Name]
		if !ok {
			t.Errorf("unexpected subcommand group: %q", sg.Name)
			continue
		}
		if len(sg.Subcommands) != expectedCount {
			t.Errorf("subcommand group %q has %d commands, want %d", sg.Name, len(sg.Subcommands), expectedCount)
		}
		delete(expectedNames, sg.Name)
	}
	for name := range expectedNames {
		t.Errorf("missing subcommand group: %q", name)
	}
}

func TestNewApp_SubcommandHandlersNotNil(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	for _, group := range app.registerCommands() {
		for _, cmd := range group.Commands {
			if cmd.Run == nil {
				t.Errorf("command %q in group %q has nil Run handler", cmd.Name, group.Title)
			}
		}
	}

	for _, sg := range app.registerSubcommandGroups() {
		for _, cmd := range sg.Subcommands {
			if cmd.Run == nil {
				t.Errorf("subcommand %q in group %q has nil Run handler", cmd.Name, sg.Name)
			}
		}
	}
}
