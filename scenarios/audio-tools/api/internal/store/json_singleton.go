package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// jsonSingleton implements a one-row table that stores arbitrary
// opaque JSON payloads keyed by id=1. Both stt_stream_config and
// tts_config use this shape; centralising the implementation avoids
// per-call duplication while keeping each public store narrowly typed
// at the boundary.
type jsonSingleton struct {
	db    *sql.DB
	table string
	cols  []string // mirror config_json / summarize_json column order
}

func (j *jsonSingleton) get(ctx context.Context) ([]string, bool, error) {
	q := fmt.Sprintf("SELECT %s FROM %s WHERE id=1", joinCols(j.cols), j.table)
	row := j.db.QueryRowContext(ctx, q)
	dest := make([]any, len(j.cols))
	out := make([]string, len(j.cols))
	for i := range dest {
		dest[i] = &out[i]
	}
	err := row.Scan(dest...)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}
	return out, true, nil
}

func (j *jsonSingleton) set(ctx context.Context, values []string) error {
	if len(values) != len(j.cols) {
		return fmt.Errorf("expected %d values, got %d", len(j.cols), len(values))
	}
	cols := joinCols(j.cols)
	placeholders := "?"
	updates := j.cols[0] + "=excluded." + j.cols[0]
	for i := 1; i < len(j.cols); i++ {
		placeholders += ",?"
		updates += "," + j.cols[i] + "=excluded." + j.cols[i]
	}
	q := fmt.Sprintf(`INSERT INTO %s(id, %s, updated_at) VALUES (1, %s, ?)
		ON CONFLICT(id) DO UPDATE SET %s, updated_at=excluded.updated_at`,
		j.table, cols, placeholders, updates)
	args := make([]any, 0, len(values)+1)
	for _, v := range values {
		args = append(args, v)
	}
	args = append(args, now().Format(time.RFC3339))
	_, err := j.db.ExecContext(ctx, q, args...)
	return err
}

func joinCols(cols []string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		out += c
	}
	return out
}

// STTStreamConfigStore persists a single JSON-encoded stream config.
type STTStreamConfigStore struct{ s jsonSingleton }

func NewSTTStreamConfigStore(db *sql.DB) *STTStreamConfigStore {
	return &STTStreamConfigStore{s: jsonSingleton{db: db, table: "stt_stream_config", cols: []string{"config_json"}}}
}

func (s *STTStreamConfigStore) Get(ctx context.Context) (string, bool, error) {
	v, ok, err := s.s.get(ctx)
	if !ok || err != nil {
		return "", false, err
	}
	return v[0], true, nil
}

func (s *STTStreamConfigStore) Set(ctx context.Context, configJSON string) error {
	return s.s.set(ctx, []string{configJSON})
}

// STTSpeakerConfigStore persists a single JSON-encoded speaker-verification
// config (mode/threshold/profile bindings). Same shape as the stream config
// store; the enrolled profiles themselves live in the speaker_profiles table.
type STTSpeakerConfigStore struct{ s jsonSingleton }

func NewSTTSpeakerConfigStore(db *sql.DB) *STTSpeakerConfigStore {
	return &STTSpeakerConfigStore{s: jsonSingleton{db: db, table: "stt_speaker_config", cols: []string{"config_json"}}}
}

func (s *STTSpeakerConfigStore) Get(ctx context.Context) (string, bool, error) {
	v, ok, err := s.s.get(ctx)
	if !ok || err != nil {
		return "", false, err
	}
	return v[0], true, nil
}

func (s *STTSpeakerConfigStore) Set(ctx context.Context, configJSON string) error {
	return s.s.set(ctx, []string{configJSON})
}

// TTSConfigStore persists both the TTS config and the summarize config
// as JSON in the same row (mirrors the in-proc Config + SummarizeConfig
// pair used by internal/tts).
type TTSConfigStore struct{ s jsonSingleton }

func NewTTSConfigStore(db *sql.DB) *TTSConfigStore {
	return &TTSConfigStore{s: jsonSingleton{db: db, table: "tts_config", cols: []string{"config_json", "summarize_json"}}}
}

func (s *TTSConfigStore) Get(ctx context.Context) (configJSON, summarizeJSON string, ok bool, err error) {
	v, ok, err := s.s.get(ctx)
	if !ok || err != nil {
		return "", "", false, err
	}
	return v[0], v[1], true, nil
}

func (s *TTSConfigStore) Set(ctx context.Context, configJSON, summarizeJSON string) error {
	return s.s.set(ctx, []string{configJSON, summarizeJSON})
}
