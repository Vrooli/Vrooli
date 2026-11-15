package skillreads

import (
	"context"
	"database/sql"
	"encoding/json"
	"time"
)

type Database interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type SQLRepository struct{ db Database }

func NewSQLRepository(db Database) *SQLRepository { return &SQLRepository{db: db} }

// Append inserts and enforces retention and the exact record cap atomically.
// Indexed deletion replaces whole-file JSON decoding and snapshot rewriting.
func (r *SQLRepository) Append(ctx context.Context, read Read, now time.Time, maxEntries int, retention time.Duration) error {
	payload, err := json.Marshal(read)
	if err != nil {
		return err
	}
	var observedAt, observedNanos any
	if at, err := time.Parse(time.RFC3339, read.At); err == nil {
		observedAt, observedNanos = at.Unix(), at.Nanosecond()
	}
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, "INSERT INTO skill_read_telemetry(observed_at, observed_at_nanos, payload) VALUES (?, ?, ?)", observedAt, observedNanos, string(payload)); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, "DELETE FROM skill_read_telemetry WHERE (observed_at, observed_at_nanos) < (?, ?)", now.Add(-retention).Unix(), now.Add(-retention).Nanosecond()); err != nil {
		return err
	}
	if maxEntries > 0 {
		var count int
		if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM skill_read_telemetry").Scan(&count); err != nil {
			return err
		}
		if count > maxEntries {
			if _, err := tx.ExecContext(ctx, "DELETE FROM skill_read_telemetry WHERE sequence IN (SELECT sequence FROM skill_read_telemetry ORDER BY sequence LIMIT ?)", count-maxEntries); err != nil {
				return err
			}
		}
	}
	return tx.Commit()
}

func (r *SQLRepository) ReadSince(ctx context.Context, window time.Duration, now time.Time) ([]Read, error) {
	query := "SELECT payload FROM skill_read_telemetry"
	var args []any
	var result []Read
	if window > 0 {
		query += " WHERE observed_at IS NULL OR (observed_at, observed_at_nanos) >= (?, ?)"
		args = append(args, now.Add(-window).Unix(), now.Add(-window).Nanosecond())
		result = []Read{}
	}
	query += " ORDER BY sequence"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, err
		}
		var read Read
		if err := json.Unmarshal([]byte(payload), &read); err != nil {
			return nil, err
		}
		result = append(result, read)
	}
	return result, rows.Err()
}

// RecordReceipt upserts one (run, skill) receipt and prunes receipts whose
// last read is older than the retention window, in one transaction.
func (r *SQLRepository) RecordReceipt(ctx context.Context, runID, skillID string, now time.Time, retention time.Duration) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	at := now.UnixNano()
	if _, err := tx.ExecContext(ctx, `INSERT INTO skill_run_receipts(run_id, skill_id, first_read_at, last_read_at, read_count)
VALUES (?, ?, ?, ?, 1)
ON CONFLICT(run_id, skill_id) DO UPDATE SET last_read_at = excluded.last_read_at, read_count = skill_run_receipts.read_count + 1`, runID, skillID, at, at); err != nil {
		return err
	}
	if retention > 0 {
		if _, err := tx.ExecContext(ctx, "DELETE FROM skill_run_receipts WHERE last_read_at < ?", now.Add(-retention).UnixNano()); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *SQLRepository) Receipt(ctx context.Context, runID, skillID string) (Receipt, bool, error) {
	rows, err := r.db.QueryContext(ctx, "SELECT first_read_at, last_read_at, read_count FROM skill_run_receipts WHERE run_id = ? AND skill_id = ?", runID, skillID)
	if err != nil {
		return Receipt{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Receipt{}, false, rows.Err()
	}
	var first, last int64
	var count int
	if err := rows.Scan(&first, &last, &count); err != nil {
		return Receipt{}, false, err
	}
	return Receipt{RunID: runID, SkillID: skillID, FirstReadAt: time.Unix(0, first).UTC(), LastReadAt: time.Unix(0, last).UTC(), ReadCount: count}, true, rows.Err()
}
