package validation

import (
	"context"
	"fmt"
	"strings"
)

func init() {
	register(&backupTargetMissing{})
}

// backupTargetMissing emits BACKUP_TARGET_MISSING (L4 advisory) when a
// data-persisting scenario that has reached a deployed stage (pilot/production)
// declares no backup target — neither a `backup` block in service.json nor a
// data-backup-manager scenario dependency.
//
// It is intentionally advisory (INFO): runtime backup registrations are
// invisible to a static analyzer, so this is the declaration-based signal that
// a deployed scenario carrying durable state has no *visible* recovery path. It
// never gates a phase; greenfield scenarios (not yet deployed) are exempt.
type backupTargetMissing struct{}

func (backupTargetMissing) Name() string { return "persistence.backup-target-missing" }

func (backupTargetMissing) Applies(ac AnalyzerContext) bool {
	if !isDataPersisting(ac.Engines) {
		return false
	}
	switch strings.ToLower(strings.TrimSpace(ac.StorageStage)) {
	case "pilot", "production":
		return true
	default:
		// greenfield (not yet deployed) and sunset (being decommissioned) do not
		// raise a backup-readiness advisory.
		return false
	}
}

func (backupTargetMissing) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	if hasBackupTarget(ac.ScenarioDir) {
		return nil, nil
	}
	engines := make([]string, 0, len(ac.Engines))
	for _, e := range ac.Engines {
		engines = append(engines, string(e))
	}
	return []Finding{{
		Code:     "BACKUP_TARGET_MISSING",
		Severity: SeverityInfo,
		Title:    "Deployed data-persisting scenario has no declared backup target",
		Message: fmt.Sprintf(
			"%s is at the %q stage and persists durable state (%s) but declares no backup target — "+
				"no `backup` block in .vrooli/service.json and no data-backup-manager dependency. "+
				"A deployed scenario without a visible recovery path risks unrecoverable data loss.",
			ac.Scenario, ac.StorageStage, strings.Join(engines, ", ")),
		Location:    ".vrooli/service.json",
		Remediation: "Add data-backup-manager to dependencies.scenarios (and register the scenario's targets via `data-backup-manager safety register-targets`), or declare a `backup` block in service.json.",
		Analyzer:    "persistence.backup-target-missing",
	}}, nil
}
