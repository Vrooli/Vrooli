package playbooksclaims

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"test-genie/internal/dbexec"
)

// SqliteRepository is the SQLite-backed Repository implementation.
type SqliteRepository struct {
	db dbexec.Executor
}

var _ Repository = (*SqliteRepository)(nil)

// NewSqliteRepository constructs a repository against the given handle.
func NewSqliteRepository(db dbexec.Executor) *SqliteRepository {
	return &SqliteRepository{db: db}
}

// TryAcquire inserts a new claim or steals an expired one. If a live
// claim already exists for the scenario, returns *ErrBusy with the holder.
//
// The acquire is atomic against concurrent callers: we use INSERT ... ON
// CONFLICT DO UPDATE WHERE expires_at < ?, then verify ownership in the
// same transaction.
func (r *SqliteRepository) TryAcquire(ctx context.Context, in AcquireInput, now time.Time, ttl time.Duration) (Claim, error) {
	if in.ScenarioName == "" {
		return Claim{}, errors.New("playbooksclaims: scenario_name required")
	}
	if in.RunID == "" {
		return Claim{}, errors.New("playbooksclaims: run_id required")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Claim{}, fmt.Errorf("playbooksclaims: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	expires := now.Add(ttl)
	nowU := now.Unix()
	expU := expires.Unix()

	// INSERT or steal-if-expired in one statement.
	_, err = tx.ExecContext(ctx, `
INSERT INTO playbooks_claims (scenario_name, run_id, mode, started_by, acquired_at, heartbeat_at, expires_at)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(scenario_name) DO UPDATE SET
    run_id = excluded.run_id,
    mode = excluded.mode,
    started_by = excluded.started_by,
    acquired_at = excluded.acquired_at,
    heartbeat_at = excluded.heartbeat_at,
    expires_at = excluded.expires_at
WHERE playbooks_claims.expires_at <= excluded.heartbeat_at
`,
		in.ScenarioName, in.RunID, string(in.Mode), in.StartedBy, nowU, nowU, expU,
	)
	if err != nil {
		return Claim{}, fmt.Errorf("playbooksclaims: insert/steal: %w", err)
	}

	// Read back current row to determine outcome.
	current, err := getInTx(ctx, tx, in.ScenarioName)
	if err != nil {
		return Claim{}, err
	}
	if current.RunID != in.RunID {
		_ = tx.Rollback()
		return Claim{}, &ErrBusy{Holder: current}
	}
	if err := tx.Commit(); err != nil {
		return Claim{}, fmt.Errorf("playbooksclaims: commit: %w", err)
	}
	return current, nil
}

// Heartbeat extends expires_at if the caller still owns the row.
func (r *SqliteRepository) Heartbeat(ctx context.Context, scenarioName, runID string, now time.Time, ttl time.Duration) (Claim, error) {
	expires := now.Add(ttl)
	res, err := r.db.ExecContext(ctx, `
UPDATE playbooks_claims
SET heartbeat_at = ?, expires_at = ?
WHERE scenario_name = ? AND run_id = ?
`, now.Unix(), expires.Unix(), scenarioName, runID)
	if err != nil {
		return Claim{}, fmt.Errorf("playbooksclaims: heartbeat: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		// Distinguish absent vs mismatched owner.
		current, getErr := r.Get(ctx, scenarioName)
		if errors.Is(getErr, ErrNotFound) {
			return Claim{}, ErrNotFound
		}
		if getErr != nil {
			return Claim{}, getErr
		}
		if current.RunID != runID {
			return Claim{}, ErrLeaseMismatch
		}
		return current, nil
	}
	return r.Get(ctx, scenarioName)
}

// Release deletes the claim iff the runID matches.
func (r *SqliteRepository) Release(ctx context.Context, scenarioName, runID string) error {
	res, err := r.db.ExecContext(ctx, `
DELETE FROM playbooks_claims
WHERE scenario_name = ? AND run_id = ?
`, scenarioName, runID)
	if err != nil {
		return fmt.Errorf("playbooksclaims: release: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		_, getErr := r.Get(ctx, scenarioName)
		if errors.Is(getErr, ErrNotFound) {
			return ErrNotFound
		}
		if getErr != nil {
			return getErr
		}
		return ErrLeaseMismatch
	}
	return nil
}

// Get returns the current claim for a scenario.
func (r *SqliteRepository) Get(ctx context.Context, scenarioName string) (Claim, error) {
	row := r.db.QueryRowContext(ctx, `
SELECT scenario_name, run_id, mode, started_by, acquired_at, heartbeat_at, expires_at
FROM playbooks_claims
WHERE scenario_name = ?
`, scenarioName)
	return scanClaim(row)
}

// List returns all current claims.
func (r *SqliteRepository) List(ctx context.Context) ([]Claim, error) {
	rows, err := r.db.QueryContext(ctx, `
SELECT scenario_name, run_id, mode, started_by, acquired_at, heartbeat_at, expires_at
FROM playbooks_claims
ORDER BY acquired_at ASC
`)
	if err != nil {
		return nil, fmt.Errorf("playbooksclaims: list: %w", err)
	}
	defer rows.Close()

	var out []Claim
	for rows.Next() {
		c, err := scanClaim(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("playbooksclaims: list scan: %w", err)
	}
	return out, nil
}

// ForceBreak deletes the claim regardless of owner. Returns the broken claim.
func (r *SqliteRepository) ForceBreak(ctx context.Context, scenarioName string) (Claim, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return Claim{}, fmt.Errorf("playbooksclaims: force-break tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	current, err := getInTx(ctx, tx, scenarioName)
	if err != nil {
		return Claim{}, err
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM playbooks_claims WHERE scenario_name = ?`, scenarioName); err != nil {
		return Claim{}, fmt.Errorf("playbooksclaims: force-break delete: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return Claim{}, fmt.Errorf("playbooksclaims: force-break commit: %w", err)
	}
	return current, nil
}

// rowScanner abstracts *sql.Row and *sql.Rows for scanClaim.
type rowScanner interface {
	Scan(dest ...interface{}) error
}

func scanClaim(s rowScanner) (Claim, error) {
	var (
		c           Claim
		modeStr     string
		acquiredAt  int64
		heartbeatAt int64
		expiresAt   int64
	)
	err := s.Scan(&c.ScenarioName, &c.RunID, &modeStr, &c.StartedBy, &acquiredAt, &heartbeatAt, &expiresAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Claim{}, ErrNotFound
		}
		return Claim{}, fmt.Errorf("playbooksclaims: scan: %w", err)
	}
	c.Mode = Mode(modeStr)
	c.AcquiredAt = time.Unix(acquiredAt, 0).UTC()
	c.HeartbeatAt = time.Unix(heartbeatAt, 0).UTC()
	c.ExpiresAt = time.Unix(expiresAt, 0).UTC()
	return c, nil
}

func getInTx(ctx context.Context, tx *sql.Tx, scenarioName string) (Claim, error) {
	row := tx.QueryRowContext(ctx, `
SELECT scenario_name, run_id, mode, started_by, acquired_at, heartbeat_at, expires_at
FROM playbooks_claims
WHERE scenario_name = ?
`, scenarioName)
	return scanClaim(row)
}
