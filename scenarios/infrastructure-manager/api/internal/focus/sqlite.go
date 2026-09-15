package focus

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type sqliteRepository struct{ db SQLExecutor }

func NewSQLiteRepository(db SQLExecutor) Repository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) SaveFindings(ctx context.Context, findings []RankedFinding) error {
	for _, finding := range findings {
		sensorRef := finding.SensorRef
		if sensorRef == "" {
			sensorRef = finding.CellRef
		}
		if finding.ID == "" || sensorRef == "" {
			return fmt.Errorf("focus finding requires id and sensor_ref")
		}
		_, err := r.db.ExecContext(ctx, `
INSERT OR REPLACE INTO focus_findings
  (finding_id, source, cell_ref, title, message, sensor_ref, expected_return, rank, rank_explanation, observed_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, finding.ID, finding.Source, finding.CellRef, finding.Title, finding.Message,
			sensorRef, finding.ExpectedReturn, finding.Rank, finding.RankExplanation, time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			return fmt.Errorf("save focus finding %q: %w", finding.ID, err)
		}
	}
	return nil
}

func (r *sqliteRepository) Efficacy(ctx context.Context, findingID string) ([]EfficacyRecord, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT finding_id, sensor_ref, expected_return, observed_return, verdict, observed_at
FROM focus_efficacy
WHERE finding_id = ?
ORDER BY observed_at DESC`, findingID)
	if err != nil {
		return nil, fmt.Errorf("list focus efficacy for %q: %w", findingID, err)
	}
	defer rows.Close()
	records := make([]EfficacyRecord, 0)
	for rows.Next() {
		var record EfficacyRecord
		var verdict, observedAt string
		if err := rows.Scan(&record.FindingID, &record.SensorRef, &record.ExpectedReturn, &record.ObservedReturn, &verdict, &observedAt); err != nil {
			return nil, fmt.Errorf("scan focus efficacy: %w", err)
		}
		record.Verdict = EfficacyVerdict(verdict)
		record.ObservedAt, err = time.Parse(time.RFC3339Nano, observedAt)
		if err != nil {
			return nil, fmt.Errorf("parse focus efficacy observed_at %q: %w", observedAt, err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate focus efficacy: %w", err)
	}
	return records, nil
}
