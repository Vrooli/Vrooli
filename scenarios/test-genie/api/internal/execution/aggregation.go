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
// to runs before the compact phase-row join, so it bounds scan cost precisely
// without parsing a JSON result document for every historical execution.
func (r *SuiteExecutionRepository) AggregatePhaseObservations(ctx context.Context, since time.Time, limit int) ([]PhaseObservation, error) {
	if limit <= 0 || limit > defaultAggregationRowCap {
		limit = defaultAggregationRowCap
	}

	const q = `
SELECT
	e.scenario_name,
	COALESCE(e.terminal_outcome, '') AS terminal_outcome,
	p.phase_name,
	p.status,
	p.classification,
	p.runnability_verdict,
	p.runnability_reason,
	p.finding_source,
	p.duration_seconds,
	p.metrics_present,
	e.completed_at
FROM (
	SELECT id, scenario_name, terminal_outcome, completed_at
	FROM suite_executions
	WHERE completed_at >= ?
	ORDER BY completed_at DESC
	LIMIT ?
) AS e
JOIN suite_execution_phases AS p ON p.execution_id = e.id
WHERE p.phase_name <> ''
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

// ScenarioRunRollup is one scenario's run-level rollup over the window: how many
// runs it had, how many produced a passed terminal outcome, and the newest run's
// completion time + outcome (the scenario-level staleness/last-status signal).
// It is the per-scenario analogue of CountRunOutcomes, used by the FLEET ledger.
type ScenarioRunRollup struct {
	ScenarioName    string
	Runs            int
	Passed          int
	LastCompletedAt time.Time
	LastOutcome     string
}

// AggregateScenarioRuns returns the per-scenario run-level rollup across the most
// recent runs (capped at limit) completed at or after since. The newest run per
// scenario (by completed_at) supplies LastCompletedAt/LastOutcome; the window
// function is evaluated before LIMIT, so the "last run" reflects the windowed
// set. This is the fleet ledger's run-count + staleness source; like the other
// windowed aggregates it holds the SQL here (storage-steer §8) so the ledger
// stays engine-portable.
func (r *SuiteExecutionRepository) AggregateScenarioRuns(ctx context.Context, since time.Time, limit int) ([]ScenarioRunRollup, error) {
	if limit <= 0 || limit > defaultAggregationRowCap {
		limit = defaultAggregationRowCap
	}

	const q = `
SELECT scenario_name,
	COUNT(*) AS runs,
	SUM(CASE WHEN outcome = 'passed' THEN 1 ELSE 0 END) AS passed,
	MAX(completed_at) AS last_completed_at,
	MAX(CASE WHEN rn = 1 THEN outcome END) AS last_outcome
FROM (
	SELECT scenario_name,
		LOWER(TRIM(COALESCE(terminal_outcome, ''))) AS outcome,
		completed_at,
		ROW_NUMBER() OVER (PARTITION BY scenario_name ORDER BY completed_at DESC, id DESC) AS rn
	FROM (
		SELECT id, scenario_name, terminal_outcome, completed_at
		FROM suite_executions
		WHERE completed_at >= ?
		ORDER BY completed_at DESC
		LIMIT ?
	)
)
GROUP BY scenario_name
ORDER BY runs DESC, scenario_name ASC
`

	rows, err := r.db.QueryContext(ctx, q, sqliteutil.FormatTimestamp(since), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rollups []ScenarioRunRollup
	for rows.Next() {
		var (
			roll        ScenarioRunRollup
			lastOutcome any
			completedAt any
		)
		if err := rows.Scan(&roll.ScenarioName, &roll.Runs, &roll.Passed, &completedAt, &lastOutcome); err != nil {
			return nil, err
		}
		if s, ok := lastOutcome.(string); ok {
			roll.LastOutcome = s
		}
		roll.LastCompletedAt, err = sqliteutil.ParseTimestamp(completedAt)
		if err != nil {
			return nil, err
		}
		rollups = append(rollups, roll)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return rollups, nil
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
