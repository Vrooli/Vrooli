package sqliterows

import (
	"context"
	"database/sql"
	"fmt"
)

// Scanner is implemented by both *sql.Rows and *sql.Row.
type Scanner interface {
	Scan(dest ...any) error
}

// Querier is the query surface implemented by *sql.DB and *sql.Tx.
type Querier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

// ErrorContext preserves the operation-specific diagnostics of the old
// hand-written row loops. Empty values leave errors unchanged.
type ErrorContext struct {
	Query, Scan, Iterate, Close string
}

// Collect reads typed values while owning the common SQLite row lifecycle.
// checkClose preserves callers that treat a cursor-close error as an operation
// failure; callers that historically deferred and ignored Close pass false.
func Collect[T any](ctx context.Context, db Querier, query string, args []any, checkClose bool, scan func(Scanner) (T, error), contexts ...ErrorContext) ([]T, error) {
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		var context ErrorContext
		if len(contexts) > 0 {
			context = contexts[0]
		}
		return nil, withContext(context.Query, err)
	}
	defer rows.Close()
	return collectRows(rows, checkClose, scan, contexts...)
}

// CollectRows applies the same lifecycle to a cursor that a caller already
// opened and configured, such as a SELECT * archive export that reads column
// names before scanning values.
func CollectRows[T any](rows *sql.Rows, checkClose bool, scan func(Scanner) (T, error), contexts ...ErrorContext) ([]T, error) {
	defer rows.Close()
	return collectRows(rows, checkClose, scan, contexts...)
}

func collectRows[T any](rows *sql.Rows, checkClose bool, scan func(Scanner) (T, error), contexts ...ErrorContext) ([]T, error) {
	var context ErrorContext
	if len(contexts) > 0 {
		context = contexts[0]
	}
	var values []T
	for rows.Next() {
		value, err := scan(rows)
		if err != nil {
			return nil, withContext(context.Scan, err)
		}
		values = append(values, value)
	}
	if err := rows.Err(); err != nil {
		return nil, withContext(context.Iterate, err)
	}
	if err := rows.Close(); checkClose && err != nil {
		return nil, withContext(context.Close, err)
	}
	return values, nil
}

func withContext(prefix string, err error) error {
	if prefix == "" {
		return err
	}
	return fmt.Errorf("%s: %w", prefix, err)
}
