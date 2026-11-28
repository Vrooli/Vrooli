package releases

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	internalreleases "deployment-manager/internal/releases"
	_ "modernc.org/sqlite"
)

func TestListOperationIDsByProfileKeysetPagesRemainStableAndProfileScoped(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "operation-index.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = db.Close() }()
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	base := time.Date(2026, 11, 24, 12, 0, 0, 0, time.UTC)
	rows := []struct {
		id, release, profile string
		created              time.Time
	}{
		{"op-a", "rel-a", "profile-a", base.Add(-3 * time.Minute)},
		{"op-b", "rel-b", "profile-a", base.Add(-2 * time.Minute)},
		{"op-c", "rel-c", "profile-a", base.Add(-2 * time.Minute)}, // ID is the deterministic tie-breaker.
		{"op-d", "rel-d", "profile-a", base.Add(-time.Minute)},
		{"op-x", "rel-x", "profile-x", base},
	}
	for _, row := range rows {
		if _, err := db.Exec(`INSERT INTO releases (id, profile_id, git_commit_hash, release_version, channel) VALUES ($1, $2, 'commit', '1.0.0', 'stable')`, row.release, row.profile); err != nil {
			t.Fatal(err)
		}
		if _, err := db.Exec(`INSERT INTO release_operations (id, release_id, profile_id, status, created_at, updated_at) VALUES ($1, $2, $3, 'queued', $4, $4)`, row.id, row.release, row.profile, row.created); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.Exec(`INSERT INTO releases (id, profile_id, git_commit_hash, release_version, channel) VALUES ('rel-mismatch', 'profile-x', 'commit', '1.0.0', 'stable')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO release_operations (id, release_id, profile_id, status, created_at, updated_at) VALUES ('op-mismatch', 'rel-mismatch', 'profile-a', 'queued', $1, $1)`, base.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	repo := NewSQLRepository(db)
	first, err := repo.ListOperationIDsByProfile(ctx, "profile-a", "", 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(first.OperationIDs) != 2 || first.OperationIDs[0] != "op-d" || first.OperationIDs[1] != "op-c" || first.NextPageToken == "" {
		t.Fatalf("first page = %+v", first)
	}
	if _, err := db.Exec(`INSERT INTO releases (id, profile_id, git_commit_hash, release_version, channel) VALUES ('rel-new', 'profile-a', 'commit', '1.0.0', 'stable')`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO release_operations (id, release_id, profile_id, status, created_at, updated_at) VALUES ('op-new', 'rel-new', 'profile-a', 'queued', $1, $1)`, base); err != nil {
		t.Fatal(err)
	}

	// Reopen the file-backed database to model process restart; the opaque cursor
	// remains sufficient because it binds only immutable persisted key fields.
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	db, err = sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	repo = NewSQLRepository(db)
	second, err := repo.ListOperationIDsByProfile(ctx, "profile-a", first.NextPageToken, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.OperationIDs) != 2 || second.OperationIDs[0] != "op-b" || second.OperationIDs[1] != "op-a" || second.NextPageToken != "" {
		t.Fatalf("second page = %+v", second)
	}
	seen := map[string]bool{}
	for _, id := range append(first.OperationIDs, second.OperationIDs...) {
		if seen[id] {
			t.Fatalf("duplicate operation across pages: %s", id)
		}
		seen[id] = true
	}
	other, err := repo.ListOperationIDsByProfile(ctx, "profile-x", "", 10)
	if err != nil || len(other.OperationIDs) != 1 || other.OperationIDs[0] != "op-x" {
		t.Fatalf("other profile page = %+v, %v", other, err)
	}
	if _, err := repo.ListOperationIDsByProfile(ctx, "profile-x", first.NextPageToken, 2); !errors.Is(err, ErrInvalidOperationPageToken) {
		t.Fatalf("cross-profile cursor error = %v", err)
	}
	if _, err := repo.ListOperationIDsByProfile(ctx, "profile-a", "not-a-cursor", 2); !errors.Is(err, ErrInvalidOperationPageToken) {
		t.Fatalf("malformed cursor error = %v", err)
	}
	var status string
	if err := db.QueryRow(`SELECT status FROM release_operations WHERE id = 'op-d'`).Scan(&status); err != nil || status != OperationQueued {
		t.Fatalf("read changed operation status: %q, %v", status, err)
	}
}

func TestListOperationIDsByProfileUsesDefaultAndRejectsInvalidSize(t *testing.T) {
	db, err := sql.Open("sqlite", "file:operation-index-size-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(internalreleases.Schema()); err != nil {
		t.Fatal(err)
	}
	page, err := NewSQLRepository(db).ListOperationIDsByProfile(context.Background(), "profile-empty", "", 0)
	if err != nil || len(page.OperationIDs) != 0 || page.NextPageToken != "" {
		t.Fatalf("empty page = %+v, %v", page, err)
	}
	for _, size := range []int{-1, 101} {
		if _, err := NewSQLRepository(db).ListOperationIDsByProfile(context.Background(), "profile-empty", "", size); err == nil {
			t.Fatalf("accepted invalid page size %d", size)
		}
	}
}
