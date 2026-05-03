package domains

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

// TestCommandGroups exercises the flat-commands aggregator. The
// template ships zero flat commands, so the contract is "returns nil";
// the test exists so a future scenario that adds CommandGroups gets
// caller-side wiring for free (the call goes through unchanged).
func TestCommandGroups(t *testing.T) {
	got := CommandGroups(&cliapp.ScenarioApp{})
	require.Nil(t, got, "CommandGroups should return nil until a domain registers a flat group")
}

// TestSubcommandGroups proves the aggregator returns whatever domains
// are wired in domains.go without panicking, and that every registered
// group has the load-bearing fields (Name + Subcommands) populated.
//
// Deliberately flexible on count and name: scenarios add and remove
// domain packages over time, and pinning "exactly 1 group named X"
// breaks the moment a scenario follows REPLACING-NOTES.md to swap the
// canonical reference for its own first domain. The catch-the-typo
// failure mode (a domain registers but forgets to set Name or has no
// subcommands) still fails this test loudly.
func TestSubcommandGroups(t *testing.T) {
	got := SubcommandGroups(&cliapp.ScenarioApp{})
	require.NotNil(t, got, "SubcommandGroups must return a slice (possibly empty), not nil")
	for i, g := range got {
		require.NotEmpty(t, g.Name, "group[%d].Name must be set", i)
		require.NotEmpty(t, g.Subcommands, "group[%d] (%s) must register at least one subcommand", i, g.Name)
	}
}
