package condition

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

func NewSQLiteRepository(db SQLExecutor) ReadingRepository { return &sqliteRepository{db: db} }

func (r *sqliteRepository) Save(ctx context.Context, readings []Observation) error {
	for _, reading := range readings {
		if reading.ID == "" || reading.CellRef == "" || reading.ObservedAt.IsZero() {
			return fmt.Errorf("condition reading requires id, cell_ref and observed_at")
		}
		_, err := r.db.ExecContext(ctx, `
INSERT OR REPLACE INTO condition_readings
  (reading_id, cell_ref, value, unit, source, observed_at, trust_verdict)
VALUES (?, ?, ?, ?, ?, ?, ?)`, reading.ID, reading.CellRef, reading.Value, reading.Unit, reading.Source,
			reading.ObservedAt.UTC().Format(time.RFC3339Nano), string(reading.Trust))
		if err != nil {
			return fmt.Errorf("save condition reading %q: %w", reading.ID, err)
		}
	}
	return nil
}

func (r *sqliteRepository) History(ctx context.Context, cellRef string, limit int) ([]Observation, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := r.db.QueryContext(ctx, `
SELECT reading_id, cell_ref, value, unit, source, observed_at, trust_verdict
FROM condition_readings
WHERE cell_ref = ?
ORDER BY observed_at DESC
LIMIT ?`, cellRef, limit)
	if err != nil {
		return nil, fmt.Errorf("list condition history for %q: %w", cellRef, err)
	}
	defer rows.Close()
	readings := make([]Observation, 0)
	for rows.Next() {
		var reading Observation
		var observedAt, trust string
		if err := rows.Scan(&reading.ID, &reading.CellRef, &reading.Value, &reading.Unit, &reading.Source, &observedAt, &trust); err != nil {
			return nil, fmt.Errorf("scan condition history: %w", err)
		}
		reading.ObservedAt, err = time.Parse(time.RFC3339Nano, observedAt)
		if err != nil {
			return nil, fmt.Errorf("parse condition observed_at %q: %w", observedAt, err)
		}
		reading.Trust = NormalizeTrust(trust)
		reading.Band.NeedsBaseline = true
		readings = append(readings, reading)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate condition history: %w", err)
	}
	return readings, nil
}
