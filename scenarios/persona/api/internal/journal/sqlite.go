package journal

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/schedule"
)

type SQLExecutor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

type sqliteRepository struct {
	db    SQLExecutor
	clock schedule.Clock
}

func NewSQLiteRepository(db SQLExecutor, clock schedule.Clock) Repository {
	return &sqliteRepository{db: db, clock: clock}
}

var _ Repository = (*sqliteRepository)(nil)

func (r *sqliteRepository) Append(ctx context.Context, entry Entry) (Entry, error) {
	entry.ID = uuid.NewString()
	if entry.At.IsZero() {
		entry.At = r.clock.Now().UTC()
	}
	if entry.Details == nil {
		entry.Details = map[string]string{}
	}
	details, err := json.Marshal(entry.Details)
	if err != nil {
		return Entry{}, fmt.Errorf("marshal journal details: %w", err)
	}
	_, err = r.db.ExecContext(ctx, `INSERT INTO persona_journal (id, persona_id, actor, verb, run_id, authorising_human, at, outcome, constraint_name, details_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, entry.ID, entry.PersonaID, entry.Actor, entry.Verb, entry.RunID, entry.AuthorisingHuman, entry.At.UTC().Format(time.RFC3339Nano), entry.Outcome, entry.Constraint, string(details))
	if err != nil {
		return Entry{}, fmt.Errorf("append journal entry: %w", err)
	}
	return entry, nil
}

func (r *sqliteRepository) List(ctx context.Context, personaID string, limit int) ([]Entry, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id, persona_id, actor, verb, run_id, authorising_human, at, outcome, constraint_name, details_json FROM persona_journal WHERE persona_id = ? ORDER BY at DESC, id DESC LIMIT ?`, personaID, limit)
	if err != nil {
		return nil, fmt.Errorf("list journal entries: %w", err)
	}
	defer rows.Close()
	var out []Entry
	for rows.Next() {
		var e Entry
		var at, details string
		if err := rows.Scan(&e.ID, &e.PersonaID, &e.Actor, &e.Verb, &e.RunID, &e.AuthorisingHuman, &at, &e.Outcome, &e.Constraint, &details); err != nil {
			return nil, fmt.Errorf("scan journal entry: %w", err)
		}
		e.At, err = time.Parse(time.RFC3339Nano, at)
		if err != nil {
			return nil, fmt.Errorf("parse journal timestamp: %w", err)
		}
		if err := json.Unmarshal([]byte(details), &e.Details); err != nil {
			return nil, fmt.Errorf("decode journal details: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
