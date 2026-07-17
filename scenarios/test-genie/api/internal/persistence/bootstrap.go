// Package persistence owns construction of an empty Test Genie SQLite store.
// Runtime startup and offline evidence cutover use this one schema authority.
package persistence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	goruntime "runtime"

	"test-genie/internal/dbexec"
	"test-genie/internal/execution"
	"test-genie/internal/playbooksclaims"
	"test-genie/internal/remediation"
	"test-genie/internal/selfhealthsnapshots"
	"test-genie/internal/storage/sqlfiles"

	repocontract "github.com/vrooli/repo-contract-go"
)

const initializationDialectDir = "sqlite"

// ApplySchema initializes an empty embedded SQLite store. Seed data is optional
// so an offline cutover never invents preview records in its replacement DB.
func ApplySchema(db dbexec.Executor, includeSeed bool) error {
	schemaPath, err := initializationFile("schema.sql")
	if err != nil {
		return fmt.Errorf("initialization schema lookup failed: %w", err)
	}
	if err := sqlfiles.ExecFile(db, schemaPath); err != nil {
		return err
	}
	for _, domain := range []struct {
		name string
		ddl  string
	}{
		{"execution", execution.Schema()},
		{"playbooksclaims", playbooksclaims.Schema()},
		{"remediation", remediation.Schema()},
		{"selfhealthsnapshots", selfhealthsnapshots.Schema()},
	} {
		if _, err := db.ExecContext(context.Background(), domain.ddl); err != nil {
			return fmt.Errorf("domain schema %q: %w", domain.name, err)
		}
	}
	for _, migration := range []struct {
		name string
		run  func(context.Context, dbexec.Executor) error
	}{
		{"execution", execution.Migrate},
		{"remediation", remediation.Migrate},
	} {
		if err := migration.run(context.Background(), db); err != nil {
			return fmt.Errorf("domain migration %q: %w", migration.name, err)
		}
	}
	if !includeSeed {
		return nil
	}
	if seedPath, err := initializationFile("seed.sql"); err == nil {
		return sqlfiles.ExecFile(db, seedPath)
	}
	return nil
}

func initializationFile(name string) (string, error) {
	_, currentFile, _, ok := goruntime.Caller(0)
	if !ok {
		return "", fmt.Errorf("locate persistence package")
	}
	root, err := repocontract.FindRepoRoot(currentFile)
	if err != nil {
		return "", err
	}
	scenarioDir, err := repocontract.ResolveScenarioPath(root, "test-genie")
	if err != nil {
		return "", err
	}
	target := filepath.Join(scenarioDir, "initialization", initializationDialectDir, name)
	if _, err := os.Stat(target); err != nil {
		return "", fmt.Errorf("initialization file not accessible (%s): %w", target, err)
	}
	return target, nil
}
