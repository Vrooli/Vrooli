package execution

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	pq "github.com/lib/pq"

	"test-genie/internal/orchestrator"
)

// SuiteExecutionRepository persists execution records.
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
	$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
)`

	var suiteRequestID interface{}
	if record.SuiteRequestID != nil {
		suiteRequestID = *record.SuiteRequestID
	}

	_, err = r.db.ExecContext(
		ctx,
		q,
		record.ID,
		suiteRequestID,
		record.ScenarioName,
		sql.NullString{String: record.PresetUsed, Valid: record.PresetUsed != ""},
		sql.NullString{String: record.RequestedPreset, Valid: record.RequestedPreset != ""},
		pq.Array(record.RequestedPhases),
		pq.Array(record.RequestedSkipPhases),
		pq.Array(record.PlannedPhases),
		record.FailFast,
		record.Success,
		payload,
		record.StartedAt,
		record.CompletedAt,
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
	var args []interface{}
	argPos := 1
	if scenario := strings.TrimSpace(scenario); scenario != "" {
		baseQuery += fmt.Sprintf(" WHERE scenario_name = $%d", argPos)
		args = append(args, scenario)
		argPos++
	}
	baseQuery += fmt.Sprintf(" ORDER BY completed_at DESC LIMIT $%d", argPos)
	args = append(args, limit)
	argPos++
	baseQuery += fmt.Sprintf(" OFFSET $%d", argPos)
	args = append(args, offset)

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
WHERE id = $1
`
	row := r.db.QueryRowContext(ctx, q, id)
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

	const q = `
SELECT
	scenario_name,
	LOWER(TRIM(phase->>'name')) AS phase_name,
	LOWER(TRIM(COALESCE(phase->>'status', ''))) AS status,
	GREATEST(COALESCE((phase->>'durationSeconds')::int, 0), 0) AS duration_seconds,
	completed_at
FROM suite_executions
CROSS JOIN LATERAL jsonb_array_elements(phases) AS phase
WHERE LOWER(TRIM(phase->>'name')) = ANY($1)
  AND completed_at >= $2
ORDER BY completed_at DESC
LIMIT $3
`

	rows, err := r.db.QueryContext(ctx, q, pq.Array(normalized), since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var samples []PhaseDurationSample
	for rows.Next() {
		var sample PhaseDurationSample
		if err := rows.Scan(
			&sample.ScenarioName,
			&sample.PhaseName,
			&sample.Status,
			&sample.DurationSeconds,
			&sample.CompletedAt,
		); err != nil {
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
	var rawSuite sql.NullString
	var preset sql.NullString
	var requestedPreset sql.NullString
	var requestedPhases pq.StringArray
	var requestedSkipPhases pq.StringArray
	var plannedPhases pq.StringArray
	var phasesJSON []byte

	if err := scanner.Scan(
		&record.ID,
		&rawSuite,
		&record.ScenarioName,
		&preset,
		&requestedPreset,
		&requestedPhases,
		&requestedSkipPhases,
		&plannedPhases,
		&record.FailFast,
		&record.Success,
		&phasesJSON,
		&record.StartedAt,
		&record.CompletedAt,
	); err != nil {
		return record, err
	}

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
	if len(requestedPhases) > 0 {
		record.RequestedPhases = append([]string(nil), requestedPhases...)
	}
	if len(requestedSkipPhases) > 0 {
		record.RequestedSkipPhases = append([]string(nil), requestedSkipPhases...)
	}
	if len(plannedPhases) > 0 {
		record.PlannedPhases = append([]string(nil), plannedPhases...)
	}
	if len(phasesJSON) > 0 {
		if err := json.Unmarshal(phasesJSON, &record.Phases); err != nil {
			return record, err
		}
	}
	return record, nil
}
