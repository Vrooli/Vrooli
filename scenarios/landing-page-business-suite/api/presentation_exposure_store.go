package main

import (
	"context"
	"database/sql"

	"github.com/vrooli/api-core/database"
)

// strictPresentationExposureStore is deliberately narrower than RoutedDB's
// general ExecContext surface. Presentation attribution is a testable write:
// a marked request must have an active lease, never silently fall back to the
// primary pool.
type strictPresentationExposureStore struct {
	routed *database.RoutedDB
}

func (s strictPresentationExposureStore) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	pool, err := s.routed.PoolForContext(ctx)
	if err != nil {
		return nil, err
	}
	return pool.ExecContext(ctx, query, args...)
}
