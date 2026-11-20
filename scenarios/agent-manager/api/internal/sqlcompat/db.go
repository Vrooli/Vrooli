// Package sqlcompat defines the narrow sqlx-shaped interface shared by
// Agent Manager's routed production database and focused sqlx test fixtures.
package sqlcompat

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/jmoiron/sqlx"
)

// TransientError recognizes SQLite contention through the driver's code,
// including extended BUSY/LOCKED codes, without parsing diagnostic messages.
func TransientError(err error) bool {
	if errors.Is(err, sql.ErrConnDone) {
		return true
	}
	var coded interface{ Code() int }
	if errors.As(err, &coded) {
		code := coded.Code() & 0xff
		return code == 5 || code == 6
	}
	return false
}

// TransientMessage recognizes SQLite contention from a driver that exposes no
// error code. Message text is untrusted for a wrapped application error, so
// this is deliberately separate from TransientError; callers use it only where
// the error came straight from the driver or a legacy row recorded its text.
func TransientMessage(message string) bool {
	for _, marker := range []string{"SQLITE_BUSY", "SQLITE_LOCKED", "database is locked", "database table is locked"} {
		if strings.Contains(message, marker) {
			return true
		}
	}
	return false
}

type DB interface {
	Exec(string, ...any) (sql.Result, error)
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
	QueryxContext(context.Context, string, ...any) (*sqlx.Rows, error)
	GetContext(context.Context, any, string, ...any) error
	SelectContext(context.Context, any, string, ...any) error
	Conn(context.Context) (*sql.Conn, error)
}
