package categories

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterBindsEveryCategoryCommand(t *testing.T) {
	manifest, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	group, err := Register(&cliapp.ScenarioApp{}, manifest)
	require.NoError(t, err)
	require.Equal(t, GroupName, group.Name)
	require.Len(t, group.Subcommands, 6)
}
