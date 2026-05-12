package verify

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterReturnsNonEmptyVerifyGroup(t *testing.T) {
	g := Register(&cliapp.ScenarioApp{})
	require.Equal(t, "verify", g.Name)
	require.NotEmpty(t, g.Description)
	require.NotEmpty(t, g.Subcommands, "verify group must register at least one subcommand")

	names := map[string]bool{}
	for _, c := range g.Subcommands {
		require.NotEmpty(t, c.Name)
		require.False(t, names[c.Name], "duplicate subcommand %q", c.Name)
		names[c.Name] = true
	}
	for _, expected := range []string{"run", "check"} {
		require.True(t, names[expected], "missing subcommand %q", expected)
	}
}
