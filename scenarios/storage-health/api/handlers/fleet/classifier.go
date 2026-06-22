package fleet

import (
	"context"

	"storage-health/internal/autofix"
	"storage-health/internal/fleet"
	"storage-health/internal/validation"
)

// validator is the slice of the validation engine the classifier needs.
type validator interface {
	ValidateScenario(ctx context.Context, scenario string) (validation.Report, error)
}

// classifier turns a scenario into a fleet.ScenarioEntry by running the full
// storage validation and projecting its Report onto the inventory fields. It is
// the production fleet.Classifier; tests drive a fake classifier against the
// pure engine instead.
//
// isolation_ready is the safety throughline: a scenario is isolation-ready iff
// its L2 isolation rung is clean — no ROUTED_SEAMS_UNWIRED (Go seams unproven)
// and no STORAGE_ISOLATION_UNVERIFIED (non-Go fail-safe). namespace_adopted is
// the inverse of a STORAGE_NAMESPACE_HARDCODED finding.
type classifier struct {
	validator validator
}

func newClassifier(v validator) *classifier { return &classifier{validator: v} }

var _ fleet.Classifier = (*classifier)(nil)

func (c *classifier) Classify(ctx context.Context, scenario string) (fleet.ScenarioEntry, error) {
	report, err := c.validator.ValidateScenario(ctx, scenario)
	if err != nil {
		return fleet.ScenarioEntry{}, err
	}

	entry := fleet.ScenarioEntry{
		Scenario:         scenario,
		Engines:          enginesToStrings(report.Engines),
		PrimaryEngine:    primaryEngine(report.Engines),
		Language:         report.Language,
		StorageStage:     report.StorageStage,
		IsolationReady:   true,
		NamespaceAdopted: true,
		HasBackupTarget:  report.ScenarioDir != "" && validation.HasBackupTarget(report.ScenarioDir),
	}

	for _, f := range report.Findings {
		entry.FindingCount++
		if f.Severity == validation.SeverityError {
			entry.ErrorCount++
		}
		if f.AutofixAvailable || autofix.CoveredCodes[f.Code] {
			entry.AutofixableCount++
		}
		switch f.Code {
		case "ROUTED_SEAMS_UNWIRED", "STORAGE_ISOLATION_UNVERIFIED":
			entry.IsolationReady = false
			if entry.IsolationReason == "" {
				entry.IsolationReason = f.Title
			}
		case "STORAGE_NAMESPACE_HARDCODED":
			entry.NamespaceAdopted = false
		}
	}
	return entry, nil
}

// enginesToStrings projects the typed engine set onto wire strings.
func enginesToStrings(engines []validation.Engine) []string {
	if len(engines) == 0 {
		return nil
	}
	out := make([]string, 0, len(engines))
	for _, e := range engines {
		out = append(out, string(e))
	}
	return out
}

// primaryEngine picks the dominant relational engine (Postgres beats SQLite),
// falling back to the first declared engine, or "" when none.
func primaryEngine(engines []validation.Engine) string {
	hasPG, hasSQLite := false, false
	for _, e := range engines {
		switch e {
		case validation.EnginePostgres:
			hasPG = true
		case validation.EngineSQLite:
			hasSQLite = true
		}
	}
	switch {
	case hasPG:
		return string(validation.EnginePostgres)
	case hasSQLite:
		return string(validation.EngineSQLite)
	case len(engines) > 0:
		return string(engines[0])
	default:
		return ""
	}
}
