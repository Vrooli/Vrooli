package validation

import "testing"

// Parity tests for hygiene-handle-capture vs the original scenario-auditor rule
// routed_database_handle_capture
// (scenarios/scenario-auditor/api/rules/api/routed_database_handle_capture.go,
// CheckRoutedDatabaseHandleCapture). Inputs/paths/verdicts mirror that rule's
// <test-case> blocks plus the *Handle* unit tests in
// routed_database_drivers_unit_test.go.
func TestHygieneHandleCapture_Parity(t *testing.T) {
	a := hygieneHandleCapture{}
	cases := []struct {
		name    string
		path    string
		src     string
		wantHit bool
	}{
		// PASS-uses-routed-db-field
		{"pass_routed_db_field", "api/main.go", `package main

import "github.com/vrooli/api-core/database"

type Server struct {
    DB *database.RoutedDB
}
`, false},
		// FAIL-struct-field-sqldb
		{"fail_struct_field", "api/server.go", `package main

import "database/sql"

type Server struct {
    DB *sql.DB
}
`, true},
		// FAIL-package-var-sqldb
		{"fail_package_var", "api/db.go", `package main

import "database/sql"

var db *sql.DB
`, true},
		// FAIL-func-param-sqldb
		{"fail_func_param", "api/handlers.go", `package main

import "database/sql"

func handle(db *sql.DB) error { return nil }
`, true},
		// PASS-test-file-exempt
		{"pass_test_file", "api/handlers_test.go", `package main

import (
    "database/sql"
    "testing"
)

type fixture struct {
    DB *sql.DB
}

func TestX(t *testing.T) { _ = fixture{} }
`, false},
		// unit test: api-core path exempt.
		{"pass_api_core_exempt", "packages/api-core/database/pool.go", `package database

import "database/sql"

type pool struct { db *sql.DB }
`, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := a.analyzeSource(tc.src, tc.path)
			if tc.wantHit && len(got) == 0 {
				t.Fatalf("expected SQL_DB_HANDLE_CAPTURE finding, got none")
			}
			if !tc.wantHit && len(got) != 0 {
				t.Fatalf("expected no finding, got %+v", got)
			}
			for _, f := range got {
				if f.Code != "SQL_DB_HANDLE_CAPTURE" {
					t.Fatalf("unexpected code %q", f.Code)
				}
				if f.Severity != SeverityWarning {
					t.Fatalf("severity = %v, want WARNING", f.Severity)
				}
			}
		})
	}
}
