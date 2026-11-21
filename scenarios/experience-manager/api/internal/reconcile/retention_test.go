package reconcile

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"
	testdb "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

func TestPruneSupersededEvidenceKeepsNewestPerClaimKeyAndRecentRows(t *testing.T) {
	db := testdb.NewSQLite(t)
	ctx := context.Background()
	if err := apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	save := func(id, viewport string, age time.Duration) {
		t.Helper()
		ev := Evidence{
			ID: id, Scenario: "demo", DocumentKind: "page", PageID: "home", Route: "/", StateID: "default",
			ViewportID: viewport, ViewportWidth: 390, ViewportHeight: 844, ClaimID: "cta", ClaimType: "element-present",
			Verdict: "passed", CaptureRef: "ref", AXNodeJSON: `{"role":"button"}`, Message: "ok",
			CheckedAt: now.Add(-age).Format(evidenceTimeFormat),
		}
		if err := repo.SaveEvidence(ctx, ev); err != nil {
			t.Fatalf("SaveEvidence %s: %v", id, err)
		}
	}
	// Mobile: six old rows; the newest three must survive regardless of age.
	for i := 0; i < 6; i++ {
		save(fmt.Sprintf("mobile-%d", i), "mobile", time.Duration(30+i)*day)
	}
	// Desktop: one old row is its claim key's only verdict and must survive
	// even though mobile has many newer rows.
	save("desktop-0", "desktop", 60*day)
	// Recent superseded rows inside MinAge are kept.
	for i := 0; i < 5; i++ {
		save(fmt.Sprintf("tablet-%d", i), "tablet", time.Duration(i)*time.Hour)
	}

	policy := EvidenceRetentionPolicy{KeepPerClaim: 3, MinAge: 14 * day, Batch: 2}
	res, err := PruneSupersededEvidence(ctx, db, db, policy, now)
	if err != nil {
		t.Fatalf("PruneSupersededEvidence: %v", err)
	}
	if res.DeletedRows != 3 || res.Candidates != 3 {
		t.Errorf("deleted=%d candidates=%d, want 3 and 3 (mobile-3..5)", res.DeletedRows, res.Candidates)
	}
	remaining := map[string]bool{}
	rows, err := db.QueryContext(ctx, `SELECT id FROM reconcile_evidence`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		remaining[id] = true
	}
	for _, id := range []string{"mobile-0", "mobile-1", "mobile-2", "desktop-0", "tablet-0", "tablet-4"} {
		if !remaining[id] {
			t.Errorf("%s was pruned; newest-per-key and recent rows must be kept", id)
		}
	}
	for _, id := range []string{"mobile-3", "mobile-4", "mobile-5"} {
		if remaining[id] {
			t.Errorf("%s survived; superseded old rows must be pruned", id)
		}
	}
	var orphans int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM reconcile_evidence_viewports WHERE evidence_id NOT IN (SELECT id FROM reconcile_evidence)`).Scan(&orphans); err != nil {
		t.Fatal(err)
	}
	if orphans != 0 {
		t.Errorf("orphaned viewport rows = %d, want 0", orphans)
	}
}

func TestPruneSupersededEvidenceRecheckRefusesStaleCandidates(t *testing.T) {
	db := testdb.NewSQLite(t)
	ctx := context.Background()
	if err := apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	ev := Evidence{ID: "only", Scenario: "demo", DocumentKind: "page", PageID: "home", Route: "/", StateID: "default", ClaimID: "cta", ClaimType: "x", Verdict: "passed", CaptureRef: "r", AXNodeJSON: "{}", Message: "m", CheckedAt: now.Add(-90 * 24 * time.Hour).Format(evidenceTimeFormat)}
	if err := repo.SaveEvidence(ctx, ev); err != nil {
		t.Fatal(err)
	}
	// A stale reader claimed "only" was superseded; the write-side re-check
	// must refuse because no newer verdict exists.
	deleted, err := deleteSupersededBatch(ctx, db, []string{"only"}, now.Format(evidenceTimeFormat), 1)
	if err != nil {
		t.Fatal(err)
	}
	if deleted != 0 {
		t.Fatal("re-check deleted a claim's only verdict")
	}
}

func TestPruneSupersededEvidenceReleasesPagesOnIncrementalFile(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "em.db")
	dsn, err := storage.SQLiteDSNAt(path, storage.SQLiteTuning{})
	if err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	db.SetMaxOpenConns(1)
	if err := apidb.EnsureSchemas(ctx, db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatal(err)
	}
	// A converted file: the pragma takes effect through a full VACUUM.
	for _, stmt := range []string{"PRAGMA auto_vacuum=INCREMENTAL", "VACUUM"} {
		if _, err := db.Exec(stmt); err != nil {
			t.Fatal(err)
		}
	}
	var mode int
	if err := db.QueryRow("PRAGMA auto_vacuum").Scan(&mode); err != nil || mode != 2 {
		t.Fatalf("setup auto_vacuum = %d (%v), want 2", mode, err)
	}
	repo := NewSQLiteRepository(db)
	now := time.Date(2026, 10, 28, 12, 0, 0, 0, time.UTC)
	big := strings.Repeat("a", 64*1024)
	for i := 0; i < 20; i++ {
		ev := Evidence{ID: fmt.Sprintf("ev-%02d", i), Scenario: "demo", DocumentKind: "page", PageID: "home", Route: "/", StateID: "s", ClaimID: "c", ClaimType: "x", Verdict: "passed", CaptureRef: "r", AXNodeJSON: big, Message: "m", CheckedAt: now.Add(-time.Duration(30+i) * 24 * time.Hour).Format(evidenceTimeFormat)}
		if err := repo.SaveEvidence(ctx, ev); err != nil {
			t.Fatal(err)
		}
	}
	res, err := PruneSupersededEvidence(ctx, db, db, EvidenceRetentionPolicy{KeepPerClaim: 3, MinAge: 14 * 24 * time.Hour, Batch: 5}, now)
	if err != nil {
		t.Fatal(err)
	}
	if res.DeletedRows != 17 || res.ReleasedBytes < int64(len(big))*10 {
		t.Errorf("deleted=%d released=%d; want 17 rows and their pages returned", res.DeletedRows, res.ReleasedBytes)
	}
	var free int64
	if err := db.QueryRow("PRAGMA freelist_count").Scan(&free); err != nil {
		t.Fatal(err)
	}
	if free != 0 {
		t.Errorf("freelist_count = %d after prune, want 0", free)
	}
}
