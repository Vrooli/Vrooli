package store

import (
	"context"
	"database/sql"
	"errors"
	"time"
)

type SpeakerProfile struct {
	ID                string
	Name              string
	Embedding         []byte
	BoundUserIdentity string // empty when unbound
	CreatedAt         time.Time
}

type SpeakerStore struct{ db *sql.DB }

func NewSpeakerStore(db *sql.DB) *SpeakerStore { return &SpeakerStore{db: db} }

func (s *SpeakerStore) Upsert(ctx context.Context, p SpeakerProfile) error {
	if p.CreatedAt.IsZero() {
		p.CreatedAt = now()
	}
	var bound any
	if p.BoundUserIdentity != "" {
		bound = p.BoundUserIdentity
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO speaker_profiles(id, name, embedding, bound_user_identity, created_at)
		VALUES (?,?,?,?,?)
		ON CONFLICT(id) DO UPDATE SET
			name=excluded.name,
			embedding=excluded.embedding,
			bound_user_identity=excluded.bound_user_identity
	`, p.ID, p.Name, p.Embedding, bound, p.CreatedAt.UTC().Format(time.RFC3339))
	return err
}

func (s *SpeakerStore) Delete(ctx context.Context, id string) (bool, error) {
	res, err := s.db.ExecContext(ctx, `DELETE FROM speaker_profiles WHERE id=?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SpeakerStore) ClearBinding(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE speaker_profiles SET bound_user_identity=NULL WHERE id=?`, id)
	return err
}

func (s *SpeakerStore) Get(ctx context.Context, id string) (SpeakerProfile, bool, error) {
	row := s.db.QueryRowContext(ctx, `SELECT id, name, embedding, bound_user_identity, created_at FROM speaker_profiles WHERE id=?`, id)
	var p SpeakerProfile
	var bound sql.NullString
	var created string
	err := row.Scan(&p.ID, &p.Name, &p.Embedding, &bound, &created)
	if errors.Is(err, sql.ErrNoRows) {
		return SpeakerProfile{}, false, nil
	}
	if err != nil {
		return SpeakerProfile{}, false, err
	}
	if bound.Valid {
		p.BoundUserIdentity = bound.String
	}
	p.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return p, true, nil
}

func (s *SpeakerStore) List(ctx context.Context) ([]SpeakerProfile, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, embedding, bound_user_identity, created_at FROM speaker_profiles ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SpeakerProfile
	for rows.Next() {
		var p SpeakerProfile
		var bound sql.NullString
		var created string
		if err := rows.Scan(&p.ID, &p.Name, &p.Embedding, &bound, &created); err != nil {
			return nil, err
		}
		if bound.Valid {
			p.BoundUserIdentity = bound.String
		}
		p.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, p)
	}
	return out, rows.Err()
}
