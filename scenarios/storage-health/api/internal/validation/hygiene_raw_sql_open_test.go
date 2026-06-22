package validation

import "testing"

// Parity tests for hygiene-raw-sql-open vs the original scenario-auditor rule
// database_backoff (scenarios/scenario-auditor/api/rules/api/database_backoff.go,
// CheckDatabaseBackoff). Each case feeds the SAME source + path the original
// rule's doc-test (<test-case> blocks) and unit test
// (database_backoff_unit_test.go) use, and asserts the SAME verdict
// (finding vs no-finding).
func TestHygieneRawSQLOpen_Parity(t *testing.T) {
	a := hygieneRawSQLOpen{}
	cases := []struct {
		name    string
		path    string
		src     string
		wantHit bool
	}{
		// <test-case id="PASS-uses-api-core-database"> — imports api-core/database.
		{"pass_uses_api_core", "api/main.go", `package main

import (
    "context"
    "log"

    "github.com/vrooli/api-core/database"
    _ "github.com/lib/pq"
)

func main() {
    db, err := database.Connect(context.Background(), database.Config{Driver: "postgres"})
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
`, false},
		// <test-case id="FAIL-direct-sql-open">
		{"fail_direct_sql_open", "api/main.go", `package main

import (
    "database/sql"
    "log"

    _ "github.com/lib/pq"
)

func main() {
    db, err := sql.Open("postgres", "postgres://user:pass@host:5432/db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
`, true},
		// <test-case id="FAIL-sql-open-with-ping-no-apicore">
		{"fail_sql_open_with_ping", "api/main.go", `package main

import (
    "database/sql"
    "log"

    _ "github.com/lib/pq"
)

func main() {
    db, err := sql.Open("postgres", "dsn")
    if err != nil {
        log.Fatal(err)
    }
    if err := db.Ping(); err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
`, true},
		// <test-case id="PASS-test-file-with-sql-open"> — _test.go is exempt.
		{"pass_test_file", "main_test.go", `package main

import (
    "database/sql"
    "testing"

    _ "github.com/mattn/go-sqlite3"
)

func TestDatabaseLogic(t *testing.T) {
    db, _ := sql.Open("sqlite3", ":memory:")
    defer db.Close()
}
`, false},
		// <test-case id="PASS-migration-script"> — migrations/ is exempt.
		{"pass_migration", "migrations/001_initial.go", `package migrations

import (
    "database/sql"

    _ "github.com/lib/pq"
)

func Run(db *sql.DB) error {
    _, err := db.Exec("CREATE TABLE users (id SERIAL PRIMARY KEY)")
    return err
}
`, false},
		// database_backoff_unit_test.go: internal/testutil is exempt.
		{"pass_internal_testutil", "api/internal/testutil/db/sqlite.go", `package db

import "database/sql"

func NewSQLite() (*sql.DB, error) {
    return sql.Open("sqlite", "file:test.db")
}
`, false},
		// database_backoff_unit_test.go: production sql.Open is flagged.
		{"fail_production_open", "api/internal/database/postgres.go", `package database

import "database/sql"

func Open() (*sql.DB, error) {
    return sql.Open("postgres", "postgres://example")
}
`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := a.analyzeSource(tc.src, tc.path)
			if tc.wantHit && len(got) == 0 {
				t.Fatalf("expected RAW_SQL_OPEN finding, got none")
			}
			if !tc.wantHit && len(got) != 0 {
				t.Fatalf("expected no finding, got %+v", got)
			}
			for _, f := range got {
				if f.Code != "RAW_SQL_OPEN" {
					t.Fatalf("unexpected code %q", f.Code)
				}
				if f.Severity != SeverityError {
					t.Fatalf("severity = %v, want ERROR", f.Severity)
				}
			}
		})
	}
}
