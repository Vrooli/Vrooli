package handoffs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type (
	SQLExecutor interface {
		ExecContext(context.Context, string, ...any) (sql.Result, error)
		QueryContext(context.Context, string, ...any) (*sql.Rows, error)
		QueryRowContext(context.Context, string, ...any) *sql.Row
	}
	sqliteRepository struct {
		db    SQLExecutor
		clock schedule.Clock
	}
)

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) Create(ctx context.Context, h Handoff) (Handoff, error) {
	h.ID = uuid.NewString()
	h.CreatedAt = r.clock.Now().UTC()
	h.UpdatedAt = h.CreatedAt
	if h.OpenedByRunID == "" {
		h.OpenedByRunID = "operator"
	}
	if h.AuthorisingHuman == "" {
		h.AuthorisingHuman = "operator"
	}
	data, err := json.Marshal(h.Checkpoint)
	if err != nil {
		return Handoff{}, err
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO persona_handoffs (id, persona_id, kind, title, human_action, checkpoint_json, state, opened_by_run_id, authorising_human, deadline, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, h.ID, h.PersonaID, h.Kind, h.Title, h.HumanAction, string(data), h.State, h.OpenedByRunID, h.AuthorisingHuman, h.Deadline.Format(time.RFC3339Nano), h.CreatedAt.Format(time.RFC3339Nano), h.UpdatedAt.Format(time.RFC3339Nano))
	if err != nil {
		return Handoff{}, fmt.Errorf("insert handoff: %w", err)
	}
	return h, nil
}

func (r *sqliteRepository) Get(ctx context.Context, id string) (Handoff, error) {
	h, err := scanHandoff(r.db.QueryRowContext(ctx, `SELECT id, persona_id, kind, title, human_action, checkpoint_json, state, opened_by_run_id, authorising_human, deadline, created_at, updated_at, relay_state FROM persona_handoffs WHERE id = ?`, id))
	if errors.Is(err, sql.ErrNoRows) {
		return Handoff{}, ErrNotFound
	}
	return h, err
}

func (r *sqliteRepository) List(ctx context.Context, personaID string, limit int) ([]Handoff, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, persona_id, kind, title, human_action, checkpoint_json, state, opened_by_run_id, authorising_human, deadline, created_at, updated_at, relay_state FROM persona_handoffs WHERE persona_id = ? ORDER BY updated_at DESC, id DESC LIMIT ?`, personaID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Handoff
	for rows.Next() {
		h, err := scanHandoff(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

func (r *sqliteRepository) UpdateState(ctx context.Context, id string, state State, actor, reason string) (Handoff, error) {
	h, err := r.Get(ctx, id)
	if err != nil {
		return Handoff{}, err
	}
	if !AllowedTransitions[h.State][state] {
		return Handoff{}, fmt.Errorf("%w: %s to %s", ErrInvalidTransition, h.State, state)
	}
	h.State = state
	h.UpdatedAt = r.clock.Now().UTC()
	_, err = r.db.ExecContext(ctx, `UPDATE persona_handoffs SET state = ?, updated_at = ? WHERE id = ?`, state, h.UpdatedAt.Format(time.RFC3339Nano), id)
	if err != nil {
		return Handoff{}, fmt.Errorf("update handoff state: %w", err)
	}
	return h, nil
}

func (r *sqliteRepository) SetRelayState(ctx context.Context, id, relayState string) (Handoff, error) {
	_, err := r.db.ExecContext(ctx, `UPDATE persona_handoffs SET relay_state = ?, updated_at = ? WHERE id = ?`, relayState, r.clock.Now().UTC().Format(time.RFC3339Nano), id)
	if err != nil {
		return Handoff{}, fmt.Errorf("update handoff relay state: %w", err)
	}
	return r.Get(ctx, id)
}

type rowScanner interface{ Scan(...any) error }

func scanHandoff(row rowScanner) (Handoff, error) {
	var h Handoff
	var checkpoint, state, deadline, created, updated string
	if err := row.Scan(&h.ID, &h.PersonaID, &h.Kind, &h.Title, &h.HumanAction, &checkpoint, &state, &h.OpenedByRunID, &h.AuthorisingHuman, &deadline, &created, &updated, &h.RelayState); err != nil {
		return Handoff{}, err
	}
	if err := json.Unmarshal([]byte(checkpoint), &h.Checkpoint); err != nil {
		return Handoff{}, fmt.Errorf("decode handoff checkpoint: %w", err)
	}
	h.State = State(state)
	var err error
	h.Deadline, err = time.Parse(time.RFC3339Nano, deadline)
	if err != nil {
		return Handoff{}, err
	}
	h.CreatedAt, err = time.Parse(time.RFC3339Nano, created)
	if err != nil {
		return Handoff{}, err
	}
	h.UpdatedAt, err = time.Parse(time.RFC3339Nano, updated)
	return h, err
}
