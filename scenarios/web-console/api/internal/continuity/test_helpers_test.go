package continuity

import (
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"
)

func newContinuityTestDB(t testing.TB, dsn string) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		t.Fatalf("open continuity test database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

// execContinuitySQL keeps each test's schema and seed rows visible at its callsite.
func execContinuitySQL(t testing.TB, db *sql.DB, statements []string) {
	t.Helper()
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("execute continuity test SQL: %v", err)
		}
	}
}
