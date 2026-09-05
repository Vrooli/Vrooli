// Package persistence owns construction of an empty Test Genie SQLite store.
// Runtime startup and offline evidence cutover use this one schema authority.
package persistence

import (
	"context"
	"fmt"

	"test-genie/internal/dbexec"
	"test-genie/internal/execution"
	"test-genie/internal/playbooksclaims"
	"test-genie/internal/remediation"
	"test-genie/internal/selfhealthsnapshots"
)

// ApplySchema initializes an empty embedded SQLite store. Seed data is optional
// so an offline cutover never invents preview records in its replacement DB.
func ApplySchema(db dbexec.Executor, includeSeed bool) error {
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
	return nil
}
