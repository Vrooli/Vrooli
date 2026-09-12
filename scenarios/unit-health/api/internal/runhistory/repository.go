package runhistory

import (
	"context"
	"database/sql"
	"encoding/json"
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
		`INSERT INTO unit_runs (run_id, scenario, started_at, status, maturity_rung) VALUES (?, ?, ?, ?, ?)`,
		rec.RunID, rec.Scenario, started, rec.Status, rec.MaturityRung); err != nil {
		return fmt.Errorf("runhistory: insert run: %w", err)
	}
	for _, c := range rec.Commands {
		result, err := tx.ExecContext(ctx,
			`INSERT INTO unit_run_commands (run_id, scenario, started_at, workspace, command, duration_ms, status, failure_class) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			rec.RunID, rec.Scenario, started, c.WorkspaceID, c.Command, c.DurationMS, c.Status, c.FailureClass)
		if err != nil {
			return fmt.Errorf("runhistory: insert command: %w", err)
		}
		if c.Identity != nil {
			id, err := result.LastInsertId()
			if err != nil {
				return fmt.Errorf("runhistory: command identity id: %w", err)
			}
			raw, err := json.Marshal(c.Identity)
			if err != nil {
				return fmt.Errorf("runhistory: encode identity: %w", err)
			}
			if _, err := tx.ExecContext(ctx, `INSERT INTO unit_run_command_identity (command_id, identity_json) VALUES (?, ?)`, id, string(raw)); err != nil {
				return fmt.Errorf("runhistory: insert identity: %w", err)
			}
		}
	}
	for _, cv := range rec.Coverage {
		if _, err := tx.ExecContext(ctx,
			`INSERT INTO unit_run_coverage (run_id, scenario, workspace, file, percent) VALUES (?, ?, ?, ?, ?)`,
			rec.RunID, rec.Scenario, cv.WorkspaceID, cv.File, cv.Percent); err != nil {
			return fmt.Errorf("runhistory: insert coverage: %w", err)
		}
	}
	if len(rec.NativeTests) > 0 {
		raw, err := json.Marshal(rec.NativeTests)
		if err != nil {
			return fmt.Errorf("runhistory: encode native tests: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO unit_run_native_tests (run_id, scenario, observations_json) VALUES (?, ?, ?)`, rec.RunID, rec.Scenario, string(raw)); err != nil {
			return fmt.Errorf("runhistory: insert native tests: %w", err)
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
	if _, err := tx.ExecContext(ctx, `DELETE FROM unit_run_command_identity WHERE command_id IN
		(SELECT id FROM unit_run_commands WHERE scenario = ? AND run_id NOT IN (`+keepSub+`))`, scenario, scenario, keep); err != nil {
		return fmt.Errorf("runhistory: prune identities: %w", err)
	}
	for _, table := range []string{"unit_run_commands", "unit_run_coverage", "unit_run_native_tests", "unit_runs"} {
		stmt := fmt.Sprintf(`DELETE FROM %s WHERE scenario = ? AND run_id NOT IN (%s)`, table, keepSub)
		if _, err := tx.ExecContext(ctx, stmt, scenario, scenario, keep); err != nil {
			return fmt.Errorf("runhistory: prune %s: %w", table, err)
		}
	}
	return nil
}

// NativeTestHistory reads retained native metadata independently of command
// diagnostics. Missing legacy rows mean no native observations were recorded.
func (r *Repository) NativeTestHistory(ctx context.Context, scenario string, runLimit int) ([]NativeTestSample, error) {
	if r == nil || r.db == nil {
		return nil, nil
	}
	if runLimit <= 0 {
		runLimit = DefaultRetention
	}
	rows, err := r.db.QueryContext(ctx, `SELECT n.observations_json FROM unit_run_native_tests n
		JOIN unit_runs r ON r.run_id = n.run_id WHERE n.scenario = ? ORDER BY r.started_at DESC LIMIT ?`, scenario, runLimit)
	if err != nil {
		return nil, fmt.Errorf("runhistory: query native tests: %w", err)
	}
	defer rows.Close()
	var out []NativeTestSample
	for rows.Next() {
		var raw string
		if err := rows.Scan(&raw); err != nil {
			return nil, err
		}
		var samples []NativeTestSample
		if err := json.Unmarshal([]byte(raw), &samples); err != nil {
			return nil, fmt.Errorf("runhistory: malformed native observations: %w", err)
		}
		out = append(out, samples...)
	}
	return out, rows.Err()
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
SELECT c.run_id, c.started_at, c.workspace, c.command, c.duration_ms, c.status, c.failure_class, i.identity_json
FROM unit_run_commands c
LEFT JOIN unit_run_command_identity i ON i.command_id = c.id
WHERE c.scenario = ?
  AND c.run_id IN (SELECT run_id FROM unit_runs WHERE scenario = ? ORDER BY started_at DESC LIMIT ?)
ORDER BY c.started_at DESC, c.id DESC`
	rows, err := r.db.QueryContext(ctx, q, scenario, scenario, runLimit)
	if err != nil {
		return nil, fmt.Errorf("runhistory: query history: %w", err)
	}
	defer rows.Close()

	var out []CommandSample
	for rows.Next() {
		var (
			s        CommandSample
			started  int64
			identity sql.NullString
		)
		if err := rows.Scan(&s.RunID, &started, &s.WorkspaceID, &s.Command, &s.DurationMS, &s.Status, &s.FailureClass, &identity); err != nil {
			return nil, fmt.Errorf("runhistory: scan history: %w", err)
		}
		if identity.Valid {
			var decoded ComparisonIdentity
			// Malformed optional provenance must not erase the historical outcome
			// or prevent reading the rest of the run history.
			if json.Unmarshal([]byte(identity.String), &decoded) == nil {
				s.Identity = &decoded
			}
		}
		s.StartedAt = time.Unix(started, 0).UTC()
		out = append(out, s)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("runhistory: iterate history: %w", err)
	}
	return out, nil
}
