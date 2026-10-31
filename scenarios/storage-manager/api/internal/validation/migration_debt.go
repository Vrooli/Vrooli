package validation

import (
	"context"
	"fmt"
	"strings"
)

func init() {
	register(&migrationDebt{})
}

// migrationDebt emits MIGRATION_DEBT (L4 advisory) for a stage/migrations
// mismatch: a deployed (pilot/production) relational scenario with no committed
// migrations/ path (schema changes would have no forward-migration route), or a
// greenfield scenario that already carries a migrations/ directory (greenfield
// expects schema-as-desired-state, not accumulated deltas).
//
// Migration *choices* are informational per the locked decision (§5.5): this is
// advisory (INFO) and never hard-fails. Unambiguous schema violations
// (ALTER-in-schema, non-idempotent, centralized) are owned by the dedicated
// schema-structure analyzers and fail at any stage.
type migrationDebt struct{}

func (migrationDebt) Name() string { return "migration.debt" }

func (migrationDebt) Applies(ac AnalyzerContext) bool {
	// Only relational scenarios accrue migration debt; non-relational stores
	// (Qdrant/Redis/file) have no embedded SQL schema to migrate.
	return ac.HasRelationalStore()
}

func (migrationDebt) Analyze(_ context.Context, ac AnalyzerContext) ([]Finding, error) {
	stage := strings.ToLower(strings.TrimSpace(ac.StorageStage))
	note, remediation := migrationDebtNote(stage, ac.HasMigrations)
	if note == "" {
		return nil, nil
	}
	return []Finding{{
		Code:        "MIGRATION_DEBT",
		Severity:    SeverityInfo,
		Title:       "Migration posture does not match deploy stage",
		Message:     fmt.Sprintf("%s (stage %q): %s", ac.Scenario, ac.StorageStage, note),
		Location:    ".vrooli/service.json",
		Remediation: remediation,
		Analyzer:    "migration.debt",
	}}, nil
}

// migrationDebtNote returns the stage-aware debt note (and remediation), or ""
// when the posture matches the stage. Pure for direct unit testing.
func migrationDebtNote(stage string, hasMigrations bool) (note, remediation string) {
	switch stage {
	case "pilot", "production":
		if !hasMigrations {
			return "a deployed relational scenario has no committed migrations/ directory, so a schema change has no forward-migration path against live data.",
				"Add a migrations/ directory and route schema deltas through versioned migrations; keep the embedded schema.sql to idempotent CREATE … IF NOT EXISTS only."
		}
	case "greenfield", "":
		if hasMigrations {
			return "a greenfield (not-yet-deployed) scenario already carries a migrations/ directory; greenfield expects schema-as-desired-state (idempotent schema.sql), not accumulated migration deltas.",
				"Fold the migrations into the embedded schema.sql while greenfield, or set maturity to pilot/production in service.json if the scenario is in fact deployed."
		}
	}
	return "", ""
}
