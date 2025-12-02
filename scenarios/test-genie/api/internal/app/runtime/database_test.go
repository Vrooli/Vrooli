package runtime

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
)

type scenarioInitPaths struct {
	apiDir  string
	schema  string
	seed    string
	hasSeed bool
}

func newScenarioInitFiles(t *testing.T, schemaContent, seedContent string, includeSeed bool) scenarioInitPaths {
	t.Helper()

	root := t.TempDir()
	apiDir := filepath.Join(root, "api")
	if err := os.MkdirAll(apiDir, 0o755); err != nil {
		t.Fatalf("failed to create api dir: %v", err)
	}

	initDir := filepath.Join(root, "initialization", "postgres")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatalf("failed to create initialization dir: %v", err)
	}

	schemaPath := filepath.Join(initDir, "schema.sql")
	if err := os.WriteFile(schemaPath, []byte(schemaContent), 0o644); err != nil {
		t.Fatalf("failed to write schema file: %v", err)
	}

	var seedPath string
	if includeSeed {
		seedPath = filepath.Join(initDir, "seed.sql")
		if err := os.WriteFile(seedPath, []byte(seedContent), 0o644); err != nil {
			t.Fatalf("failed to write seed file: %v", err)
		}
	}

	return scenarioInitPaths{
		apiDir:  apiDir,
		schema:  schemaPath,
		seed:    seedPath,
		hasSeed: includeSeed,
	}
}

func withWorkingDir(t *testing.T, dir string) {
	t.Helper()
	original, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd failed: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("failed to chdir to %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(original); err != nil {
			t.Errorf("failed to restore working directory: %v", err)
		}
	})
}

func TestResolveInitializationFileSuccess(t *testing.T) {
	files := newScenarioInitFiles(t, "CREATE TABLE example (id INT);", "", false)
	withWorkingDir(t, files.apiDir)

	got, err := resolveInitializationFile("schema.sql")
	if err != nil {
		t.Fatalf("resolveInitializationFile returned error: %v", err)
	}
	if got != files.schema {
		t.Fatalf("expected schema path %s, got %s", files.schema, got)
	}
}

func TestResolveInitializationFileErrorWhenMissing(t *testing.T) {
	files := newScenarioInitFiles(t, "CREATE TABLE example (id INT);", "", false)
	withWorkingDir(t, files.apiDir)

	if _, err := resolveInitializationFile("seed.sql"); err == nil {
		t.Fatal("expected error when seed file is missing")
	}
}

func TestExecSQLFileRunsStatements(t *testing.T) {
	tmp := t.TempDir()
	sqlPath := filepath.Join(tmp, "script.sql")
	sqlContent := `
-- comment
CREATE TABLE foo (
    id INT
);

INSERT INTO foo VALUES (1);
`
	if err := os.WriteFile(sqlPath, []byte(sqlContent), 0o644); err != nil {
		t.Fatalf("failed to write sql script: %v", err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE foo (\n    id INT\n)")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO foo VALUES (1)")).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := execSQLFile(db, sqlPath); err != nil {
		t.Fatalf("execSQLFile returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestExecSQLFilePropagatesExecErrors(t *testing.T) {
	tmp := t.TempDir()
	sqlPath := filepath.Join(tmp, "script.sql")
	if err := os.WriteFile(sqlPath, []byte("INSERT INTO foo VALUES (1);"), 0o644); err != nil {
		t.Fatalf("failed to write sql script: %v", err)
	}

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO foo VALUES (1)")).WillReturnError(os.ErrInvalid)

	if err := execSQLFile(db, sqlPath); err == nil {
		t.Fatal("expected execSQLFile to return error")
	}
}

func TestEnsureDatabaseSchemaExecutesSchemaAndSeed(t *testing.T) {
	schemaContent := `
CREATE TABLE foo (
    id INT
);
CREATE INDEX idx_foo ON foo (id);
`
	seedContent := `
INSERT INTO foo VALUES (1);
`
	files := newScenarioInitFiles(t, schemaContent, seedContent, true)
	withWorkingDir(t, files.apiDir)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE foo (\n    id INT\n)")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("CREATE INDEX idx_foo ON foo (id)")).WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectExec(regexp.QuoteMeta("INSERT INTO foo VALUES (1)")).WillReturnResult(sqlmock.NewResult(0, 1))

	if err := ensureDatabaseSchema(db); err != nil {
		t.Fatalf("ensureDatabaseSchema returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

func TestEnsureDatabaseSchemaSkipsMissingSeed(t *testing.T) {
	schemaContent := "CREATE TABLE foo (id INT);"
	files := newScenarioInitFiles(t, schemaContent, "", false)
	withWorkingDir(t, files.apiDir)

	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("sqlmock.New error: %v", err)
	}
	defer db.Close()

	mock.ExpectExec(regexp.QuoteMeta("CREATE TABLE foo (id INT)")).WillReturnResult(sqlmock.NewResult(0, 0))

	if err := ensureDatabaseSchema(db); err != nil {
		t.Fatalf("ensureDatabaseSchema returned error: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
