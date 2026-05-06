package api

import "testing"

func TestCheckDatabaseBackoffExemptsInternalTestutil(t *testing.T) {
	source := []byte(`package db

import "database/sql"

func NewSQLite() (*sql.DB, error) {
	return sql.Open("sqlite", "file:test.db")
}
`)

	violations := CheckDatabaseBackoff(source, "api/internal/testutil/db/sqlite.go")
	if len(violations) != 0 {
		t.Fatalf("expected internal testutil sql.Open to be exempt, got %d violation(s): %+v", len(violations), violations)
	}
}

func TestCheckDatabaseBackoffFlagsProductionSQLOpen(t *testing.T) {
	source := []byte(`package database

import "database/sql"

func Open() (*sql.DB, error) {
	return sql.Open("postgres", "postgres://example")
}
`)

	violations := CheckDatabaseBackoff(source, "api/internal/database/postgres.go")
	if len(violations) != 1 {
		t.Fatalf("expected production sql.Open to be flagged once, got %d violation(s): %+v", len(violations), violations)
	}
}
