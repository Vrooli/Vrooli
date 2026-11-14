package inference

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
)

func TestManifestBindsInferenceRun(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	group, err := Register(&cliapp.ScenarioApp{}, manifest)
	require.NoError(t, err)
	require.Equal(t, GroupName, group.Name)
	require.Len(t, group.Subcommands, 5)
}

func TestManifestExposesPrivacyContractForEveryTypedCaller(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	var root struct {
		Groups []struct {
			Name     string `json:"name"`
			Commands []struct {
				Name  string `json:"name"`
				Flags []struct {
					Name    string `json:"name"`
					Default string `json:"default"`
				} `json:"flags"`
			} `json:"commands"`
		} `json:"groups"`
	}
	require.NoError(t, json.Unmarshal(manifest, &root))
	commands := map[string][]struct {
		Name    string `json:"name"`
		Default string `json:"default"`
	}{}
	for _, group := range root.Groups {
		if group.Name != GroupName {
			continue
		}
		for _, command := range group.Commands {
			flags := make([]struct {
				Name    string `json:"name"`
				Default string `json:"default"`
			}, 0, len(command.Flags))
			for _, flag := range command.Flags {
				flags = append(flags, struct {
					Name    string `json:"name"`
					Default string `json:"default"`
				}{Name: flag.Name, Default: flag.Default})
			}
			commands[command.Name] = flags
		}
	}
	for _, name := range []string{"run", "run-batch", "decide", "decide-batch"} {
		flags, ok := commands[name]
		require.True(t, ok, "inference %s command missing", name)
		seen := map[string]string{}
		for _, flag := range flags {
			seen[flag.Name] = flag.Default
		}
		require.Contains(t, seen, "profile", "inference %s must expose profile", name)
		require.Equal(t, "local-first", seen["profile"])
		require.Contains(t, seen, "privacy-class", "inference %s must expose privacy-class", name)
		require.Empty(t, seen["privacy-class"], "privacy classification must never be synthesized by a CLI default")
	}
}
