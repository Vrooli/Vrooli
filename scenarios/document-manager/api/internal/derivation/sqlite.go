package derivation

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"
)

type SQLiteStore struct{ db queryer }

type queryer interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func NewSQLiteStore(db queryer) *SQLiteStore { return &SQLiteStore{db: db} }

func (s *SQLiteStore) NextVersion(ctx context.Context, hash string) (int, error) {
	var version int
	if err := s.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(version), 0) + 1 FROM derivation_versions WHERE document_hash = ?`, hash).Scan(&version); err != nil {
		return 0, fmt.Errorf("next derivation version: %w", err)
	}
	return version, nil
}

func (s *SQLiteStore) Append(ctx context.Context, result Result) error {
	chain, err := json.Marshal(result.Chain)
	if err != nil {
		return err
	}
	handlers, err := json.Marshal(result.Handlers)
	if err != nil {
		return err
	}
	model, err := json.Marshal(result.Model)
	if err != nil {
		return err
	}
	skipped, err := json.Marshal(result.Skipped)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO derivation_versions (document_hash, version, tier, chain_json, handlers_json, model_json, terminal_state, reason, remedy, skipped_json, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, result.DocumentHash, result.Version, result.Tier, chain, handlers, model, result.State, result.Reason, result.Remedy, skipped, time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("append derivation version: %w", err)
	}
	return nil
}
