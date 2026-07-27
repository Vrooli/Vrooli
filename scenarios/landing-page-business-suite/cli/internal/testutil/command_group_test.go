package testutil

import (
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestAssertCommandGroupAcceptsWellFormedGroup(t *testing.T) {
	AssertCommandGroup(t, cliapp.CommandGroup{
		Title:    "Example",
		Commands: []cliapp.Command{{Name: "example", Description: "An example command"}},
	})
}
