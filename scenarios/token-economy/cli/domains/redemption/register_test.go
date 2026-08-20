package redemption

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterBuildsTypedRedemptionGroup(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "..", "manifest.json"))
	require.NoError(t, err)
	group, err := Register(nil, raw)
	require.NoError(t, err)
	require.Equal(t, GroupName, group.Name)
	require.Len(t, group.Subcommands, 4)
	for _, command := range group.Subcommands {
		require.NotEmpty(t, command.PrimitiveEvidence())
	}
}
