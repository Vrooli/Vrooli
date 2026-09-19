package database_test

import (
	"context"
	"strings"
	"testing"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "audio-tools/internal/database"
)

// TestSystemSchema_IsCrossCuttingOnly guards the storage ownership boundary.
// system.sql contains only cross-cutting documentation; each domain exposes
// its own Schema and the module registry composes those providers at boot.
func TestSystemSchema_IsCrossCuttingOnly(t *testing.T) {
	if got := strings.TrimSpace(stripComments(localdb.SystemSchema())); got != "" {
		t.Fatalf("system.sql must not declare domain tables, got %q", got)
	}
}

// TestEnsureSchemas_AppliesSystem proves the canonical bootstrap path
// works against a real sqlite handle with the system provider only.
// Per-domain providers own their own apply-and-query coverage in their
// respective packages.
//
// This test deliberately does NOT import any per-domain package. Domain
// deletion must leave this package's tests passing — coupling to a domain
// here would break the deletability invariant Pass-3 establishes.
func TestEnsureSchemas_AppliesSystem(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
	))
	// Idempotency: a second apply against the same handle must succeed.
	require.NoError(t, apidb.EnsureSchemas(ctx, d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
	))
}

// stripComments removes SQL line comments so the empty-by-default check
// passes when the file contains only header documentation.
func stripComments(s string) string {
	var b strings.Builder
	for _, line := range strings.Split(s, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") {
			continue
		}
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
