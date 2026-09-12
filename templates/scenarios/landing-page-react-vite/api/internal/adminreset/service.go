// Package adminreset performs the env-gated demo-data reset: TRUNCATE the demo
// tables (respecting FK order) inside a transaction, then reseed defaults via an
// injected callback (main.go's seedDefaultData). The AdminReset Connect handler
// in handlers/reset adapts this Service.
package adminreset

import (
	"context"
	"database/sql"
	"fmt"
)

// demoTables are truncated on reset, ordered so FK children precede parents.
var demoTables = []string{
	"content_sections",
	"variant_axes",
	"variants",
	"download_assets",
	"download_apps",
	"bundle_prices",
	"bundle_products",
}

// Service truncates and reseeds the demo tables.
type Service struct {
	db     *sql.DB
	reseed func(ctx context.Context) error
}

// NewService wires the database and the reseed callback (seedDefaultData).
func NewService(db *sql.DB, reseed func(ctx context.Context) error) *Service {
	return &Service{db: db, reseed: reseed}
}

// Reset truncates every demo table then reseeds default data.
func (s *Service) Reset(ctx context.Context) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	for _, table := range demoTables {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("TRUNCATE TABLE %s CASCADE", table)); err != nil {
			return fmt.Errorf("truncate %s: %w", table, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	if s.reseed == nil {
		return nil
	}
	return s.reseed(ctx)
}
