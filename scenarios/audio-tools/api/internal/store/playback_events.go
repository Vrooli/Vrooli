package store

import (
	"context"
	"database/sql"
	"time"
)

type PlaybackEvent struct {
	EventID      string
	EmittedAt    time.Time
	Kind         string
	Version      int32
	Voice        string
	ProviderTier string
	ProviderID   string
}

type PlaybackStore struct{ db *sql.DB }

func NewPlaybackStore(db *sql.DB) *PlaybackStore { return &PlaybackStore{db: db} }

// Insert is idempotent on EventID; duplicate events from retried writes
// silently no-op.
func (s *PlaybackStore) Insert(ctx context.Context, e PlaybackEvent) error {
	if e.EmittedAt.IsZero() {
		e.EmittedAt = now()
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO playback_events(event_id, emitted_at, kind, version, voice, provider_tier, provider_id)
		VALUES (?,?,?,?,?,?,?)`,
		e.EventID, e.EmittedAt.UTC().Format(time.RFC3339Nano), e.Kind, e.Version,
		nullableString(e.Voice), nullableString(e.ProviderTier), nullableString(e.ProviderID),
	)
	return err
}

// List returns up to `limit` most-recent events.
func (s *PlaybackStore) List(ctx context.Context, limit int) ([]PlaybackEvent, error) {
	if limit <= 0 || limit > 1000 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT event_id, emitted_at, kind, version, voice, provider_tier, provider_id
		FROM playback_events ORDER BY emitted_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []PlaybackEvent
	for rows.Next() {
		var e PlaybackEvent
		var emitted string
		var voice, tier, prov sql.NullString
		if err := rows.Scan(&e.EventID, &emitted, &e.Kind, &e.Version, &voice, &tier, &prov); err != nil {
			return nil, err
		}
		e.EmittedAt, _ = time.Parse(time.RFC3339Nano, emitted)
		if voice.Valid {
			e.Voice = voice.String
		}
		if tier.Valid {
			e.ProviderTier = tier.String
		}
		if prov.Valid {
			e.ProviderID = prov.String
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}
