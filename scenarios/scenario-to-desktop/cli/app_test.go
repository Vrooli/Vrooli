package main

import (
	"os"
	"path/filepath"
	"testing"

	"scenario-to-desktop/cli/domains"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
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

func TestAppRun_RegistersProductionCommandTree(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	if err := app.Run([]string{"--help"}); err != nil {
		t.Fatalf("Run(--help) error: %v", err)
	}
	if app.core == nil {
		t.Fatal("production command registration did not initialize app core")
	}
}

func TestPrimitiveEvidenceArtifact(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}
	manifest, err := os.ReadFile("manifest.json")
	if err != nil {
		t.Fatalf("read manifest: %v", err)
	}
	scenarioRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve scenario root: %v", err)
	}
	groups := domains.CommandGroups(app.dependencies())
	flatCommands := make([]cliapp.Command, 0)
	for _, group := range groups {
		flatCommands = append(flatCommands, group.Commands...)
	}
	subcommandGroups := append(domains.SubcommandGroups(app.dependencies()), cliapp.SubcommandGroup{Name: "desktop", Subcommands: flatCommands})
	cliapptest.RequirePrimitiveEvidence(t, cliapp.EvidenceArtifactPath(scenarioRoot), cliapp.EvidenceExportInput{
		Scenario:    appName,
		ManifestRaw: manifest,
		Groups:      subcommandGroups,
	}, os.Getenv("UPDATE_CLI_EVIDENCE") == "1")
}

func TestNewApp_AllCommandGroupsRegistered(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	groups := domains.CommandGroups(app.dependencies())
	// Domain groups exclude the standard status and configure commands, which are
	// handled by cli-core directly (status is disabled via IncludeStatusCommand).
	if len(groups) != 4 {
		t.Errorf("NewApp command callback returned %d groups, want 4", len(groups))
	}

	expectedTitles := map[string]bool{
		"Templates": false,
		"Records":   false,
		"Download":  false,
		"Scenarios": false,
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

	subgroups := domains.SubcommandGroups(app.dependencies())
	if len(subgroups) != 12 {
		t.Errorf("NewApp subcommand callback returned %d groups, want 12", len(subgroups))
	}

	expectedNames := map[string]int{
		"pipeline":      11,
		"preflight":     1,
		"build":         1,
		"bundle":        1,
		"docs":          1,
		"deploy-target": 6,
		"evidence":      4,
		"signing":       8,
		"state":         1,
		"tasks":         1,
		"telemetry":     6,
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

func TestNewApp_CommandsHaveExecutableHandlers(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() error: %v", err)
	}

	for _, group := range domains.CommandGroups(app.dependencies()) {
		for _, cmd := range group.Commands {
			if cmd.Run == nil && cmd.RunCtx == nil {
				t.Errorf("command %q in group %q has no executable handler", cmd.Name, group.Title)
			}
		}
	}

	for _, sg := range domains.SubcommandGroups(app.dependencies()) {
		for _, cmd := range sg.Subcommands {
			if cmd.Run == nil && cmd.RunCtx == nil {
				t.Errorf("subcommand %q in group %q has no executable handler", cmd.Name, sg.Name)
			}
		}
	}
}
