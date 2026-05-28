package database_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	localdb "audio-tools/internal/database"
	"audio-tools/internal/testutil/db"
)

// TestApplyMigrations_AddsColumnsToLegacyTable proves the forward-only ALTERs
// bring a speaker_profiles table created with the PRE-metadata schema up to the
// current shape — the upgrade path for the operator's existing on-disk DB.
func TestApplyMigrations_AddsColumnsToLegacyTable(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()

	// Recreate the legacy table exactly as it shipped before the metadata
	// columns existed (no enrollment_audio_seconds / sample_rate / etc.).
	_, err := d.ExecContext(ctx, `CREATE TABLE speaker_profiles (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		embedding BLOB,
		bound_user_identity TEXT,
		created_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `INSERT INTO speaker_profiles(id,name,created_at) VALUES('legacy','Old','2026-01-01T00:00:00Z')`)
	require.NoError(t, err)

	require.NoError(t, localdb.ApplyMigrations(ctx, d))

	// The new columns now exist and the legacy row reads back with defaults.
	row := d.QueryRowContext(ctx, `SELECT clip_count, total_voiced_seconds, sample_rate, embedding_dim, model_name FROM speaker_profiles WHERE id='legacy'`)
	var clips int
	var voiced float64
	var sr, dim int
	var model string
	require.NoError(t, row.Scan(&clips, &voiced, &sr, &dim, &model))
	require.Zero(t, clips)
	require.Zero(t, voiced)
	require.Zero(t, sr)
	require.Zero(t, dim)
	require.Empty(t, model)
}

// TestApplyMigrations_Idempotent proves a second apply is a clean no-op (the
// "duplicate column name" error is the success signal, not a failure) when the
// columns are already present from the declarative schema.
func TestApplyMigrations_Idempotent(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	// Start from the current declarative schema (columns already present).
	require.NoError(t, apidb.EnsureSchemas(ctx, d, apidb.SchemaProviderFunc(localdb.SystemSchema)))
	require.NoError(t, localdb.ApplyMigrations(ctx, d))
	require.NoError(t, localdb.ApplyMigrations(ctx, d))
}
