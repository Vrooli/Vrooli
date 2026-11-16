package reviewjobs

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// AdvisorySummary is one advisory lifecycle row without its immutable input,
// which can be megabytes. Background owners scan these, then load one Job.
type AdvisorySummary struct {
	ID              string
	RepositoryID    string
	ExecutionID     string
	State           State
	CancelRequested bool
	Error           string
	ResultJSON      string
	UpdatedAt       time.Time
}

// ListAdvisory returns at most limit rows of one operation in the given
// states, most recently updated first.
func (s *Store) ListAdvisory(ctx context.Context, operation string, states []State, limit int) ([]AdvisorySummary, error) {
	return s.listAdvisory(ctx, operation, states, limit, "", "", "updated_at DESC")
}

// ListRecoverableAdvisory returns a stable ID-ordered page of active rows a
// background owner may advance. Handle-less rows are included: a crash can
// happen after durable admission or claim but before the owner call. Advancing
// those rows is safe because dispatch uses the saved input and durable owner
// idempotency key. Filtering cancelled rows before LIMIT prevents them from
// starving recoverable work; afterID rotates through rows that remain active.
func (s *Store) ListRecoverableAdvisory(ctx context.Context, operation string, states []State, limit int, afterID string) ([]AdvisorySummary, error) {
	return s.listAdvisory(ctx, operation, states, limit, " AND ((execution_id <> '' AND cancel_requested = 0) OR (execution_id = '' AND state IN ('queued','dispatching') AND cancel_requested = 0) OR state='cancelling')", afterID, "id ASC")
}

func (s *Store) listAdvisory(ctx context.Context, operation string, states []State, limit int, filter, afterID, order string) ([]AdvisorySummary, error) {
	if strings.TrimSpace(operation) == "" || len(states) == 0 || limit <= 0 {
		return nil, fmt.Errorf("operation, states, and a positive limit are required")
	}
	args := []any{operation}
	marks := make([]string, 0, len(states))
	for _, state := range states {
		marks = append(marks, "?")
		args = append(args, string(state))
	}
	if afterID != "" {
		filter += " AND id > ?"
		args = append(args, afterID)
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `SELECT id,repository_id,execution_id,state,cancel_requested,error,result_json,updated_at FROM gct_review_jobs WHERE operation=? AND state IN (`+strings.Join(marks, ",")+")"+filter+" ORDER BY "+order+" LIMIT ?", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AdvisorySummary
	for rows.Next() {
		var row AdvisorySummary
		var updated string
		if err := rows.Scan(&row.ID, &row.RepositoryID, &row.ExecutionID, &row.State, &row.CancelRequested, &row.Error, &row.ResultJSON, &updated); err != nil {
			return nil, err
		}
		if !row.State.Valid() {
			return nil, fmt.Errorf("decode advisory job %q: unknown state %q", row.ID, row.State)
		}
		if row.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
