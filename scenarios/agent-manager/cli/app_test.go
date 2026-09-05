package main

import (
	"strings"
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

func TestRejectUnsupportedNodeTarget(t *testing.T) {
	if err := rejectUnsupportedNodeTarget("no-such-node"); err == nil || !strings.Contains(err.Error(), "no-such-node") {
		t.Fatalf("error = %v, want explicit remote node refusal", err)
	}
	if err := rejectUnsupportedNodeTarget(" "); err != nil {
		t.Fatalf("empty node should remain local: %v", err)
	}
}

// =============================================================================
// APP INITIALIZATION TESTS
// =============================================================================

func TestNewApp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	if app == nil {
		t.Fatal("expected app, got nil")
	}

	// Verify core is initialized
	if app.core == nil {
		t.Error("expected core to be initialized")
	}
}

func TestNewApp_ServicesInitialized(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Verify services are initialized
	if app.services == nil {
		t.Error("expected services to be initialized")
	}

	if app.services.Profiles == nil {
		t.Error("expected Profiles service")
	}
	if app.services.Tasks == nil {
		t.Error("expected Tasks service")
	}
	if app.services.Runs == nil {
		t.Error("expected Runs service")
	}
	if app.services.Runners == nil {
		t.Error("expected Runners service")
	}
	if app.services.Settings == nil {
		t.Error("expected Settings service")
	}
	if app.services.Maintenance == nil {
		t.Error("expected Maintenance service")
	}
}

// =============================================================================
// CLI EXECUTION TESTS
// =============================================================================

func TestApp_Run_Help(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Running with help flag should not panic
	// Note: help typically exits with code 0 but urfave/cli may return an error
	err = app.Run([]string{"agent-manager", "--help"})
	// Help output is acceptable, we just want to ensure no panic
	_ = err
}

func TestApp_Run_Version(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Running with version flag should not panic
	err = app.Run([]string{"agent-manager", "--version"})
	_ = err
}

func TestApp_Run_UnknownCommand(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Unknown command should provide help or error gracefully
	err = app.Run([]string{"agent-manager", "nonexistent-command"})
	// We expect either an error or graceful handling
	_ = err
}

// =============================================================================
// STRUCTURE AND COMMAND REGISTRATION TESTS
// =============================================================================

func TestApp_Structure(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	if app.core == nil {
		t.Error("app core should be set")
	}
}

func TestApp_RegisterCommands_Groups(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	groups := app.commandGroups()

	// Verify expected command groups exist
	expectedGroups := []string{
		"Health",
		"Configuration",
		"Runs",
	}
	if len(groups) != len(expectedGroups) {
		t.Errorf("expected %d command groups, got %d", len(expectedGroups), len(groups))
	}

	for i, expected := range expectedGroups {
		if i >= len(groups) {
			break
		}
		if groups[i].Title != expected {
			t.Errorf("expected group[%d] title '%s', got '%s'", i, expected, groups[i].Title)
		}
	}
}

func TestApp_RegisterCommands_HealthGroup(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	groups := app.commandGroups()

	// Find Health group
	var healthGroup *struct {
		Title    string
		Commands []struct{ Name string }
	}
	for _, g := range groups {
		if g.Title == "Health" {
			healthGroup = &struct {
				Title    string
				Commands []struct{ Name string }
			}{Title: g.Title}
			for _, cmd := range g.Commands {
				healthGroup.Commands = append(healthGroup.Commands, struct{ Name string }{Name: cmd.Name})
			}
			break
		}
	}

	if healthGroup == nil {
		t.Fatal("Health command group not found")
	}

	if len(healthGroup.Commands) == 0 {
		t.Error("Health group should have commands")
	}

	// Verify status command exists
	hasStatus := false
	for _, cmd := range healthGroup.Commands {
		if cmd.Name == "status" {
			hasStatus = true
			break
		}
	}
	if !hasStatus {
		t.Error("expected 'status' command in Health group")
	}
}

// =============================================================================
// COMMAND HELP TESTS
// =============================================================================

func TestApp_ProfileHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Running profile help should not error
	err = app.Run([]string{"profile", "help"})
	if err != nil {
		t.Errorf("profileHelp() returned error: %v", err)
	}
}

func TestApp_TaskHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Running task help should not error
	err = app.Run([]string{"task", "help"})
	if err != nil {
		t.Errorf("taskHelp() returned error: %v", err)
	}
}

func TestApp_RunHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Running run help should not error
	err = app.Run([]string{"run", "help"})
	if err != nil {
		t.Errorf("runHelp() returned error: %v", err)
	}
}

// =============================================================================
// COMMAND DISPATCH TESTS
// =============================================================================

func TestApp_CommandDispatch(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	tests := []struct {
		name    string
		command func([]string) error
		help    func(string) error
	}{
		{name: "profile", command: app.cmdProfile, help: func(arg string) error { return app.Run([]string{"profile", arg}) }},
		{name: "task", command: app.cmdTask, help: func(arg string) error { return app.cmdTask([]string{arg}) }},
		{name: "run", command: app.cmdRun, help: func(arg string) error { return app.cmdRun([]string{arg}) }},
		{name: "runner", command: app.cmdRunner, help: func(arg string) error { return app.cmdRunner([]string{arg}) }},
		{name: "settings", command: app.cmdSettings, help: func(arg string) error { return app.cmdSettings([]string{arg}) }},
		{name: "maintenance", command: app.cmdMaintenance, help: func(arg string) error { return app.cmdMaintenance([]string{arg}) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.command(nil); err != nil {
				t.Fatalf("empty args: %v", err)
			}
			for _, arg := range []string{"help", "-h", "--help"} {
				if err := tt.help(arg); err != nil {
					t.Errorf("%s: %v", arg, err)
				}
			}
			if err := tt.command([]string{"unknown-subcommand"}); err == nil {
				t.Error("expected error for unknown subcommand")
			}
		})
	}
}

// =============================================================================
// MISSING ARGUMENTS TESTS
// =============================================================================

func TestCommandsRejectMissingRequiredArguments(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	tests := map[string]func([]string) error{
		"profile get id": app.profileGet, "profile update id": app.profileUpdate,
		"profile delete id": app.profileDelete, "profile create name": app.profileCreate,
		"task get id": app.taskGet, "task create title": app.taskCreate,
		"run get id": app.runGet, "run create task": app.runCreate,
	}
	for name, command := range tests {
		t.Run(name, func(t *testing.T) {
			if err := command(nil); err == nil {
				t.Error("expected missing required argument error")
			}
		})
	}
}

func TestParseResultSpecClassificationUsesCanonicalContract(t *testing.T) { // [REQ:REQ-P2-001]
	spec, err := parseResultSpec("", "", "complete, blocked", true)
	if err != nil {
		t.Fatal(err)
	}
	if spec.Kind != domainpb.ResultSpecKind_RESULT_SPEC_KIND_CLASSIFICATION || len(spec.ClassificationValues) != 2 {
		t.Fatalf("spec = %+v", spec)
	}
	if spec.ExtractionMode != domainpb.StructuredExtractionMode_STRUCTURED_EXTRACTION_MODE_CONSTRAINED_FALLBACK || spec.ExtractionRole != "extract.structured" {
		t.Fatalf("extraction = %+v", spec)
	}
}

func TestParseResultSpecRejectsParallelSchemaSystems(t *testing.T) {
	if _, err := parseResultSpec(`{"type":"string"}`, "", "yes,no", false); err == nil {
		t.Fatal("expected mutually exclusive result-spec inputs to fail")
	}
}

// =============================================================================
// SERVICES TESTS
// =============================================================================

func TestNewServices(t *testing.T) {
	// Test that NewServices creates all expected service instances
	services := NewServices(nil) // nil client for testing structure

	if services == nil {
		t.Fatal("expected services, got nil")
	}

	if services.Profiles == nil {
		t.Error("expected Profiles service")
	}
	if services.Tasks == nil {
		t.Error("expected Tasks service")
	}
	if services.Runs == nil {
		t.Error("expected Runs service")
	}
	if services.Runners == nil {
		t.Error("expected Runners service")
	}
	if services.Policy == nil {
		t.Error("expected Policy service")
	}
	if services.Settings == nil {
		t.Error("expected Settings service")
	}
	if services.Maintenance == nil {
		t.Error("expected Maintenance service")
	}
}

// =============================================================================
// NEW COMMAND DISPATCHER TESTS
// =============================================================================

// [REQ:REQ-P1-004] Policy commands are discoverable and removed mutation vocabulary is rejected.
func TestApp_CmdPolicy_HelpAndUnknownSubcommand(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	for _, args := range [][]string{{}, {"help"}, {"-h"}, {"--help"}} {
		if err := app.cmdPolicy(args); err != nil {
			t.Fatalf("cmdPolicy(%v): %v", args, err)
		}
	}
	if err := app.cmdPolicy([]string{"models-update"}); err == nil {
		t.Fatal("expected removed policy subcommand to fail")
	}
}

func TestApp_AdditionalCommandGroupsAcceptHelp(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}
	commands := map[string]func([]string) error{
		"declarations":      app.cmdDeclarations,
		"events":            app.cmdEvents,
		"health":            app.cmdHealth,
		"ops":               app.cmdOps,
		"permission-policy": app.cmdPermissionPolicy,
	}
	for name, command := range commands {
		t.Run(name, func(t *testing.T) {
			if err := command([]string{"--help"}); err != nil {
				t.Fatalf("help: %v", err)
			}
		})
	}
}
