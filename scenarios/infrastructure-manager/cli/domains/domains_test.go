package domains

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/api-core/spacedoc"
	"github.com/vrooli/cli-core/cliapp"
)

// TestCommandGroups pins the flat-commands aggregator to the cross-scenario
// space contract. This scenario is the interim space owner for `capacity` and
// `commissioning`, and the `space` verb is how every other owner publishes its
// denominator. A denominator readable only as a file path is the failure this
// asserts against: the instrument would be the sole reader of a document it
// does not own.
func TestCommandGroups(t *testing.T) {
	got := CommandGroups(&cliapp.ScenarioApp{})
	require.Len(t, got, 1, "the space command group must be registered")
	require.Len(t, got[0].Commands, 1)
	require.Equal(t, "space", got[0].Commands[0].Name)
}

// TestSpaceVerbServesOwnedProjectionsOnly proves the verb emits the two
// projections this scenario owns and refuses the nine it does not. Serving
// another layer's space from the instrument would put the denominator and the
// bar in the same hand, which is the split COVERAGE-MODEL.md exists to keep.
func TestSpaceVerbServesOwnedProjectionsOnly(t *testing.T) {
	run := CommandGroups(&cliapp.ScenarioApp{})[0].Commands[0].Run

	for _, projection := range []string{"capacity", "commissioning"} {
		t.Run(projection, func(t *testing.T) {
			stdout := captureStdout(t, func() {
				require.NoError(t, run([]string{"--projection", projection, "--json"}))
			})
			var def spacedoc.SpaceDefinition
			require.NoError(t, json.Unmarshal([]byte(stdout), &def))
			require.Equal(t, spacedoc.Projection(projection), def.Projection)
			require.NotEmpty(t, def.Cells, "an owned space must publish its cells")
			require.NotEmpty(t, def.DenominatorConfidence, "confidence is never omitted")
			for _, cell := range def.Cells {
				require.NotEmpty(t, cell.ID)
				if cell.Status == spacedoc.StatusMissing {
					require.NotEmpty(t, cell.GapOpenedOn,
						"cell %s is MISSING and must carry gap_opened_on, or gap aging degrades to declared-once-and-forgotten", cell.ID)
				}
			}
		})
	}

	for _, projection := range []string{"substrate", "supervision", "availability", "recovery", "headroom", "durability", "attribution", "validation-cost", "agent-throughput"} {
		t.Run("refuses/"+projection, func(t *testing.T) {
			err := run([]string{"--projection", projection, "--json"})
			require.Error(t, err, "the instrument must not serve a space it does not own")
			require.Contains(t, err.Error(), "owns")
		})
	}
}

// captureStdout redirects os.Stdout for the duration of fn. spacecli writes to
// os.Stdout by default, which is the same path the real CLI takes.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	read, write, err := os.Pipe()
	require.NoError(t, err)
	original := os.Stdout
	os.Stdout = write
	defer func() { os.Stdout = original }()
	fn()
	require.NoError(t, write.Close())
	out, err := io.ReadAll(read)
	require.NoError(t, err)
	return string(out)
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
