package audio

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"

	testutil "audio-tools/cli/internal/testutil"
)

// operationCatalogSeed mirrors the subset of api/internal/operationcatalog that
// this CLI coverage gate needs. The CLI is a separate Go module, so it reads the
// same checked-in seed rather than importing the API package.
type operationCatalogSeed struct {
	Operations []catalogOperation `json:"operations"`
}

type catalogOperation struct {
	ID     string     `json:"id"`
	Family string     `json:"family"`
	CLI    catalogCLI `json:"cli"`
}

type catalogCLI struct {
	Group   string        `json:"group"`
	Verb    string        `json:"verb"`
	Service string        `json:"service"`
	Method  string        `json:"method"`
	Flags   []catalogFlag `json:"flags"`
}

type catalogFlag struct {
	Name      string `json:"name"`
	Field     string `json:"field"`
	Required  bool   `json:"required"`
	LocalOnly bool   `json:"localOnly"`
	Repeated  bool   `json:"repeated"`
}

func loadOperationCatalog(t *testing.T) operationCatalogSeed {
	t.Helper()
	path := filepath.Join("..", "..", "..", "api", "internal", "operationcatalog", "operations.seed.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "read operation catalog seed")
	var seed operationCatalogSeed
	require.NoError(t, json.Unmarshal(raw, &seed))
	return seed
}

// TestAudioManifestCoversEveryOperation proves the manifest's audio commands
// match the declarative operation catalog: every catalog operation declares a
// CLI command, the Connect binding agrees, and every declared flag exists with
// the same required/repeated contract.
func TestAudioManifestCoversEveryOperation(t *testing.T) {
	seed := loadOperationCatalog(t)
	manifest, err := cliapp.ParseManifest(testutil.ReadManifest(t))
	require.NoError(t, err)

	transforms := 0
	for _, op := range seed.Operations {
		if op.CLI.Method == "" {
			continue
		}
		cmd := findCommand(manifest, op.CLI.Group, op.CLI.Verb)
		require.NotNilf(t, cmd, "operation %q: group %q command %q missing from cli/manifest.json", op.ID, op.CLI.Group, op.CLI.Verb)
		require.Equalf(t, "connect-rpc", cmd.Binding.Kind, "operation %q binding kind", op.ID)
		require.Equalf(t, op.CLI.Service, cmd.Binding.Service, "operation %q binding service", op.ID)
		require.Equalf(t, op.CLI.Method, cmd.Binding.Method, "operation %q binding method", op.ID)

		flags := make(map[string]cliapp.ManifestFlag, len(cmd.Flags))
		for _, f := range cmd.Flags {
			flags[f.Name] = f
		}
		for _, want := range op.CLI.Flags {
			got, ok := flags[want.Name]
			require.Truef(t, ok, "operation %q: command %s lacks flag --%s", op.ID, op.CLI.Verb, want.Name)
			require.Equalf(t, want.Required, got.Required, "operation %q: --%s required", op.ID, want.Name)
			require.Equalf(t, want.Repeated, got.Repeated || got.Repeatable, "operation %q: --%s repeated", op.ID, want.Name)
			require.Equalf(t, want.LocalOnly, got.LocalOnly, "operation %q: --%s local_only", op.ID, want.Name)
		}
		if op.Family == "transform" {
			transforms++
			require.NotEmptyf(t, op.CLI.Flags, "operation %q must declare CLI flags", op.ID)
		}
	}
	require.Equal(t, 8, transforms, "every audio transform must be present in the operation catalog")
}

func findCommand(m *cliapp.Manifest, group, command string) *cliapp.ManifestCommand {
	for gi := range m.Groups {
		if m.Groups[gi].Name != group {
			continue
		}
		for ci := range m.Groups[gi].Commands {
			if m.Groups[gi].Commands[ci].Name == command {
				return &m.Groups[gi].Commands[ci]
			}
		}
	}
	return nil
}
