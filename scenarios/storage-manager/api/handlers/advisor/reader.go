package advisor

import (
	"context"

	"storage-manager/internal/advisor"
	"storage-manager/internal/validation"
)

// validator is the slice of the validation engine the reader needs.
type validator interface {
	ValidateScenario(ctx context.Context, scenario string) (validation.Report, error)
}

// migrationNoteCodes are the finding codes whose messages become migration-debt
// notes: the stage-aware MIGRATION_DEBT plus the schema-hygiene codes that a
// migration must resolve before it is safe.
var migrationNoteCodes = map[string]bool{
	"MIGRATION_DEBT":        true,
	"SCHEMA_HAS_ALTER":      true,
	"SCHEMA_NOT_IDEMPOTENT": true,
	"SCHEMA_CENTRALIZED":    true,
}

// reader is the production advisor.Reader: it runs the storage validation and
// projects its Report onto the normalized facts the advisor reasons over.
type reader struct {
	validator validator
}

func newReader(v validator) *reader { return &reader{validator: v} }

var _ advisor.Reader = (*reader)(nil)

func (r *reader) Read(ctx context.Context, scenario string) (advisor.ScenarioFacts, error) {
	report, err := r.validator.ValidateScenario(ctx, scenario)
	if err != nil {
		return advisor.ScenarioFacts{}, err
	}
	facts := advisor.ScenarioFacts{
		Scenario:      scenario,
		StorageStage:  report.StorageStage,
		HasMigrations: report.HasMigrations,
	}
	for _, e := range report.Engines {
		facts.Engines = append(facts.Engines, string(e))
	}
	for _, f := range report.Findings {
		switch f.Code {
		case "SCHEMA_HAS_ALTER":
			facts.HasAlterInSchema = true
		case "SCHEMA_NOT_IDEMPOTENT":
			facts.NonIdempotentSchema = true
		}
		if migrationNoteCodes[f.Code] {
			facts.DebtNotes = append(facts.DebtNotes, f.Message)
		}
	}
	return facts, nil
}
