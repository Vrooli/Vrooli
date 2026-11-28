package channelmanager

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
)

// SQLExecutor matches RoutedDB and *sql.DB, preserving Test Genie's per-request
// database routing while keeping persistence independently testable.
type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

const schema = `CREATE TABLE IF NOT EXISTS channel_manager_state (id INTEGER PRIMARY KEY CHECK(id=1), state_json TEXT NOT NULL);`

func Schema() string { return schema }

type Store struct{ db SQLExecutor }

func NewStore(db SQLExecutor) Store { return Store{db: db} }
func (s Store) Load(ctx context.Context, service *Service) error {
	var raw string
	err := s.db.QueryRowContext(ctx, "SELECT state_json FROM channel_manager_state WHERE id=1").Scan(&raw)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("load channel-manager state: %w", err)
	}
	var state State
	if err := json.Unmarshal([]byte(raw), &state); err != nil {
		return fmt.Errorf("decode channel-manager state: %w", err)
	}
	service.Restore(state)
	return nil
}

func (s Store) Save(ctx context.Context, service *Service) error {
	raw, err := json.Marshal(service.State())
	if err != nil {
		return fmt.Errorf("encode channel-manager state: %w", err)
	}
	_, err = s.db.ExecContext(ctx, "INSERT INTO channel_manager_state(id,state_json) VALUES(1,?) ON CONFLICT(id) DO UPDATE SET state_json=excluded.state_json", string(raw))
	if err != nil {
		return fmt.Errorf("save channel-manager state: %w", err)
	}
	return nil
}
