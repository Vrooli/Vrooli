package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/database"

	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"
	_ "modernc.org/sqlite" // pure-Go SQLite driver registration for test DB
)

// SetupTestDB creates a temporary SQLite database with the full schema applied.
// Returns a database.DB and a cleanup function. The cleanup function closes the
// database and removes the temp file.
func SetupTestDB(t *testing.T) (*database.DB, func()) {
	t.Helper()

	tmpDir := t.TempDir()
	dbPath := filepath.Join(tmpDir, "agent-manager-test.db")
	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)",
		dbPath,
	)

	db, err := sqlx.Connect("sqlite", dsn)
	if err != nil {
		t.Fatalf("connect sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)

	// Read and execute schema
	schemaPath := filepath.Join(getSchemaDir(), "schema.sql")
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		db.Close()
		t.Fatalf("read schema: %v", err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		db.Close()
		t.Fatalf("exec schema: %v", err)
	}

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	dbWrapper := database.NewDB(db, logger)

	cleanup := func() {
		db.Close()
	}
	return dbWrapper, cleanup
}

// SetupTestRepos creates a temporary SQLite database and returns all repositories
// plus the event store and a cleanup function.
func SetupTestRepos(t *testing.T) (*database.Repositories, event.Store, func()) {
	t.Helper()
	db, cleanup := SetupTestDB(t)

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	repos := database.NewRepositories(db, logger)
	eventStore := event.NewSQLiteStore(db.DB, logger)

	return repos, eventStore, cleanup
}

// SetupTestReposWithDB is like SetupTestRepos but uses an externally provided DB.
// This is useful when the test needs direct access to the DB for raw SQL operations.
func SetupTestReposWithDB(t *testing.T, db *database.DB) (*database.Repositories, event.Store, func()) {
	t.Helper()

	logger := logrus.New()
	logger.SetLevel(logrus.WarnLevel)

	repos := database.NewRepositories(db, logger)
	eventStore := event.NewSQLiteStore(db.DB, logger)

	return repos, eventStore, func() {}
}

// getSchemaDir returns the path to the database package directory containing schema.sql.
func getSchemaDir() string {
	_, filename, _, _ := runtime.Caller(0)
	// testutil is at internal/testutil/testdb.go
	// schema.sql is at internal/database/schema.sql
	return filepath.Join(filepath.Dir(filename), "..", "database")
}
