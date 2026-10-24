package validation

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

// schemaACFor builds an AnalyzerContext for a Go scenario laid out under a temp
// repo root by the test, with the given domains and engines.
func schemaACFor(t *testing.T, repoRoot, scenario string, domains []string, engines ...Engine) AnalyzerContext {
	t.Helper()
	scenarioDir := filepath.Join(repoRoot, "scenarios", scenario)
	if len(engines) == 0 {
		engines = []Engine{EngineSQLite}
	}
	return AnalyzerContext{
		Scenario:    scenario,
		ScenarioDir: scenarioDir,
		APIDir:      filepath.Join(scenarioDir, "api"),
		Language:    "go",
		Engines:     engines,
		Domains:     domains,
	}
}

// findingCodes returns the set of finding codes produced, for assertions.
func findingCodes(findings []Finding) map[string]int {
	out := map[string]int{}
	for _, f := range findings {
		out[f.Code]++
	}
	return out
}

func runAnalyzer(t *testing.T, a Analyzer, ac AnalyzerContext) []Finding {
	t.Helper()
	if !a.Applies(ac) {
		return nil
	}
	got, err := a.Analyze(context.Background(), ac)
	if err != nil {
		t.Fatalf("%s.Analyze: %v", a.Name(), err)
	}
	return got
}

// -----------------------------------------------------------------------------
// SCHEMA_HAS_ALTER
// -----------------------------------------------------------------------------

func TestSchemaHasAlter(t *testing.T) {
	a := schemaHasAlter{}

	t.Run("positive", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\nALTER TABLE orders ADD COLUMN total INTEGER;\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if findingCodes(got)["SCHEMA_HAS_ALTER"] != 1 {
			t.Fatalf("want 1 SCHEMA_HAS_ALTER, got %+v", got)
		}
		if !strings.Contains(got[0].Location, "schema.sql") {
			t.Fatalf("location %q should point at schema.sql", got[0].Location)
		}
	})

	t.Run("clean", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT, total INTEGER);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if len(got) != 0 {
			t.Fatalf("want clean, got %+v", got)
		}
	})

	t.Run("exempt-migration", func(t *testing.T) {
		root := t.TempDir()
		// An ALTER inside migrations/ is a legitimate migration — must NOT trigger.
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "migrations", "0001.sql"),
			"ALTER TABLE orders ADD COLUMN total INTEGER;\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if len(got) != 0 {
			t.Fatalf("migration ALTER must be exempt, got %+v", got)
		}
	})

	t.Run("comment-not-counted", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"-- ALTER TABLE orders ADD COLUMN total INTEGER; (do this in a migration)\nCREATE TABLE IF NOT EXISTS orders (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if len(got) != 0 {
			t.Fatalf("ALTER in a comment must not trigger, got %+v", got)
		}
	})
}

// -----------------------------------------------------------------------------
// SCHEMA_NOT_IDEMPOTENT
// -----------------------------------------------------------------------------

func TestSchemaNotIdempotent(t *testing.T) {
	a := schemaNotIdempotent{}

	t.Run("positive-table", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE orders (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if findingCodes(got)["SCHEMA_NOT_IDEMPOTENT"] != 1 {
			t.Fatalf("want 1 SCHEMA_NOT_IDEMPOTENT, got %+v", got)
		}
	})

	t.Run("positive-index", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\nCREATE INDEX orders_id ON orders(id);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if findingCodes(got)["SCHEMA_NOT_IDEMPOTENT"] != 1 {
			t.Fatalf("want 1 SCHEMA_NOT_IDEMPOTENT for the bare index, got %+v", got)
		}
	})

	t.Run("clean", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\nCREATE INDEX IF NOT EXISTS orders_id ON orders(id);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if len(got) != 0 {
			t.Fatalf("want clean, got %+v", got)
		}
	})

	t.Run("exempt-migration", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "migrations", "0001.sql"),
			"CREATE TABLE orders (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if len(got) != 0 {
			t.Fatalf("migration CREATE must be exempt, got %+v", got)
		}
	})
}

// -----------------------------------------------------------------------------
// SCHEMA_CENTRALIZED
// -----------------------------------------------------------------------------

func TestSchemaCentralized(t *testing.T) {
	a := schemaCentralized{}

	t.Run("positive-api-root", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\nCREATE TABLE IF NOT EXISTS users (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders", "users"}))
		if findingCodes(got)["SCHEMA_CENTRALIZED"] != 1 {
			t.Fatalf("want 1 SCHEMA_CENTRALIZED, got %+v", got)
		}
	})

	t.Run("clean-per-domain", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\n")
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "users", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS users (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders", "users"}))
		if len(got) != 0 {
			t.Fatalf("per-domain schema must be clean, got %+v", got)
		}
	})

	t.Run("clean-empty-system", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "database", "system.sql"),
			"-- system\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", nil))
		if len(got) != 0 {
			t.Fatalf("empty system home must be clean, got %+v", got)
		}
	})

	t.Run("exempt-migration", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "migrations", "0001.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if len(got) != 0 {
			t.Fatalf("migration file must be exempt, got %+v", got)
		}
	})
}

// -----------------------------------------------------------------------------
// SCHEMA_NOT_PER_DOMAIN
// -----------------------------------------------------------------------------

func TestSchemaNotPerDomain(t *testing.T) {
	a := schemaNotPerDomain{}

	t.Run("positive-loose-tables", func(t *testing.T) {
		root := t.TempDir()
		// A loose .sql at the api root (no domain, not the system home) owning
		// tables, while the scenario partitions into >=2 domains.
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\nCREATE TABLE IF NOT EXISTS users (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders", "users"}))
		if findingCodes(got)["SCHEMA_NOT_PER_DOMAIN"] != 1 {
			t.Fatalf("want 1 SCHEMA_NOT_PER_DOMAIN, got %+v", got)
		}
	})

	t.Run("clean-per-domain", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\n")
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "users", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS users (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders", "users"}))
		if len(got) != 0 {
			t.Fatalf("per-domain schema must be clean, got %+v", got)
		}
	})

	t.Run("gated-single-domain", func(t *testing.T) {
		root := t.TempDir()
		// Loose tables but only ONE domain → analyzer does not apply, no finding.
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "store", "all.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if len(got) != 0 {
			t.Fatalf("single-domain scenario must be gated out, got %+v", got)
		}
	})

	t.Run("exempt-migration", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "migrations", "0001.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\nCREATE TABLE IF NOT EXISTS users (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders", "users"}))
		if len(got) != 0 {
			t.Fatalf("migration tables must be exempt, got %+v", got)
		}
	})
}

// -----------------------------------------------------------------------------
// ENSURE_SCHEMAS_NOT_WIRED
// -----------------------------------------------------------------------------

func TestSchemaEnsureNotWired(t *testing.T) {
	a := schemaEnsureNotWired{}

	t.Run("positive-no-call", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\n")
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.go"),
			"package orders\n\nimport _ \"embed\"\n\n//go:embed schema.sql\nvar schemaSQL string\n\nfunc Schema() string { return schemaSQL }\n")
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "main.go"),
			"package main\n\nfunc main() {}\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if findingCodes(got)["ENSURE_SCHEMAS_NOT_WIRED"] != 1 {
			t.Fatalf("want 1 ENSURE_SCHEMAS_NOT_WIRED (no call), got %+v", got)
		}
	})

	t.Run("positive-no-embed", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\n")
		// EnsureSchemas is called, but the domain schema has no sibling //go:embed.
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "main.go"),
			"package main\n\nfunc main() { database.EnsureSchemas(ctx, db, modules.AllSchemas()...) }\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if findingCodes(got)["ENSURE_SCHEMAS_NOT_WIRED"] != 1 {
			t.Fatalf("want 1 ENSURE_SCHEMAS_NOT_WIRED (no embed), got %+v", got)
		}
		if !strings.Contains(got[0].Location, "orders/schema.sql") {
			t.Fatalf("location %q should point at the unembedded domain schema", got[0].Location)
		}
	})

	t.Run("clean", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT);\n")
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.go"),
			"package orders\n\nimport _ \"embed\"\n\n//go:embed schema.sql\nvar schemaSQL string\n\nfunc Schema() string { return schemaSQL }\n")
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "main.go"),
			"package main\n\nfunc main() { database.EnsureSchemas(ctx, db, modules.AllSchemas()...) }\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if len(got) != 0 {
			t.Fatalf("wired schema must be clean, got %+v", got)
		}
	})

	t.Run("clean-no-tables", func(t *testing.T) {
		root := t.TempDir()
		// Only an empty system home → nothing to wire, no findings even without a call.
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "database", "system.sql"),
			"-- system\n")
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "main.go"),
			"package main\n\nfunc main() {}\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", nil))
		if len(got) != 0 {
			t.Fatalf("empty schema must be clean, got %+v", got)
		}
	})
}

// -----------------------------------------------------------------------------
// SYSTEM_SQL_NOT_EMPTY
// -----------------------------------------------------------------------------

func TestSchemaSystemNotEmpty(t *testing.T) {
	a := schemaSystemNotEmpty{}

	t.Run("positive", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "database", "system.sql"),
			"CREATE TABLE IF NOT EXISTS widgets (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", nil))
		if findingCodes(got)["SYSTEM_SQL_NOT_EMPTY"] != 1 {
			t.Fatalf("want 1 SYSTEM_SQL_NOT_EMPTY, got %+v", got)
		}
	})

	t.Run("clean-empty", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "database", "system.sql"),
			"-- system\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", nil))
		if len(got) != 0 {
			t.Fatalf("empty system home must be clean, got %+v", got)
		}
	})

	t.Run("clean-extensions-only", func(t *testing.T) {
		root := t.TempDir()
		// Cross-cutting infra (extensions/views) is fine — only tables are flagged.
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "database", "system.sql"),
			"CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\";\nCREATE VIEW IF NOT EXISTS active AS SELECT 1;\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", nil))
		if len(got) != 0 {
			t.Fatalf("extensions/views must be clean, got %+v", got)
		}
	})

	t.Run("exempt-migration", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "migrations", "0001.sql"),
			"CREATE TABLE IF NOT EXISTS widgets (id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", nil))
		if len(got) != 0 {
			t.Fatalf("migration tables must be exempt, got %+v", got)
		}
	})
}

// -----------------------------------------------------------------------------
// CROSS_DOMAIN_HARD_FK
// -----------------------------------------------------------------------------

func TestSchemaCrossDomainFK(t *testing.T) {
	a := schemaCrossDomainFK{}

	t.Run("positive", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "users", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY);\n")
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT, user_id TEXT REFERENCES users(id));\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders", "users"}))
		if findingCodes(got)["CROSS_DOMAIN_HARD_FK"] != 1 {
			t.Fatalf("want 1 CROSS_DOMAIN_HARD_FK, got %+v", got)
		}
	})

	t.Run("clean-same-domain", func(t *testing.T) {
		root := t.TempDir()
		// FK within the same domain is fine.
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT PRIMARY KEY);\n"+
				"CREATE TABLE IF NOT EXISTS order_lines (id TEXT, order_id TEXT REFERENCES orders(id));\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders"}))
		if len(got) != 0 {
			t.Fatalf("same-domain FK must be clean, got %+v", got)
		}
	})

	t.Run("clean-soft-id", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "users", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY);\n")
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "orders", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT, user_id TEXT);\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders", "users"}))
		if len(got) != 0 {
			t.Fatalf("soft-id cross-domain reference must be clean, got %+v", got)
		}
	})

	t.Run("exempt-migration", func(t *testing.T) {
		root := t.TempDir()
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "internal", "users", "schema.sql"),
			"CREATE TABLE IF NOT EXISTS users (id TEXT PRIMARY KEY);\n")
		writeFile(t, filepath.Join(root, "scenarios", "s", "api", "migrations", "0001.sql"),
			"CREATE TABLE IF NOT EXISTS orders (id TEXT, user_id TEXT REFERENCES users(id));\n")
		got := runAnalyzer(t, a, schemaACFor(t, root, "s", []string{"orders", "users"}))
		if len(got) != 0 {
			t.Fatalf("migration FK must be exempt, got %+v", got)
		}
	})
}

// -----------------------------------------------------------------------------
// Dogfood: storage-manager itself must be clean across every Tier-1 analyzer.
// -----------------------------------------------------------------------------

func TestSchemaAnalyzers_DogfoodStorageHealthClean(t *testing.T) {
	// Resolve the real storage-manager scenario dir from this test's location:
	// .../scenarios/storage-manager/api/internal/validation → up 3 to the scenario.
	scenarioDir, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("abs: %v", err)
	}
	ac := AnalyzerContext{
		Scenario:    "storage-manager",
		ScenarioDir: scenarioDir,
		APIDir:      filepath.Join(scenarioDir, "api"),
		Language:    "go",
		Engines:     []Engine{EngineSQLite},
		Domains:     filesystemDomains(scenarioDir),
	}

	analyzers := []Analyzer{
		schemaHasAlter{},
		schemaNotIdempotent{},
		schemaCentralized{},
		schemaNotPerDomain{},
		schemaEnsureNotWired{},
		schemaSystemNotEmpty{},
		schemaCrossDomainFK{},
	}
	for _, a := range analyzers {
		got := runAnalyzer(t, a, ac)
		if len(got) != 0 {
			t.Errorf("storage-manager is not clean for %s: %+v", a.Name(), got)
		}
	}
}
