package eval

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"search-hub/internal/clock"

	"google.golang.org/protobuf/encoding/protojson"

	evalv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/eval"
)

// SQLExecutor is the narrow database surface the store depends on. Declared at
// the consumer per seam-discovery: both *sql.DB (store tests) and
// *database.RoutedDB (production main.go) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// Store is the persistence seam the eval handler and runner depend on.
// Production wires the SQLite-backed impl; unit tests wire a fake.
type Store interface {
	// UpsertSuite validates and persists s (upsert keyed by suite_id). Returns
	// created=true on insert, false on update. Returns ErrInvalidSuite on
	// validation failure.
	UpsertSuite(ctx context.Context, s *evalv1.EvalSuite) (created bool, err error)
	// ListSuites returns suites matching filter, ordered by suite_id.
	ListSuites(ctx context.Context, filter ListSuitesFilter) ([]*evalv1.EvalSuite, error)
	// GetSuite returns the suite for id or ErrSuiteNotFound.
	GetSuite(ctx context.Context, id string) (*evalv1.EvalSuite, error)

	// AppendRun stores an immutable run. The run's created_at is stamped by the
	// store (clock seam) if unset, and run_id must be provided by the caller.
	AppendRun(ctx context.Context, run *evalv1.EvalRun) error
	// ListRuns returns runs for a suite (newest first), optionally filtered by
	// tag and capped by limit.
	ListRuns(ctx context.Context, filter ListRunsFilter) ([]*evalv1.EvalRun, error)
	// GetRun returns the run for id or ErrRunNotFound.
	GetRun(ctx context.Context, id string) (*evalv1.EvalRun, error)
}

// sqliteStore is the production Store impl. Unexported so callers depend on the
// Store interface and substitute the fake without reaching inside.
type sqliteStore struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteStore constructs the production Store. db is the connection pool
// opened in main.go (*database.RoutedDB in production, *sql.DB in unit tests);
// clk supplies created_at/updated_at timestamps so tests can advance time
// deterministically.
func NewSQLiteStore(db SQLExecutor, clk clock.Clock) Store {
	return &sqliteStore{db: db, clock: clk}
}

// Compile-time guarantee.
var _ Store = (*sqliteStore)(nil)

const timeFormat = time.RFC3339Nano

// marshalOpts keeps blobs compact (EmitUnpopulated false) and human-inspectable
// (proto field names) — same convention as the registry store.
var marshalOpts = protojson.MarshalOptions{UseProtoNames: true}

func (s *sqliteStore) UpsertSuite(ctx context.Context, suite *evalv1.EvalSuite) (bool, error) {
	Normalize(suite)
	if err := Validate(suite); err != nil {
		return false, err
	}

	now := s.clock.Now().UTC().Format(timeFormat)

	// Determine insert vs update so we can report `created` honestly and so the
	// immutable created_at survives an update. SQLite is a single writer, so the
	// read-then-write race is benign.
	var existingCreatedAt string
	err := s.db.QueryRowContext(ctx, `SELECT created_at FROM eval_suites WHERE suite_id = ?`, suite.SuiteId).
		Scan(&existingCreatedAt)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		suite.CreatedAt = now
		suite.UpdatedAt = now
		blob, mErr := marshalOpts.Marshal(suite)
		if mErr != nil {
			return false, fmt.Errorf("marshal suite: %w", mErr)
		}
		_, insErr := s.db.ExecContext(ctx, `
INSERT INTO eval_suites (suite_id, provider_id, name, state, descriptor, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?, ?)`,
			suite.SuiteId, suite.ProviderId, suite.Name, suite.State, string(blob), now, now)
		if insErr != nil {
			return false, fmt.Errorf("insert suite: %w", insErr)
		}
		return true, nil
	case err != nil:
		return false, fmt.Errorf("probe suite: %w", err)
	default:
		suite.CreatedAt = existingCreatedAt
		suite.UpdatedAt = now
		blob, mErr := marshalOpts.Marshal(suite)
		if mErr != nil {
			return false, fmt.Errorf("marshal suite: %w", mErr)
		}
		_, updErr := s.db.ExecContext(ctx, `
UPDATE eval_suites
SET provider_id = ?, name = ?, state = ?, descriptor = ?, updated_at = ?
WHERE suite_id = ?`,
			suite.ProviderId, suite.Name, suite.State, string(blob), now, suite.SuiteId)
		if updErr != nil {
			return false, fmt.Errorf("update suite: %w", updErr)
		}
		return false, nil
	}
}

func (s *sqliteStore) ListSuites(ctx context.Context, filter ListSuitesFilter) ([]*evalv1.EvalSuite, error) {
	var (
		clauses []string
		args    []any
	)
	if strings.TrimSpace(filter.ProviderID) != "" {
		clauses = append(clauses, "provider_id = ?")
		args = append(args, filter.ProviderID)
	}
	query := "SELECT descriptor FROM eval_suites"
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}
	query += " ORDER BY suite_id ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query suites: %w", err)
	}
	defer rows.Close()

	out := make([]*evalv1.EvalSuite, 0)
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, fmt.Errorf("scan suite: %w", err)
		}
		suite, uErr := unmarshalSuite(blob)
		if uErr != nil {
			return nil, uErr
		}
		out = append(out, suite)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate suites: %w", err)
	}
	return out, nil
}

func (s *sqliteStore) GetSuite(ctx context.Context, id string) (*evalv1.EvalSuite, error) {
	var blob string
	err := s.db.QueryRowContext(ctx, `SELECT descriptor FROM eval_suites WHERE suite_id = ?`, id).Scan(&blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrSuiteNotFound{SuiteID: id}
	}
	if err != nil {
		return nil, fmt.Errorf("get suite: %w", err)
	}
	return unmarshalSuite(blob)
}

func (s *sqliteStore) AppendRun(ctx context.Context, run *evalv1.EvalRun) error {
	if run == nil {
		return fmt.Errorf("nil run")
	}
	if strings.TrimSpace(run.RunId) == "" {
		return fmt.Errorf("run_id required")
	}
	if strings.TrimSpace(run.CreatedAt) == "" {
		run.CreatedAt = s.clock.Now().UTC().Format(timeFormat)
	}
	blob, err := marshalOpts.Marshal(run)
	if err != nil {
		return fmt.Errorf("marshal run: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `
INSERT INTO eval_runs (run_id, suite_id, tag, created_at, result)
VALUES (?, ?, ?, ?, ?)`,
		run.RunId, run.SuiteId, run.Tag, run.CreatedAt, string(blob))
	if err != nil {
		return fmt.Errorf("insert run: %w", err)
	}
	return nil
}

func (s *sqliteStore) ListRuns(ctx context.Context, filter ListRunsFilter) ([]*evalv1.EvalRun, error) {
	clauses := []string{"suite_id = ?"}
	args := []any{filter.SuiteID}
	if strings.TrimSpace(filter.Tag) != "" {
		clauses = append(clauses, "tag = ?")
		args = append(args, filter.Tag)
	}
	// Newest first: created_at is RFC3339Nano (lexicographically sortable), with
	// run_id as a stable tiebreaker for runs sharing a timestamp.
	query := "SELECT result FROM eval_runs WHERE " + strings.Join(clauses, " AND ") +
		" ORDER BY created_at DESC, run_id DESC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query runs: %w", err)
	}
	defer rows.Close()

	out := make([]*evalv1.EvalRun, 0)
	for rows.Next() {
		var blob string
		if err := rows.Scan(&blob); err != nil {
			return nil, fmt.Errorf("scan run: %w", err)
		}
		run, uErr := unmarshalRun(blob)
		if uErr != nil {
			return nil, uErr
		}
		out = append(out, run)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate runs: %w", err)
	}
	return out, nil
}

func (s *sqliteStore) GetRun(ctx context.Context, id string) (*evalv1.EvalRun, error) {
	var blob string
	err := s.db.QueryRowContext(ctx, `SELECT result FROM eval_runs WHERE run_id = ?`, id).Scan(&blob)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrRunNotFound{RunID: id}
	}
	if err != nil {
		return nil, fmt.Errorf("get run: %w", err)
	}
	return unmarshalRun(blob)
}

func unmarshalSuite(blob string) (*evalv1.EvalSuite, error) {
	s := &evalv1.EvalSuite{}
	if err := protojson.Unmarshal([]byte(blob), s); err != nil {
		return nil, fmt.Errorf("unmarshal suite: %w", err)
	}
	return s, nil
}

func unmarshalRun(blob string) (*evalv1.EvalRun, error) {
	r := &evalv1.EvalRun{}
	if err := protojson.Unmarshal([]byte(blob), r); err != nil {
		return nil, fmt.Errorf("unmarshal run: %w", err)
	}
	return r, nil
}
