package suite

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// SuiteExecutionRecord captures a persisted execution outcome.
type SuiteExecutionRecord struct {
	ID             uuid.UUID
	SuiteRequestID *uuid.UUID
	ScenarioName   string
	PresetUsed     string
	Success        bool
	Phases         []PhaseExecutionResult
	StartedAt      time.Time
	CompletedAt    time.Time
}

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
	success,
	phases,
	started_at,
	completed_at
) VALUES (
	$1, $2, $3, $4, $5, $6, $7, $8
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
		record.Success,
		payload,
		record.StartedAt,
		record.CompletedAt,
	)
	return err
}

func (r *SuiteExecutionRepository) ListRecent(ctx context.Context, scenario string, limit int, offset int) ([]SuiteExecutionRecord, error) {
	if limit <= 0 || limit > MaxExecutionHistory {
		limit = MaxExecutionHistory
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

func scanSuiteExecutionRecord(scanner rowScanner) (SuiteExecutionRecord, error) {
	var record SuiteExecutionRecord
	var rawSuite sql.NullString
	var preset sql.NullString
	var phasesJSON []byte

	if err := scanner.Scan(
		&record.ID,
		&rawSuite,
		&record.ScenarioName,
		&preset,
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
	if len(phasesJSON) > 0 {
		if err := json.Unmarshal(phasesJSON, &record.Phases); err != nil {
			return record, err
		}
	}
	return record, nil
}
