package execution

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"test-genie/internal/dbexec"
	"test-genie/internal/orchestrator"
	"test-genie/internal/storage/sqliteutil"

	"github.com/google/uuid"
)

// SuiteExecutionRepository persists execution records in Test Genie's embedded
// SQLite database.
type SuiteExecutionRepository struct {
	db dbexec.Executor
}

func NewSuiteExecutionRepository(db dbexec.Executor) *SuiteExecutionRepository {
	return &SuiteExecutionRepository{db: db}
}

func (r *SuiteExecutionRepository) Create(ctx context.Context, record *SuiteExecutionRecord) error {
	requestedPhases, err := sqliteutil.MarshalStringSlice(record.RequestedPhases)
	if err != nil {
		return err
	}
	requestedSkipPhases, err := sqliteutil.MarshalStringSlice(record.RequestedSkipPhases)
	if err != nil {
		return err
	}
	plannedPhases, err := sqliteutil.MarshalStringSlice(record.PlannedPhases)
	if err != nil {
		return err
	}

	const q = `
INSERT INTO suite_executions (
	id,
	run_id,
	scenario_name,
	target_kind,
	target_id,
	preset_used,
	requested_preset,
	requested_phases,
	requested_skip_phases,
	planned_phases,
	phase_set_digest,
	descriptor_snapshot_digest,
	configuration_fingerprint,
	fail_fast,
	success,
	terminal_outcome,
		started_at,
		completed_at
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)`

	// terminal_outcome is the run-level classification. A caller that did not
	// set it (the normal completed-run path) gets passed/failed derived from
	// the success flag; catastrophic callers set errored/aborted/timeout.
	outcome := record.TerminalOutcome
	if outcome == "" {
		outcome = outcomeForSuccess(record.Success)
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(
		ctx,
		q,
		record.ID.String(),
		nullIfEmpty(record.RunID),
		record.ScenarioName,
		orDefault(record.TargetKind, "scenario"),
		orDefault(record.TargetID, record.ScenarioName),
		nullIfEmpty(record.PresetUsed),
		nullIfEmpty(record.RequestedPreset),
		requestedPhases,
		requestedSkipPhases,
		plannedPhases,
		nullIfEmpty(record.PhaseSetDigest),
		nullIfEmpty(record.DescriptorSnapshotDigest),
		nullIfEmpty(record.ConfigurationFingerprint),
		boolToInt(record.FailFast),
		boolToInt(record.Success),
		outcome.String(),
		sqliteutil.FormatTimestamp(record.StartedAt),
		sqliteutil.FormatTimestamp(record.CompletedAt),
	)
	if err != nil {
		return err
	}
	if err := insertPhaseHistory(ctx, tx, record.ID, record.Phases); err != nil {
		return err
	}
	if err := insertStageHistory(ctx, tx, record.ID, record.PreparationStages); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SuiteExecutionRepository) ListRecent(ctx context.Context, scenario string, limit int, offset int) ([]SuiteExecutionRecord, error) {
	if limit <= 0 || limit > orchestrator.MaxExecutionHistory {
		limit = orchestrator.MaxExecutionHistory
	}
	if offset < 0 {
		offset = 0
	}

	baseQuery := `
SELECT
	id,
	run_id,
	scenario_name,
	target_kind,
	target_id,
	preset_used,
	requested_preset,
	requested_phases,
	requested_skip_phases,
	planned_phases,
	phase_set_digest,
	descriptor_snapshot_digest,
	configuration_fingerprint,
	fail_fast,
	success,
	terminal_outcome,
	started_at,
	completed_at
FROM suite_executions`
	args := make([]any, 0, 3)
	if scenario = strings.TrimSpace(scenario); scenario != "" {
		baseQuery += " WHERE scenario_name = ?"
		args = append(args, scenario)
	}
	// List surfaces are summary-only. Selecting NULL rather than the JSON blob is
	// load-bearing: historical rows can contain hundreds of megabytes of phase
	// detail, and list polling must never hydrate it into the API process.
	baseQuery += " ORDER BY completed_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := r.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var records []SuiteExecutionRecord
	for rows.Next() {
		record, err := scanSuiteExecutionRecord(rows)
		if err != nil {
			return nil, err
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return records, nil
}

func (r *SuiteExecutionRepository) GetByID(ctx context.Context, id uuid.UUID) (*SuiteExecutionRecord, error) {
	const q = `
SELECT
	id,
	run_id,
	scenario_name,
	target_kind,
	target_id,
	preset_used,
	requested_preset,
	requested_phases,
	requested_skip_phases,
	planned_phases,
	phase_set_digest,
	descriptor_snapshot_digest,
	configuration_fingerprint,
	fail_fast,
	success,
	terminal_outcome,
	started_at,
	completed_at
FROM suite_executions
WHERE id = ?
`
	row := r.db.QueryRowContext(ctx, q, id.String())
	record, err := scanSuiteExecutionRecord(row)
	if err != nil {
		return nil, err
	}
	record.Phases, err = r.loadPhaseHistory(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	record.PreparationStages, err = r.loadStageHistory(ctx, record.ID)
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func (r *SuiteExecutionRepository) Latest(ctx context.Context) (*SuiteExecutionRecord, error) {
	records, err := r.ListRecent(ctx, "", 1, 0)
	if err != nil {
		return nil, err
	}
	if len(records) == 0 {
		return nil, nil
	}
	return &records[0], nil
}

// DeleteByRunID removes compact execution-history rows as part of the one
// retention lifecycle. It deliberately has no public handler of its own:
// artifact, log, index, and SQLite deletion must converge through
// shared/runs.RetentionService and its tombstone recovery.
func (r *SuiteExecutionRepository) DeleteByRunID(ctx context.Context, runID string) error {
	if strings.TrimSpace(runID) == "" {
		return fmt.Errorf("run id is required")
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// Do not depend on connection-local foreign_keys pragmas for lifecycle
	// correctness: the compact child rows are removed explicitly with their
	// owning execution headers.
	if _, err := tx.ExecContext(ctx, `DELETE FROM suite_execution_phases WHERE execution_id IN (SELECT id FROM suite_executions WHERE run_id = ?)`, runID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM suite_execution_stages WHERE execution_id IN (SELECT id FROM suite_executions WHERE run_id = ?)`, runID); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM suite_executions WHERE run_id = ?`, runID); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *SuiteExecutionRepository) ListPhaseSamples(ctx context.Context, phaseNames []string, since time.Time, limit int) ([]PhaseDurationSample, error) {
	if len(phaseNames) == 0 {
		return nil, nil
	}
	if limit <= 0 {
		limit = 2000
	}

	normalized := make([]string, 0, len(phaseNames))
	seen := make(map[string]struct{}, len(phaseNames))
	for _, phase := range phaseNames {
		key := strings.ToLower(strings.TrimSpace(phase))
		if key == "" {
			continue
		}
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		normalized = append(normalized, key)
	}
	if len(normalized) == 0 {
		return nil, nil
	}

	placeholders := strings.TrimSuffix(strings.Repeat("?,", len(normalized)), ",")
	q := fmt.Sprintf(`
SELECT e.scenario_name, p.phase_name, p.status, p.duration_seconds, e.completed_at
FROM suite_execution_phases AS p
JOIN suite_executions AS e ON e.id = p.execution_id
WHERE p.phase_name IN (%s)
  AND e.completed_at >= ?
ORDER BY completed_at DESC
LIMIT ?
`, placeholders)

	args := make([]any, 0, len(normalized)+2)
	for _, phase := range normalized {
		args = append(args, phase)
	}
	args = append(args, sqliteutil.FormatTimestamp(since), limit)

	rows, err := r.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []PhaseDurationSample
	for rows.Next() {
		var sample PhaseDurationSample
		var completedAt any
		if err := rows.Scan(
			&sample.ScenarioName,
			&sample.PhaseName,
			&sample.Status,
			&sample.DurationSeconds,
			&completedAt,
		); err != nil {
			return nil, err
		}
		sample.CompletedAt, err = sqliteutil.ParseTimestamp(completedAt)
		if err != nil {
			return nil, err
		}
		samples = append(samples, sample)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return samples, nil
}

// ListPlanSamples returns same-scenario terminal runs with their immutable
// comparability key. The plan preview uses these rows before flattened phase
// samples: exact full-run evidence captures orchestration/startup cost that a
// sum of phase medians cannot see.
func (r *SuiteExecutionRepository) ListPlanSamples(ctx context.Context, scenario string, since time.Time, limit int) ([]PlanDurationSample, error) {
	if limit <= 0 {
		limit = 2000
	}
	const q = `
SELECT scenario_name, phase_set_digest, descriptor_snapshot_digest,
       configuration_fingerprint, COALESCE(terminal_outcome, ''),
       started_at, completed_at
FROM suite_executions
WHERE scenario_name = ? AND completed_at >= ?
ORDER BY completed_at DESC
LIMIT ?`
	rows, err := r.db.QueryContext(ctx, q, strings.TrimSpace(scenario), sqliteutil.FormatTimestamp(since), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var samples []PlanDurationSample
	for rows.Next() {
		var sample PlanDurationSample
		var startedAt, completedAt any
		var phaseSetDigest, descriptorSnapshotDigest, configurationFingerprint sql.NullString
		if err := rows.Scan(&sample.ScenarioName, &phaseSetDigest, &descriptorSnapshotDigest,
			&configurationFingerprint, &sample.TerminalOutcome, &startedAt, &completedAt); err != nil {
			return nil, err
		}
		if phaseSetDigest.Valid {
			sample.PhaseSetDigest = phaseSetDigest.String
		}
		if descriptorSnapshotDigest.Valid {
			sample.DescriptorSnapshotDigest = descriptorSnapshotDigest.String
		}
		if configurationFingerprint.Valid {
			sample.ConfigurationFingerprint = configurationFingerprint.String
		}
		var err error
		sample.StartedAt, err = sqliteutil.ParseTimestamp(startedAt)
		if err != nil {
			return nil, err
		}
		sample.CompletedAt, err = sqliteutil.ParseTimestamp(completedAt)
		if err != nil {
			return nil, err
		}
		sample.DurationSeconds = maxInt(0, int(sample.CompletedAt.Sub(sample.StartedAt).Round(time.Second).Seconds()))
		samples = append(samples, sample)
	}
	return samples, rows.Err()
}

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanSuiteExecutionRecord(scanner rowScanner) (SuiteExecutionRecord, error) {
	var record SuiteExecutionRecord
	var rawID string
	var rawRun sql.NullString
	var targetKind sql.NullString
	var targetID sql.NullString
	var preset sql.NullString
	var requestedPreset sql.NullString
	var requestedPhases any
	var requestedSkipPhases any
	var plannedPhases any
	var phaseSetDigest sql.NullString
	var descriptorSnapshotDigest sql.NullString
	var configurationFingerprint sql.NullString
	var failFast int
	var success int
	var terminalOutcome sql.NullString
	var startedAt any
	var completedAt any

	if err := scanner.Scan(
		&rawID,
		&rawRun,
		&record.ScenarioName,
		&targetKind,
		&targetID,
		&preset,
		&requestedPreset,
		&requestedPhases,
		&requestedSkipPhases,
		&plannedPhases,
		&phaseSetDigest,
		&descriptorSnapshotDigest,
		&configurationFingerprint,
		&failFast,
		&success,
		&terminalOutcome,
		&startedAt,
		&completedAt,
	); err != nil {
		return record, err
	}

	parsedID, err := uuid.Parse(rawID)
	if err != nil {
		return record, err
	}
	record.ID = parsedID
	if rawRun.Valid {
		record.RunID = rawRun.String
	}
	if targetKind.Valid {
		record.TargetKind = targetKind.String
	}
	if targetID.Valid {
		record.TargetID = targetID.String
	}
	if record.TargetKind == "" {
		record.TargetKind = "scenario"
	}
	if record.TargetID == "" {
		record.TargetID = record.ScenarioName
	}

	if preset.Valid {
		record.PresetUsed = preset.String
	}
	if requestedPreset.Valid {
		record.RequestedPreset = requestedPreset.String
	}
	record.RequestedPhases, err = sqliteutil.UnmarshalStringSlice(requestedPhases)
	if err != nil {
		return record, err
	}
	record.RequestedSkipPhases, err = sqliteutil.UnmarshalStringSlice(requestedSkipPhases)
	if err != nil {
		return record, err
	}
	record.PlannedPhases, err = sqliteutil.UnmarshalStringSlice(plannedPhases)
	if err != nil {
		return record, err
	}
	if phaseSetDigest.Valid {
		record.PhaseSetDigest = phaseSetDigest.String
	}
	if descriptorSnapshotDigest.Valid {
		record.DescriptorSnapshotDigest = descriptorSnapshotDigest.String
	}
	if configurationFingerprint.Valid {
		record.ConfigurationFingerprint = configurationFingerprint.String
	}
	record.FailFast = failFast == 1
	record.Success = success == 1
	if terminalOutcome.Valid {
		record.TerminalOutcome = TerminalOutcome(terminalOutcome.String)
	}

	record.StartedAt, err = sqliteutil.ParseTimestamp(startedAt)
	if err != nil {
		return record, err
	}
	record.CompletedAt, err = sqliteutil.ParseTimestamp(completedAt)
	if err != nil {
		return record, err
	}
	return record, nil
}

func nullIfEmpty(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func orDefault(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}

func maxInt(left, right int) int {
	if left > right {
		return left
	}
	return right
}
