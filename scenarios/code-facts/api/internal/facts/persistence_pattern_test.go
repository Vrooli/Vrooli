package facts

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPersistencePatternJoinsSchemaMigrationRepositoryAndAbstains(t *testing.T) {
	root, err := filepath.Abs("../../../")
	if err != nil {
		t.Fatal(err)
	}
	repository, err := os.ReadFile(filepath.Join(root, "api", "internal", "catalog", "sqlite.go"))
	if err != nil {
		t.Fatal(err)
	}
	refs, err := exactSQLRepositoryReferences(repository)
	if err != nil {
		t.Fatal(err)
	}
	if refs["code_facts_generations"] == 0 {
		t.Fatalf("contextual SQL query argument was not parsed into repository references: refs=%+v", refs)
	}
	positive, err := scanPersistencePatterns(root, "code-facts", "catalog generations sqlite migration repository", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(positive.Results) == 0 || positive.Results[0].Entity != "code_facts_generations" {
		t.Fatalf("expected joined generation persistence evidence, got %#v", positive.Results)
	}
	hit := positive.Results[0]
	if hit.Schema.SHA256 == "" || hit.SchemaBinding.Line == 0 || hit.SchemaBinding.SHA256 == "" || hit.Migration.Line == 0 || hit.Repository.Line == 0 || hit.Classification != "JOINED_STATIC_SOURCE_EVIDENCE" {
		t.Fatalf("incomplete provenance join: %#v", hit)
	}
	negative, err := scanPersistencePatterns(root, "code-facts", "underwater penguin telescope", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(negative.Results) != 0 {
		t.Fatalf("unrelated query should abstain: %#v", negative.Results)
	}
}

func TestPersistencePatternRejectsRepositorySubstringAndCommentMerges(t *testing.T) {
	root := writePersistencePatternFixture(t, `CREATE TABLE IF NOT EXISTS orders (
id TEXT PRIMARY KEY
);`, `//go:embed schema.sql
var schemaSQL string
func Schema() string { return schemaSQL }`, `package catalog
type Migration struct { Version int; SQL string }
func Migrate() { ApplyMigrations([]Migration{{Version: 1, SQL: Schema()}}) }
func Insert() {
  // INSERT INTO orders (commented table name must not count)
  _, _ = db.ExecContext(ctx, "INSERT INTO orders_archive(id) VALUES(?)")
  _, _ = db.ExecContext(ctx, "INSERT INTO orders_archive(id) VALUES(?) -- INSERT INTO orders from a SQL comment")
}
var documentation = "INSERT INTO orders(id) VALUES(?)"
`)
	got, err := scanPersistencePatterns(root, "code-facts", "orders schema migration repository", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 0 || len(got.Unknowns) != 1 || got.Unknowns[0].Entity != "orders" || got.Unknowns[0].Status != "UNKNOWN" || got.Unknowns[0].Reason != "exact_repository_table_reference_not_proven" {
		t.Fatalf("substring/comment references must remain UNKNOWN, got results=%+v unknowns=%+v", got.Results, got.Unknowns)
	}
}

func TestPersistencePatternJoinsOnlySupportedExactSQLLiteralIdentifier(t *testing.T) {
	root := writePersistencePatternFixture(t, `CREATE TABLE IF NOT EXISTS orders (
id TEXT PRIMARY KEY
);`, `//go:embed schema.sql
var schemaSQL string
func Schema() string { return schemaSQL }`, `package catalog
type Migration struct { Version int; SQL string }
func Migrate() { ApplyMigrations([]Migration{{Version: 1, SQL: Schema()}}) }
func Insert() {
  _, _ = db.ExecContext(ctx, "INSERT INTO orders_archive(id) VALUES(?)")
  _, _ = db.ExecContext(ctx, "INSERT INTO orders(id) VALUES(?)")
}`)
	got, err := scanPersistencePatterns(root, "code-facts", "orders schema migration repository", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 || got.Results[0].Classification != "JOINED_STATIC_SOURCE_EVIDENCE" || got.Results[0].Entity != "orders" || got.Results[0].Repository.Line == 0 || len(got.Unknowns) != 0 {
		t.Fatalf("exact SQL literal should join with source line evidence, got %+v", got)
	}
}

func TestPersistencePatternJoinsSQLiteQuotedIdentifiersCaseInsensitively(t *testing.T) {
	root := writePersistencePatternFixture(t, "CREATE TABLE IF NOT EXISTS \"Orders\" (\nid TEXT PRIMARY KEY\n);", "//go:embed schema.sql\nvar schemaSQL string\nfunc Schema() string { return schemaSQL }", "package catalog\ntype Migration struct { Version int; SQL string }\nfunc Migrate() { ApplyMigrations([]Migration{{Version: 1, SQL: Schema()}}) }\nfunc Insert() { _, _ = db.ExecContext(ctx, \"INSERT INTO [orders](id) VALUES(?)\") }")
	got, err := scanPersistencePatterns(root, "code-facts", "orders schema", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 1 || got.Results[0].Entity != "orders" || got.Results[0].Repository.Line == 0 {
		t.Fatalf("equivalent SQLite quoted identifiers should join, got %+v", got)
	}
}

func TestSQLiteRepositoryIdentifierScannerAbstainsOnCTEAndQualifiedNames(t *testing.T) {
	if refs := sqliteTableReferences("WITH orders AS (SELECT 1) SELECT * FROM orders"); len(refs) != 0 {
		t.Fatalf("CTE alias must not be reported as a base table: %+v", refs)
	}
	refs := sqliteTableReferences("SELECT 'FROM orders' /* JOIN orders */ FROM main.orders JOIN [Actual] a JOIN json_each(?) AS fixture")
	if len(refs) != 1 || refs[0].name != "actual" {
		t.Fatalf("qualified names/comments/string literals should not become joins: %+v", refs)
	}
	if refs := sqliteTableReferences("SELECT * FROM harmless; INSERT INTO orders(id) VALUES(?)"); len(refs) != 0 {
		t.Fatalf("multi-statement query must remain unjoined: %+v", refs)
	}
	if refs := sqliteTableReferences("INSERT INTO \"orders\"\"archive\"(id) VALUES(?)"); len(refs) != 0 {
		t.Fatalf("escaped delimited identifier must remain unsupported: %+v", refs)
	}
	refs = sqliteTableReferences("INSERT INTO source(id) VALUES(?) ON CONFLICT(id) DO UPDATE SET value = ?")
	if len(refs) != 1 || refs[0].name != "source" {
		t.Fatalf("upsert update clause must not treat SET as a table: %+v", refs)
	}
}

func TestSQLiteRepositoryScannerHandlesConflictClause(t *testing.T) {
	refs := sqliteTableReferences("UPDATE OR IGNORE \"Orders\" SET id = ?\nWHERE id = ?")
	if len(refs) != 1 || refs[0].name != "orders" || refs[0].offset == 0 {
		t.Fatalf("supported conflict update was not normalized: %+v", refs)
	}
}

func TestSQLiteRepositoryScannerRecognizesInsertTarget(t *testing.T) {
	refs := sqliteTableReferences("INSERT INTO orders(id) VALUES(?)")
	if len(refs) != 1 || refs[0].name != "orders" {
		t.Fatalf("insert target was not recognized: tokens=%+v refs=%+v", lexSQLiteSQL("INSERT INTO orders(id) VALUES(?)"), refs)
	}
}

func writePersistencePatternFixture(t *testing.T, schema, schemaGo, repository string) string {
	t.Helper()
	root := t.TempDir()
	base := filepath.Join(root, "api", "internal", "catalog")
	if err := os.MkdirAll(base, 0o755); err != nil {
		t.Fatal(err)
	}
	schemaGo = "package catalog\nimport _ \"embed\"\n" + schemaGo
	for name, content := range map[string]string{"schema.sql": schema, "schema.go": schemaGo, "sqlite.go": repository} {
		if err := os.WriteFile(filepath.Join(base, name), []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
