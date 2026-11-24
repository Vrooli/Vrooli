package database

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// migrationExecer is the minimal database surface ApplyMigrations needs.
// *sql.DB satisfies it directly; tests supply a fake.
type migrationExecer interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
}

var _ migrationExecer = (*sql.DB)(nil)

// forwardOnlyAlters are additive ADD COLUMN statements applied AFTER the
// declarative schema (EnsureSchemas). The declarative schema already lists
// these columns in CREATE TABLE for fresh databases; these ALTERs bring an
// already-created table up to the current shape.
//
// SQLite has no `ADD COLUMN IF NOT EXISTS`, so re-running an ALTER on a column
// that already exists raises "duplicate column name". That specific error is
// the success signal for an already-migrated database and is swallowed; any
// other error is fatal. This is a flat forward-only list, not a versioned
// migration framework — add a line when a column is introduced.
var forwardOnlyAlters = []string{
	`ALTER TABLE speaker_profiles ADD COLUMN clip_count INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE speaker_profiles ADD COLUMN total_voiced_seconds REAL NOT NULL DEFAULT 0`,
	`ALTER TABLE speaker_profiles ADD COLUMN sample_rate INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE speaker_profiles ADD COLUMN embedding_dim INTEGER NOT NULL DEFAULT 0`,
	`ALTER TABLE speaker_profiles ADD COLUMN model_name TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE qualification_evidence ADD COLUMN model_id TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE qualification_evidence ADD COLUMN provider_tier TEXT NOT NULL DEFAULT ''`,
	`DROP INDEX IF EXISTS idx_qualification_evidence_cell`,
	`CREATE INDEX IF NOT EXISTS idx_qualification_evidence_cell ON qualification_evidence(engine_id, model_id, strategy, policy_profile, observed_at DESC)`,
	`ALTER TABLE corpus_clips ADD COLUMN corpus_revision TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE corpus_clips ADD COLUMN license_reference TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE corpus_clips ADD COLUMN speaker_pseudonym TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE corpus_clips ADD COLUMN split TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE corpus_clips ADD COLUMN noise_condition TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE corpus_clips ADD COLUMN snr_db REAL NOT NULL DEFAULT 0`,
	`ALTER TABLE corpus_clips ADD COLUMN content_sha256 TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE corpus_clips ADD COLUMN source_id TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE corpus_clips ADD COLUMN accent_group TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE corpus_clips ADD COLUMN manifest_transcript_sha256 TEXT NOT NULL DEFAULT ''`,
	`ALTER TABLE corpus_clips ADD COLUMN lineage_manifest_sha256 TEXT NOT NULL DEFAULT ''`,
	`CREATE INDEX IF NOT EXISTS idx_corpus_clips_source_id ON corpus_clips(source_id) WHERE source_id != ''`,
	`CREATE TABLE IF NOT EXISTS corpus_reference_reviews (clip_id TEXT PRIMARY KEY REFERENCES corpus_clips(id) ON DELETE CASCADE, review_json TEXT NOT NULL, updated_at TEXT NOT NULL)`,
}

// ApplyMigrations runs the forward-only ALTERs against db. It is idempotent:
// the "duplicate column name" error from an already-applied ALTER is treated
// as success. A partial-schema test/database can omit a domain table entirely;
// that is also skipped so migrations remain usable by isolated domain tests.
// Bootstrap calls it before EnsureSchemas (to repair declared-shape drift) and
// again after EnsureSchemas (to cover tables created during this boot).
func ApplyMigrations(ctx context.Context, db migrationExecer) error {
	for _, stmt := range forwardOnlyAlters {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			message := strings.ToLower(err.Error())
			if strings.Contains(message, "duplicate column name") || strings.Contains(message, "no such table") {
				continue
			}
			return fmt.Errorf("apply migration %q: %w", stmt, err)
		}
	}
	return nil
}

// MigrateBYOKDuplexCapability rebuilds the BYOK table when its original closed
// CHECK constraint predates the duplex capability. SQLite cannot alter a CHECK
// constraint in place; copying the rows preserves ciphertext and fingerprints.
func MigrateBYOKDuplexCapability(ctx context.Context, db *sql.DB) error {
	var schema sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT sql FROM sqlite_master WHERE type='table' AND name='byok_credentials'`).Scan(&schema); err != nil {
		if err == sql.ErrNoRows {
			return nil
		}
		return fmt.Errorf("inspect BYOK schema: %w", err)
	}
	if !schema.Valid || strings.Contains(schema.String, "'duplex'") {
		return nil
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin BYOK duplex migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	for _, stmt := range []string{
		`CREATE TEMP TABLE byok_credentials_duplex (
			provider_id TEXT NOT NULL,
			capability TEXT NOT NULL CHECK (capability IN ('stt','tts','summarize','duplex')),
			secret_kind TEXT NOT NULL DEFAULT 'api_key',
			secret_cipher BLOB NOT NULL,
			fingerprint TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_used_at TEXT,
			PRIMARY KEY (provider_id, capability)
		)`,
		`INSERT INTO byok_credentials_duplex (provider_id, capability, secret_kind, secret_cipher, fingerprint, created_at, last_used_at)
		 SELECT provider_id, capability, secret_kind, secret_cipher, fingerprint, created_at, last_used_at FROM byok_credentials`,
		`DROP TABLE byok_credentials`,
		`CREATE TABLE byok_credentials (
			provider_id TEXT NOT NULL,
			capability TEXT NOT NULL CHECK (capability IN ('stt','tts','summarize','duplex')),
			secret_kind TEXT NOT NULL DEFAULT 'api_key',
			secret_cipher BLOB NOT NULL,
			fingerprint TEXT NOT NULL,
			created_at TEXT NOT NULL,
			last_used_at TEXT,
			PRIMARY KEY (provider_id, capability)
		)`,
		`INSERT INTO byok_credentials (provider_id, capability, secret_kind, secret_cipher, fingerprint, created_at, last_used_at)
			 SELECT provider_id, capability, secret_kind, secret_cipher, fingerprint, created_at, last_used_at FROM byok_credentials_duplex`,
		`DROP TABLE byok_credentials_duplex`,
	} {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("migrate BYOK duplex constraint: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit BYOK duplex migration: %w", err)
	}
	return nil
}
