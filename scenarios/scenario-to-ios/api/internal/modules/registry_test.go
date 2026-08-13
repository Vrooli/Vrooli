package modules_test

import (
	"context"
	"fmt"
	"testing"

	db "github.com/vrooli/api-core/databasetest"
	"scenario-to-ios/internal/modules"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "scenario-to-ios/internal/database"
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

	// An empty list is valid for a scenario whose current API surface is made
	// entirely of documented REST exceptions (health is the template case).
	files := modules.AllProtoFiles()

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
