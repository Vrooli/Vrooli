package capture

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

const (
	StatePending   = "pending"
	StateDelivered = "delivered"
	StateFailed    = "failed"
	leaseDuration  = 2 * time.Minute
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Attempt struct {
	AttemptID        string
	TaskID           string
	Payload          []byte
	PayloadSHA256    string
	DeliveryIdentity string
	State            string
	Attempts         int
	NextAttemptAt    time.Time
	LeaseUntil       time.Time
	LastError        string
	CreatedAt        time.Time
	DeliveredAt      time.Time
}

type Counts struct {
	Pending, Delivered, Failed int
	OldestPending              time.Time
}

// Drain performs one bounded delivery pass. The repository remains the
// durable owner; the callback may safely lose its acknowledgement because a
// later pass replays the same delivery identity.
func (r *Repository) Drain(ctx context.Context, limit int, deliver func(context.Context, Attempt) error) (int, error) {
	if deliver == nil {
		return 0, fmt.Errorf("delivery callback is required")
	}
	claimed, err := r.Claim(ctx, limit)
	if err != nil {
		return 0, err
	}
	delivered := 0
	for _, attempt := range claimed {
		if err := deliver(ctx, attempt); err != nil {
			retryAt := r.now().UTC().Add(time.Duration(attempt.Attempts) * time.Minute)
			if markErr := r.MarkFailed(ctx, attempt.AttemptID, err.Error(), retryAt); markErr != nil {
				return delivered, markErr
			}
			continue
		}
		if err := r.MarkDelivered(ctx, attempt.AttemptID); err != nil {
			return delivered, err
		}
		delivered++
	}
	return delivered, nil
}

type Repository struct {
	db  SQLExecutor
	now func() time.Time
}

func NewRepository(db SQLExecutor, now func() time.Time) *Repository {
	if now == nil {
		now = time.Now
	}
	return &Repository{db: db, now: now}
}

func PayloadHash(payload []byte) string {
	sum := sha256.Sum256(payload)
	return hex.EncodeToString(sum[:])
}

func (r *Repository) Enqueue(ctx context.Context, attemptID, taskID, deliveryIdentity string, payload []byte) (Attempt, bool, error) {
	if attemptID == "" || deliveryIdentity == "" || len(payload) == 0 {
		return Attempt{}, false, fmt.Errorf("attempt identity and payload are required")
	}
	hash := PayloadHash(payload)
	now := r.now().UTC()
	res, err := r.db.ExecContext(ctx, `INSERT INTO research_attempt_outbox (attempt_id, task_id, payload, payload_sha256, delivery_identity, state, next_attempt_at, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?) ON CONFLICT(attempt_id) DO NOTHING`, attemptID, taskID, payload, hash, deliveryIdentity, StatePending, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano))
	if err != nil {
		return Attempt{}, false, fmt.Errorf("enqueue attempt: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		a, err := r.Get(ctx, attemptID)
		if err != nil {
			return Attempt{}, false, err
		}
		if a.PayloadSHA256 != hash || a.DeliveryIdentity != deliveryIdentity {
			return Attempt{}, false, fmt.Errorf("attempt identity already has a different payload")
		}
		return a, true, nil
	}
	a, err := r.Get(ctx, attemptID)
	return a, false, err
}

func (r *Repository) Get(ctx context.Context, id string) (Attempt, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT attempt_id, task_id, payload, payload_sha256, delivery_identity, state, attempts, next_attempt_at, lease_until, last_error, created_at, delivered_at FROM research_attempt_outbox WHERE attempt_id = ?`, id)
	if err != nil {
		return Attempt{}, err
	}
	defer rows.Close()
	if !rows.Next() {
		return Attempt{}, sql.ErrNoRows
	}
	return scan(rows)
}

func (r *Repository) Claim(ctx context.Context, limit int) ([]Attempt, error) {
	if limit <= 0 || limit > 100 {
		return nil, fmt.Errorf("claim limit must be between 1 and 100")
	}
	now := r.now().UTC()
	rows, err := r.db.QueryContext(ctx, `SELECT attempt_id, task_id, payload, payload_sha256, delivery_identity, state, attempts, next_attempt_at, lease_until, last_error, created_at, delivered_at FROM research_attempt_outbox WHERE state IN (?, ?) AND next_attempt_at <= ? AND (lease_until = '' OR lease_until <= ?) ORDER BY created_at, attempt_id LIMIT ?`, StatePending, StateFailed, now.Format(time.RFC3339Nano), now.Format(time.RFC3339Nano), limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Attempt
	for rows.Next() {
		a, err := scan(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	claimed := make([]Attempt, 0, len(out))
	for i := range out {
		lease := now.Add(leaseDuration)
		result, err := r.db.ExecContext(ctx, `UPDATE research_attempt_outbox SET lease_until = ?, attempts = attempts + 1 WHERE attempt_id = ? AND (lease_until = '' OR lease_until <= ?)`, lease.Format(time.RFC3339Nano), out[i].AttemptID, now.Format(time.RFC3339Nano))
		if err != nil {
			return nil, err
		}
		if affected, _ := result.RowsAffected(); affected != 1 {
			continue
		}
		out[i].LeaseUntil, out[i].Attempts = lease, out[i].Attempts+1
		claimed = append(claimed, out[i])
	}
	return claimed, nil
}

func (r *Repository) MarkDelivered(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE research_attempt_outbox SET state = ?, lease_until = '', delivered_at = ?, last_error = '' WHERE attempt_id = ?`, StateDelivered, r.now().UTC().Format(time.RFC3339Nano), id)
	return err
}

func (r *Repository) MarkFailed(ctx context.Context, id, reason string, retryAt time.Time) error {
	_, err := r.db.ExecContext(ctx, `UPDATE research_attempt_outbox SET state = ?, lease_until = '', last_error = ?, next_attempt_at = ? WHERE attempt_id = ?`, StateFailed, reason, retryAt.UTC().Format(time.RFC3339Nano), id)
	return err
}

func (r *Repository) Counts(ctx context.Context) (Counts, error) {
	return r.counts(ctx, "", "")
}

// CountsInWindow returns the admitted-attempt population created in the
// half-open UTC interval [from, to). A window with no rows remains distinct
// from a reachable zero-valued rate at the measure layer.
func (r *Repository) CountsInWindow(ctx context.Context, from, to time.Time) (Counts, error) {
	if !from.Before(to) {
		return Counts{}, fmt.Errorf("capture window must be ordered")
	}
	return r.counts(ctx, from.UTC().Format(time.RFC3339Nano), to.UTC().Format(time.RFC3339Nano))
}

func (r *Repository) counts(ctx context.Context, from, to string) (Counts, error) {
	var c Counts
	query := `SELECT state, COUNT(*), MIN(CASE WHEN state IN (?, ?) THEN created_at END) FROM research_attempt_outbox`
	args := []any{StatePending, StateFailed}
	if from != "" || to != "" {
		query += ` WHERE created_at >= ? AND created_at < ?`
		args = append(args, from, to)
	}
	query += ` GROUP BY state`
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return c, err
	}
	defer rows.Close()
	for rows.Next() {
		var state string
		var count int
		var oldest sql.NullString
		if err := rows.Scan(&state, &count, &oldest); err != nil {
			return c, err
		}
		switch state {
		case StatePending:
			c.Pending = count
		case StateDelivered:
			c.Delivered = count
		case StateFailed:
			c.Failed = count
		}
		if oldest.Valid && oldest.String != "" {
			candidate, _ := time.Parse(time.RFC3339Nano, oldest.String)
			if c.OldestPending.IsZero() || candidate.Before(c.OldestPending) {
				c.OldestPending = candidate
			}
		}
	}
	return c, rows.Err()
}

func scan(rows interface{ Scan(...any) error }) (Attempt, error) {
	var a Attempt
	var next, lease, created, delivered string
	if err := rows.Scan(&a.AttemptID, &a.TaskID, &a.Payload, &a.PayloadSHA256, &a.DeliveryIdentity, &a.State, &a.Attempts, &next, &lease, &a.LastError, &created, &delivered); err != nil {
		return a, err
	}
	var err error
	a.NextAttemptAt, err = time.Parse(time.RFC3339Nano, next)
	if err != nil {
		return a, err
	}
	if lease != "" {
		a.LeaseUntil, err = time.Parse(time.RFC3339Nano, lease)
		if err != nil {
			return a, err
		}
	}
	a.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return a, err
	}
	if delivered != "" {
		a.DeliveredAt, err = time.Parse(time.RFC3339Nano, delivered)
	}
	return a, err
}
