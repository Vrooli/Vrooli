package database_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	db "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	apidb "github.com/vrooli/api-core/database"

	localdb "nutrition-planner/internal/database"
	"nutrition-planner/internal/modules"
	"nutrition-planner/internal/planning"
	"nutrition-planner/internal/recipe"
	"nutrition-planner/internal/workspace"
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

func TestInitializeDatabaseUpgradesPopulatedPredecessorAndReopensIdempotently(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "predecessor.sqlite")
	open := func() *sql.DB {
		db, err := sql.Open("sqlite", path)
		require.NoError(t, err)
		db.SetMaxOpenConns(1)
		return db
	}
	db := open()
	_, err := db.Exec(`
CREATE TABLE workspaces (id TEXT PRIMARY KEY, name TEXT NOT NULL, owner_subject TEXT NOT NULL, revision INTEGER NOT NULL DEFAULT 1, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE recipes (id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, current_revision INTEGER NOT NULL, created_at TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE recipe_revisions (recipe_id TEXT NOT NULL, revision INTEGER NOT NULL, workspace_id TEXT NOT NULL, name TEXT NOT NULL, notes TEXT NOT NULL DEFAULT '', source_url TEXT NOT NULL DEFAULT '', source_type TEXT NOT NULL DEFAULT '', original_text TEXT NOT NULL DEFAULT '', status TEXT NOT NULL, methods_json TEXT NOT NULL DEFAULT '[]', groups_json TEXT NOT NULL DEFAULT '[]', required_appliances_json TEXT NOT NULL DEFAULT '[]', allergen_evidence_json TEXT NOT NULL DEFAULT '{}', canonical_yield TEXT NOT NULL DEFAULT '', serving_unit TEXT NOT NULL DEFAULT 'serving', ingredients_json TEXT NOT NULL DEFAULT '[]', created_at TEXT NOT NULL, PRIMARY KEY(recipe_id,revision));
CREATE TABLE plans (workspace_id TEXT PRIMARY KEY, revision INTEGER NOT NULL, plan_json TEXT NOT NULL, updated_at TEXT NOT NULL);
CREATE TABLE shopping_checks (workspace_id TEXT NOT NULL, line_key TEXT NOT NULL, checked INTEGER NOT NULL DEFAULT 0, updated_at TEXT NOT NULL, PRIMARY KEY(workspace_id,line_key));
CREATE TABLE inventory_batches (workspace_id TEXT NOT NULL, batch_id TEXT NOT NULL, recipe_id TEXT NOT NULL DEFAULT '', recipe_revision INTEGER NOT NULL DEFAULT 0, yield_amount TEXT NOT NULL, available_amount TEXT NOT NULL, unit TEXT NOT NULL, PRIMARY KEY(workspace_id,batch_id));
INSERT INTO workspaces VALUES ('w1','Home','subject-1',3,'2026-10-01T00:00:00Z','2026-10-02T00:00:00Z');
INSERT INTO recipes VALUES ('r1','w1',2,'2026-10-01T00:00:00Z','2026-10-02T00:00:00Z');
INSERT INTO recipe_revisions(recipe_id,revision,workspace_id,name,status,created_at) VALUES ('r1',1,'w1','Soup old','active','2026-10-01T00:00:00Z'),('r1',2,'w1','Soup current','active','2026-10-02T00:00:00Z');
INSERT INTO shopping_checks VALUES ('w1','ingredient:rice',1,'checked-at');
INSERT INTO inventory_batches VALUES ('w1','batch-1','r1',1,'unknown','unknown','serving');
`)
	require.NoError(t, err)
	legacyPlan := `{"occurrences":[{"date":"2026-10-03","slotName":"dinner","mode":"fixed","quantity":"unknown","recipeId":"r1","recipeRevision":1,"recipeName":"Soup old","reason":"historical","locked":true}],"unresolved":[{"date":"2026-10-04","code":"missing_amount","message":"amount remains unknown"}],"inputReferences":["recipe:r1:1","recipe:r1:2","catalog:future:value"],"runId":"legacy-run","seed":41}`
	_, err = db.Exec(`INSERT INTO plans VALUES(?,?,?,?)`, "w1", 7, legacyPlan, "legacy-updated")
	require.NoError(t, err)
	require.NoError(t, modules.InitializeDatabase(ctx, db))
	require.NoError(t, modules.InitializeDatabase(ctx, db))
	require.NoError(t, db.Close())

	db = open()
	defer db.Close()
	var schemaVersion int64
	require.NoError(t, db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&schemaVersion))
	require.EqualValues(t, modules.NativeSchemaVersion, schemaVersion)
	workspaces := workspace.NewSQLiteRepository(db, schedule.System())
	gotWorkspace, err := workspaces.Get(ctx, "w1", "subject-1")
	require.NoError(t, err)
	require.Equal(t, "subject-1", gotWorkspace.OwnerSubject)
	recipes := recipe.NewSQLiteRepository(db, schedule.System())
	oldRecipe, err := recipes.GetRevision(ctx, "r1", "w1", 1)
	require.NoError(t, err)
	require.Equal(t, "Soup old", oldRecipe.Name)
	var revisionCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM recipe_revisions WHERE recipe_id='r1'`).Scan(&revisionCount))
	require.Equal(t, 2, revisionCount)
	var checked int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT checked FROM shopping_checks WHERE workspace_id='w1' AND line_key='ingredient:rice'`).Scan(&checked))
	require.Equal(t, 1, checked)
	var available string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT available_amount FROM inventory_batches WHERE workspace_id='w1' AND batch_id='batch-1'`).Scan(&available))
	require.Equal(t, "unknown", available)
	var archived string
	require.NoError(t, db.QueryRowContext(ctx, `SELECT plan_json FROM plans WHERE workspace_id='w1'`).Scan(&archived))
	require.Equal(t, legacyPlan, archived)
	repo := planning.NewSQLiteRepository(db, schedule.System())
	revision, raw, err := repo.Get(ctx, "w1")
	require.NoError(t, err)
	require.EqualValues(t, 7, revision)
	var migrated planning.Draft
	require.NoError(t, json.Unmarshal([]byte(raw), &migrated))
	require.Equal(t, "unknown", migrated.Occurrences[0].Quantity)
	require.EqualValues(t, 1, migrated.Occurrences[0].RecipeRevision)
	require.Equal(t, "catalog:future:value", migrated.InputReferences[2])
	require.Equal(t, "missing_amount", migrated.Unresolved[0].Code)
	var migrationCount int
	require.NoError(t, db.QueryRowContext(ctx, `SELECT count(*) FROM plan_migrations WHERE name='legacy-plan-json-to-occurrences-v1'`).Scan(&migrationCount))
	require.Equal(t, 1, migrationCount)
}

func TestInitializeDatabaseMalformedLegacyPlanLeavesMigrationStateRetryable(t *testing.T) {
	db := db.NewSQLite(t)
	_, err := db.Exec(`CREATE TABLE plans(workspace_id TEXT PRIMARY KEY, revision INTEGER NOT NULL, plan_json TEXT NOT NULL, updated_at TEXT NOT NULL);
INSERT INTO plans VALUES ('a',1,'{"runId":"good","occurrences":[{"date":"2026-10-03","recipeId":"r1"}]}','now'),('z',1,'not-json','now');`)
	require.NoError(t, err)
	err = modules.InitializeDatabase(context.Background(), db)
	require.Error(t, err)
	var version int64
	require.NoError(t, db.QueryRow(`PRAGMA user_version`).Scan(&version))
	require.Zero(t, version)
	for _, table := range []string{"plan_state", "plan_occurrences", "plan_migrations"} {
		var count int
		require.NoError(t, db.QueryRow(`SELECT count(*) FROM `+table).Scan(&count))
		require.Zero(t, count, table)
	}
	var sources int
	require.NoError(t, db.QueryRow(`SELECT count(*) FROM plans`).Scan(&sources))
	require.Equal(t, 2, sources)
	_, err = db.Exec(`UPDATE plans SET plan_json='{"runId":"fixed","occurrences":[]}' WHERE workspace_id='z'`)
	require.NoError(t, err)
	require.NoError(t, modules.InitializeDatabase(context.Background(), db))
	require.NoError(t, db.QueryRow(`PRAGMA user_version`).Scan(&version))
	require.EqualValues(t, modules.NativeSchemaVersion, version)
}
