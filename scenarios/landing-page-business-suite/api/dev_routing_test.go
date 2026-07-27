package main

import "testing"

func TestLeaseDatabaseName(t *testing.T) {
	got, err := leaseDatabaseName("workflow-health/ABC-123")
	if err != nil {
		t.Fatalf("leaseDatabaseName returned error: %v", err)
	}
	if got != "lpbs_workflow_workflow_health_abc_123" {
		t.Fatalf("lease database name = %q", got)
	}
}

func TestPostgresLeaseURLsPreserveCredentialsAndReplaceOnlyDatabase(t *testing.T) {
	admin, pool, err := postgresLeaseURLs("postgres://user:secret@localhost:5433/primary?sslmode=disable", "lpbs_workflow_lease")
	if err != nil {
		t.Fatalf("postgresLeaseURLs returned error: %v", err)
	}
	if admin != "postgres://user:secret@localhost:5433/postgres?sslmode=disable" {
		t.Fatalf("admin URL = %q", admin)
	}
	if pool != "postgres://user:secret@localhost:5433/lpbs_workflow_lease?sslmode=disable" {
		t.Fatalf("pool URL = %q", pool)
	}
}

func TestIsSQLitePlaceholder(t *testing.T) {
	if !isSQLitePlaceholder("file:/tmp/test.db?_pragma=foreign_keys(ON)") {
		t.Fatal("file DSN should be recognized as a SQLite placeholder")
	}
	if isSQLitePlaceholder("postgres://user:secret@localhost:5433/db") {
		t.Fatal("Postgres DSN must not be recognized as a SQLite placeholder")
	}
}
