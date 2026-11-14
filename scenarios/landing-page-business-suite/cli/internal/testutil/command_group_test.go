package testutil

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestAssertCommandGroupAcceptsWellFormedGroup(t *testing.T) {
	group := cliapp.CommandGroup{
		Title:    "Example",
		Commands: []cliapp.Command{{Name: "example", Description: "An example command"}},
	}
	if err := ValidateCommandGroup(group); err != nil {
		t.Fatalf("ValidateCommandGroup() error = %v", err)
	}
}
