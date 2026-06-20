package runtime

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"

	"test-genie/internal/dbexec"
	"test-genie/internal/execution"
	"test-genie/internal/playbooksclaims"
	"test-genie/internal/storage/sqlfiles"

	repocontract "github.com/vrooli/repo-contract-go"
)

const initializationDialectDir = "sqlite"

// ApplySchema initializes the embedded SQLite schema for Test Genie.
// Seed data is optional so one-time migration utilities can provision a clean
// target without adding preview records. The handle is the narrow
// dbexec.Executor seam (production *database.RoutedDB or a test *sql.DB).
func ApplySchema(db dbexec.Executor, includeSeed bool) error {
	schemaPath, err := resolveInitializationFile("schema.sql")
	if err != nil {
		return fmt.Errorf("initialization schema lookup failed: %w", err)
	}
	if err := sqlfiles.ExecFile(db, schemaPath); err != nil {
		return err
	}
	if err := applyDomainSchemas(db); err != nil {
		return err
	}
	if err := applyDomainMigrations(db); err != nil {
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

func ensureDatabaseSchema(db dbexec.Executor) error {
	return ApplySchema(db, true)
}

// applyDomainSchemas runs the declarative DDL owned by per-domain packages.
// Each schema must be idempotent (CREATE TABLE IF NOT EXISTS) so this is
// safe to call on every boot.
func applyDomainSchemas(db dbexec.Executor) error {
	domains := []struct {
		name string
		ddl  string
	}{
		{"execution", execution.Schema()},
		{"playbooksclaims", playbooksclaims.Schema()},
	}
	for _, d := range domains {
		if _, err := db.ExecContext(context.Background(), d.ddl); err != nil {
			return fmt.Errorf("domain schema %q: %w", d.name, err)
		}
	}
	return nil
}

// applyDomainMigrations runs guarded, idempotent column-evolution migrations
// owned by per-domain packages, after applyDomainSchemas. Unlike the declarative
// CREATE TABLE IF NOT EXISTS schemas (which cannot add a column to an existing
// data-bearing table), these introspect current state and ALTER in place,
// preserving accumulated data. Each migration must be safe to run on every boot.
func applyDomainMigrations(db dbexec.Executor) error {
	migrations := []struct {
		name string
		run  func(context.Context, dbexec.Executor) error
	}{
		{"execution", execution.Migrate},
	}
	for _, m := range migrations {
		if err := m.run(context.Background(), db); err != nil {
			return fmt.Errorf("domain migration %q: %w", m.name, err)
		}
	}
	return nil
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
		root, err := repocontract.FindRepoRoot(currentFile)
		if err == nil {
			path, resolveErr := repocontract.ResolveScenarioPath(root, "test-genie")
			if resolveErr == nil {
				return path, nil
			}
		}
	}

	root, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		return "", fmt.Errorf("failed to determine test-genie scenario root: %w", err)
	}
	path, err := repocontract.ResolveScenarioPath(root, "test-genie")
	if err != nil {
		return "", fmt.Errorf("resolve test-genie scenario path: %w", err)
	}
	return path, nil
}
