package agents

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type Record struct {
	ID string
	Binding
	CreatedAt string
}

type Store struct{ db SQLExecutor }

func NewStore(db SQLExecutor) *Store { return &Store{db: db} }

func (s *Store) List(ctx context.Context, agentID string) ([]Record, error) {
	query := `SELECT id,agent_id,channel_id,address,thread_key,created_at FROM switchboard_bindings`
	args := []any{}
	if strings.TrimSpace(agentID) != "" {
		query += ` WHERE agent_id=?`
		args = append(args, agentID)
	}
	query += ` ORDER BY created_at,id`
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list bindings: %w", err)
	}
	defer rows.Close()
	out := make([]Record, 0)
	for rows.Next() {
		var record Record
		if err := rows.Scan(&record.ID, &record.AgentID, &record.ChannelID, &record.Address, &record.ThreadKey, &record.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan binding: %w", err)
		}
		out = append(out, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read bindings: %w", err)
	}
	return out, nil
}

func (s *Store) Create(ctx context.Context, binding Binding) (Record, error) {
	binding.AgentID = strings.TrimSpace(binding.AgentID)
	binding.ChannelID = strings.TrimSpace(binding.ChannelID)
	binding.Address = strings.TrimSpace(binding.Address)
	binding.ThreadKey = strings.TrimSpace(binding.ThreadKey)
	if binding.AgentID == "" || binding.ChannelID == "" || binding.Address == "" {
		return Record{}, fmt.Errorf("agent_id, channel_id, and address are required")
	}
	record := Record{ID: uuid.NewString(), Binding: binding, CreatedAt: time.Now().UTC().Format(time.RFC3339Nano)}
	_, err := s.db.ExecContext(ctx, `INSERT INTO switchboard_bindings(id,agent_id,channel_id,address,thread_key,created_at) VALUES(?,?,?,?,?,?)`, record.ID, record.AgentID, record.ChannelID, record.Address, record.ThreadKey, record.CreatedAt)
	if err != nil {
		return Record{}, fmt.Errorf("create binding: %w", err)
	}
	return record, nil
}
