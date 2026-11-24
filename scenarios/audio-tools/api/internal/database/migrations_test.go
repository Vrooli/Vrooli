package database_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"

	localdb "audio-tools/internal/database"

	db "github.com/vrooli/api-core/databasetest"
)

type migrationExecStub struct {
	failQuery string
	failure   error
}

func (s migrationExecStub) ExecContext(_ context.Context, query string, _ ...any) (sql.Result, error) {
	if query == s.failQuery {
		return nil, s.failure
	}
	return nil, nil
}

func TestApplyMigrationsReturnsUnexpectedExecutionError(t *testing.T) {
	query := `ALTER TABLE speaker_profiles ADD COLUMN clip_count INTEGER NOT NULL DEFAULT 0`
	wantErr := errors.New("database is read-only")
	err := localdb.ApplyMigrations(context.Background(), migrationExecStub{failQuery: query, failure: wantErr})
	require.ErrorIs(t, err, wantErr)
	require.ErrorContains(t, err, "apply migration")
}

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

func TestApplyMigrations_AddsModelAndProviderTierToLegacyQualificationEvidence(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	_, err := d.ExecContext(ctx, `CREATE TABLE qualification_evidence (
		id TEXT PRIMARY KEY,
		engine_id TEXT NOT NULL,
		strategy TEXT NOT NULL,
		policy_profile TEXT NOT NULL DEFAULT '',
		kind TEXT NOT NULL,
		fault_profile TEXT NOT NULL DEFAULT '',
		passed INTEGER NOT NULL,
		artifact_ref TEXT NOT NULL,
		notes TEXT NOT NULL DEFAULT '',
		machine_json TEXT NOT NULL DEFAULT '{}',
		observed_at TEXT NOT NULL
	)`)
	require.NoError(t, err)
	require.NoError(t, localdb.ApplyMigrations(ctx, d))

	_, err = d.ExecContext(ctx, `INSERT INTO qualification_evidence (id, engine_id, model_id, strategy, kind, passed, artifact_ref, observed_at) VALUES ('legacy', 'kyutai', '', 'passthrough', 'fault', 1, 'bas:legacy', '2026-07-12T00:00:00Z')`)
	require.NoError(t, err)
	row := d.QueryRowContext(ctx, `SELECT model_id, provider_tier FROM qualification_evidence WHERE id = 'legacy'`)
	var model, providerTier string
	require.NoError(t, row.Scan(&model, &providerTier))
	require.Empty(t, model)
	require.Empty(t, providerTier)
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

func TestMigrateBYOKDuplexCapabilityPreservesEncryptedRows(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	_, err := d.ExecContext(ctx, `CREATE TABLE byok_credentials (
		provider_id TEXT NOT NULL,
		capability TEXT NOT NULL CHECK (capability IN ('stt','tts','summarize')),
		secret_kind TEXT NOT NULL DEFAULT 'api_key',
		secret_cipher BLOB NOT NULL,
		fingerprint TEXT NOT NULL,
		created_at TEXT NOT NULL,
		last_used_at TEXT,
		PRIMARY KEY (provider_id, capability)
	)`)
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `INSERT INTO byok_credentials(provider_id, capability, secret_cipher, fingerprint, created_at) VALUES ('openai-tts','tts',X'0102','sk-***abcd','2026-01-01')`)
	require.NoError(t, err)

	require.NoError(t, localdb.MigrateBYOKDuplexCapability(ctx, d))
	require.NoError(t, localdb.MigrateBYOKDuplexCapability(ctx, d), "migration is idempotent")
	_, err = d.ExecContext(ctx, `INSERT INTO byok_credentials(provider_id, capability, secret_cipher, fingerprint, created_at) VALUES ('openai-gpt-live','duplex',X'0304','sk-***efgh','2026-01-02')`)
	require.NoError(t, err)

	var cipher []byte
	var fingerprint string
	require.NoError(t, d.QueryRowContext(ctx, `SELECT secret_cipher, fingerprint FROM byok_credentials WHERE provider_id='openai-tts' AND capability='tts'`).Scan(&cipher, &fingerprint))
	require.Equal(t, []byte{1, 2}, cipher)
	require.Equal(t, "sk-***abcd", fingerprint)
	var stagingTables int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='byok_credentials_duplex'`).Scan(&stagingTables))
	require.Zero(t, stagingTables, "the staging table must not remain as a persisted entity")
}

func TestMigrateBYOKDuplexCapabilitySkipsMissingTable(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, localdb.MigrateBYOKDuplexCapability(context.Background(), d))
}

func TestMigrateBYOKDuplexCapabilityReportsInspectionFailure(t *testing.T) {
	d := db.NewSQLite(t)
	require.NoError(t, d.Close())
	err := localdb.MigrateBYOKDuplexCapability(context.Background(), d)
	require.ErrorContains(t, err, "inspect BYOK schema")
}

func TestMigrateBYOKDuplexCapabilityRollsBackIncompleteLegacySchema(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()
	_, err := d.ExecContext(ctx, `CREATE TABLE byok_credentials (
		provider_id TEXT NOT NULL,
		capability TEXT NOT NULL CHECK (capability IN ('stt','tts','summarize')),
		secret_cipher BLOB NOT NULL,
		fingerprint TEXT NOT NULL,
		created_at TEXT NOT NULL,
		PRIMARY KEY (provider_id, capability)
	)`)
	require.NoError(t, err)
	_, err = d.ExecContext(ctx, `INSERT INTO byok_credentials(provider_id, capability, secret_cipher, fingerprint, created_at) VALUES ('openai-tts','tts',X'0102','fingerprint','2026-01-01')`)
	require.NoError(t, err)

	err = localdb.MigrateBYOKDuplexCapability(ctx, d)
	require.ErrorContains(t, err, "migrate BYOK duplex constraint")

	var count int
	require.NoError(t, d.QueryRowContext(ctx, `SELECT COUNT(*) FROM byok_credentials WHERE provider_id='openai-tts'`).Scan(&count))
	require.Equal(t, 1, count, "the legacy table and encrypted row must survive a failed migration")
	_, err = d.ExecContext(ctx, `INSERT INTO byok_credentials(provider_id, capability, secret_cipher, fingerprint, created_at) VALUES ('openai-gpt-live','duplex',X'0304','fingerprint','2026-01-02')`)
	require.Error(t, err, "the old capability check must remain intact after rollback")
}
