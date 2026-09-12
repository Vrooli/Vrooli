package federation

import (
	"testing"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
)

func TestProviderIDForScopeUsesStableSourceLedgerNamespace(t *testing.T) {
	require.Equal(t, "source-ledger.agent-memory", ProviderIDForScope("agent-memory"))
	require.Equal(t, "source-ledger.scope.marketing", ProviderIDForScope("marketing"))
	require.NotContains(t, ProviderIDForScope("marketing"), "vrooli-memory")
}

func TestAppendScopeProviderDerivesIsolatedRecallDescriptor(t *testing.T) {
	path := writeSearchFixture(t)
	require.NoError(t, AppendScopeProvider(path, "marketing"))
	file, err := aisearch.LoadSearchFile(path)
	require.NoError(t, err)
	require.Len(t, file.Providers, 1)
	require.Equal(t, "source-ledger.agent-memory", file.Providers[0].ProviderID)
}
