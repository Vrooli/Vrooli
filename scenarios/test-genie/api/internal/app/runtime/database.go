package runtime

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"

	"test-genie/internal/storage/sqlfiles"
)

const initializationDialectDir = "sqlite"

// ApplySchema initializes the embedded SQLite schema for Test Genie.
// Seed data is optional so one-time migration utilities can provision a clean
// target without adding preview records.
func ApplySchema(db *sql.DB, includeSeed bool) error {
	schemaPath, err := resolveInitializationFile("schema.sql")
	if err != nil {
		return fmt.Errorf("initialization schema lookup failed: %w", err)
	}
	if err := sqlfiles.ExecFile(db, schemaPath); err != nil {
		return err
	}
	if !includeSeed {
		return nil
	}
	if seedPath, err := resolveInitializationFile("seed.sql"); err == nil {
		if err := sqlfiles.ExecFile(db, seedPath); err != nil {
			return err
		}
	}
	return nil
}

func ensureDatabaseSchema(db *sql.DB) error {
	return ApplySchema(db, true)
}

func resolveInitializationFile(name string) (string, error) {
	scenarioDir, err := scenarioRoot()
	if err != nil {
		return "", err
	}
	target := filepath.Join(scenarioDir, "initialization", initializationDialectDir, name)
	if _, err := os.Stat(target); err != nil {
		return "", fmt.Errorf("initialization file not accessible (%s): %w", target, err)
	}
	return target, nil
}

func scenarioRoot() (string, error) {
	_, currentFile, _, ok := goruntime.Caller(0)
	if ok {
		return filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", "..", "..")), nil
	}

	wd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("failed to determine working directory: %w", err)
	}
	return filepath.Dir(wd), nil
}
