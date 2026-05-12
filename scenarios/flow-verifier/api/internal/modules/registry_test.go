package modules_test

import (
	"context"
	"testing"

	"flow-verifier/internal/modules"
	"flow-verifier/internal/testutil/db"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "flow-verifier/internal/database"
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
