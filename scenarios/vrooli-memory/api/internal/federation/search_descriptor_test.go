package federation

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

func TestSearchDescriptorParsesAndMapsRecallHits(t *testing.T) { // [REQ:VMEM-P0-009]
	path := filepath.Join("..", "..", "..", ".vrooli", "search.json")
	file, err := aisearch.LoadSearchFile(path)
	require.NoError(t, err)
	var provider aisearch.ProviderConfig
	for _, candidate := range file.Providers {
		if candidate.ProviderID == "vrooli-memory.memories" {
			provider = candidate
			break
		}
	}
	require.NotEmpty(t, provider.ProviderID)
	require.Equal(t, "vrooli-memory.memories", provider.ProviderID)
	require.NotEmpty(t, provider.Tests.Cases)

	descriptors, err := searchregister.Descriptors(file)
	require.NoError(t, err)
	var descriptor *registryv1.ProviderDescriptor
	for _, candidate := range descriptors {
		if candidate.GetProviderId() == provider.ProviderID {
			descriptor = candidate
			break
		}
	}
	require.NotNil(t, descriptor)
	require.Equal(t, "hits", descriptor.GetResultMapping().GetResultsPath())
	require.Equal(t, "entryId", descriptor.GetResultMapping().GetIdField())
}
