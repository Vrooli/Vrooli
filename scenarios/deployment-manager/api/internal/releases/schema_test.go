package releases

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

func TestSchemaIsTheReleaseDomainSchema(t *testing.T) {
	if len(Schema()) == 0 {
		t.Fatal("release schema is empty")
	}
}

func TestLegacyReleaseSchemaReconcilesBeforeDependentIndex(t *testing.T) {
	db, err := sql.Open("sqlite", "file:release-schema-compat?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	legacy := `CREATE TABLE releases (
        id TEXT PRIMARY KEY,
        profile_id TEXT NOT NULL,
        deployment_id TEXT,
        profile_version INTEGER,
        git_commit_hash TEXT NOT NULL,
        release_version TEXT NOT NULL,
        channel TEXT NOT NULL DEFAULT 'stable',
        status TEXT NOT NULL DEFAULT 'pending',
        release_notes TEXT,
        released_by TEXT,
        promoted_from_release_id TEXT,
        verification_evidence TEXT,
        created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        published_at TIMESTAMP,
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
    );`
	if _, err := db.ExecContext(context.Background(), legacy); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}

	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("reconcile legacy schema: %v", err)
	}
	if err := database.ApplySchemas(context.Background(), db, database.SchemaProviderFunc(IndexesSchema)); err != nil {
		t.Fatalf("apply dependent indexes after reconciliation: %v", err)
	}

	if _, err := db.ExecContext(context.Background(), `SELECT idempotency_key FROM releases`); err != nil {
		t.Fatalf("reconciled idempotency_key is unavailable: %v", err)
	}
	rows, err := db.QueryContext(context.Background(), `PRAGMA index_list('releases')`)
	if err != nil {
		t.Fatalf("list release indexes: %v", err)
	}
	defer rows.Close()
	var found bool
	for rows.Next() {
		var seq int
		var name string
		var unique, partial int
		var origin string
		if err := rows.Scan(&seq, &name, &unique, &origin, &partial); err != nil {
			t.Fatalf("scan release index: %v", err)
		}
		if strings.EqualFold(name, "idx_releases_idempotency") {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate release indexes: %v", err)
	}
	if !found {
		t.Fatal("dependent idempotency index was not created")
	}
}

func TestLegacyCommitUniqueMigrationPreservesHistoryAndAllowsNewCandidate(t *testing.T) {
	db, err := sql.Open("sqlite", "file:release-schema-identity-migration?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	legacy := `CREATE TABLE releases (
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
        updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
        UNIQUE(profile_id, git_commit_hash, channel)
    );`
	if _, err := db.Exec(legacy); err != nil {
		t.Fatalf("create legacy schema: %v", err)
	}
	if err := database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("reconcile legacy schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO releases (id, profile_id, git_commit_hash, artifact_digest, candidate_id, destination_revision_id, release_version, channel) VALUES ('legacy-release', 'p1', 'commit-1', 'sha256:old', 'candidate-old', 'destination-1', '1.0.0', 'stable')`); err != nil {
		t.Fatalf("insert legacy release: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO release_platforms (release_id, platform, status) VALUES ('legacy-release', 'linux-x64', 'published')`); err != nil {
		t.Fatalf("insert legacy platform: %v", err)
	}

	if err := MigrateLegacyReleaseConstraints(context.Background(), db); err != nil {
		t.Fatalf("migrate legacy uniqueness: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO releases (id, profile_id, git_commit_hash, artifact_digest, candidate_id, destination_revision_id, release_version, channel) VALUES ('new-release', 'p1', 'commit-1', 'sha256:new', 'candidate-new', 'destination-1', '1.1.0', 'stable')`); err != nil {
		t.Fatalf("insert same-commit new candidate: %v", err)
	}
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM releases WHERE profile_id = 'p1' AND git_commit_hash = 'commit-1'`).Scan(&count); err != nil {
		t.Fatalf("count migrated releases: %v", err)
	}
	if count != 2 {
		t.Fatalf("migrated release history count = %d, want 2", count)
	}
	var platforms int
	if err := db.QueryRow(`SELECT COUNT(*) FROM release_platforms WHERE release_id = 'legacy-release'`).Scan(&platforms); err != nil {
		t.Fatalf("count migrated platform rows: %v", err)
	}
	if platforms != 1 {
		t.Fatalf("migrated platform history count = %d, want 1", platforms)
	}
}

func TestReviewBindingsRequireStoredReleaseIdentities(t *testing.T) {
	db, err := sql.Open("sqlite", "file:release-schema-review-fk?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO release_review_bindings
		(review_id, canonical_json, candidate_id, destination_revision_id, status)
		VALUES ('review-1', '{}', 'missing-candidate', 'missing-destination', 'approved')`); err == nil {
		t.Fatal("review binding accepted missing candidate and destination identities")
	}
}
