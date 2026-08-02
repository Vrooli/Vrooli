package database

import (
	"context"
	"database/sql"
	"reflect"
	"sort"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

// The verbatim search-hub registry DDL — the schema whose evolution (adding
// control_token) exposed the silent-no-op trap this drift check guards against.
const providersDDL = `
-- Registry table — owned by internal/registry/.
CREATE TABLE IF NOT EXISTS providers (
  provider_id    TEXT PRIMARY KEY,
  provider_group TEXT NOT NULL DEFAULT '',
  bucket         INTEGER NOT NULL DEFAULT 0,
  type           TEXT NOT NULL DEFAULT '',
  state          INTEGER NOT NULL DEFAULT 0,
  scope          INTEGER NOT NULL DEFAULT 0,
  descriptor     TEXT NOT NULL,
  control_token  TEXT NOT NULL DEFAULT '',
  created_at     TEXT NOT NULL,
  updated_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_providers_group  ON providers(provider_group);
CREATE INDEX IF NOT EXISTS idx_providers_bucket ON providers(bucket);
`

func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

func TestDeclaredColumns_ProvidersDDL(t *testing.T) {
	got := DeclaredColumns(providersDDL)
	cols, ok := got["providers"]
	if !ok {
		t.Fatalf("expected a 'providers' entry, got tables %v", keysOf(got))
	}
	want := []string{
		"provider_id", "provider_group", "bucket", "type", "state",
		"scope", "descriptor", "control_token", "created_at", "updated_at",
	}
	if !reflect.DeepEqual(sortedCopy(cols), sortedCopy(want)) {
		t.Fatalf("columns mismatch:\n got %v\nwant %v", sortedCopy(cols), sortedCopy(want))
	}
	if len(got) != 1 {
		t.Fatalf("CREATE INDEX must not be parsed as a table; got tables %v", keysOf(got))
	}
}

func TestDeclaredColumns_SkipsTableConstraintsAndHandlesTrickyDefs(t *testing.T) {
	// Exercises: within-domain FK (REFERENCES with parens), NUMERIC(10,2)
	// column with a comma inside parens, a string default containing a paren,
	// a CHECK column constraint, quoted identifier, and table-level PRIMARY
	// KEY / FOREIGN KEY / UNIQUE / CONSTRAINT lines that must be excluded.
	const ddl = `
CREATE TABLE order_items (
    id        TEXT PRIMARY KEY,
    order_id  TEXT NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    price     NUMERIC(10,2) NOT NULL DEFAULT 0,
    label     TEXT NOT NULL DEFAULT '(none)',
    qty       INTEGER NOT NULL CHECK (qty > 0),
    "order"   TEXT,
    UNIQUE (order_id, id),
    PRIMARY KEY (id),
    FOREIGN KEY (order_id) REFERENCES orders(id),
    CONSTRAINT chk_price CHECK (price >= 0)
);`
	got := DeclaredColumns(ddl)["order_items"]
	want := []string{"id", "order_id", "price", "label", "qty", "order"}
	if !reflect.DeepEqual(sortedCopy(got), sortedCopy(want)) {
		t.Fatalf("columns mismatch:\n got %v\nwant %v", sortedCopy(got), sortedCopy(want))
	}
}

func TestDeclaredColumns_MultipleTablesAndIfNotExists(t *testing.T) {
	const ddl = `
CREATE TABLE IF NOT EXISTS a (x TEXT, y INTEGER);
CREATE TABLE b (z TEXT);`
	got := DeclaredColumns(ddl)
	if !reflect.DeepEqual(sortedCopy(got["a"]), []string{"x", "y"}) {
		t.Fatalf("table a: got %v", got["a"])
	}
	if !reflect.DeepEqual(got["b"], []string{"z"}) {
		t.Fatalf("table b: got %v", got["b"])
	}
}

func TestDeclaredColumns_Empty(t *testing.T) {
	if got := DeclaredColumns(""); len(got) != 0 {
		t.Fatalf("empty schema must yield no tables; got %v", got)
	}
	if got := DeclaredColumns("-- just a comment\nCREATE INDEX IF NOT EXISTS i ON t(c);"); len(got) != 0 {
		t.Fatalf("no CREATE TABLE must yield no tables; got %v", got)
	}
}

func keysOf(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// --- Integration tests against a real in-memory SQLite DB ---

func openMemSQLite(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// TestEnsureSchemas_DriftDetectedOnExistingTable reproduces the search-hub bug:
// a DB created before a column was added does NOT gain it from CREATE TABLE IF
// NOT EXISTS, and EnsureSchemas must now fail loudly rather than silently.
func TestEnsureSchemas_DriftDetectedOnExistingTable(t *testing.T) {
	db := openMemSQLite(t)
	ctx := context.Background()

	// Old DB: providers without control_token.
	oldSchema := SchemaProviderFunc(func() string {
		return `CREATE TABLE IF NOT EXISTS providers (
			provider_id TEXT PRIMARY KEY,
			descriptor  TEXT NOT NULL
		);`
	})
	if err := EnsureSchemas(ctx, db, oldSchema); err != nil {
		t.Fatalf("initial apply: %v", err)
	}

	// New code adds control_token to the CREATE TABLE block.
	newSchema := SchemaProviderFunc(func() string {
		return `CREATE TABLE IF NOT EXISTS providers (
			provider_id   TEXT PRIMARY KEY,
			descriptor    TEXT NOT NULL,
			control_token TEXT NOT NULL DEFAULT ''
		);`
	})
	err := EnsureSchemas(ctx, db, newSchema)
	if err == nil {
		t.Fatal("expected drift error for missing control_token, got nil")
	}
	if !strings.Contains(err.Error(), "providers.control_token") {
		t.Fatalf("error should name the missing column; got: %v", err)
	}
	if !strings.Contains(err.Error(), "migration") {
		t.Fatalf("error should point at the migration fix; got: %v", err)
	}
}

// TestEnsureSchemas_NoDriftOnFreshOrMatchingDB confirms the check is silent
// when the live table already has every declared column (fresh DB, or after a
// migration brought it into shape) — no false positives.
func TestEnsureSchemas_NoDriftOnFreshOrMatchingDB(t *testing.T) {
	db := openMemSQLite(t)
	ctx := context.Background()
	schema := SchemaProviderFunc(func() string { return providersDDL })

	// Fresh DB: table created with all columns this run.
	if err := EnsureSchemas(ctx, db, schema); err != nil {
		t.Fatalf("fresh apply must not drift: %v", err)
	}
	// Re-apply (idempotent boot): still no drift.
	if err := EnsureSchemas(ctx, db, schema); err != nil {
		t.Fatalf("re-apply must not drift: %v", err)
	}
}

// TestEnsureSchemas_MigrationClearsDrift confirms that once a one-shot
// migration adds the missing column, EnsureSchemas passes — the documented
// recovery path.
func TestEnsureSchemas_MigrationClearsDrift(t *testing.T) {
	db := openMemSQLite(t)
	ctx := context.Background()

	if err := EnsureSchemas(ctx, db, SchemaProviderFunc(func() string {
		return `CREATE TABLE IF NOT EXISTS providers (provider_id TEXT PRIMARY KEY, descriptor TEXT NOT NULL);`
	})); err != nil {
		t.Fatalf("initial apply: %v", err)
	}

	// The one-shot migration a developer would run with the scenario stopped.
	if _, err := db.ExecContext(ctx, `ALTER TABLE providers ADD COLUMN control_token TEXT NOT NULL DEFAULT '';`); err != nil {
		t.Fatalf("migration: %v", err)
	}

	if err := EnsureSchemas(ctx, db, SchemaProviderFunc(func() string {
		return `CREATE TABLE IF NOT EXISTS providers (
			provider_id   TEXT PRIMARY KEY,
			descriptor    TEXT NOT NULL,
			control_token TEXT NOT NULL DEFAULT ''
		);`
	})); err != nil {
		t.Fatalf("post-migration apply must pass: %v", err)
	}
}

// TestEnsureSchemas_NonQuerierSkipsCheck confirms a write-only execer (no
// QueryContext) keeps the original apply-only behavior — verification is
// best-effort and never required.
func TestEnsureSchemas_NonQuerierSkipsCheck(t *testing.T) {
	f := &fakeExecer{}
	// providersDDL adds columns a bare CREATE TABLE no-op would miss, but the
	// fake can't be queried, so no drift error is possible.
	if err := EnsureSchemas(context.Background(), f, SchemaProviderFunc(func() string { return providersDDL })); err != nil {
		t.Fatalf("non-querier execer must skip the drift check: %v", err)
	}
}

// TestGeneratedColumnIsNotReportedAsDrift locks in the table_xinfo fix.
//
// PRAGMA table_info omits generated columns, so verifying against it reported a
// freshly created table as missing its own generated column on every boot — and
// the remediation text pointed at an ADD COLUMN migration that SQLite rejects
// for generated columns.
func TestGeneratedColumnIsNotReportedAsDrift(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()

	schema := `CREATE TABLE IF NOT EXISTS metrics (
    id TEXT PRIMARY KEY,
    a REAL,
    b REAL,
    avg_ab REAL GENERATED ALWAYS AS ((a + b) / 2) STORED,
    virt REAL GENERATED ALWAYS AS (a * 2) VIRTUAL
);`

	for i := range 2 {
		if err := EnsureSchemas(context.Background(), db, SchemaProviderFunc(func() string { return schema })); err != nil {
			t.Fatalf("apply %d: generated columns must not read as drift: %v", i+1, err)
		}
	}
}
