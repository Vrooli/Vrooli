package catalog

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRegisterRejectsManifestBackedAndUnknownGroups(t *testing.T) {
	for _, name := range []string{"ledger", "missing"} {
		t.Run(name, func(t *testing.T) {
			group, err := Register(name)
			require.Error(t, err)
			require.Empty(t, group.Name)
			require.Empty(t, group.Subcommands)
		})
	}
}
