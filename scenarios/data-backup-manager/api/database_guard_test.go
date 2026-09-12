package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDatabasePreflightAllowsFirstInitialization(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data-backup-manager.db")
	if err := databasePreflight(path); err != nil {
		t.Fatalf("first initialization preflight: %v", err)
	}
}

func TestDatabasePreflightRejectsEmptyDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data-backup-manager.db")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := databasePreflight(path); err == nil || !strings.Contains(err.Error(), "empty") {
		t.Fatalf("databasePreflight = %v, want empty-database fault", err)
	}
}

func TestDatabasePreflightRejectsMissingPreviouslyInitializedDatabase(t *testing.T) {
	path := filepath.Join(t.TempDir(), "data-backup-manager.db")
	if err := os.WriteFile(path+".initialized", []byte("initialized\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := databasePreflight(path); err == nil || !strings.Contains(err.Error(), "absent after prior initialization") {
		t.Fatalf("databasePreflight = %v, want prior-initialization fault", err)
	}
}
