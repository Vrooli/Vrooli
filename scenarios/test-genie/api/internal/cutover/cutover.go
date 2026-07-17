// Package cutover composes the separately owned evidence and SQLite cutovers
// into the one operator operation. It owns only cross-store ordering and
// compensation; each store retains its inventory and mutation authority.
package cutover

import (
	"fmt"
	"path/filepath"

	"test-genie/internal/executionevidence"
	"test-genie/internal/persistence"
)

const Confirmation = executionevidence.CutoverConfirmation

type Plan struct {
	Evidence executionevidence.CutoverPlan
	Database persistence.DatabaseCutoverPlan
}

func PlanOffline(scenarioDir, evidenceArchive, liveDatabase, databaseArchive string) (Plan, error) {
	evidence, err := executionevidence.PlanCutover(scenarioDir, evidenceArchive)
	if err != nil {
		return Plan{}, err
	}
	database, err := persistence.PlanDatabaseCutover(liveDatabase, databaseArchive)
	if err != nil {
		return Plan{}, err
	}
	return Plan{Evidence: evidence, Database: database}, nil
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
