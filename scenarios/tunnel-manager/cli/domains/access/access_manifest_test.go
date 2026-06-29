package access

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

// readManifestForTest reads cli/manifest.json from two directories up
// (this test runs in cli/domains/access/). Mirrors the sibling domains_test.
func readManifestForTest(t *testing.T) []byte {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err, "read cli/manifest.json")
	return raw
}

// TestAccessGroupRegisters asserts the access group loads from the manifest
// with the read-only status command bound to GetAccessStatus, exposing the
// --dry-run projection flag. The CLI manifest contract binds exactly one
// command per RPC, so the dry-run preview is a flag on status, not a separate
// verb; this test pins that shape.
func TestAccessGroupRegisters(t *testing.T) {
	group, err := Register(&cliapp.ScenarioApp{}, readManifestForTest(t))
	require.NoError(t, err, "access group must build from cli/manifest.json")
	require.Equal(t, GroupName, group.Name)
	require.Len(t, group.Subcommands, 1, "access group binds one command per RPC")

	status := group.Subcommands[0]
	require.Equal(t, "status", status.Name)

	var dryRun *cliapp.Flag
	for i := range status.Args.Flags {
		if status.Args.Flags[i].Name == "dry-run" {
			dryRun = &status.Args.Flags[i]
		}
	}
	require.NotNil(t, dryRun, "status must declare a --dry-run flag")
	require.True(t, dryRun.Bool, "--dry-run must be a presence (bool) flag")
}
