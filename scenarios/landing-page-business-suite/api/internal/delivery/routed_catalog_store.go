package delivery

import (
	"context"
	"database/sql"

	"github.com/vrooli/api-core/database"
)

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
	return s.db.QueryContext(ctx, query, args...)
}

func (s routedCatalogStore) QueryRow(query string, args ...any) *sql.Row {
	// #nosec G701 -- CatalogService owns call sites and supplies package-constant SQL.
	return s.db.Primary().QueryRow(query, args...)
}

func (s routedCatalogStore) Exec(query string, args ...any) (sql.Result, error) {
	return s.db.Primary().Exec(query, args...)
}

func (s routedCatalogStore) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.db.ExecContext(ctx, query, args...)
}

func (s routedCatalogStore) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, opts)
}
