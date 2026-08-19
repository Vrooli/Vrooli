package database_test

import (
	"context"
	"strings"
	"testing"

	db "github.com/vrooli/api-core/databasetest"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "treasury/internal/database"
)

// TestSystemSchema_IsEmpty is a deliberate tripwire. The system file
// ships empty by intent — if a future agent adds a CREATE TABLE here
// instead of creating a domain package, this test fails and forces a
// "yes, this is a genuinely cross-cutting concern" decision. Update the
// test (or the comment) when the first real entry lands.
//
// Lives here (not under a per-domain test) because the system home is
// owned by this package; the tripwire belongs with its subject.
func TestSystemSchema_IsEmpty(t *testing.T) {
	got := strings.TrimSpace(stripComments(localdb.SystemSchema()))
	if got != "" {
		t.Fatalf("system.sql is meant to be empty by default; got:\n%s\n\n"+
			"If this is an intentional cross-cutting addition, update this test.",
			got)
	}
}

// TestEnsureSchemas_AppliesSystem proves the canonical bootstrap path
// works against a real sqlite handle with the system provider only.
// Per-domain providers own their own apply-and-query coverage in their
// own *_test.go files.
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
