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

// TestSubcommandGroups proves the aggregator wires the notes domain.
// New domain packages should land here as additional asserts so the
// aggregator's failure mode (forgot to add the new domain) is caught
// in CI rather than on a bug report.
func TestSubcommandGroups(t *testing.T) {
	got := SubcommandGroups(&cliapp.ScenarioApp{})
	require.Len(t, got, 1, "expected exactly one registered subcommand group")
	require.Equal(t, "notes", got[0].Name, "the canonical CRUD reference must always be present")
}
