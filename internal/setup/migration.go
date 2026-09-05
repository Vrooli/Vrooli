package setup

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/vrooli/vrooli/internal/tuning"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/projectstate"
)

//nolint:gocyclo // ownership migration coordinates discovery, confirmation, mutation, and recovery reporting.
func runOwnershipMigration(locator projectstate.Locator, stdout, stderr io.Writer) error {
	ledger, err := projectstate.LoadMigrationLedger(locator)
	if err != nil {
		return err
	}
	record, exists := ledger.Migrations[config.RuntimeHomeOwnershipMigration]
	if exists && record.Status == projectstate.MigrationComplete && record.AppliedThrough >= ownershipMigrationVersion {
		_, _ = fmt.Fprintln(stdout, "Filesystem ownership migration: already complete; broad scan skipped.")
		return nil
	}
	uid, gid := config.RepairIdentity()
	scope := projectstate.MigrationScope{Kind: "runtime_home", Classes: append([]string(nil), ownershipMigrationClasses...)}
	if !exists || record.AppliedThrough != ownershipMigrationVersion || !sameMigrationScope(record.Scope, scope) {
		record = projectstate.MigrationRecord{
			AppliedThrough: ownershipMigrationVersion,
			Scope:          scope,
			Expected:       projectstate.MigrationExpectedIdentity{UID: uid, GID: gid},
			Cursors:        map[string]string{},
		}
	} else {
		if record.Cursors == nil {
			record.Cursors = map[string]string{}
		}
		// Ledgers written by the first version of this migration had no
		// continuation cursor. Their counters describe a failed prefix, not
		// durable progress, so retry from the root without double-counting it.
		if len(record.Cursors) == 0 && record.Result.Scanned > 0 && len(record.Completed) == 0 {
			record.Result = projectstate.MigrationResult{}
		}
	}
	record.AppliedThrough = ownershipMigrationVersion
	record.Scope = scope
	record.Expected = projectstate.MigrationExpectedIdentity{UID: uid, GID: gid}
	record.Status = projectstate.MigrationRunning
	record.StartedAt = time.Now().UTC().Format(time.RFC3339)
	record.CompletedAt = ""
	record.LastError = ""
	ledger.Migrations[config.RuntimeHomeOwnershipMigration] = record
	if err := projectstate.SaveMigrationLedger(locator, ledger); err != nil {
		return err
	}

	service := config.RepairService{ResolveRoot: func(class string) (string, error) {
		key := class
		switch class {
		case "artifacts":
			key = "artifacts"
		case "backups":
			key = "backups"
		}
		return repocontract.RuntimeHomeEntryPath(locator.Home(), key)
	}}
	_, _, apply := hostreqkit.InvokingUserIDs()
	if !apply {
		record.Status = projectstate.MigrationPending
		record.LastError = "setup was not elevated; ownership migration deferred"
		ledger.Migrations[config.RuntimeHomeOwnershipMigration] = record
		if err := projectstate.SaveMigrationLedger(locator, ledger); err != nil {
			return err
		}
		_, _ = fmt.Fprintln(stdout, "Filesystem ownership migration: setup was not elevated; migration deferred.")
		return nil
	}
	deadline := time.Now().Add(tuning.RepairDeadline())
	for _, class := range ownershipMigrationClasses {
		if containsMigrationClass(record.Completed, class) {
			continue
		}
		for {
			request := config.RepairRequest{
				Scope:       config.RepairScope{RootClass: class, Legacy: true},
				ExpectedUID: uid,
				ExpectedGID: gid,
				Apply:       apply,
				MaxEntries:  ownershipMigrationBatchEntries,
				Deadline:    deadline,
				ResumeAfter: record.Cursors[class],
			}
			result, repairErr := service.Repair(context.Background(), request)
			if repairErr != nil {
				record.Status = projectstate.MigrationFailed
				record.LastError = repairErr.Error()
				ledger.Migrations[config.RuntimeHomeOwnershipMigration] = record
				_ = projectstate.SaveMigrationLedger(locator, ledger)
				return fmt.Errorf("ownership migration %s: %w", class, repairErr)
			}
			record.Result.Scanned += result.Scanned
			record.Result.Repaired += result.Repaired
			record.Result.Skipped += result.Skipped
			record.Result.Failed += result.Failed
			if result.Failed > 0 {
				record.Status = projectstate.MigrationInterrupted
				record.LastError = fmt.Sprintf("repair of %s failed", class)
				ledger.Migrations[config.RuntimeHomeOwnershipMigration] = record
				_ = projectstate.SaveMigrationLedger(locator, ledger)
				return fmt.Errorf("ownership migration %s incomplete", class)
			}
			if result.Status == config.RepairComplete {
				delete(record.Cursors, class)
				record.Completed = appendUniqueMigrationClass(record.Completed, class)
				ledger.Migrations[config.RuntimeHomeOwnershipMigration] = record
				if err := projectstate.SaveMigrationLedger(locator, ledger); err != nil {
					return err
				}
				break
			}
			if result.LastPath == "" {
				record.Status = projectstate.MigrationInterrupted
				record.LastError = fmt.Sprintf("repair of %s incomplete without a continuation point", class)
				ledger.Migrations[config.RuntimeHomeOwnershipMigration] = record
				_ = projectstate.SaveMigrationLedger(locator, ledger)
				return fmt.Errorf("ownership migration %s incomplete", class)
			}
			record.Cursors[class] = result.LastPath
			ledger.Migrations[config.RuntimeHomeOwnershipMigration] = record
			if err := projectstate.SaveMigrationLedger(locator, ledger); err != nil {
				return err
			}
			if time.Now().After(deadline) {
				record.Status = projectstate.MigrationInterrupted
				record.LastError = fmt.Sprintf("repair of %s exceeded migration deadline", class)
				ledger.Migrations[config.RuntimeHomeOwnershipMigration] = record
				_ = projectstate.SaveMigrationLedger(locator, ledger)
				return fmt.Errorf("ownership migration %s incomplete", class)
			}
		}
	}
	record.Status = projectstate.MigrationComplete
	record.CompletedAt = time.Now().UTC().Format(time.RFC3339)
	record.Result.Duration = time.Since(parseMigrationStart(record.StartedAt)).Milliseconds()
	record.LastError = ""
	record.Cursors = nil
	record.Completed = nil
	ledger.Migrations[config.RuntimeHomeOwnershipMigration] = record
	if err := projectstate.SaveMigrationLedger(locator, ledger); err != nil {
		return err
	}
	_, _ = fmt.Fprintf(stdout, "Filesystem ownership migration: scanned=%d repaired=%d skipped=%d failed=%d.\n", record.Result.Scanned, record.Result.Repaired, record.Result.Skipped, record.Result.Failed)
	_ = stderr
	return nil
}

func sameMigrationScope(left, right projectstate.MigrationScope) bool {
	if left.Kind != right.Kind || len(left.Classes) != len(right.Classes) {
		return false
	}
	for i := range left.Classes {
		if left.Classes[i] != right.Classes[i] {
			return false
		}
	}
	return true
}

func containsMigrationClass(classes []string, class string) bool {
	for _, candidate := range classes {
		if candidate == class {
			return true
		}
	}
	return false
}

func appendUniqueMigrationClass(classes []string, class string) []string {
	if containsMigrationClass(classes, class) {
		return classes
	}
	return append(classes, class)
}

func parseMigrationStart(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return time.Now()
	}
	return parsed
}

// installSelectedCLIs keeps CLI installation aligned with the same selectors
// used for host requirements and resource installation. In particular, an
// explicit "none" must not build every CLI in the repository: a minimal fresh
// host should only pay for the capabilities the operator selected.
