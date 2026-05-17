package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type WakeWordTemplate struct {
	ID        string
	Phrase    string
	Embedding []byte
	CreatedAt time.Time
}

type WakeWordStore struct{ db *sql.DB }

func NewWakeWordStore(db *sql.DB) *WakeWordStore { return &WakeWordStore{db: db} }

func (s *WakeWordStore) Upsert(ctx context.Context, t WakeWordTemplate) error {
	if t.CreatedAt.IsZero() {
		t.CreatedAt = now()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO wakeword_templates(id, phrase, embedding, created_at)
		VALUES (?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET phrase=excluded.phrase, embedding=excluded.embedding
	`, t.ID, t.Phrase, t.Embedding, t.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *WakeWordStore) Delete(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM wakeword_templates WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *WakeWordStore) Get(ctx context.Context, id string) (WakeWordTemplate, bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, phrase, embedding, created_at FROM wakeword_templates WHERE id=?`, id)
	var t WakeWordTemplate
	var created string
	err := row.Scan(&t.ID, &t.Phrase, &t.Embedding, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return WakeWordTemplate{}, false, nil
	}
	if err != nil {
		return WakeWordTemplate{}, false, err
	}
	t.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return t, true, nil
}

func (s *WakeWordStore) List(ctx context.Context) ([]WakeWordTemplate, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, phrase, embedding, created_at FROM wakeword_templates ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WakeWordTemplate
	for rows.Next() {
		var t WakeWordTemplate
		var created string
		if err := rows.Scan(&t.ID, &t.Phrase, &t.Embedding, &created); err != nil {
			return nil, err
		}
		t.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, t)
	}
	return out, rows.Err()
}
