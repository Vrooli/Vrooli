// Package audit records security-relevant authentication events (sign-in/out,
// token-family revoke, account lockout) to a durable SQLite table. Ported from
// the old handlers/audit.go + logAuthEvent with the placeholders translated to
// SQLite `?`. Writes are best-effort by policy at the call site — a logging
// failure never fails the auth operation — but the Logger surfaces the error so
// the caller can decide.
package audit

import (
	"context"
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"time"

	"github.com/vrooli/api-core/schedule"

	"github.com/google/uuid"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the audit_log SQL contribution for the modules registry.
func Schema() string { return schemaSQL }

const timeFormat = time.RFC3339Nano

// Event is a single auth event to record.
type Event struct {
	UserID    string
	RealmID   string
	Action    string
	IPAddress string
	UserAgent string
	Success   bool
	Metadata  map[string]any
}

// Record is a persisted audit row (read side).
type Record struct {
	ID        string
	UserID    string
	RealmID   string
	Action    string
	IPAddress string
	UserAgent string
	Success   bool
	Metadata  map[string]any
	CreatedAt time.Time
}

// SQLExecutor is the narrow DB surface the logger needs.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error)
}

// Logger writes and reads audit events.
type Logger interface {
	Log(ctx context.Context, e Event) error
	List(ctx context.Context, filter Filter) ([]Record, error)
}

// Filter narrows an audit query.
type Filter struct {
	UserID string
	Action string
	Limit  int
	Offset int
}

type sqliteLogger struct {
	db    SQLExecutor
	clock schedule.Clock
}

// NewSQLiteLogger constructs the production Logger.
func NewSQLiteLogger(db SQLExecutor, clk schedule.Clock) Logger {
	return &sqliteLogger{db: db, clock: clk}
}

var _ Logger = (*sqliteLogger)(nil)

func (l *sqliteLogger) Log(ctx context.Context, e Event) error {
	metaJSON, err := json.Marshal(e.Metadata)
	if err != nil {
		metaJSON = []byte("{}")
	}
	_, err = l.db.ExecContext(ctx, `
INSERT INTO audit_log (id, user_id, realm_id, action, ip_address, user_agent, success, metadata, created_at)
VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		uuid.NewString(), e.UserID, e.RealmID, e.Action, e.IPAddress, e.UserAgent,
		boolToInt(e.Success), string(metaJSON), l.clock.Now().UTC().Format(timeFormat),
	)
	if err != nil {
		return fmt.Errorf("write audit event %q: %w", e.Action, err)
	}
	return nil
}

func (l *sqliteLogger) List(ctx context.Context, f Filter) ([]Record, error) {
	limit := f.Limit
	if limit <= 0 || limit > 1000 {
		limit = 50
	}
	query := `
SELECT id, user_id, realm_id, action, ip_address, user_agent, success, metadata, created_at
FROM audit_log WHERE 1=1`
	args := []any{}
	if f.UserID != "" {
		query += " AND user_id = ?"
		args = append(args, f.UserID)
	}
	if f.Action != "" {
		query += " AND action = ?"
		args = append(args, f.Action)
	}
	query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, f.Offset)

	rows, err := l.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit log: %w", err)
	}
	defer rows.Close()

	var out []Record
	for rows.Next() {
		var (
			r          Record
			successInt int
			metaRaw    string
			createdRaw string
		)
		if err := rows.Scan(&r.ID, &r.UserID, &r.RealmID, &r.Action, &r.IPAddress,
			&r.UserAgent, &successInt, &metaRaw, &createdRaw); err != nil {
			return nil, fmt.Errorf("scan audit row: %w", err)
		}
		r.Success = successInt != 0
		if metaRaw != "" {
			_ = json.Unmarshal([]byte(metaRaw), &r.Metadata)
		}
		if t, err := time.Parse(timeFormat, createdRaw); err == nil {
			r.CreatedAt = t
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
