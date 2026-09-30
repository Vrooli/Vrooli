package validation

import "testing"

// Parity tests for hygiene-routed-driver vs the original scenario-auditor rule
// routed_database_drivers
// (scenarios/scenario-auditor/api/rules/api/routed_database_drivers.go,
// CheckRoutedDatabaseDrivers). Inputs/paths/verdicts mirror that rule's
// <test-case> blocks plus routed_database_drivers_unit_test.go.
func TestHygieneRoutedDriver_Parity(t *testing.T) {
	a := hygieneRoutedDriver{}
	cases := []struct {
		name    string
		path    string
		src     string
		wantHit bool
	}{
		// PASS-uses-routed-db
		{"pass_uses_routed_db", "api/main.go", `package main

import (
    "context"
    "log"

    "github.com/vrooli/api-core/database"
)

func main() {
    db, err := database.Open(context.Background(), database.Config{Driver: "postgres"})
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
}
`, false},
		// FAIL-imports-pgx
		{"fail_imports_pgx", "api/main.go", `package main

import (
    "github.com/jackc/pgx/v5"
)

var _ = pgx.Connect
`, true},
		// FAIL-imports-pgxpool
		{"fail_imports_pgxpool", "api/db.go", `package db

import (
    "github.com/jackc/pgx/v5/pgxpool"
)

var _ = pgxpool.Connect
`, true},
		// FAIL-imports-lib-pq-non-blank
		{"fail_imports_lib_pq_named", "api/db.go", `package db

import (
    pq "github.com/lib/pq"
)

var _ = pq.Driver{}
`, true},
		// FAIL-imports-mattn-sqlite3
		{"fail_imports_mattn", "api/db.go", `package db

import (
    "github.com/mattn/go-sqlite3"
)

var _ = sqlite3.Version
`, true},
		// FAIL-imports-modernc-sqlite
		{"fail_imports_modernc", "api/db.go", `package db

import (
    sqlite "modernc.org/sqlite"
)

var _ = sqlite.Version
`, true},
		// PASS-blank-import-allowed-in-main
		{"pass_blank_import", "api/main.go", `package main

import (
    "github.com/vrooli/api-core/database"

    _ "github.com/lib/pq"
)

var _ = database.DriverPostgres
`, false},
		// FAIL-sql-opendb
		{"fail_sql_opendb", "api/main.go", `package main

import (
    "database/sql"
    "database/sql/driver"
)

var connector driver.Connector

func main() {
    _ = sql.OpenDB(connector)
}
`, true},
		// PASS-test-file-exempt
		{"pass_test_file", "api/main_test.go", `package main

import (
    "testing"

    "github.com/jackc/pgx/v5"
)

func TestX(t *testing.T) { _ = pgx.Connect }
`, false},
		// routed_database_drivers_unit_test.go: api-core path exempt.
		{"pass_api_core_exempt", "packages/api-core/database/routed.go", `package database

import "modernc.org/sqlite"

var _ = sqlite.Version
`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := a.analyzeSource(tc.src, tc.path)
			if tc.wantHit && len(got) == 0 {
				t.Fatalf("expected ROUTED_DRIVER_IMPORT finding, got none")
			}
			if !tc.wantHit && len(got) != 0 {
				t.Fatalf("expected no finding, got %+v", got)
			}
			for _, f := range got {
				if f.Code != "ROUTED_DRIVER_IMPORT" {
					t.Fatalf("unexpected code %q", f.Code)
				}
				if f.Severity != SeverityError {
					t.Fatalf("severity = %v, want ERROR", f.Severity)
				}
			}
		})
	}
}
