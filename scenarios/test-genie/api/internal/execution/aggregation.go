package execution

import (
	"context"
	"time"

	"test-genie/internal/storage/sqliteutil"
)

// PhaseObservation is a single flattened per-phase row drawn from a windowed set
// of runs. It carries the run-level terminal_outcome alongside the per-phase
// signal so the reliability ledger can correlate phase behaviour with
// catastrophic run results. All SQL that produces these lives in the
// engine-neutral SuiteExecutionRepository (storage-steer §8); the selfhealth
// ledger composes over these values without embedding SQL.
type PhaseObservation struct {
	ScenarioName       string
	TerminalOutcome    string
	PhaseName          string
	Status             string
	Classification     string
	RunnabilityVerdict string
	RunnabilityReason  string
	FindingSource      string
	DurationSeconds    int
	MetricsPresent     bool
	CompletedAt        time.Time
}

// RunOutcomeCount is one bucket of the run-level terminal_outcome histogram over
// the window. Catastrophic outcomes (errored/aborted/timeout) are only correctly
// represented here because Phase B4a persists a row for every terminal run.
type RunOutcomeCount struct {
	TerminalOutcome string
	Count           int
}

// defaultAggregationRowCap bounds the number of runs scanned by the windowed
// aggregation queries. It is deliberately distinct from (and larger than) the
// 50-row UI history cap (orchestrator.MaxExecutionHistory): the ledger samples a
// wider window for statistical signal, not the most-recent-runs UI list.
const defaultAggregationRowCap = 5000

// AggregatePhaseObservations returns the per-phase observations across the most
// recent runs (capped at limit) completed at or after since. The cap is applied
// to runs BEFORE the json_each explosion so it bounds scan cost precisely (one
// run yields many phase rows). Mirrors the ListPhaseSamples json_each precedent.
func (r *SuiteExecutionRepository) AggregatePhaseObservations(ctx context.Context, since time.Time, limit int) ([]PhaseObservation, error) {
	if limit <= 0 || limit > defaultAggregationRowCap {
		limit = defaultAggregationRowCap
	}

	const q = `
SELECT
	e.scenario_name,
	COALESCE(e.terminal_outcome, '') AS terminal_outcome,
	LOWER(TRIM(json_extract(phase.value, '$.name'))) AS phase_name,
	LOWER(TRIM(COALESCE(json_extract(phase.value, '$.status'), ''))) AS status,
	LOWER(TRIM(COALESCE(json_extract(phase.value, '$.classification'), ''))) AS classification,
	LOWER(TRIM(COALESCE(json_extract(phase.value, '$.runnabilityVerdict'), ''))) AS runnability_verdict,
	TRIM(COALESCE(json_extract(phase.value, '$.runnabilityReason'), '')) AS runnability_reason,
	LOWER(TRIM(COALESCE(json_extract(phase.value, '$.findingSource'), ''))) AS finding_source,
	MAX(CAST(COALESCE(json_extract(phase.value, '$.durationSeconds'), 0) AS INTEGER), 0) AS duration_seconds,
	CASE WHEN json_extract(phase.value, '$.metrics') IS NOT NULL THEN 1 ELSE 0 END AS metrics_present,
	e.completed_at
FROM (
	SELECT id, scenario_name, terminal_outcome, phases, completed_at
	FROM suite_executions
	WHERE completed_at >= ?
	ORDER BY completed_at DESC
	LIMIT ?
) AS e
JOIN json_each(e.phases) AS phase
WHERE LENGTH(LOWER(TRIM(json_extract(phase.value, '$.name')))) > 0
`

	rows, err := r.db.QueryContext(ctx, q, sqliteutil.FormatTimestamp(since), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var observations []PhaseObservation
	for rows.Next() {
		var (
			obs            PhaseObservation
			metricsPresent int
			completedAt    any
		)
		if err := rows.Scan(
			&obs.ScenarioName,
			&obs.TerminalOutcome,
			&obs.PhaseName,
			&obs.Status,
			&obs.Classification,
			&obs.RunnabilityVerdict,
			&obs.RunnabilityReason,
			&obs.FindingSource,
			&obs.DurationSeconds,
			&metricsPresent,
			&completedAt,
		); err != nil {
			return nil, err
		}
		obs.MetricsPresent = metricsPresent == 1
		obs.CompletedAt, err = sqliteutil.ParseTimestamp(completedAt)
		if err != nil {
			return nil, err
		}
		observations = append(observations, obs)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return observations, nil
}

// CountRunOutcomes returns the run-level terminal_outcome histogram across the
// most recent runs (capped at limit) completed at or after since. It is the
// denominator source for suite availability — correct only because Phase B4a
// persists catastrophic outcomes.
func (r *SuiteExecutionRepository) CountRunOutcomes(ctx context.Context, since time.Time, limit int) ([]RunOutcomeCount, error) {
	if limit <= 0 || limit > defaultAggregationRowCap {
		limit = defaultAggregationRowCap
	}

	const q = `
SELECT outcome, COUNT(*) AS n
FROM (
	SELECT COALESCE(terminal_outcome, '') AS outcome
	FROM suite_executions
	WHERE completed_at >= ?
	ORDER BY completed_at DESC
	LIMIT ?
)
GROUP BY outcome
ORDER BY n DESC
`

	rows, err := r.db.QueryContext(ctx, q, sqliteutil.FormatTimestamp(since), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var counts []RunOutcomeCount
	for rows.Next() {
		var c RunOutcomeCount
		if err := rows.Scan(&c.TerminalOutcome, &c.Count); err != nil {
			return nil, err
		}
		counts = append(counts, c)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return counts, nil
}
