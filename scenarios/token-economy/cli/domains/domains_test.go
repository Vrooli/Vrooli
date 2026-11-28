package domains

import (
	"os"
	"path/filepath"
	"sort"
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
// breaks the moment a scenario swaps the canonical reference for its
// own first domain. The catch-the-typo failure mode (a domain registers
// but forgets to set Name or has no
// subcommands) still fails this test loudly.
func TestSubcommandGroups(t *testing.T) {
	manifest := readManifestForTest(t)
	got, err := SubcommandGroups(&cliapp.ScenarioApp{}, manifest)
	require.NoError(t, err, "SubcommandGroups must build cleanly from cli/manifest.json")
	require.NotNil(t, got, "SubcommandGroups must return a slice (possibly empty), not nil")
	for i, g := range got {
		require.NotEmpty(t, g.Name, "group[%d].Name must be set", i)
		require.NotEmpty(t, g.Subcommands, "group[%d] (%s) must register at least one subcommand", i, g.Name)
	}
	names := make([]string, 0, len(got))
	for _, group := range got {
		names = append(names, group.Name)
	}
	sort.Strings(names)
	require.Equal(t, []string{"catalog", "earning", "grants", "holders", "journal", "mints", "redemption"}, names)
}

func TestMintsGroupCannotAcquireHolderClientAuthority(t *testing.T) {
	manifest, err := cliapp.ParseManifest(readManifestForTest(t))
	require.NoError(t, err)
	group := manifest.FindGroup("mints")
	require.NotNil(t, group)
	for _, command := range group.Commands {
		require.Equal(t, "MinterService", command.Binding.Service, "mints/%s must never bind a holder client", command.Name)
	}
}

// readManifestForTest reads cli/manifest.json from the parent cli/ directory
// (this test runs in cli/domains/). Mirrors what the embed in main does,
// but at test time the embed isn't available since this package is not main.
func readManifestForTest(t *testing.T) []byte {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("..", "manifest.json"))
	require.NoError(t, err, "read cli/manifest.json")
	return bytes
}
