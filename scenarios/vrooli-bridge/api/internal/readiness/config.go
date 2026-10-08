// Package readiness owns the durable Bridge host endpoint selection. It is
// deliberately separate from onboarding: one host configuration supplies many
// onboarding attempts, while every attempt still records its immutable choice.
package readiness

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"vrooli-bridge/internal/onboard"
)

const schema = `
CREATE TABLE IF NOT EXISTS bridge_endpoint_config (
  singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
  endpoint TEXT NOT NULL DEFAULT '',
  reachability_mode TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL DEFAULT ''
);`

// Schema contributes the singleton host configuration table.
func Schema() string { return schema }

// SQLExecutor is satisfied by both *sql.DB and api-core's routed database.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

// Endpoint is the canonical endpoint decision presented to an owner and used
// as the default for future onboarding. Source is never user-provided.
type Endpoint struct {
	URL    string
	Mode   string
	Source string
}

// Store persists an owner-approved endpoint while retaining a lifecycle-safe
// fallback derived at process start when no explicit configuration exists.
type Store struct {
	db       SQLExecutor
	fallback Endpoint
}

func NewStore(db SQLExecutor, fallback Endpoint) *Store {
	return &Store{db: db, fallback: fallback}
}

// Resolve returns the saved configuration when present, otherwise the known
// configured/tunnel/derived fallback. It validates persisted data defensively
// so a corrupt database cannot silently make onboarding unsafe.
func (s *Store) Resolve(ctx context.Context) (Endpoint, error) {
	if s == nil {
		return Endpoint{}, fmt.Errorf("bridge endpoint store is not configured")
	}
	if s.db == nil {
		return s.fallback, nil
	}
	var endpoint, mode string
	err := s.db.QueryRowContext(ctx, `SELECT endpoint, reachability_mode FROM bridge_endpoint_config WHERE singleton = 1`).Scan(&endpoint, &mode)
	if err == sql.ErrNoRows {
		return s.fallback, nil
	}
	if err != nil {
		return Endpoint{}, fmt.Errorf("read bridge endpoint configuration: %w", err)
	}
	modeValue, err := onboard.NormalizeReachabilityMode(mode)
	if err != nil {
		return Endpoint{}, fmt.Errorf("stored bridge endpoint configuration has invalid mode: %w", err)
	}
	mode = string(modeValue)
	endpoint, err = onboard.ValidateControlPlaneURL(strings.TrimSpace(endpoint), modeValue)
	if err != nil {
		return Endpoint{}, fmt.Errorf("stored bridge endpoint configuration is invalid: %w", err)
	}
	return Endpoint{URL: endpoint, Mode: mode, Source: "configured"}, nil
}

// Save validates before writing and is an atomic singleton upsert. It stores no
// credentials because endpoint validation rejects URL userinfo.
func (s *Store) Save(ctx context.Context, endpoint, mode string) (Endpoint, error) {
	modeValue, err := onboard.NormalizeReachabilityMode(mode)
	if err != nil {
		return Endpoint{}, err
	}
	endpoint, err = onboard.ValidateControlPlaneURL(strings.TrimSpace(endpoint), modeValue)
	if err != nil {
		return Endpoint{}, err
	}
	if _, err := s.db.ExecContext(ctx, `
INSERT INTO bridge_endpoint_config (singleton, endpoint, reachability_mode, updated_at)
VALUES (1, ?, ?, strftime('%Y-%m-%dT%H:%M:%fZ', 'now'))
ON CONFLICT(singleton) DO UPDATE SET endpoint = excluded.endpoint, reachability_mode = excluded.reachability_mode, updated_at = excluded.updated_at`, endpoint, string(modeValue)); err != nil {
		return Endpoint{}, fmt.Errorf("save bridge endpoint configuration: %w", err)
	}
	return Endpoint{URL: endpoint, Mode: string(modeValue), Source: "configured"}, nil
}
