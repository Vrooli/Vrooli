package database_test

import (
	"context"
	"strings"
	"testing"

	"audio-tools/internal/testutil/db"

	"github.com/stretchr/testify/require"

	apidb "github.com/vrooli/api-core/database"

	localdb "audio-tools/internal/database"
)

// TestSystemSchema_DeclaresCoreTables guards the canonical table set.
// system.sql owns the full audio-tools schema; per-domain handlers
// keep Schema() empty and route I/O through internal/store/*.
func TestSystemSchema_DeclaresCoreTables(t *testing.T) {
	got := stripComments(localdb.SystemSchema())
	for _, want := range []string{
		"byok_credentials",
		"provider_config",
		"voice_overrides",
		"usage_rows",
		"wakeword_templates",
		"speaker_profiles",
		"stt_stream_config",
		"tts_config",
		"playback_events",
	} {
		if !strings.Contains(got, want) {
			t.Fatalf("system.sql missing table %q", want)
		}
	}
}

// TestEnsureSchemas_AppliesSystem proves the canonical bootstrap path
// works against a real sqlite handle with the system provider only.
// Per-domain providers (notes, etc.) own their own apply-and-query
// coverage in their own *_test.go files (see internal/notes/sqlite_test.go).
//
// This test deliberately does NOT import any per-domain package. Domain
// deletion must leave this package's tests passing — coupling to notes
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
