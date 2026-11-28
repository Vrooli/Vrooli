package workload

import (
	"context"
	"database/sql"
	_ "embed"
	"errors"
	"time"

	sweepv1 "github.com/vrooli/vrooli/packages/proto/gen/go/performance-health/v1/sweep"
	"google.golang.org/protobuf/encoding/protojson"
)

//go:embed schema.sql
var schema string

func Schema() string { return schema }

type Executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Store struct{ db Executor }

func NewStore(db Executor) *Store { return &Store{db: db} }

// Insert admits an attempt before producer effects. Its initial unavailable
// reading prevents an older success from resurfacing after a terminal-write fault.
func (s *Store) Insert(ctx context.Context, r *sweepv1.WorkloadReading) error {
	if s == nil || s.db == nil {
		return errors.New("workload store unavailable")
	}
	b, err := protojson.Marshal(r)
	if err != nil {
		return err
	}
	started, err := time.Parse(time.RFC3339Nano, r.CapturedAt)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO performance_workload_receipts
		(operation_id, scenario, workload, started_at_ns, reading_json) VALUES (?, ?, ?, ?, ?)`,
		r.OperationId, r.Scenario, r.Workload, started.UnixNano(), string(b))
	return err
}

// Complete commits one terminal reading. Repeated finalization cannot overwrite
// an existing outcome, and a failed write leaves the admitted attempt unknown.
func (s *Store) Complete(ctx context.Context, r *sweepv1.WorkloadReading) error {
	if s == nil || s.db == nil {
		return errors.New("workload store unavailable")
	}
	b, err := protojson.Marshal(r)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE performance_workload_receipts SET reading_json = ?, completed = 1
		WHERE operation_id = ? AND completed = 0`, string(b), r.OperationId)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n != 1 {
		return errors.New("workload admission missing or already finalized")
	}
	return nil
}

func (s *Store) Latest(ctx context.Context, scenario, name string) (*sweepv1.WorkloadReading, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("workload store unavailable")
	}
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT reading_json FROM performance_workload_receipts
		WHERE scenario = ? AND workload = ? ORDER BY started_at_ns DESC, rowid DESC LIMIT 1`, scenario, name).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	r := &sweepv1.WorkloadReading{}
	if err := protojson.Unmarshal([]byte(raw), r); err != nil {
		return nil, err
	}
	return r, nil
}
