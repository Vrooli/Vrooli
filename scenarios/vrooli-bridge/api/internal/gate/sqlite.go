package gate

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteRepository depends on. Both
// *sql.DB (repository unit tests) and *database.RoutedDB (production) satisfy
// it, so production participates in per-request routing without the test
// fixture wrapping its handle.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteRepository constructs the production Repository.
func NewSQLiteRepository(db SQLExecutor, clk schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Repository = (*sqliteRepository)(nil)

// gateTimeFormat sorts lexicographically in time order for a fixed zone, so a
// string order comparison is a correct newest-first filter.
const gateTimeFormat = time.RFC3339Nano

const (
	insertGateSQL = `
INSERT INTO gates (id, scenario, target_revision, verb, args_json, verdict, total_targets, passed, failed, pending, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	insertResultSQL = `
INSERT OR IGNORE INTO gate_os_results (gate_id, os, node_id, run_id, disposition, exit_code, detail)
VALUES (?, ?, ?, ?, ?, ?, ?)
`

	selectGateColumns = `
SELECT id, scenario, target_revision, verb, args_json, verdict, total_targets, passed, failed, pending, created_at
FROM gates
`

	selectGateByIDSQL = selectGateColumns + `WHERE id = ?`

	selectResultsSQL = `
SELECT os, node_id, run_id, disposition, exit_code, detail
FROM gate_os_results
WHERE gate_id = ?
ORDER BY os ASC
`
)

func (s *sqliteRepository) Create(ctx context.Context, g Gate, results []OSResult) (Gate, error) {
	if g.ID == "" {
		g.ID = uuid.NewString()
	}
	if g.CreatedAt.IsZero() {
		g.CreatedAt = s.clock.Now().UTC()
	}
	argsJSON, err := json.Marshal(nonNilArgs(g.Args))
	if err != nil {
		return Gate{}, fmt.Errorf("marshal gate args: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, insertGateSQL,
		g.ID, g.Scenario, g.TargetRevision, g.Verb, string(argsJSON), int(g.Verdict),
		g.TotalTargets, g.Passed, g.Failed, g.Pending, g.CreatedAt.Format(gateTimeFormat),
	); err != nil {
		return Gate{}, fmt.Errorf("insert gate %q: %w", g.ID, err)
	}
	for _, r := range results {
		if _, err := s.db.ExecContext(ctx, insertResultSQL,
			g.ID, r.OS, r.NodeID, r.RunID, int(r.Disposition), r.ExitCode, r.Detail,
		); err != nil {
			return Gate{}, fmt.Errorf("insert gate result for os %q: %w", r.OS, err)
		}
	}
	return g, nil
}

func (s *sqliteRepository) Get(ctx context.Context, id string) (Gate, error) {
	row := s.db.QueryRowContext(ctx, selectGateByIDSQL, id)
	g, err := scanGate(row)
	if errors.Is(err, sql.ErrNoRows) {
		return Gate{}, ErrGateNotFound{ID: id}
	}
	if err != nil {
		return Gate{}, fmt.Errorf("get gate %q: %w", id, err)
	}
	return g, nil
}

func (s *sqliteRepository) Results(ctx context.Context, gateID string) ([]OSResult, error) {
	rows, err := s.db.QueryContext(ctx, selectResultsSQL, gateID)
	if err != nil {
		return nil, fmt.Errorf("list results for gate %q: %w", gateID, err)
	}
	defer rows.Close()

	var out []OSResult
	for rows.Next() {
		var (
			r           OSResult
			disposition int
		)
		if err := rows.Scan(&r.OS, &r.NodeID, &r.RunID, &disposition, &r.ExitCode, &r.Detail); err != nil {
			return nil, fmt.Errorf("scan gate result: %w", err)
		}
		r.Disposition = OSDisposition(disposition)
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gate results: %w", err)
	}
	return out, nil
}

func (s *sqliteRepository) List(ctx context.Context, filter ListFilter) ([]Gate, error) {
	query := selectGateColumns + `ORDER BY created_at DESC, id DESC`
	args := make([]any, 0, 1)
	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list gates: %w", err)
	}
	defer rows.Close()

	var out []Gate
	for rows.Next() {
		g, err := scanGate(rows)
		if err != nil {
			return nil, fmt.Errorf("scan gate: %w", err)
		}
		out = append(out, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate gates: %w", err)
	}
	return out, nil
}

// rowScanner unifies *sql.Row and *sql.Rows under their common Scan surface.
type rowScanner interface {
	Scan(dest ...any) error
}

func scanGate(sc rowScanner) (Gate, error) {
	var (
		g          Gate
		verdict    int
		argsJSON   string
		createdRaw string
	)
	if err := sc.Scan(&g.ID, &g.Scenario, &g.TargetRevision, &g.Verb, &argsJSON, &verdict,
		&g.TotalTargets, &g.Passed, &g.Failed, &g.Pending, &createdRaw); err != nil {
		return Gate{}, err
	}
	g.Verdict = GateVerdict(verdict)
	if argsJSON != "" {
		if err := json.Unmarshal([]byte(argsJSON), &g.Args); err != nil {
			return Gate{}, fmt.Errorf("unmarshal gate args %q: %w", argsJSON, err)
		}
	}
	created, err := time.Parse(gateTimeFormat, createdRaw)
	if err != nil {
		return Gate{}, fmt.Errorf("parse created_at %q: %w", createdRaw, err)
	}
	g.CreatedAt = created
	return g, nil
}

// nonNilArgs returns a non-nil slice so the JSON encoding is `[]` not `null`.
func nonNilArgs(args []string) []string {
	if args == nil {
		return []string{}
	}
	return args
}
