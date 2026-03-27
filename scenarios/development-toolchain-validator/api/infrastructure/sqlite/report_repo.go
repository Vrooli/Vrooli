package sqlite

import (
	"context"
	"database/sql"

	"development-toolchain-validator/domain/report"
)

// ReportRepository reads aggregated validation results from SQLite.
type ReportRepository struct {
	db *sql.DB
}

// NewReportRepository creates a new report repository.
func NewReportRepository(db *sql.DB) *ReportRepository {
	return &ReportRepository{db: db}
}

// CLIResultsByReference returns CLI assertion results for the latest
// validation run of the given reference, joined with cli_assertions
// to expose connection_id.
func (r *ReportRepository) CLIResultsByReference(ctx context.Context, referenceID string) ([]*report.CLIResultRow, error) {
	// Find the latest validation run for this reference.
	var runID string
	err := r.db.QueryRowContext(ctx,
		`SELECT id FROM validation_runs
		 WHERE reference_id = ?
		 ORDER BY started_at DESC
		 LIMIT 1`,
		referenceID,
	).Scan(&runID)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT cr.assertion_id, ca.connection_id, ca.command, ca.json_path,
		        cr.status, COALESCE(cr.actual_value, ''), COALESCE(cr.error_message, '')
		 FROM cli_results cr
		 JOIN cli_assertions ca ON ca.id = cr.assertion_id
		 WHERE cr.run_id = ?`,
		runID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []*report.CLIResultRow
	for rows.Next() {
		var row report.CLIResultRow
		if err := rows.Scan(
			&row.AssertionID,
			&row.ConnectionID,
			&row.Command,
			&row.JSONPath,
			&row.Status,
			&row.ActualValue,
			&row.ErrorMessage,
		); err != nil {
			return nil, err
		}
		results = append(results, &row)
	}
	return results, rows.Err()
}

// Compile-time check.
var _ report.ValidationResultReader = (*ReportRepository)(nil)
