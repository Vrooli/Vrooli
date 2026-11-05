package database

import (
	"context"
	"database/sql"
	"strconv"
	"strings"
	"time"

	"agent-manager/internal/pricing"
)

var _ pricing.QuotaObservationCompactor = (*pricingRepository)(nil)

const (
	quotaCompactionPage  = 5000
	quotaCompactionBatch = 500
	// quotaCompactionPause yields the write lock between delete batches.
	quotaCompactionPause = 5 * time.Millisecond
)

// quotaReadingRow is the part of a quota observation that identifies its
// reading: what QuotaReadingSpans groups by within a scope.
type quotaReadingRow struct {
	id          string
	observedAt  string
	resetAt     sql.NullString
	usedPercent sql.NullFloat64
	standing    string
}

// reading is the span grouping key: reset minute (or, without a reset time,
// the hour of observation), used percent and standing.
func (r quotaReadingRow) reading() string {
	bucket := "hour:" + prefix(r.observedAt, 13)
	if r.resetAt.Valid {
		bucket = "reset:" + prefix(r.resetAt.String, 16)
	}
	percent := "none"
	if r.usedPercent.Valid {
		percent = strconv.FormatFloat(r.usedPercent.Float64, 'g', -1, 64)
	}
	return strings.Join([]string{bucket, percent, r.standing}, "|")
}

func (r quotaReadingRow) at() time.Time {
	t, _ := time.Parse(time.RFC3339Nano, r.observedAt)
	return t
}

func prefix(value string, n int) string {
	if len(value) < n {
		return value
	}
	return value[:n]
}

// quotaRunPlanner decides, over one scope's rows in time order, which rows
// are redundant: in each run of rows with the same reading it keeps the
// first, the last and the first row at least pricing.QuotaHeartbeat after
// the previously kept one. Every span's first and last time survive, so
// QuotaReadingSpans and the forecast read the same history. Replaying the
// planner over its own output deletes nothing (idempotent).
type quotaRunPlanner struct {
	run       string
	keptAt    time.Time
	pending   *quotaReadingRow
	redundant []string
}

func (p *quotaRunPlanner) next(row quotaReadingRow) {
	reading, at := row.reading(), row.at()
	switch {
	case reading != p.run:
		// The pending row, if any, was the last of its run: keep it.
		p.run, p.keptAt, p.pending = reading, at, nil
	case at.Sub(p.keptAt) >= pricing.QuotaHeartbeat:
		if p.pending != nil {
			p.redundant = append(p.redundant, p.pending.id)
		}
		p.keptAt, p.pending = at, nil
	default:
		if p.pending != nil {
			p.redundant = append(p.redundant, p.pending.id)
		}
		p.pending = &row
	}
}

// CompactQuotaObservations removes the repeated readings stored before
// write-time deduplication. It pages through each scope by time and deletes
// in small batches, yielding between them, so it never holds a long write
// lock on a large database; rerunning it deletes nothing new.
func (r *pricingRepository) CompactQuotaObservations(ctx context.Context) (pricing.QuotaCompaction, error) {
	var result pricing.QuotaCompaction
	scopes, err := r.quotaScopes(ctx)
	if err != nil {
		return result, err
	}
	for _, scope := range scopes {
		planner := &quotaRunPlanner{}
		after := quotaReadingRow{}
		for {
			page, err := r.quotaScopePage(ctx, scope, after)
			if err != nil {
				return result, err
			}
			for _, row := range page {
				planner.next(row)
			}
			result.Scanned += int64(len(page))
			// The pending row may still be the last of its run; it is
			// decided on the next page.
			if err := r.deleteQuotaRows(ctx, planner.redundant, &result); err != nil {
				return result, err
			}
			planner.redundant = planner.redundant[:0]
			if len(page) < quotaCompactionPage {
				break
			}
			after = page[len(page)-1]
		}
	}
	return result, nil
}

func (r *pricingRepository) quotaScopes(ctx context.Context) ([][4]string, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT DISTINCT account_scope, provider, pool, window FROM quota_observations`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var scopes [][4]string
	for rows.Next() {
		var scope [4]string
		if err := rows.Scan(&scope[0], &scope[1], &scope[2], &scope[3]); err != nil {
			return nil, err
		}
		scopes = append(scopes, scope)
	}
	return scopes, rows.Err()
}

// quotaScopePage reads the next page of a scope after the given row, in the
// (observed_at, id) order the planner needs.
func (r *pricingRepository) quotaScopePage(ctx context.Context, scope [4]string, after quotaReadingRow) ([]quotaReadingRow, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, observed_at, reset_at, used_percent, standing FROM quota_observations
		WHERE account_scope = ? AND provider = ? AND pool = ? AND window = ? AND (observed_at > ? OR (observed_at = ? AND id > ?))
		ORDER BY observed_at, id LIMIT ?`, scope[0], scope[1], scope[2], scope[3], after.observedAt, after.observedAt, after.id, quotaCompactionPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	page := make([]quotaReadingRow, 0, quotaCompactionPage)
	for rows.Next() {
		var row quotaReadingRow
		if err := rows.Scan(&row.id, &row.observedAt, &row.resetAt, &row.usedPercent, &row.standing); err != nil {
			return nil, err
		}
		page = append(page, row)
	}
	return page, rows.Err()
}

func (r *pricingRepository) deleteQuotaRows(ctx context.Context, ids []string, result *pricing.QuotaCompaction) error {
	for start := 0; start < len(ids); start += quotaCompactionBatch {
		batch := ids[start:min(len(ids), start+quotaCompactionBatch)]
		args := make([]any, len(batch))
		for i, id := range batch {
			args[i] = id
		}
		deleted, err := r.db.ExecContext(ctx, `DELETE FROM quota_observations WHERE id IN (`+strings.TrimSuffix(strings.Repeat("?,", len(batch)), ",")+`)`, args...)
		if err != nil {
			return err
		}
		n, _ := deleted.RowsAffected()
		result.Deleted += n
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(quotaCompactionPause):
		}
	}
	return nil
}
