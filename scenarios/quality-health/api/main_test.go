package main

import (
	"strings"
	"testing"
)

func TestSQLiteFileDSNLeavesFileDSNUnchanged(t *testing.T) {
	const dsn = "file:/tmp/quality-health.db?_pragma=foreign_keys(ON)"
	got, err := sqliteFileDSN(dsn)
	if err != nil {
		t.Fatalf("sqliteFileDSN returned error: %v", err)
	}
	if got != dsn {
		t.Fatalf("sqliteFileDSN() = %q, want %q", got, dsn)
	}
}

func TestSQLiteFileDSNBuildsFileDSN(t *testing.T) {
	got, err := sqliteFileDSN(t.TempDir() + "/quality-health.db")
	if err != nil {
		t.Fatalf("sqliteFileDSN returned error: %v", err)
	}
	if !strings.HasPrefix(got, "file:") {
		t.Fatalf("sqliteFileDSN() = %q, want file: prefix", got)
	}
	if !strings.Contains(got, "foreign_keys(ON)") {
		t.Fatalf("sqliteFileDSN() = %q, want pragma settings", got)
	}
}
