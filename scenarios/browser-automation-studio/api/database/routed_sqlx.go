package database

import (
	"context"
	"database/sql"
	"fmt"
	"reflect"

	"github.com/jmoiron/sqlx"
)

// ExecContext, QueryContext, QueryRowContext, GetContext, SelectContext, and
// NamedExecContext preserve the repository's sqlx-shaped API while routing by
// request context whenever DB.Routed is configured. Keeping this compatibility
// layer at the database boundary prevents handlers and domain services from
// choosing their own persistence substrate during the migration.
func (db *DB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	if db.Routed != nil {
		return db.Routed.ExecContext(ctx, query, args...)
	}
	return db.DB.ExecContext(ctx, query, args...)
}

func (db *DB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	if db.Routed != nil {
		return db.Routed.QueryContext(ctx, query, args...)
	}
	return db.DB.QueryContext(ctx, query, args...)
}

func (db *DB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	if db.Routed != nil {
		return db.Routed.QueryRowContext(ctx, query, args...)
	}
	return db.DB.QueryRowContext(ctx, query, args...)
}

func (db *DB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	if db.Routed != nil {
		return db.Routed.BeginTx(ctx, opts)
	}
	return db.DB.BeginTx(ctx, opts)
}

func (db *DB) GetContext(ctx context.Context, dest any, query string, args ...any) error {
	if db.Routed == nil {
		return db.DB.GetContext(ctx, dest, query, args...)
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("GetContext destination must be a non-nil pointer")
	}
	if value.Elem().Kind() == reflect.Struct {
		// sqlx.StructScan deliberately accepts a slice destination and advances
		// the cursor itself. Calling it after rows.Next (as the original routed
		// adapter did) both skips the current row and returns "expected slice but
		// got struct". Scan through a temporary slice, then preserve GetContext's
		// single-row contract by assigning its first result.
		result := reflect.New(reflect.SliceOf(value.Elem().Type()))
		if err := sqlx.StructScan(rows, result.Interface()); err != nil {
			return err
		}
		if result.Elem().Len() == 0 {
			return sql.ErrNoRows
		}
		value.Elem().Set(result.Elem().Index(0))
		return nil
	}
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return err
		}
		return sql.ErrNoRows
	}
	if err := scanCurrentRow(rows, dest); err != nil {
		return err
	}
	return rows.Err()
}

func (db *DB) SelectContext(ctx context.Context, dest any, query string, args ...any) error {
	if db.Routed == nil {
		return db.DB.SelectContext(ctx, dest, query, args...)
	}
	slice, err := sliceDestination(dest)
	if err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	elementType := slice.Type().Elem()
	if elementType.Kind() == reflect.Struct || (elementType.Kind() == reflect.Pointer && elementType.Elem().Kind() == reflect.Struct) {
		return sqlx.StructScan(rows, dest)
	}
	for rows.Next() {
		var element reflect.Value
		if elementType.Kind() == reflect.Pointer {
			element = reflect.New(elementType.Elem())
		} else {
			element = reflect.New(elementType)
		}
		if err := scanCurrentRow(rows, element.Interface()); err != nil {
			return err
		}
		if elementType.Kind() == reflect.Pointer {
			slice.Set(reflect.Append(slice, element))
		} else {
			slice.Set(reflect.Append(slice, element.Elem()))
		}
	}
	return rows.Err()
}

func (db *DB) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	if db.Routed == nil {
		return db.DB.NamedExecContext(ctx, query, arg)
	}
	bound, args, err := sqlx.Named(query, arg)
	if err != nil {
		return nil, err
	}
	return db.ExecContext(ctx, bound, args...)
}

func sliceDestination(dest any) (reflect.Value, error) {
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Slice {
		return reflect.Value{}, fmt.Errorf("SelectContext destination must be a non-nil pointer to a slice")
	}
	return value.Elem(), nil
}

func scanCurrentRow(rows *sql.Rows, dest any) error {
	value := reflect.ValueOf(dest)
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return fmt.Errorf("scan destination must be a non-nil pointer")
	}
	return rows.Scan(dest)
}
