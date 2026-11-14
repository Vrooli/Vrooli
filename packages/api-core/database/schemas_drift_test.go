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

// oldProvidersDDL / newProvidersDDL model the search-hub bug: a DB created
// before control_token existed does not gain it from CREATE TABLE IF NOT
// EXISTS, because that statement is a no-op once the table is there.
const oldProvidersDDL = `CREATE TABLE IF NOT EXISTS providers (
	provider_id TEXT PRIMARY KEY,
	descriptor  TEXT NOT NULL
);`

const newProvidersDDL = `CREATE TABLE IF NOT EXISTS providers (
	provider_id   TEXT PRIMARY KEY,
	descriptor    TEXT NOT NULL,
	control_token TEXT NOT NULL DEFAULT ''
);`

// TestEnsureSchemas_AdditiveDriftIsRepaired covers the headline behavior: a
// declared column missing from a pre-existing table is added automatically, by
// replaying the column's own definition, rather than failing boot.
//
// Declaring a column used to be only half the job — the other half was a
// hand-written ALTER that had to run at exactly the right point in boot, and
// getting that wrong took the whole API down. Adding a column is now a
// one-place change again.
func TestEnsureSchemas_AdditiveDriftIsRepaired(t *testing.T) {
	db := openMemSQLite(t)
	ctx := context.Background()

	if err := EnsureSchemas(ctx, db, SchemaProviderFunc(func() string { return oldProvidersDDL })); err != nil {
		t.Fatalf("initial apply: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO providers (provider_id, descriptor) VALUES ('p1', 'desc')`); err != nil {
		t.Fatalf("seed row: %v", err)
	}

	if err := EnsureSchemas(ctx, db, SchemaProviderFunc(func() string { return newProvidersDDL })); err != nil {
		t.Fatalf("additive drift must be repaired, not reported: %v", err)
	}

	// The column landed with its declared default, and the existing row survived.
	var descriptor, token string
	if err := db.QueryRowContext(ctx,
		`SELECT descriptor, control_token FROM providers WHERE provider_id = 'p1'`,
	).Scan(&descriptor, &token); err != nil {
		t.Fatalf("read repaired row: %v", err)
	}
	if descriptor != "desc" {
		t.Errorf("existing data lost: descriptor = %q", descriptor)
	}
	if token != "" {
		t.Errorf("control_token = %q, want the declared default", token)
	}

	// Idempotent: the next boot has nothing left to do.
	if err := EnsureSchemas(ctx, db, SchemaProviderFunc(func() string { return newProvidersDDL })); err != nil {
		t.Fatalf("re-apply after repair: %v", err)
	}
}

// TestVerifyDeclaredColumns_ReportsWithoutWriting keeps the read-only escape
// hatch honest: it must still detect the drift AND leave the database alone,
// for callers that require every schema change to be deliberate.
func TestVerifyDeclaredColumns_ReportsWithoutWriting(t *testing.T) {
	db := openMemSQLite(t)
	ctx := context.Background()

	if err := ApplySchemas(ctx, db, SchemaProviderFunc(func() string { return oldProvidersDDL })); err != nil {
		t.Fatalf("initial apply: %v", err)
	}

	newSchema := SchemaProviderFunc(func() string { return newProvidersDDL })
	err := VerifyDeclaredColumns(ctx, db, newSchema)
	if err == nil {
		t.Fatal("expected drift error for missing control_token, got nil")
	}
	if !strings.Contains(err.Error(), "providers.control_token") {
		t.Fatalf("error should name the missing column; got: %v", err)
	}
	if !strings.Contains(err.Error(), "migration") {
		t.Fatalf("error should point at the migration fix; got: %v", err)
	}

	// Still absent — verification must not have repaired anything.
	if _, err := db.ExecContext(ctx, `SELECT control_token FROM providers`); err == nil {
		t.Fatal("VerifyDeclaredColumns wrote to the database; it must be read-only")
	}
}

// TestReconcile_LeavesUnaddableColumnsToTheOperator pins the safety boundary.
// SQLite cannot add these to an existing table, so the check must say so
// precisely instead of issuing a statement it knows will fail.
func TestReconcile_LeavesUnaddableColumnsToTheOperator(t *testing.T) {
	for _, tc := range []struct {
		name   string
		column string
		reason string
	}{
		{"unique", "tag TEXT UNIQUE", "UNIQUE"},
		{"not null without default", "tag TEXT NOT NULL", "NOT NULL"},
		{"non-constant default", "seen_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP", "non-constant"},
		{"foreign key", "owner_id TEXT NOT NULL DEFAULT '' REFERENCES providers(provider_id)", "foreign-key"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			db := openMemSQLite(t)
			ctx := context.Background()
			if err := EnsureSchemas(ctx, db, SchemaProviderFunc(func() string { return oldProvidersDDL })); err != nil {
				t.Fatalf("initial apply: %v", err)
			}
			err := EnsureSchemas(ctx, db, SchemaProviderFunc(func() string {
				return "CREATE TABLE IF NOT EXISTS providers (provider_id TEXT PRIMARY KEY, descriptor TEXT NOT NULL, " + tc.column + ");"
			}))
			if err == nil {
				t.Fatal("expected a drift error; SQLite cannot add this column to an existing table")
			}
			if !strings.Contains(err.Error(), tc.reason) {
				t.Fatalf("error should explain why (%q); got: %v", tc.reason, err)
			}
		})
	}
}

// TestReconcile_DefaultValueCannotFakeAConstraint guards the keyword scan: a
// default whose text contains SQL keywords must not be mistaken for one.
func TestReconcile_DefaultValueCannotFakeAConstraint(t *testing.T) {
	db := openMemSQLite(t)
	ctx := context.Background()
	if err := EnsureSchemas(ctx, db, SchemaProviderFunc(func() string { return oldProvidersDDL })); err != nil {
		t.Fatalf("initial apply: %v", err)
	}
	if err := EnsureSchemas(ctx, db, SchemaProviderFunc(func() string {
		return `CREATE TABLE IF NOT EXISTS providers (
			provider_id TEXT PRIMARY KEY,
			descriptor  TEXT NOT NULL,
			note        TEXT NOT NULL DEFAULT 'PRIMARY KEY UNIQUE REFERENCES'
		);`
	})); err != nil {
		t.Fatalf("a literal default must not read as a constraint: %v", err)
	}
	var note string
	if err := db.QueryRowContext(ctx,
		`SELECT note FROM providers WHERE 1=0 UNION ALL SELECT 'PRIMARY KEY UNIQUE REFERENCES'`).Scan(&note); err != nil {
		t.Fatalf("column not added: %v", err)
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
