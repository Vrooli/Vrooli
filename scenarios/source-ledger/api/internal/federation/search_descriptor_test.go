package federation

import (
	"testing"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
	searchregister "github.com/vrooli/searchregister-go"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

func TestSearchDescriptorParsesAndMapsRecallHits(t *testing.T) { // [REQ:VMEM-P0-009]
	path := writeSearchFixture(t)
	file, err := aisearch.LoadSearchFile(path)
	require.NoError(t, err)
	var provider aisearch.ProviderConfig
	for _, candidate := range file.Providers {
		if candidate.ProviderID == "source-ledger.scopes" {
			provider = candidate
			break
		}
	}
	require.NotEmpty(t, provider.ProviderID)
	require.Equal(t, "source-ledger.scopes", provider.ProviderID)
	require.Equal(t, "source-ledger", provider.ProviderGroup)
	require.Contains(t, string(provider.Endpoint), "source-ledger")
	require.NotContains(t, string(provider.Endpoint), "vrooli-memory")
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

func TestCommittedSearchDescriptorUsesOneScopeAwareProvider(t *testing.T) { // [REQ:SL-P1-002]
	file, err := aisearch.LoadSearchFile("../../../.vrooli/search.json")
	require.NoError(t, err)
	require.Len(t, file.Providers, 1, "duplicate providers run default recall twice and ignore explicit scopes")
	require.Equal(t, "source-ledger.scopes", file.Providers[0].ProviderID)
	require.Contains(t, string(file.Providers[0].Endpoint), "{{scope_value}}")
}
