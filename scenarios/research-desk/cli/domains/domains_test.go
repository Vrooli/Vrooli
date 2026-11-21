package domains

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

// TestCommandGroups proves every flat domain group in the manifest is
// registered with its commands, and that no command name repeats across
// groups (flat commands share one namespace).
func TestCommandGroups(t *testing.T) {
	manifest := readManifestForTest(t)
	got, err := CommandGroups(&cliapp.ScenarioApp{}, manifest)
	require.NoError(t, err)
	parsed, err := cliapp.ParseManifest(manifest)
	require.NoError(t, err)
	want := map[string]bool{}
	for _, g := range parsed.Groups {
		if g.Flat {
			for _, c := range g.Commands {
				require.False(t, want[c.Name], "flat command %q is declared twice", c.Name)
				want[c.Name] = true
			}
		}
	}
	registered := map[string]bool{}
	for _, c := range TopLevel(got) {
		registered[c.Name] = true
	}
	require.Equal(t, want, registered)
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
