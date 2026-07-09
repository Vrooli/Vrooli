package facts

import (
	"database/sql"
	"testing"

	"code-facts/internal/module"

	"github.com/stretchr/testify/require"
)

func TestModuleWithoutDatabaseStillMountsFactsEndpoints(t *testing.T) {
	got := Module((*sql.DB)(nil), nil, DefaultCacheMaxBytes())

	require.Equal(t, "facts", got.Name)
	require.NotNil(t, got.Mount)
	require.NotEmpty(t, got.Endpoints)
	requireEndpointIDs(t, got.Endpoints,
		"facts_describe",
		"facts_surfaces",
		"facts_cache_status",
		"facts_cache_inspect",
		"facts_cache_clear",
	)
}

func TestCacheStatusEndpointDocumentsBudgetFields(t *testing.T) {
	endpoint := endpointByID(t, Endpoints, "facts_cache_status")

	require.Equal(t, "cache", endpoint.Category)
	require.NotNil(t, endpoint.Response)
	for _, field := range []string{
		"total_rows",
		"total_payload_bytes",
		"budget_bytes",
		"utilization",
		"scopes",
		"last_sweep_at_unix",
	} {
		require.Contains(t, endpoint.Response.Properties, field)
	}
}

func TestCacheClearEndpointDocumentsAllFlag(t *testing.T) {
	endpoint := endpointByID(t, Endpoints, "facts_cache_clear")

	require.NotNil(t, endpoint.Request)
	require.Contains(t, endpoint.Request.Properties, "all")
	require.Contains(t, endpoint.Request.Properties["target"], "unless all is true")
}

func TestSchemaHelpersExposeCacheInfrastructure(t *testing.T) {
	require.Contains(t, Schema(), "code_facts_cache_entries")
	require.Positive(t, DefaultCacheMaxBytes())
}

func endpointByID(t *testing.T, endpoints []module.EndpointDescriptor, id string) module.EndpointDescriptor {
	t.Helper()
	for _, endpoint := range endpoints {
		if endpoint.ID == id {
			return endpoint
		}
	}
	t.Fatalf("endpoint %q not found", id)
	return module.EndpointDescriptor{}
}

func requireEndpointIDs(t *testing.T, endpoints []module.EndpointDescriptor, ids ...string) {
	t.Helper()
	seen := map[string]bool{}
	for _, endpoint := range endpoints {
		seen[endpoint.ID] = true
	}
	for _, id := range ids {
		require.True(t, seen[id], "endpoint %s must be registered", id)
	}
}
