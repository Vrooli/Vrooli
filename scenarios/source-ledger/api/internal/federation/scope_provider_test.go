package federation

import (
	"testing"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
)

func TestAppendScopeProviderDerivesIsolatedRecallDescriptor(t *testing.T) {
	path := writeSearchFixture(t)
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
