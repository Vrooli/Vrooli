package flows

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterReturnsNonEmptyFlowsGroup(t *testing.T) {
	g := Register(&cliapp.ScenarioApp{})
	require.Equal(t, "flows", g.Name)
	require.NotEmpty(t, g.Description)
	require.NotEmpty(t, g.Subcommands, "flows group must register at least one subcommand")

	names := map[string]bool{}
	for _, c := range g.Subcommands {
		require.NotEmpty(t, c.Name, "subcommand name must be set")
		require.NotNil(t, c.RunCtx, "subcommand %q must have a RunCtx", c.Name)
		require.False(t, names[c.Name], "duplicate subcommand %q", c.Name)
		names[c.Name] = true
	}
	for _, expected := range []string{"list", "validate", "new", "explain"} {
		require.True(t, names[expected], "missing subcommand %q", expected)
	}
}
