package releases

import (
	"context"
	"fmt"
	"strings"

	"deployment-manager/shared"
)

import _ "embed"

//go:embed schema.sql
var schemaSQL string

func Schema() string { return schemaSQL }

//go:embed indexes.sql
var indexesSQL string

// IndexesSchema returns indexes that depend on columns added to an existing
// release ledger. Call it after database.EnsureSchemas has reconciled additive
// column drift; applying these statements inside the declarative schema would
// make a legacy database fail before reconciliation can run.
func IndexesSchema() string { return indexesSQL }

// MigrateLegacyReleaseConstraints removes the pre-identity uniqueness rule
// that treated one source commit as one release. A rebuilt candidate can
// legitimately reuse that commit while carrying different signed bytes. The
// migration preserves every release row and leaves dependent tables pointing
// at the recreated releases table.
func MigrateLegacyReleaseConstraints(ctx context.Context, db shared.RoutedDBTX) error {
	columns, err := legacyCommitUniqueColumns(ctx, db)
	if err != nil {
		return err
	}
	if len(columns) == 0 {
		return nil
	}

	conn, err := db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("open release schema migration connection: %w", err)
	}
	defer conn.Close()
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = OFF`); err != nil {
		return fmt.Errorf("disable foreign keys for release schema migration: %w", err)
	}
	foreignKeysRestored := false
	defer func() {
		if !foreignKeysRestored {
			_, _ = conn.ExecContext(context.Background(), `PRAGMA foreign_keys = ON`)
		}
	}()

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin release schema migration: %w", err)
	}
	rollback := func(cause error) error {
		_ = tx.Rollback()
		return cause
	}
	const backupTable = "releases_legacy_identity_migration"
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`CREATE TABLE %s AS SELECT id, profile_id, deployment_id, profile_version, git_commit_hash, artifact_digest, candidate_id, destination_revision_id, authorization_epoch, idempotency_key, readiness_review_key, release_version, channel, status, release_notes, released_by, promoted_from_release_id, readiness_goal_ref, approved_at_commit, verification_evidence, created_at, published_at, updated_at FROM releases`, backupTable)); err != nil {
		return rollback(fmt.Errorf("copy releases before identity migration: %w", err))
	}
	if _, err := tx.ExecContext(ctx, `DROP TABLE releases`); err != nil {
		return rollback(fmt.Errorf("replace releases table for identity migration: %w", err))
	}
	if _, err := tx.ExecContext(ctx, releaseTableWithoutCommitUnique); err != nil {
		return rollback(fmt.Errorf("create migrated releases table: %w", err))
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO releases (id, profile_id, deployment_id, profile_version, git_commit_hash, artifact_digest, candidate_id, destination_revision_id, authorization_epoch, idempotency_key, readiness_review_key, release_version, channel, status, release_notes, released_by, promoted_from_release_id, readiness_goal_ref, approved_at_commit, verification_evidence, created_at, published_at, updated_at) SELECT id, profile_id, deployment_id, profile_version, git_commit_hash, artifact_digest, candidate_id, destination_revision_id, authorization_epoch, idempotency_key, readiness_review_key, release_version, channel, status, release_notes, released_by, promoted_from_release_id, readiness_goal_ref, approved_at_commit, verification_evidence, created_at, published_at, updated_at FROM %s`, backupTable)); err != nil {
		return rollback(fmt.Errorf("restore releases after identity migration: %w", err))
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`DROP TABLE %s`, backupTable)); err != nil {
		return rollback(fmt.Errorf("remove release migration backup: %w", err))
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit release schema migration: %w", err)
	}
	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys = ON`); err != nil {
		return fmt.Errorf("restore foreign keys after release schema migration: %w", err)
	}
	foreignKeysRestored = true
	return nil
}

func legacyCommitUniqueColumns(ctx context.Context, db shared.RoutedDBTX) ([]string, error) {
	rows, err := db.QueryContext(ctx, `PRAGMA index_list('releases')`)
	if err != nil {
		return nil, fmt.Errorf("inspect release indexes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var seq, unique, partial int
		var name, origin string
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			return nil, fmt.Errorf("read release index: %w", err)
		}
		if unique != 1 {
			continue
		}
		quoted := strings.ReplaceAll(name, `'`, `''`)
		info, err := db.QueryContext(ctx, `PRAGMA index_info('`+quoted+`')`)
		if err != nil {
			return nil, fmt.Errorf("inspect release index %q: %w", name, err)
		}
		defer info.Close()
		var columns []string
		for info.Next() {
			var indexSeq, columnSeq int
			var column string
			if err := info.Scan(&indexSeq, &columnSeq, &column); err != nil {
				_ = info.Close()
				return nil, fmt.Errorf("read release index %q: %w", name, err)
			}
			columns = append(columns, column)
		}
		if err := info.Err(); err != nil {
			_ = info.Close()
			return nil, fmt.Errorf("iterate release index %q: %w", name, err)
		}
		_ = info.Close()
		if len(columns) == 3 && columns[0] == "profile_id" && columns[1] == "git_commit_hash" && columns[2] == "channel" {
			return columns, nil
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate release indexes: %w", err)
	}
	return nil, nil
}

const releaseTableWithoutCommitUnique = `CREATE TABLE releases (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    deployment_id TEXT,
    profile_version INTEGER,
    git_commit_hash TEXT NOT NULL,
    artifact_digest TEXT,
    candidate_id TEXT,
    destination_revision_id TEXT,
    authorization_epoch INTEGER NOT NULL DEFAULT 1,
    idempotency_key TEXT,
    readiness_review_key TEXT,
    release_version TEXT NOT NULL,
    channel TEXT NOT NULL DEFAULT 'stable',
    status TEXT NOT NULL DEFAULT 'pending',
    release_notes TEXT,
    released_by TEXT,
    promoted_from_release_id TEXT,
    readiness_goal_ref TEXT,
    approved_at_commit TEXT,
    verification_evidence TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    published_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
)`
