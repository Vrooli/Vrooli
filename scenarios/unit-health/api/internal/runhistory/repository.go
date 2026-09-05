package runhistory

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// DefaultRetention is the number of most-recent runs kept per scenario. Older
// runs (and their commands/coverage) are pruned on each Record so the table
// stays bounded. It is deterministic — exactly the newest N survive.
const DefaultRetention = 50

// Repository is the SQLite-backed Store. It is safe under MaxOpenConns:1: every
// read is a single SELECT (no nested query inside an open rows loop) and every
// write runs in one transaction.
type Repository struct {
	db        *sql.DB
	retention int
}

// NewRepository builds a Repository over the shared *sql.DB.
func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db, retention: DefaultRetention}
}

var _ Store = (*Repository)(nil)

// Record persists the run and prunes history beyond the retention window in a
// single transaction.
func (r *Repository) Record(ctx context.Context, rec RunRecord) error {
	if r == nil || r.db == nil {
		return nil
	}
	if rec.RunID == "" || rec.Scenario == "" {
		return fmt.Errorf("runhistory: run_id and scenario are required")
	}
	started := rec.StartedAt.UTC().Unix()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("runhistory: begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.ExecContext(ctx,
		`INSERT OR REPLACE INTO unit_runs (run_id, scenario, started_at, status, maturity_rung) VALUES (?, ?, ?, ?, ?)`,
		rec.RunID, rec.Scenario, started, rec.Status, rec.MaturityRung); err != nil {
		return fmt.Errorf("runhistory: insert run: %w", err)
	}
	for _, c := range rec.Commands {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO unit_run_commands (run_id, scenario, started_at, workspace, command, duration_ms, status, failure_class) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			rec.RunID, rec.Scenario, started, c.WorkspaceID, c.Command, c.DurationMS, c.Status, c.FailureClass); err != nil {
			return fmt.Errorf("runhistory: insert command: %w", err)
		}
	}
	for _, cv := range rec.Coverage {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO unit_run_coverage (run_id, scenario, workspace, file, percent) VALUES (?, ?, ?, ?, ?)`,
			rec.RunID, rec.Scenario, cv.WorkspaceID, cv.File, cv.Percent); err != nil {
			return fmt.Errorf("runhistory: insert coverage: %w", err)
		}
	}

	if err := pruneTx(ctx, tx, rec.Scenario, r.retention); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("runhistory: commit: %w", err)
	}
	return nil
}

// pruneTx deletes all but the newest `keep` runs (and their child rows) for a
// scenario, in one transaction.
func pruneTx(ctx context.Context, tx *sql.Tx, scenario string, keep int) error {
	if keep <= 0 {
		return nil
	}
	// Newest `keep` run_ids survive; everything older for this scenario is
	// deleted from all three tables via a NOT IN subquery (single statements).
	keepSub := `SELECT run_id FROM unit_runs WHERE scenario = ? ORDER BY started_at DESC LIMIT ?`
	for _, table := range []string{"unit_run_commands", "unit_run_coverage", "unit_runs"} {
		stmt := fmt.Sprintf(`DELETE FROM %s WHERE scenario = ? AND run_id NOT IN (%s)`, table, keepSub)
		if _, err := tx.ExecContext(ctx, stmt, scenario, scenario, keep); err != nil {
			return fmt.Errorf("runhistory: prune %s: %w", table, err)
		}
	}
	return nil
}

// CommandHistory returns command samples for the scenario across the most recent
// runLimit runs, newest first. It is a single SELECT bounded by a subquery on
// the run list, so it never opens a nested query inside the rows loop.
func (r *Repository) CommandHistory(ctx context.Context, scenario string, runLimit int) ([]CommandSample, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if runLimit <= 0 {
		runLimit = DefaultRetention
	}
	const q = `
SELECT run_id, started_at, workspace, command, duration_ms, status, failure_class
FROM unit_run_commands
WHERE scenario = ?
  AND run_id IN (SELECT run_id FROM unit_runs WHERE scenario = ? ORDER BY started_at DESC LIMIT ?)
ORDER BY started_at DESC, id DESC`
	rows, err := r.db.QueryContext(ctx, q, scenario, scenario, runLimit)
	if err != nil {
		return nil, fmt.Errorf("runhistory: query history: %w", err)
	}
	defer rows.Close()

	var out []CommandSample
	for rows.Next() {
		var (
			s       CommandSample
			started int64
		)
		if err := rows.Scan(&s.RunID, &started, &s.WorkspaceID, &s.Command, &s.DurationMS, &s.Status, &s.FailureClass); err != nil {
			return nil, fmt.Errorf("runhistory: scan history: %w", err)
		}
		s.StartedAt = time.Unix(started, 0).UTC()
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("runhistory: iterate history: %w", err)
	}
	return out, nil
}
