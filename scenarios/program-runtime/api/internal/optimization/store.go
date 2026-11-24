package optimization

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

//go:embed schema.sql
var schemaFS embed.FS

// SQLStore is the durable run journal. It depends only on database/sql so the
// scenario's storage owner can choose the configured local engine.
type SQLStore struct{ db *sql.DB }

// RFC3339Nano trims trailing fractional zeroes. SQLite compares the stored
// TEXT timestamps lexically, so mixing whole-second and fractional forms can
// put a later fractional timestamp before an earlier whole-second timestamp.
// Keep the journal's sort key at a fixed precision.
const journalTimestampLayout = "2006-01-02T15:04:05.000000000Z07:00"

func journalTimestamp(t time.Time) string {
	return t.UTC().Format(journalTimestampLayout)
}

func NewSQLStore(ctx context.Context, db *sql.DB) (*SQLStore, error) {
	if db == nil {
		return nil, fmt.Errorf("optimization database is required")
	}
	schema, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		return nil, err
	}
	if _, err := db.ExecContext(ctx, string(schema)); err != nil {
		return nil, fmt.Errorf("create optimization journal: %w", err)
	}
	return &SQLStore{db: db}, nil
}

func (s *SQLStore) Create(ctx context.Context, record Record) (Record, error) {
	if s == nil || s.db == nil {
		return Record{}, fmt.Errorf("optimization database is required")
	}
	if record.Revision != 0 {
		return Record{}, fmt.Errorf("new optimization record must start at revision 0")
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return Record{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO optimization_runs(run_id, revision, stage, record_json, updated_at) VALUES(?, ?, ?, ?, ?)`, record.Request.RunID, record.Revision, record.Stage, string(encoded), journalTimestamp(record.UpdatedAt))
	if err != nil {
		return Record{}, fmt.Errorf("create optimization run: %w", err)
	}
	return record, nil
}

func (s *SQLStore) Load(ctx context.Context, runID string) (Record, error) {
	if s == nil || s.db == nil {
		return Record{}, fmt.Errorf("optimization database is required")
	}
	var encoded string
	err := s.db.QueryRowContext(ctx, `SELECT record_json FROM optimization_runs WHERE run_id = ?`, runID).Scan(&encoded)
	if errors.Is(err, sql.ErrNoRows) {
		return Record{}, ErrNotFound
	}
	if err != nil {
		return Record{}, fmt.Errorf("load optimization run: %w", err)
	}
	var record Record
	if err := json.Unmarshal([]byte(encoded), &record); err != nil {
		return Record{}, fmt.Errorf("decode optimization run: %w", err)
	}
	return record, nil
}

func (s *SQLStore) CompareAndSwap(ctx context.Context, expectedRevision uint64, record Record) (Record, error) {
	if s == nil || s.db == nil {
		return Record{}, fmt.Errorf("optimization database is required")
	}
	if record.Revision != expectedRevision+1 {
		return Record{}, fmt.Errorf("optimization revision must advance exactly once")
	}
	encoded, err := json.Marshal(record)
	if err != nil {
		return Record{}, err
	}
	result, err := s.db.ExecContext(ctx, `UPDATE optimization_runs SET revision=?, stage=?, record_json=?, updated_at=? WHERE run_id=? AND revision=?`, record.Revision, record.Stage, string(encoded), journalTimestamp(record.UpdatedAt), record.Request.RunID, expectedRevision)
	if err != nil {
		return Record{}, fmt.Errorf("advance optimization run: %w", err)
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return Record{}, err
	}
	if updated != 1 {
		return Record{}, ErrRevision
	}
	return record, nil
}

func (s *SQLStore) ListActive(ctx context.Context, limit int) ([]Record, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("optimization database is required")
	}
	if limit <= 0 || limit > 1001 {
		limit = 100
	}
	rows, err := s.db.QueryContext(ctx, `SELECT record_json FROM optimization_runs WHERE stage NOT IN (?, ?) ORDER BY updated_at, run_id LIMIT ?`, StageComplete, StageRefused, limit)
	if err != nil {
		return nil, fmt.Errorf("list active optimization runs: %w", err)
	}
	defer rows.Close()
	out := make([]Record, 0)
	for rows.Next() {
		var encoded string
		if err := rows.Scan(&encoded); err != nil {
			return nil, err
		}
		var record Record
		if err := json.Unmarshal([]byte(encoded), &record); err != nil {
			return nil, fmt.Errorf("decode active optimization run: %w", err)
		}
		out = append(out, record)
	}
	return out, rows.Err()
}
