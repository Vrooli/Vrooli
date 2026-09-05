package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strconv"
	"time"
)

// DialectHelper provides SQL dialect-specific expressions for databases
// that need to support both PostgreSQL and SQLite.
type DialectHelper struct {
	dialect string // "postgres" or "sqlite"
}

// NewDialectHelper creates a new dialect helper.
// If dialect is empty, defaults to "postgres".
func NewDialectHelper(dialect string) *DialectHelper {
	if dialect == "" {
		dialect = "postgres"
	}
	return &DialectHelper{dialect: dialect}
}

// NowExpr returns the appropriate SQL expression for current timestamp.
// PostgreSQL: NOW()
// SQLite: datetime('now')
func (d *DialectHelper) NowExpr() string {
	if d.dialect == "sqlite" {
		return "datetime('now')"
	}
	return "NOW()"
}

// Placeholder returns the appropriate placeholder for the given index.
// PostgreSQL uses $1, $2, etc.
// SQLite uses ?, but we use numbered placeholders for consistency.
// Note: For SQLite compatibility, consider using sqlx or named parameters.
func (d *DialectHelper) Placeholder(index int) string {
	// Currently both use PostgreSQL-style placeholders
	// as the database/sql driver handles translation
	return "$" + strconv.Itoa(index)
}

// IsSQLite returns true if the dialect is SQLite.
func (d *DialectHelper) IsSQLite() bool {
	return d.dialect == "sqlite"
}

// IsPostgres returns true if the dialect is PostgreSQL.
func (d *DialectHelper) IsPostgres() bool {
	return d.dialect == "postgres" || d.dialect == ""
}

// Dialect returns the current dialect string.
func (d *DialectHelper) Dialect() string {
	return d.dialect
}

// NullStringValue extracts the value from a sql.NullString.
// Returns nil if the NullString is not valid, otherwise returns a pointer to the string value.
func NullStringValue(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	return &ns.String
}

// NullInt64Value extracts the value from a sql.NullInt64.
// Returns nil if the NullInt64 is not valid, otherwise returns a pointer to the int64 value.
func NullInt64Value(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	return &ni.Int64
}

// NullFloat64Value extracts the value from a sql.NullFloat64.
// Returns nil if the NullFloat64 is not valid, otherwise returns a pointer to the float64 value.
func NullFloat64Value(nf sql.NullFloat64) *float64 {
	if !nf.Valid {
		return nil
	}
	return &nf.Float64
}

// NullBoolValue extracts the value from a sql.NullBool.
// Returns nil if the NullBool is not valid, otherwise returns a pointer to the bool value.
func NullBoolValue(nb sql.NullBool) *bool {
	if !nb.Valid {
		return nil
	}
	return &nb.Bool
}

// StringToNullString converts a *string to sql.NullString.
// Returns an invalid NullString if the pointer is nil.
func StringToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// Int64ToNullInt64 converts a *int64 to sql.NullInt64.
// Returns an invalid NullInt64 if the pointer is nil.
func Int64ToNullInt64(i *int64) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: *i, Valid: true}
}

// NullTimeValue extracts the value from a sql.NullTime.
// Returns nil if the NullTime is not valid, otherwise returns a pointer to the time value.
func NullTimeValue(nt sql.NullTime) *time.Time {
	if !nt.Valid {
		return nil
	}
	return &nt.Time
}

// TimeToNullTime converts a *time.Time to sql.NullTime.
// Returns an invalid NullTime if the pointer is nil.
func TimeToNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

// QueryRowResult represents the result of a single-row query.
// It provides a convenient way to handle sql.ErrNoRows without explicit checks.
type QueryRowResult struct {
	err   error
	found bool
}

// Err returns the error from the query, or nil if successful (including not found).
func (r *QueryRowResult) Err() error {
	return r.err
}

// Found returns true if a row was found and scanned successfully.
func (r *QueryRowResult) Found() bool {
	return r.found
}

// NotFound returns true if no row was found (sql.ErrNoRows).
func (r *QueryRowResult) NotFound() bool {
	return !r.found && r.err == nil
}

// ScanSingleRow executes a query expected to return at most one row and scans into dest.
// This handles the common pattern of QueryRowContext + Scan with ErrNoRows handling.
//
// Returns a QueryRowResult that can be checked for:
//   - Err() for actual database errors
//   - Found() for successful scan
//   - NotFound() for no matching row
//
// Example:
//
//	result := ScanSingleRow(ctx, db, "SELECT name FROM users WHERE id = $1", []any{userID}, &name)
//	if result.Err() != nil {
//	    return fmt.Errorf("query failed: %w", result.Err())
//	}
//	if result.NotFound() {
//	    return nil, nil // No user found
//	}
type QueryRowContextStore interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func ScanSingleRow(ctx context.Context, db QueryRowContextStore, query string, args []any, dest ...any) QueryRowResult {
	err := db.QueryRowContext(ctx, query, args...).Scan(dest...)
	if err == sql.ErrNoRows {
		return QueryRowResult{err: nil, found: false}
	}
	if err != nil {
		return QueryRowResult{err: err, found: false}
	}
	return QueryRowResult{err: nil, found: true}
}

// ScanSingleRowTx is like ScanSingleRow but uses an existing transaction.
func ScanSingleRowTx(ctx context.Context, tx *sql.Tx, query string, args []any, dest ...any) QueryRowResult {
	err := tx.QueryRowContext(ctx, query, args...).Scan(dest...)
	if err == sql.ErrNoRows {
		return QueryRowResult{err: nil, found: false}
	}
	if err != nil {
		return QueryRowResult{err: err, found: false}
	}
	return QueryRowResult{err: nil, found: true}
}

// TransactionFunc is the function type used by WithTransaction.
// It receives a transaction and returns an error.
// If the function returns nil, the transaction is committed.
// If it returns an error, the transaction is rolled back.
type TransactionFunc func(tx *sql.Tx) error

// WithTransaction executes fn within a database transaction.
// If fn returns nil, the transaction is committed.
// If fn returns an error, the transaction is rolled back.
// If the commit fails, that error is returned.
//
// Example:
//
//	err := WithTransaction(ctx, db, nil, func(tx *sql.Tx) error {
//	    if _, err := tx.ExecContext(ctx, "UPDATE users SET active = true WHERE id = $1", userID); err != nil {
//	        return err
//	    }
//	    if _, err := tx.ExecContext(ctx, "INSERT INTO audit_log (user_id, action) VALUES ($1, 'activated')", userID); err != nil {
//	        return err
//	    }
//	    return nil
//	})
type TransactionStarter interface {
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

func WithTransaction(ctx context.Context, db TransactionStarter, opts *sql.TxOptions, fn TransactionFunc) error {
	tx, err := db.BeginTx(ctx, opts)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}

	// Ensure rollback is called if commit doesn't happen
	defer func() {
		if p := recover(); p != nil {
			if err := tx.Rollback(); err != nil {
				log.Printf("transaction rollback failed during panic recovery: %v", err)
			}
			panic(p) // Re-throw panic after rollback
		}
	}()

	if err := fn(tx); err != nil {
		rbErr := tx.Rollback()
		if rbErr != nil {
			return fmt.Errorf("rollback failed after error: %w (original: %w)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// WithSerializableTransaction executes fn within a serializable transaction.
// This provides the highest isolation level, preventing phantom reads.
// Use this for operations that require strict consistency.
//
// Example:
//
//	err := WithSerializableTransaction(ctx, db, func(tx *sql.Tx) error {
//	    // Read current balance
//	    var balance int64
//	    if err := tx.QueryRowContext(ctx, "SELECT balance FROM accounts WHERE id = $1 FOR UPDATE", accountID).Scan(&balance); err != nil {
//	        return err
//	    }
//	    // Update with new balance
//	    if _, err := tx.ExecContext(ctx, "UPDATE accounts SET balance = $1 WHERE id = $2", balance-amount, accountID); err != nil {
//	        return err
//	    }
//	    return nil
//	})
func WithSerializableTransaction(ctx context.Context, db TransactionStarter, fn TransactionFunc) error {
	return WithTransaction(ctx, db, &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	}, fn)
}

// WithReadCommittedTransaction executes fn within a read committed transaction.
// This is the default isolation level for PostgreSQL.
func WithReadCommittedTransaction(ctx context.Context, db TransactionStarter, fn TransactionFunc) error {
	return WithTransaction(ctx, db, &sql.TxOptions{
		Isolation: sql.LevelReadCommitted,
	}, fn)
}

// ExecWithAffectedRows executes a query and returns an error if no rows were affected.
// Useful for UPDATE/DELETE operations where you expect at least one row to be modified.
//
// Example:
//
//	if err := ExecWithAffectedRows(ctx, db, "DELETE FROM users WHERE id = $1", userID); err != nil {
//	    return fmt.Errorf("user not found or already deleted")
//	}
type ExecContextStore interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

func ExecWithAffectedRows(ctx context.Context, db ExecContextStore, query string, args ...any) error {
	result, err := db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}

// ExecWithAffectedRowsTx is like ExecWithAffectedRows but uses an existing transaction.
func ExecWithAffectedRowsTx(ctx context.Context, tx *sql.Tx, query string, args ...any) error {
	result, err := tx.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if affected == 0 {
		return sql.ErrNoRows
	}

	return nil
}
