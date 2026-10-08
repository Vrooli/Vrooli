package readiness

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
)

// [REQ:PH-CLI-001] The readiness group loads from the manifest and binds every
// ReadinessService verb (validate / fix / apply).
func TestRegisterBindsManifestVerbs(t *testing.T) {
	manifest := loadManifest(t)
	group, err := Register(&cliapp.ScenarioApp{}, manifest)
	require.NoError(t, err, "Register must build cleanly from cli/manifest.json")
	require.Equal(t, GroupName, group.Name)

	want := map[string]bool{"validate": false, "fix": false, "apply": false}
	for _, c := range group.Subcommands {
		if _, ok := want[c.Name]; ok {
			want[c.Name] = true
		}
	}
	for name, found := range want {
		require.Truef(t, found, "readiness verb %q not registered", name)
	}
}

// loadManifest reads cli/manifest.json from the cli/ root (this test runs in
// cli/domains/readiness/).
func loadManifest(t *testing.T) []byte {
	t.Helper()
	bytes, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err, "read cli/manifest.json")
	return bytes
}
