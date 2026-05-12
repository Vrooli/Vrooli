package artifacts

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegister_ExposesStatusGenerateClear(t *testing.T) {
	g := Register(&cliapp.ScenarioApp{})
	require.Equal(t, "artifacts", g.Name)
	require.NotEmpty(t, g.Description)
	names := map[string]bool{}
	for _, c := range g.Subcommands {
		require.NotNil(t, c.RunCtx, "subcommand %q must have a RunCtx", c.Name)
		names[c.Name] = true
	}
	for _, want := range []string{"status", "generate", "clear"} {
		require.True(t, names[want], "missing subcommand %q", want)
	}
}

func TestTruthy_AcceptsCommonForms(t *testing.T) {
	for _, s := range []string{"1", "true", "TRUE", "yes", "y", "on"} {
		require.True(t, truthy(s), "truthy(%q) should be true", s)
	}
	for _, s := range []string{"", "false", "no", "off", "0"} {
		require.False(t, truthy(s), "truthy(%q) should be false", s)
	}
}
