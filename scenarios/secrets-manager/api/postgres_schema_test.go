package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPostgresSchemaPathUsesLifecycleScenarioDirectory(t *testing.T) {
	scenarioDir := t.TempDir()
	t.Setenv("VROOLI_SCENARIO_DIR", scenarioDir)

	path, err := postgresSchemaPath()
	if err != nil {
		t.Fatalf("postgres schema path: %v", err)
	}
	want := filepath.Join(scenarioDir, "initialization", "storage", "postgres", "schema.sql")
	if path != want {
		t.Fatalf("postgres schema path = %q, want %q", path, want)
	}
}

func TestPostgresSchemaPathResolvesExistingScenarioSchema(t *testing.T) {
	t.Setenv("VROOLI_SCENARIO_DIR", "")

	path, err := postgresSchemaPath()
	if err != nil {
		t.Fatalf("postgres schema path: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("scenario PostgreSQL schema is not available at %s: %v", path, err)
	}
}
