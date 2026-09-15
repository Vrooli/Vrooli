package reconcile

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"time"

	"compute-manager/internal/provider"
)

// DailyCostRunner compares one closed UTC day of locally-recorded lifecycle
// usage with a provider statement. It is deliberately report-only: a billing
// mismatch must never rewrite usage or settle money automatically.
type DailyCostRunner struct {
	Source provider.BillingStatementSource
	DB     interface {
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	}
	Provider  string
	Threshold int64
	Now       func() time.Time
	Report    func([]CostObservation)
}

func (r DailyCostRunner) Run(ctx context.Context, day time.Time) error {
	if r.Source == nil {
		return fmt.Errorf("daily cost statement source is unavailable")
	}
	if r.DB == nil {
		return fmt.Errorf("daily cost database is unavailable")
	}
	from := day.UTC().Truncate(24 * time.Hour)
	to := from.Add(24 * time.Hour)
	statements, err := r.Source.BillingStatements(ctx, from, to)
	if err != nil {
		return err
	}
	metered, err := r.loadMinutes(ctx, from, to)
	if err != nil {
		return err
	}
	observations := CompareCost(metered, statements, r.Threshold)
	if r.Report != nil {
		r.Report(observations)
	}
	return nil
}

func (r DailyCostRunner) loadMinutes(ctx context.Context, from, to time.Time) (map[string]int64, error) {
	rows, err := r.DB.QueryContext(ctx, `SELECT provider_instance_id,created_at,destroyed_at FROM instances WHERE provider=? AND created_at < ? AND (destroyed_at='' OR destroyed_at>?)`, r.Provider, to.Format(time.RFC3339Nano), from.Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	minutes := make(map[string]int64)
	for rows.Next() {
		var id, created, destroyed string
		if err := rows.Scan(&id, &created, &destroyed); err != nil {
			return nil, err
		}
		start, err := time.Parse(time.RFC3339Nano, created)
		if err != nil {
			return nil, fmt.Errorf("parse instance creation time: %w", err)
		}
		end := to
		if destroyed != "" {
			end, err = time.Parse(time.RFC3339Nano, destroyed)
			if err != nil {
				return nil, fmt.Errorf("parse instance destruction time: %w", err)
			}
		}
		if start.Before(from) {
			start = from
		}
		if end.After(to) {
			end = to
		}
		if end.After(start) && id != "" {
			minutes[id] += int64(math.Ceil(end.Sub(start).Minutes()))
		}
	}
	return minutes, rows.Err()
}
