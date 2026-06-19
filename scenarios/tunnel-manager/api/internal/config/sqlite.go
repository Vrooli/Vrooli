package config

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// singletonID is the fixed primary key of the one config row. The config
// is a singleton: there is always exactly one logical row.
const singletonID = "singleton"

// SQLExecutor is the narrow database surface sqliteRepository depends on.
// Declared at the consumer per seam-discovery: both *sql.DB (repository
// unit tests via testutil/db.NewSQLite) and *database.RoutedDB (production
// main.go) satisfy it.
type SQLExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

type sqliteRepository struct {
	db SQLExecutor
}

// NewSQLiteRepository constructs the production ConfigRepository.
func NewSQLiteRepository(db SQLExecutor) ConfigRepository {
	return &sqliteRepository{db: db}
}

// Compile-time guarantee.
var _ ConfigRepository = (*sqliteRepository)(nil)

const (
	configColumns = `mode, tunnel_id, account_id, cred_ref, prom_endpoint`

	upsertConfigSQL = `
INSERT INTO tunnel_config (id, mode, tunnel_id, account_id, cred_ref, prom_endpoint)
VALUES (?, ?, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  mode = excluded.mode,
  tunnel_id = excluded.tunnel_id,
  account_id = excluded.account_id,
  cred_ref = excluded.cred_ref,
  prom_endpoint = excluded.prom_endpoint
`
)

func (s *sqliteRepository) Get(ctx context.Context) (TunnelConfig, error) {
	row := s.db.QueryRowContext(ctx, "SELECT "+configColumns+" FROM tunnel_config WHERE id = ?", singletonID)
	var (
		cfg     TunnelConfig
		modeRaw string
	)
	err := row.Scan(&modeRaw, &cfg.TunnelID, &cfg.AccountID, &cfg.CredRef, &cfg.PromEndpoint)
	if errors.Is(err, sql.ErrNoRows) {
		// No config persisted yet: return domain defaults. There is always
		// exactly one logical config.
		return TunnelConfig{Mode: DefaultMode, PromEndpoint: DefaultPromEndpoint}, nil
	}
	if err != nil {
		return TunnelConfig{}, fmt.Errorf("get config: %w", err)
	}
	cfg.Mode = Mode(modeRaw)
	return cfg, nil
}

func (s *sqliteRepository) Upsert(ctx context.Context, cfg TunnelConfig) (TunnelConfig, error) {
	if cfg.Mode == ModeUnspecified {
		cfg.Mode = DefaultMode
	}
	if cfg.PromEndpoint == "" {
		cfg.PromEndpoint = DefaultPromEndpoint
	}
	_, err := s.db.ExecContext(ctx, upsertConfigSQL,
		singletonID, string(cfg.Mode), cfg.TunnelID, cfg.AccountID, cfg.CredRef, cfg.PromEndpoint,
	)
	if err != nil {
		return TunnelConfig{}, fmt.Errorf("upsert config: %w", err)
	}
	return cfg, nil
}
