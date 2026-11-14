package delivery

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/vrooli/api-core/database"
)

var ErrRequestStorageLeaseUnavailable = errors.New("request storage lease unavailable")

// NewRoutedCatalogStore adapts RoutedDB to the delivery catalog persistence
// boundary. Context-free operations use primary; request-aware operations keep
// the request routing marker used by tests and read/write policy.
func NewRoutedCatalogStore(db *database.RoutedDB) CatalogStore {
	return routedCatalogStore{db: db}
}

type routedCatalogStore struct {
	db *database.RoutedDB
}

func (s routedCatalogStore) Query(query string, args ...any) (*sql.Rows, error) {
	return s.db.Primary().Query(query, args...)
}

func (s routedCatalogStore) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	pool, err := s.db.PoolForContext(ctx)
	if err != nil {
		return nil, err
	}
	return pool.QueryContext(ctx, query, args...)
}

func (s routedCatalogStore) QueryRow(query string, args ...any) *sql.Row {
	// #nosec G701 -- CatalogService owns call sites and supplies package-constant SQL.
	return s.db.Primary().QueryRow(query, args...)
}

func (s routedCatalogStore) Exec(query string, args ...any) (sql.Result, error) {
	return s.db.Primary().Exec(query, args...)
}

func (s routedCatalogStore) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	pool, err := s.db.PoolForContext(ctx)
	if err != nil {
		return nil, err
	}
	return pool.ExecContext(ctx, query, args...)
}

func (s routedCatalogStore) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	pool, err := s.db.PoolForContext(ctx)
	if err != nil {
		return nil, err
	}
	return pool.BeginTx(ctx, opts)
}

// RequestStoreResolver binds one request to one concrete database pool. The
// returned Store never re-routes, so a lease cannot expire or be cleared
// between validation and a later QueryRowContext call.
type RequestStoreResolver interface {
	ResolveRequestStore(context.Context) (Store, error)
}

type strictRoutedStore struct{ db *database.RoutedDB }

// NewStrictRoutedStore returns the request-safe resolver used by
// artifact/settings reads and presigning.
func NewStrictRoutedStore(db *database.RoutedDB) RequestStoreResolver {
	return strictRoutedStore{db: db}
}

func (s strictRoutedStore) ResolveRequestStore(ctx context.Context) (Store, error) {
	if s.db == nil {
		return nil, ErrRequestStorageLeaseUnavailable
	}
	pool, err := s.db.PoolForContext(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRequestStorageLeaseUnavailable, err)
	}
	return boundRoutedStore{db: pool}, nil
}

// boundRoutedStore is a concrete-pool Store. It deliberately has no lease
// lookup: the resolver performed the one admission check for the operation.
type boundRoutedStore struct{ db *sql.DB }

func (s boundRoutedStore) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return s.db.QueryRowContext(ctx, query, args...)
}

func (s boundRoutedStore) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.db.QueryContext(ctx, query, args...)
}

func (s boundRoutedStore) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}
