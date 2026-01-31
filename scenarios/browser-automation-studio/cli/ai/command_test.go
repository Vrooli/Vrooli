package ai

import (
	"testing"

	"browser-automation-studio/cli/internal/appctx"

	"github.com/vrooli/cli-core/cliapp"
)

func TestCommands(t *testing.T) {
	// Create a minimal context for testing
	ctx := &appctx.Context{
		Name:    "test-cli",
		Version: "1.0.0",
	}

	group := Commands(ctx)

	if group.Title != "AI" {
		t.Errorf("Commands().Title = %q, want %q", group.Title, "AI")
	}

	if len(group.Commands) != 1 {
		t.Fatalf("Commands() returned %d commands, want 1", len(group.Commands))
	}

	cmd := group.Commands[0]
	if cmd.Name != "ai" {
		t.Errorf("Commands()[0].Name = %q, want %q", cmd.Name, "ai")
	}
	if !cmd.NeedsAPI {
		t.Error("Commands()[0].NeedsAPI = false, want true")
	}
	if cmd.Description == "" {
		t.Error("Commands()[0].Description should not be empty")
	}
}

func TestRunAI_Help(t *testing.T) {
	ctx := &appctx.Context{
		Name:    "test-cli",
		Version: "1.0.0",
	}

	// Test --help flag
	err := runAI(ctx, []string{"--help"})
	if err != nil {
		t.Errorf("runAI(--help) error = %v, want nil", err)
	}

	// Test -h flag
	err = runAI(ctx, []string{"-h"})
	if err != nil {
		t.Errorf("runAI(-h) error = %v, want nil", err)
	}
}

func TestRunAI_NoSubcommand(t *testing.T) {
	ctx := &appctx.Context{
		Name:    "test-cli",
		Version: "1.0.0",
	}

	err := runAI(ctx, []string{})
	if err == nil {
		t.Error("runAI() with no args should return error")
	}
}

func TestRunAI_UnknownSubcommand(t *testing.T) {
	ctx := &appctx.Context{
		Name:    "test-cli",
		Version: "1.0.0",
	}

	err := runAI(ctx, []string{"unknown"})
	if err == nil {
		t.Error("runAI(unknown) should return error")
	}
}

// TestCommandGroupStructure verifies the command group matches the expected cliapp interface.
func TestCommandGroupStructure(t *testing.T) {
	ctx := &appctx.Context{
		Name:    "test-cli",
		Version: "1.0.0",
	}

	group := Commands(ctx)

	// Verify it's a valid CommandGroup
	var _ cliapp.CommandGroup = group

	// Verify the command is valid
	for _, cmd := range group.Commands {
		var _ cliapp.Command = cmd

		if cmd.Run == nil {
			t.Errorf("Command %q has nil Run function", cmd.Name)
		}
	}
}
