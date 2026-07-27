// Package testutil contains shared assertions for Landing Page Business Suite CLI tests.
package testutil

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// AssertCommandGroup verifies the registration contract shared by every CLI domain.
// A malformed group otherwise leaves commands undiscoverable even when their handlers compile.
func AssertCommandGroup(t *testing.T, group cliapp.CommandGroup) {
	t.Helper()
	if group.Title == "" {
		t.Fatal("command group must have a title")
	}
	if len(group.Commands) == 0 {
		t.Fatalf("command group %q must expose at least one command", group.Title)
	}

	names := make(map[string]struct{}, len(group.Commands))
	for _, command := range group.Commands {
		if command.Name == "" {
			t.Fatalf("command group %q contains a command without a name", group.Title)
		}
		if command.Description == "" {
			t.Errorf("command %q in group %q must have a description", command.Name, group.Title)
		}
		if _, exists := names[command.Name]; exists {
			t.Errorf("command group %q registers duplicate command %q", group.Title, command.Name)
		}
		names[command.Name] = struct{}{}
	}
}
