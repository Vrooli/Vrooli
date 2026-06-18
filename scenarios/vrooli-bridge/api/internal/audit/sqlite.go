package audit

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"vrooli-bridge/internal/clock"

	"github.com/google/uuid"
)

// SQLExecutor is the narrow database surface sqliteStore depends on. Both
// *sql.DB (tests) and *database.RoutedDB (production) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteStore struct {
	db    SQLExecutor
	clock clock.Clock
}

// NewSQLiteStore constructs the production Store (Sink + Reader). This is the
// default accountability substrate; it is append-only by construction (only the
// INSERT and SELECT below exist).
func NewSQLiteStore(db SQLExecutor, clk clock.Clock) Store {
	return &sqliteStore{db: db, clock: clk}
}

// Compile-time guarantees.
var (
	_ Store  = (*sqliteStore)(nil)
	_ Sink   = (*sqliteStore)(nil)
	_ Reader = (*sqliteStore)(nil)
)

const auditTimeFormat = time.RFC3339Nano

const (
	insertAuditSQL = `
INSERT INTO audit_records (id, action, actor, node_id, scenario, verb, args, outcome, detail, run_id, recorded_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`

	selectAuditColumns = `
SELECT id, action, actor, node_id, scenario, verb, args, outcome, detail, run_id, recorded_at
FROM audit_records
`
)

func (s *sqliteStore) Append(ctx context.Context, r Record) (Record, error) {
	if strings.TrimSpace(r.Actor) == "" {
		return Record{}, ErrInvalidRecord{Field: "actor", Reason: "required"}
	}
	if strings.TrimSpace(r.NodeID) == "" {
		return Record{}, ErrInvalidRecord{Field: "node_id", Reason: "required"}
	}
	if r.ID == "" {
		r.ID = uuid.NewString()
	}
	if r.RecordedAt.IsZero() {
		r.RecordedAt = s.clock.Now().UTC()
	}
	args, err := marshalStrings(r.Args)
	if err != nil {
		return Record{}, fmt.Errorf("encode args: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, insertAuditSQL,
		r.ID, int(r.Action), r.Actor, r.NodeID, r.Scenario, r.Verb, args,
		int(r.Outcome), r.Detail, r.RunID, r.RecordedAt.Format(auditTimeFormat),
	); err != nil {
		return Record{}, fmt.Errorf("append audit record: %w", err)
	}
	return r, nil
}

func (s *sqliteStore) List(ctx context.Context, filter ListFilter) ([]Record, error) {
	query := selectAuditColumns
	clauses := make([]string, 0, 2)
	args := make([]any, 0, 3)
	if filter.NodeID != "" {
		clauses = append(clauses, "node_id = ?")
		args = append(args, filter.NodeID)
	}
	if filter.RunID != "" {
		clauses = append(clauses, "run_id = ?")
		args = append(args, filter.RunID)
	}
	if len(clauses) > 0 {
		query += "WHERE " + strings.Join(clauses, " AND ") + " "
	}
	query += "ORDER BY recorded_at DESC, id DESC"
	if filter.Limit > 0 {
		query += " LIMIT ?"
		args = append(args, filter.Limit)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list audit records: %w", err)
	}
	defer rows.Close()

	var records []Record
	for rows.Next() {
		var (
			r          Record
			action     int
			outcome    int
			argsRaw    string
			recordedAt string
		)
		if err := rows.Scan(&r.ID, &action, &r.Actor, &r.NodeID, &r.Scenario, &r.Verb,
			&argsRaw, &outcome, &r.Detail, &r.RunID, &recordedAt); err != nil {
			return nil, fmt.Errorf("scan audit record: %w", err)
		}
		r.Action = Action(action)
		r.Outcome = Outcome(outcome)
		if r.Args, err = unmarshalStrings(argsRaw); err != nil {
			return nil, fmt.Errorf("decode args: %w", err)
		}
		if r.RecordedAt, err = time.Parse(auditTimeFormat, recordedAt); err != nil {
			return nil, fmt.Errorf("parse recorded_at %q: %w", recordedAt, err)
		}
		records = append(records, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit records: %w", err)
	}
	return records, nil
}

func marshalStrings(v []string) (string, error) {
	if v == nil {
		v = []string{}
	}
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

func unmarshalStrings(raw string) ([]string, error) {
	if raw == "" {
		return nil, nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, err
	}
	if len(out) == 0 {
		return nil, nil
	}
	return out, nil
}
