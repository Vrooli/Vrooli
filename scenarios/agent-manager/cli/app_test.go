package main

import (
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

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
		"Profiles",
		"Declarations",
		"Workflows",
		"Tasks",
		"Runs",
		"Runners",
		"Role Policy",
		"Settings",
		"Maintenance",
		"Operational Stats",
		"Health",
		"Events",
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
	err = app.profileHelp()
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
	err = app.taskHelp()
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
	err = app.runHelp()
	if err != nil {
		t.Errorf("runHelp() returned error: %v", err)
	}
}

// =============================================================================
// COMMAND DISPATCH TESTS
// =============================================================================

func TestApp_CmdProfile_EmptyArgs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Empty args should show help, not error
	err = app.cmdProfile([]string{})
	if err != nil {
		t.Errorf("cmdProfile with empty args should show help, got error: %v", err)
	}
}

func TestApp_CmdProfile_Help(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Help subcommand should not error
	for _, helpArg := range []string{"help", "-h", "--help"} {
		err = app.cmdProfile([]string{helpArg})
		if err != nil {
			t.Errorf("cmdProfile with '%s' returned error: %v", helpArg, err)
		}
	}
}

func TestApp_CmdProfile_UnknownSubcommand(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Unknown subcommand should return error
	err = app.cmdProfile([]string{"unknown-subcommand"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestApp_CmdTask_EmptyArgs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Empty args should show help
	err = app.cmdTask([]string{})
	if err != nil {
		t.Errorf("cmdTask with empty args should show help, got error: %v", err)
	}
}

func TestApp_CmdTask_Help(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	for _, helpArg := range []string{"help", "-h", "--help"} {
		err = app.cmdTask([]string{helpArg})
		if err != nil {
			t.Errorf("cmdTask with '%s' returned error: %v", helpArg, err)
		}
	}
}

func TestApp_CmdTask_UnknownSubcommand(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	err = app.cmdTask([]string{"unknown-subcommand"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestApp_CmdRun_EmptyArgs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Empty args should show help
	err = app.cmdRun([]string{})
	if err != nil {
		t.Errorf("cmdRun with empty args should show help, got error: %v", err)
	}
}

func TestApp_CmdRun_Help(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	for _, helpArg := range []string{"help", "-h", "--help"} {
		err = app.cmdRun([]string{helpArg})
		if err != nil {
			t.Errorf("cmdRun with '%s' returned error: %v", helpArg, err)
		}
	}
}

func TestApp_CmdRun_UnknownSubcommand(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	err = app.cmdRun([]string{"unknown-subcommand"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

// =============================================================================
// MISSING ARGUMENTS TESTS
// =============================================================================

func TestApp_ProfileGet_MissingID(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Get without ID should error
	err = app.profileGet([]string{})
	if err == nil {
		t.Error("expected error for missing profile ID")
	}
}

func TestApp_ProfileUpdate_MissingID(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Update without ID should error
	err = app.profileUpdate([]string{})
	if err == nil {
		t.Error("expected error for missing profile ID")
	}
}

func TestApp_ProfileDelete_MissingID(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Delete without ID should error
	err = app.profileDelete([]string{})
	if err == nil {
		t.Error("expected error for missing profile ID")
	}
}

func TestApp_ProfileCreate_MissingName(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Create without name should error
	err = app.profileCreate([]string{})
	if err == nil {
		t.Error("expected error for missing profile name")
	}
}

func TestApp_TaskGet_MissingID(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	err = app.taskGet([]string{})
	if err == nil {
		t.Error("expected error for missing task ID")
	}
}

func TestApp_TaskCreate_MissingTitle(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Create without required fields should error
	err = app.taskCreate([]string{})
	if err == nil {
		t.Error("expected error for missing task title")
	}
}

func TestApp_RunGet_MissingID(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	err = app.runGet([]string{})
	if err == nil {
		t.Error("expected error for missing run ID")
	}
}

func TestApp_RunCreate_MissingTaskID(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Create without task ID should error
	err = app.runCreate([]string{})
	if err == nil {
		t.Error("expected error for missing task ID")
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

func TestApp_CmdRunner_EmptyArgs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Empty args should show help
	err = app.cmdRunner([]string{})
	if err != nil {
		t.Errorf("cmdRunner with empty args should show help, got error: %v", err)
	}
}

func TestApp_CmdRunner_Help(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	for _, helpArg := range []string{"help", "-h", "--help"} {
		err = app.cmdRunner([]string{helpArg})
		if err != nil {
			t.Errorf("cmdRunner with '%s' returned error: %v", helpArg, err)
		}
	}
}

func TestApp_CmdRunner_UnknownSubcommand(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	err = app.cmdRunner([]string{"unknown-subcommand"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

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

func TestApp_CmdSettings_EmptyArgs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Empty args should show help
	err = app.cmdSettings([]string{})
	if err != nil {
		t.Errorf("cmdSettings with empty args should show help, got error: %v", err)
	}
}

func TestApp_CmdSettings_Help(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	for _, helpArg := range []string{"help", "-h", "--help"} {
		err = app.cmdSettings([]string{helpArg})
		if err != nil {
			t.Errorf("cmdSettings with '%s' returned error: %v", helpArg, err)
		}
	}
}

func TestApp_CmdSettings_UnknownSubcommand(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	err = app.cmdSettings([]string{"unknown-subcommand"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}

func TestApp_CmdMaintenance_EmptyArgs(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	// Empty args should show help
	err = app.cmdMaintenance([]string{})
	if err != nil {
		t.Errorf("cmdMaintenance with empty args should show help, got error: %v", err)
	}
}

func TestApp_CmdMaintenance_Help(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	for _, helpArg := range []string{"help", "-h", "--help"} {
		err = app.cmdMaintenance([]string{helpArg})
		if err != nil {
			t.Errorf("cmdMaintenance with '%s' returned error: %v", helpArg, err)
		}
	}
}

func TestApp_CmdMaintenance_UnknownSubcommand(t *testing.T) {
	app, err := NewApp()
	if err != nil {
		t.Fatalf("NewApp() failed: %v", err)
	}

	err = app.cmdMaintenance([]string{"unknown-subcommand"})
	if err == nil {
		t.Error("expected error for unknown subcommand")
	}
}
