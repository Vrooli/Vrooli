package studio

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

// StateStore is the persistence seam for the policy aggregate. It allows the
// Connect adapter to remain an adapter while tests use the in-memory default.
type StateStore interface {
	Load(context.Context) (*Studio, error)
	Save(context.Context, *Studio) error
}

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type sqliteStore struct{ db SQLExecutor }

func NewSQLiteStore(db SQLExecutor) StateStore { return &sqliteStore{db: db} }

func (s *sqliteStore) Load(ctx context.Context) (*Studio, error) {
	var raw string
	err := s.db.QueryRowContext(ctx, `SELECT state_json FROM studio_runtime_state WHERE singleton = 1`).Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return New(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("load studio state: %w", err)
	}
	state := New()
	if err := json.Unmarshal([]byte(raw), state); err != nil {
		return nil, fmt.Errorf("decode studio state: %w", err)
	}
	if state.Identities == nil {
		state.Identities = map[string]Identity{}
	}
	if state.Specs == nil {
		state.Specs = map[string]Spec{}
	}
	if state.Renders == nil {
		state.Renders = map[string]*Render{}
	}
	if state.Assets == nil {
		state.Assets = map[string]*Asset{}
	}
	if state.ImportHashes == nil {
		state.ImportHashes = map[string]string{}
	}
	if state.CampaignBudgets == nil {
		state.CampaignBudgets = map[string]*CampaignBudget{}
	}
	return state, nil
}

func (s *sqliteStore) Save(ctx context.Context, state *Studio) error {
	raw, err := json.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode studio state: %w", err)
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO studio_runtime_state(singleton, state_json, updated_at) VALUES (1, ?, ?) ON CONFLICT(singleton) DO UPDATE SET state_json = excluded.state_json, updated_at = excluded.updated_at`, string(raw), time.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return fmt.Errorf("save studio state: %w", err)
	}
	return nil
}
