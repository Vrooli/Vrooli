package database_test

import (
	"context"
	"testing"
	"testing/fstest"

	db "github.com/vrooli/api-core/databasetest"

	"image-tools/internal/database"
)

func TestMigrateDropsTemplateTables(t *testing.T) {
	dbc := db.NewSQLite(t)
	ctx := context.Background()

	for _, name := range []string{"attachments", "notes"} {
		if _, err := dbc.ExecContext(ctx, "CREATE TABLE "+name+" (id TEXT PRIMARY KEY)"); err != nil {
			t.Fatalf("create %s: %v", name, err)
		}
	}

	if err := database.Migrate(ctx, dbc); err != nil {
		t.Fatalf("Migrate: %v", err)
	}

	for _, name := range []string{"attachments", "notes"} {
		var n int
		if err := dbc.QueryRowContext(ctx,
			`SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name=?`, name).Scan(&n); err != nil {
			t.Fatalf("query %s: %v", name, err)
		}
		if n != 0 {
			t.Errorf("table %q still exists after migration", name)
		}
	}

	var applied int
	if err := dbc.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations`).Scan(&applied); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if applied == 0 {
		t.Error("migration was not recorded in schema_migrations")
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	dbc := db.NewSQLite(t)
	ctx := context.Background()

	if err := database.Migrate(ctx, dbc); err != nil {
		t.Fatalf("first Migrate: %v", err)
	}
	var first int
	if err := dbc.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations`).Scan(&first); err != nil {
		t.Fatalf("count: %v", err)
	}
	if err := database.Migrate(ctx, dbc); err != nil {
		t.Fatalf("second Migrate: %v", err)
	}
	var second int
	if err := dbc.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations`).Scan(&second); err != nil {
		t.Fatalf("count: %v", err)
	}
	if second != first {
		t.Errorf("second run recorded more migrations: %d -> %d", first, second)
	}
}

func TestMigrationNamesAreOrderedAndNonEmpty(t *testing.T) {
	names, err := database.MigrationNames()
	if err != nil {
		t.Fatalf("MigrationNames: %v", err)
	}
	if len(names) == 0 {
		t.Fatal("no migrations embedded")
	}
	for i := 1; i < len(names); i++ {
		if names[i-1] >= names[i] {
			t.Fatalf("migration names not strictly ordered: %q then %q", names[i-1], names[i])
		}
	}
}

func TestMigrateFSRollsBackOnFailure(t *testing.T) {
	dbc := db.NewSQLite(t)
	ctx := context.Background()
	fsys := fstest.MapFS{
		"migrations/0001_bad.sql": &fstest.MapFile{
			Data: []byte("CREATE TABLE ok_table (id TEXT); THIS IS NOT SQL;"),
		},
	}
	if err := database.MigrateFS(ctx, dbc, fsys, "migrations"); err == nil {
		t.Fatal("expected the invalid migration to fail")
	}
	var n int
	if err := dbc.QueryRowContext(ctx, `SELECT COUNT(1) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatalf("count schema_migrations: %v", err)
	}
	if n != 0 {
		t.Errorf("failed migration was recorded (%d rows); rollback did not happen", n)
	}
}

// TestMigrateCarriesOverridesAcrossWithUnknownProvenance proves 0002 moves the
// pre-provenance model_state and adapter_state rows into the override tables
// without changing their effect, marks their author unknown and their seed
// baseline unknown, and drops the old tables.
func TestMigrateCarriesOverridesAcrossWithUnknownProvenance(t *testing.T) {
	dbc := db.NewSQLite(t)
	ctx := context.Background()
	for _, stmt := range []string{
		`CREATE TABLE model_state (id TEXT PRIMARY KEY, enabled INTEGER NOT NULL)`,
		`INSERT INTO model_state VALUES ('restoreformer-plus-plus', 0), ('flux-2-klein-4b', 1)`,
		`CREATE TABLE adapter_state (id TEXT PRIMARY KEY, enabled INTEGER NOT NULL)`,
		`INSERT INTO adapter_state VALUES ('lcm-lora-sdv1-5', 1)`,
	} {
		if _, err := dbc.ExecContext(ctx, stmt); err != nil {
			t.Fatalf("%s: %v", stmt, err)
		}
	}
	if err := database.Migrate(ctx, dbc); err != nil {
		t.Fatalf("Migrate: %v", err)
	}
	var enabled, seed int
	var setBy string
	if err := dbc.QueryRowContext(ctx, `SELECT enabled, set_by, seed_enabled FROM model_override WHERE id='restoreformer-plus-plus'`).Scan(&enabled, &setBy, &seed); err != nil {
		t.Fatalf("read carried override: %v", err)
	}
	if enabled != 0 || setBy != "unknown (before provenance)" || seed != -1 {
		t.Fatalf("carried override = enabled %d set_by %q seed %d", enabled, setBy, seed)
	}
	var n int
	if err := dbc.QueryRowContext(ctx, `SELECT COUNT(1) FROM model_override`).Scan(&n); err != nil || n != 2 {
		t.Fatalf("model_override rows = %d (%v)", n, err)
	}
	if err := dbc.QueryRowContext(ctx, `SELECT COUNT(1) FROM adapter_override`).Scan(&n); err != nil || n != 1 {
		t.Fatalf("adapter_override rows = %d (%v)", n, err)
	}
	for _, old := range []string{"model_state", "adapter_state"} {
		if err := dbc.QueryRowContext(ctx, `SELECT COUNT(1) FROM sqlite_master WHERE type='table' AND name=?`, old).Scan(&n); err != nil || n != 0 {
			t.Fatalf("%s still exists after migration", old)
		}
	}
}
