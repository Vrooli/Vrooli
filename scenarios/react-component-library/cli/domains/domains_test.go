package domains

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

func TestCommandGroups(t *testing.T) {
	got := CommandGroups(&cliapp.ScenarioApp{})
	require.Len(t, got, 2)
	if len(got) == 2 {
		require.Equal(t, "Coverage", got[0].Title)
		require.Len(t, got[0].Commands, 1)
		require.Equal(t, "coverage", got[0].Commands[0].Name)
		require.Equal(t, "Findings", got[1].Title)
		require.Equal(t, "findings", got[1].Commands[0].Name)
	}
}

func TestSubcommandGroups(t *testing.T) {
	manifest := readManifestForTest(t)
	got, err := SubcommandGroups(&cliapp.ScenarioApp{}, manifest)
	require.NoError(t, err, "SubcommandGroups must build cleanly from cli/manifest.json")
	require.NotNil(t, got, "SubcommandGroups must return a slice (possibly empty), not nil")
	for i, g := range got {
		require.NotEmpty(t, g.Name, "group[%d].Name must be set", i)
		require.NotEmpty(t, g.Subcommands, "group[%d] (%s) must register at least one subcommand", i, g.Name)
	}
}

func readManifestForTest(t *testing.T) []byte {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	require.NoError(t, err, "read cli/manifest.json")
	return bytes
}
