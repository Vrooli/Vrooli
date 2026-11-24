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

func TestPersistencePatternReportsUnsupportedQuotedSchemaIdentifierAsUnknown(t *testing.T) {
	root := writePersistencePatternFixture(t, `CREATE TABLE IF NOT EXISTS "quoted_orders" (
id TEXT PRIMARY KEY
);`, `//go:embed schema.sql
var schemaSQL string
func Schema() string { return schemaSQL }`, `package catalog
type Migration struct { Version int; SQL string }
func Migrate() { ApplyMigrations([]Migration{{Version: 1, SQL: Schema()}}) }
func Insert() { _, _ = db.ExecContext(ctx, "INSERT INTO quoted_orders(id) VALUES(?)") }`)
	got, err := scanPersistencePatterns(root, "code-facts", "quoted_orders schema", 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Results) != 0 || len(got.Unknowns) != 1 || got.Unknowns[0].Reason != "unsupported_schema_identifier_form" {
		t.Fatalf("quoted identifier should be explicitly UNKNOWN, got %+v", got)
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
