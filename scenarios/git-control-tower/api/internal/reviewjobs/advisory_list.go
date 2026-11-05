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
	if strings.TrimSpace(operation) == "" || len(states) == 0 || limit <= 0 {
		return nil, fmt.Errorf("operation, states, and a positive limit are required")
	}
	args := []any{operation}
	marks := make([]string, 0, len(states))
	for _, state := range states {
		marks = append(marks, "?")
		args = append(args, string(state))
	}
	args = append(args, limit)
	rows, err := s.db.QueryContext(ctx, `SELECT id,repository_id,execution_id,state,cancel_requested,error,result_json,updated_at FROM gct_review_jobs WHERE operation=? AND state IN (`+strings.Join(marks, ",")+`) ORDER BY updated_at DESC LIMIT ?`, args...)
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
		if row.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}
