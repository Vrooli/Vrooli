package federation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
)

func TestAppendScopeProviderDerivesIsolatedRecallDescriptor(t *testing.T) {
	root := filepath.Join("..", "..", "..", ".vrooli", "search.json")
	raw, err := os.ReadFile(root)
	require.NoError(t, err)
	path := filepath.Join(t.TempDir(), "search.json")
	require.NoError(t, os.WriteFile(path, raw, 0o644))
	require.NoError(t, AppendScopeProvider(path, "marketing"))
	file, err := aisearch.LoadSearchFile(path)
	require.NoError(t, err)
	var found bool
	for _, provider := range file.Providers {
		if provider.ProviderID == "vrooli-memory.scope.marketing" {
			found = true
			require.Contains(t, string(provider.Endpoint), `scope`)
			require.Contains(t, string(provider.Endpoint), `marketing`)
		}
	}
	require.True(t, found)
}
