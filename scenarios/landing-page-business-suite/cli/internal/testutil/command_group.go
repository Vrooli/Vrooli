// Package testutil contains shared assertions for Landing Page Business Suite CLI tests.
package testutil

import (
	"fmt"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

// ValidateCommandGroup verifies the registration contract shared by every CLI
// domain. Returning an error lets individual tests make a visible assertion,
// while AssertCommandGroup remains convenient for callers that prefer the
// testing.T helper form.
func ValidateCommandGroup(group cliapp.CommandGroup) error {
	if group.Title == "" {
		return fmt.Errorf("command group must have a title")
	}
	if len(group.Commands) == 0 {
		return fmt.Errorf("command group %q must expose at least one command", group.Title)
	}

	names := make(map[string]struct{}, len(group.Commands))
	for _, command := range group.Commands {
		if command.Name == "" {
			return fmt.Errorf("command group %q contains a command without a name", group.Title)
		}
		if command.Description == "" {
			return fmt.Errorf("command %q in group %q must have a description", command.Name, group.Title)
		}
		if _, exists := names[command.Name]; exists {
			return fmt.Errorf("command group %q registers duplicate command %q", group.Title, command.Name)
		}
		names[command.Name] = struct{}{}
	}
	return nil
}

// AssertCommandGroup is the testing.T convenience wrapper around
// ValidateCommandGroup.
func AssertCommandGroup(t *testing.T, group cliapp.CommandGroup) {
	t.Helper()
	if err := ValidateCommandGroup(group); err != nil {
		t.Fatal(err)
	}
}
