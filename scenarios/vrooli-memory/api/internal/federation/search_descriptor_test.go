package federation

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
)

func TestSearchDescriptorParsesAndMapsRecallHits(t *testing.T) { // [REQ:VMEM-P0-009]
	path := filepath.Join("..", "..", "..", ".vrooli", "search.json")
	file, err := aisearch.LoadSearchFile(path)
	require.NoError(t, err)
	require.Len(t, file.Providers, 1)
	provider := file.Providers[0]
	require.Equal(t, "vrooli-memory.memories", provider.ProviderID)
	require.NotEmpty(t, provider.Tests.Cases)

	descriptors, err := searchregister.Descriptors(file)
	require.NoError(t, err)
	require.Len(t, descriptors, 1)
	require.Equal(t, provider.ProviderID, descriptors[0].GetProviderId())
	require.Equal(t, "hits", descriptors[0].GetResultMapping().GetResultsPath())
	require.Equal(t, "entryId", descriptors[0].GetResultMapping().GetIdField())
}
