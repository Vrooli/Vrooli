// Package cutover composes the separately owned evidence and SQLite cutovers
// into the one operator operation. It owns only cross-store ordering and
// compensation; each store retains its inventory and mutation authority.
package cutover

import (
	"fmt"
	"os"
	"path/filepath"

	"test-genie/internal/executionevidence"
	"test-genie/internal/persistence"
	sharedruns "test-genie/internal/shared/runs"
)

const Confirmation = executionevidence.CutoverConfirmation

type Plan struct {
	Evidence          executionevidence.CutoverPlan
	Database          persistence.DatabaseCutoverPlan
	RequiredFreeBytes int64
}

func PlanOffline(scenarioDir, evidenceArchive, liveDatabase, databaseArchive string) (Plan, error) {
	if err := Preflight(scenarioDir); err != nil {
		return Plan{}, err
	}
	evidence, err := executionevidence.PlanCutover(scenarioDir, evidenceArchive)
	if err != nil {
		return Plan{}, err
	}
	database, err := persistence.PlanDatabaseCutover(liveDatabase, databaseArchive)
	if err != nil {
		return Plan{}, err
	}
	// The operator needs space for both rollback archives before the original
	// stores can be moved. Replacement-store overhead is intentionally not
	// guessed; the runbook requires additional headroom on the live volume.
	return Plan{Evidence: evidence, Database: database, RequiredFreeBytes: evidence.Bytes + database.Bytes}, nil
}

// Preflight rejects a cutover while the durable index still names a queued or
// running suite. WAL/SHM checks remain the SQLite-side stopped-process proof.
func Preflight(scenarioDir string) error {
	index := sharedruns.NewIndex(scenarioDir)
	// A missing index is an empty history. Avoid List in that case because its
	// advisory lock would create a file, violating the read-only plan contract.
	if _, err := os.Stat(index.Path()); os.IsNotExist(err) {
		return nil
	} else if err != nil {
		return fmt.Errorf("inspect run index for cutover preflight: %w", err)
	}
	records, err := index.List()
	if err != nil {
		return fmt.Errorf("read run index for cutover preflight: %w", err)
	}
	for _, record := range records {
		if record.Status == sharedruns.StatusQueued || record.Status == sharedruns.StatusInProgress {
			return fmt.Errorf("cutover preflight rejected: run %s is %s", record.RunID, record.Status)
		}
	}
	return nil
}

// ApplyOffline performs all read-only validation before changing either store.
// Database goes first because a later evidence failure has a deterministic
// compensating restore; successful completion leaves two independent receipts.
func ApplyOffline(plan Plan, confirmation string) error {
	if confirmation != Confirmation {
		return executionevidence.ErrCutoverNotConfirmed
	}
	if _, err := PlanOfflineFromPlan(plan); err != nil {
		return err
	}
	if err := persistence.ApplyDatabaseCutover(plan.Database, confirmation); err != nil {
		return err
	}
	if err := executionevidence.ApplyCutover(plan.Evidence, confirmation); err != nil {
		if rollbackErr := persistence.RestoreDatabaseCutover(plan.Database); rollbackErr != nil {
			return fmt.Errorf("archive evidence: %w; database compensation failed: %v", err, rollbackErr)
		}
		return fmt.Errorf("archive evidence: %w; database restored", err)
	}
	return nil
}

func PlanOfflineFromPlan(plan Plan) (Plan, error) {
	return PlanOffline(filepath.Dir(plan.Evidence.CoverageRoot), plan.Evidence.ArchiveRoot, plan.Database.LivePath, plan.Database.ArchivePath)
}
