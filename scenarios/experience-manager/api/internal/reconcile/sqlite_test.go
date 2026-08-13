package reconcile

import (
	"context"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	testdb "github.com/vrooli/api-core/databasetest"
)

func TestSQLiteRepositorySavesAndListsEvidence(t *testing.T) {
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	want := Evidence{
		ID:             "ev-test",
		Scenario:       "demo",
		DocumentKind:   "component",
		PageID:         "home",
		ComponentID:    "button",
		ComponentTitle: "Button",
		ExampleName:    "primary",
		Route:          "/",
		StateID:        "default",
		ViewportID:     "mobile",
		ViewportWidth:  390,
		ViewportHeight: 844,
		ClaimID:        "primary-present",
		ClaimType:      "element-present",
		Verdict:        "passed",
		CaptureRef:     "scenario=demo,path=/",
		AXNodeJSON:     `{"role":"button"}`,
		Message:        "claim proven",
		CheckedAt:      "2026-07-05T12:00:00Z",
	}
	if err := repo.SaveEvidence(ctx, want); err != nil {
		t.Fatalf("SaveEvidence: %v", err)
	}
	got, err := repo.ListEvidence(ctx, EvidenceFilter{Scenario: "demo", PageID: "home", ClaimID: "primary-present"})
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("evidence rows = %d, want 1: %+v", len(got), got)
	}
	if got[0] != want {
		t.Fatalf("evidence = %+v, want %+v", got[0], want)
	}
}

func TestEnsureMigrationsAddsComponentIdentityToLegacyEvidence(t *testing.T) {
	db := testdb.NewSQLite(t)
	ctx := context.Background()
	_, err := db.ExecContext(ctx, `CREATE TABLE reconcile_evidence (id TEXT PRIMARY KEY, scenario TEXT NOT NULL, page_id TEXT NOT NULL, route TEXT NOT NULL, state_id TEXT NOT NULL, claim_id TEXT NOT NULL, claim_type TEXT NOT NULL, verdict TEXT NOT NULL, capture_ref TEXT NOT NULL, ax_node_json TEXT NOT NULL, message TEXT NOT NULL, checked_at TEXT NOT NULL)`)
	if err != nil {
		t.Fatal(err)
	}
	if err := EnsureMigrations(ctx, db); err != nil {
		t.Fatal(err)
	}
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(reconcile_evidence)`)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	found := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			t.Fatal(err)
		}
		found[name] = true
	}
	for _, name := range []string{"document_kind", "component_id", "component_title", "example_name"} {
		if !found[name] {
			t.Fatalf("missing migrated column %q", name)
		}
	}
}

func TestEnsureMigrationsSkipsFreshDatabase(t *testing.T) {
	db := testdb.NewSQLite(t)
	if err := EnsureMigrations(context.Background(), db); err != nil {
		t.Fatalf("EnsureMigrations on fresh database: %v", err)
	}
}
