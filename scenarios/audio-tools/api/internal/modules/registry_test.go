package modules_test

import (
	"context"
	"fmt"
	"testing"

	"audio-tools/internal/modules"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "audio-tools/internal/database"
)

// TestAllEndpoints_NonEmpty pins the smoke contract: at minimum the
// health endpoint is registered. Failing this means the registry
// didn't import a handler package or the handler dropped its
// Endpoints slice.
//
// Deliberately loose on count (not "exactly N"): scenarios remove the
// notes reference and add their own domains, so pinning the count would
// couple this test to whichever domains happen to be present.
func TestAllEndpoints_NonEmpty(t *testing.T) {
	got := modules.AllEndpoints()
	require.NotEmpty(t, got, "AllEndpoints must include at least the health endpoint")
}

// TestAllEndpoints_StableOrder pins the stable-order contract that the
// diff-exit-code CI check on .vrooli/endpoints.json depends on. Two
// consecutive calls must return slices with the same element order;
// otherwise codegen output churns and the diff gate becomes noisy.
func TestAllEndpoints_StableOrder(t *testing.T) {
	a := modules.AllEndpoints()
	b := modules.AllEndpoints()
	require.Equal(t, len(a), len(b), "non-deterministic length")
	for i := range a {
		require.Equal(t, a[i].ID, b[i].ID,
			"endpoint order changed between calls at index %d (%s vs %s)",
			i, a[i].ID, b[i].ID)
	}
}

func TestProductAndLabEndpointPartition(t *testing.T) {
	all := modules.AllEndpoints()
	lab := modules.LabEndpoints()
	product := modules.ProductEndpoints()
	if !modules.IsLabBuild() {
		require.Empty(t, lab, "product build must not include lab endpoint descriptors")
		require.Equal(t, all, product)
		return
	}
	require.NotEmpty(t, lab)
	require.NotEmpty(t, product)
	require.Len(t, lab, len(all)-len(product), "product and lab lists must partition all endpoints")

	allIDs := make(map[string]struct{}, len(all))
	allRoutes := make(map[string]struct{}, len(all))
	for _, endpoint := range all {
		require.NotEmpty(t, endpoint.ID)
		require.NotEmpty(t, endpoint.Method)
		require.NotEmpty(t, endpoint.Path)
		_, duplicateID := allIDs[endpoint.ID]
		require.Falsef(t, duplicateID, "endpoint ID %q must be unique", endpoint.ID)
		allIDs[endpoint.ID] = struct{}{}
		route := endpoint.Method + " " + endpoint.Path
		_, duplicateRoute := allRoutes[route]
		require.Falsef(t, duplicateRoute, "endpoint route %q must have one owner", route)
		allRoutes[route] = struct{}{}
	}
	seen := make(map[string]struct{}, len(all))
	seenRoutes := make(map[string]struct{}, len(all))
	for _, endpoint := range lab {
		require.Contains(t, []string{"corpus", "experiment", "validation"}, endpoint.Category)
		seen[endpoint.ID] = struct{}{}
		seenRoutes[endpoint.Method+" "+endpoint.Path] = struct{}{}
	}
	for _, endpoint := range product {
		require.NotContains(t, seen, endpoint.ID, "endpoint %q must not be both product and lab", endpoint.ID)
		route := endpoint.Method + " " + endpoint.Path
		require.NotContainsf(t, seenRoutes, route, "route %q must not be both product and lab", route)
		seenRoutes[route] = struct{}{}
		seen[endpoint.ID] = struct{}{}
	}
	require.Equal(t, allIDs, seen, "product plus lab endpoints must cover the generated inventory")
	require.Contains(t, allIDs, "validation.soak.validate_scenario", "external validation provider paths belong in endpoint inventory")
}

// TestAllSchemas_NonEmpty proves the schema registry is always
// populated (system home is always present) and that every entry has
// a non-nil provider. Per-domain schema content is verified by each
// domain's own *_test.go (see internal/notes/sqlite_test.go for the
// canonical apply-and-query coverage).
func TestAllSchemas_NonEmpty(t *testing.T) {
	got := modules.AllSchemas()
	require.NotEmpty(t, got, "AllSchemas must return at least the system provider")
	for i, p := range got {
		require.NotNil(t, p, "AllSchemas[%d] is nil", i)
	}
}

// TestAllSchemas_FirstIsSystem pins the ordering invariant: the system
// home runs first. Postgres scenarios that put `CREATE EXTENSION ...`
// in system.sql rely on this; SQLite scenarios are unaffected (system
// is empty) but the contract holds either way.
func TestAllSchemas_FirstIsSystem(t *testing.T) {
	got := modules.AllSchemas()
	require.NotEmpty(t, got)
	require.Equal(t, localdb.SystemSchema(), got[0].Schema(),
		"system schema must be first in AllSchemas() so cross-cutting infrastructure applies before any domain")
}

func TestProductAndLabSchemaPartition(t *testing.T) {
	product := modules.ProductSchemas()
	lab := modules.LabSchemas()
	all := modules.AllSchemas()
	require.NotEmpty(t, product)
	if !modules.IsLabBuild() {
		require.Empty(t, lab, "product build must not include lab schema descriptors")
		require.Len(t, all, len(product))
		for i := range product {
			require.Equal(t, product[i].Schema(), all[i].Schema())
		}
		return
	}
	require.Len(t, all, len(product)+len(lab))
	require.Equal(t, localdb.SystemSchema(), product[0].Schema(), "system schema must remain first in product bootstrap")

	productSQL := make(map[string]struct{}, len(product))
	for _, provider := range product {
		require.NotNil(t, provider)
		productSQL[provider.Schema()] = struct{}{}
	}
	for _, provider := range lab {
		require.NotNil(t, provider)
		_, duplicate := productSQL[provider.Schema()]
		require.False(t, duplicate, "lab schema must not be part of product bootstrap")
	}
	for i, provider := range append(append([]apidb.SchemaProvider{}, product...), lab...) {
		require.Equal(t, provider.Schema(), all[i].Schema(), "complete schema inventory must preserve product then lab order")
	}
}

// TestAllSchemas_AppliesIdempotently exercises the registry against a
// real sqlite handle. Mirrors api/main.go's bootstrap: applies all
// schemas twice and asserts no error. Catches the failure mode where
// a domain's schema accidentally drops `IF NOT EXISTS`.
//
// This test calls modules.AllSchemas() (whose contents change as
// scenarios add/remove domains) but never names a specific domain, so
// feature deletion leaves it green.
func TestAllSchemas_AppliesIdempotently(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, d, modules.AllSchemas()...))
	require.NoError(t, apidb.EnsureSchemas(ctx, d, modules.AllSchemas()...),
		"second apply must succeed (uses IF NOT EXISTS guards)")
}

// TestProtoConnectParity enforces the proto/Connect-RPC anti-drift
// contract globally: every `rpc Foo(...)` method declared in any proto
// FileDescriptor returned by AllProtoFiles() must have exactly one
// matching EndpointDescriptor in AllEndpoints() whose Path equals
// "/" + service.FullName + "/" + method.Name.
//
// Lifted from a per-domain test (formerly
// api/handlers/notes/module_test.go) so the safety net applies
// automatically to every Connect-mounted domain registered in
// AllProtoFiles() — agents adding a new domain no longer have to
// remember to copy the parity test.
//
// On failure the message names the proto method, the expected path,
// and the module, so the fix is mechanical: either add the matching
// EndpointDescriptor (referencing the generated *Procedure constant
// from the *v1connect package) or remove the rpc from the proto file.
func TestProtoConnectParity(t *testing.T) {
	endpoints := modules.AllEndpoints()
	byPath := make(map[string]int, len(endpoints))
	for _, ep := range endpoints {
		byPath[ep.Path]++
	}

	files := modules.AllProtoFiles()
	require.NotEmpty(t, files,
		"AllProtoFiles() returned no entries; every Connect-mounted "+
			"domain module must be registered there")

	for _, entry := range files {
		services := entry.File.Services()
		require.NotZero(t, services.Len(),
			"module %q: proto file declares no services", entry.Module)

		for s := 0; s < services.Len(); s++ {
			svc := services.Get(s)
			methods := svc.Methods()
			for m := 0; m < methods.Len(); m++ {
				method := methods.Get(m)
				wantPath := fmt.Sprintf("/%s/%s", svc.FullName(), method.Name())
				count := byPath[wantPath]
				require.Equal(t, 1, count,
					"module %q: proto method %s.%s (expected EndpointDescriptor.Path %q) "+
						"must have exactly one matching entry in AllEndpoints(); found %d. "+
						"Fix: either add an EndpointDescriptor referencing the generated "+
						"*Procedure constant, or remove the rpc from the proto file.",
					entry.Module, svc.FullName(), method.Name(), wantPath, count)
			}
		}
	}
}
