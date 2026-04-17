package execution

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"test-genie/internal/orchestrator"
	"test-genie/internal/storage/sqliteutil"
	"time"

	"github.com/google/uuid"
)

// SuiteExecutionRepository persists execution records in Test Genie's embedded
// SQLite database.
type SuiteExecutionRepository struct {
	db *sql.DB
}

func NewSuiteExecutionRepository(db *sql.DB) *SuiteExecutionRepository {
	return &SuiteExecutionRepository{db: db}
}

func (r *SuiteExecutionRepository) Create(ctx context.Context, record *SuiteExecutionRecord) error {
	payload, err := json.Marshal(record.Phases)
	if err != nil {
		return err
	}
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
	suite_request_id,
	scenario_name,
	preset_used,
	requested_preset,
	requested_phases,
	requested_skip_phases,
	planned_phases,
	fail_fast,
	success,
	phases,
	started_at,
	completed_at
) VALUES (
	?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?
)`

	var suiteRequestID any
	if record.SuiteRequestID != nil {
		suiteRequestID = record.SuiteRequestID.String()
	}

	_, err = r.db.ExecContext(
		ctx,
		q,
		record.ID.String(),
		suiteRequestID,
		record.ScenarioName,
		nullIfEmpty(record.PresetUsed),
		nullIfEmpty(record.RequestedPreset),
		requestedPhases,
		requestedSkipPhases,
		plannedPhases,
		boolToInt(record.FailFast),
		boolToInt(record.Success),
		string(payload),
		sqliteutil.FormatTimestamp(record.StartedAt),
		sqliteutil.FormatTimestamp(record.CompletedAt),
	)
	return err
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
	suite_request_id,
	scenario_name,
	preset_used,
	requested_preset,
	requested_phases,
	requested_skip_phases,
	planned_phases,
	fail_fast,
	success,
	phases,
	started_at,
	completed_at
FROM suite_executions`
	args := make([]any, 0, 3)
	if scenario = strings.TrimSpace(scenario); scenario != "" {
		baseQuery += " WHERE scenario_name = ?"
		args = append(args, scenario)
	}
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
	suite_request_id,
	scenario_name,
	preset_used,
	requested_preset,
	requested_phases,
	requested_skip_phases,
	planned_phases,
	fail_fast,
	success,
	phases,
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
SELECT
	scenario_name,
	LOWER(TRIM(json_extract(phase.value, '$.name'))) AS phase_name,
	LOWER(TRIM(COALESCE(json_extract(phase.value, '$.status'), ''))) AS status,
	MAX(CAST(COALESCE(json_extract(phase.value, '$.durationSeconds'), 0) AS INTEGER), 0) AS duration_seconds,
	completed_at
FROM suite_executions
JOIN json_each(suite_executions.phases) AS phase
WHERE LOWER(TRIM(json_extract(phase.value, '$.name'))) IN (%s)
  AND completed_at >= ?
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

type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanSuiteExecutionRecord(scanner rowScanner) (SuiteExecutionRecord, error) {
	var record SuiteExecutionRecord
	var rawID string
	var rawSuite sql.NullString
	var preset sql.NullString
	var requestedPreset sql.NullString
	var requestedPhases any
	var requestedSkipPhases any
	var plannedPhases any
	var failFast int
	var success int
	var phasesJSON any
	var startedAt any
	var completedAt any

	if err := scanner.Scan(
		&rawID,
		&rawSuite,
		&record.ScenarioName,
		&preset,
		&requestedPreset,
		&requestedPhases,
		&requestedSkipPhases,
		&plannedPhases,
		&failFast,
		&success,
		&phasesJSON,
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

	if rawSuite.Valid {
		if parsed, err := uuid.Parse(rawSuite.String); err == nil {
			record.SuiteRequestID = &parsed
		}
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
	record.FailFast = failFast == 1
	record.Success = success == 1

	if err := sqliteutil.UnmarshalJSON(phasesJSON, &record.Phases); err != nil {
		return record, err
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

func boolToInt(value bool) int {
	if value {
		return 1
	}
	return 0
}
