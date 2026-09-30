package ingest

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/google/uuid"
	apidb "github.com/vrooli/api-core/database"

	"knowledge-observatory/internal/sqlitetime"
)

// SQLite implements Repository against SQLite.
type SQLite struct {
	DB *apidb.RoutedDB
}

var _ Repository = (*SQLite)(nil)

// NewSQLite returns a Repository backed by db.
func NewSQLite(db *apidb.RoutedDB) *SQLite { return &SQLite{DB: db} }

func (s *SQLite) InsertHistory(ctx context.Context, h HistoryEntry) (string, error) {
	if s == nil || s.DB == nil {
		return "", nil
	}
	if h.ID == "" {
		h.ID = uuid.NewString()
	}
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO ingest_history
  (id, record_id, namespace, collection_name, content_hash, visibility, source,
   source_type, status, error_message, took_ms, created_at)
VALUES (?, ?, ?, ?, NULLIF(?, ''), ?, NULLIF(?, ''), NULLIF(?, ''), ?, NULLIF(?, ''), ?, CURRENT_TIMESTAMP)
ON CONFLICT(id) DO NOTHING
`, h.ID, h.RecordID, h.Namespace, h.CollectionName, h.ContentHash, h.Visibility,
		h.Source, h.SourceType, h.Status, h.ErrorMessage, h.TookMS)
	if err != nil {
		return "", fmt.Errorf("insert ingest history: %w", err)
	}
	return h.ID, nil
}

func (s *SQLite) GetHistory(ctx context.Context, id string) (HistoryEntry, bool, error) {
	if s == nil || s.DB == nil {
		return HistoryEntry{}, false, nil
	}
	var h HistoryEntry
	err := s.DB.QueryRowContext(ctx, `
SELECT id, record_id, namespace, collection_name, COALESCE(content_hash, ''), visibility,
       COALESCE(source, ''), COALESCE(source_type, ''), status, COALESCE(error_message, ''),
       COALESCE(took_ms, 0), created_at
FROM ingest_history
WHERE id = ?
`, strings.TrimSpace(id)).Scan(&h.ID, &h.RecordID, &h.Namespace, &h.CollectionName, &h.ContentHash,
		&h.Visibility, &h.Source, &h.SourceType, &h.Status, &h.ErrorMessage, &h.TookMS, &h.CreatedAt)
	if err == sql.ErrNoRows {
		return HistoryEntry{}, false, nil
	}
	if err != nil {
		return HistoryEntry{}, false, fmt.Errorf("get ingest history: %w", err)
	}
	return h, true, nil
}

// ProvenanceForCollection answers "what does ingest history know about this
// collection". It replaces the raw aggregate that used to sit in the collections
// handler.
func (s *SQLite) ProvenanceForCollection(ctx context.Context, collection string) (Provenance, error) {
	if s == nil || s.DB == nil {
		return Provenance{}, nil
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return Provenance{}, nil
	}

	var (
		out Provenance
		// MAX(created_at) is an expression, not a column, so the driver loses
		// the TIMESTAMP affinity and hands back text. Read it as text and parse.
		last sql.NullString
	)
	err := s.DB.QueryRowContext(ctx, `
SELECT COUNT(*), COUNT(DISTINCT namespace), MAX(created_at)
FROM ingest_history
WHERE collection_name = ?
`, collection).Scan(&out.IngestAttempts, &out.DistinctNamespaces, &last)
	if err != nil {
		return Provenance{}, fmt.Errorf("collection provenance: %w", err)
	}
	out.LastIngestAt = sqlitetime.ParseNull(last)
	return out, nil
}

// HealthForCollection tallies ingest outcomes for one collection.
//
// The 24-hour window used `NOW() - INTERVAL '24 hours'` on Postgres; SQLite
// spells the same thing as datetime('now', '-24 hours'). Both are evaluated in
// UTC, and created_at is stored in UTC, so the comparison is unchanged.
func (s *SQLite) HealthForCollection(ctx context.Context, collection string) (Health, error) {
	if s == nil || s.DB == nil {
		return Health{}, nil
	}
	collection = strings.TrimSpace(collection)
	if collection == "" {
		return Health{}, nil
	}

	var (
		out Health
		// An aggregate over a TIMESTAMP column comes back as text; see
		// ProvenanceForCollection.
		lastFailure sql.NullString
	)
	err := s.DB.QueryRowContext(ctx, `
SELECT
    COUNT(*),
    COUNT(*) FILTER (WHERE status = 'success'),
    COUNT(*) FILTER (WHERE status = 'failure'),
    COUNT(*) FILTER (WHERE status = 'failure' AND created_at >= datetime('now', '-24 hours')),
    MAX(created_at) FILTER (WHERE status = 'failure')
FROM ingest_history
WHERE collection_name = ?
`, collection).Scan(&out.TotalAttempts, &out.SuccessCount, &out.FailureCount,
		&out.FailureCountLast24H, &lastFailure)
	if err != nil {
		return Health{}, fmt.Errorf("collection ingest health: %w", err)
	}
	out.LastFailureAt = sqlitetime.ParseNull(lastFailure)
	return out, nil
}

func (s *SQLite) DeleteHistoryByCollection(ctx context.Context, collection string) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, nil
	}
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM ingest_history WHERE collection_name = ?`, strings.TrimSpace(collection))
	if err != nil {
		return 0, fmt.Errorf("delete ingest history: %w", err)
	}
	return res.RowsAffected()
}

func (s *SQLite) UpsertJob(ctx context.Context, j Job) (string, error) {
	if s == nil || s.DB == nil {
		return "", nil
	}
	if j.ID == "" {
		j.ID = uuid.NewString()
	}
	if strings.TrimSpace(j.RequestJSON) == "" {
		j.RequestJSON = "{}"
	}
	if strings.TrimSpace(j.Status) == "" {
		j.Status = "pending"
	}
	_, err := s.DB.ExecContext(ctx, `
INSERT INTO ingest_jobs
  (id, request_json, status, error_message, created_at, started_at, finished_at,
   total_chunks, completed_chunks)
VALUES (?, ?, ?, NULLIF(?, ''), CURRENT_TIMESTAMP, ?, ?, ?, ?)
ON CONFLICT(id) DO UPDATE SET
  status = excluded.status,
  error_message = excluded.error_message,
  started_at = COALESCE(excluded.started_at, ingest_jobs.started_at),
  finished_at = COALESCE(excluded.finished_at, ingest_jobs.finished_at),
  total_chunks = excluded.total_chunks,
  completed_chunks = excluded.completed_chunks
`, j.ID, j.RequestJSON, j.Status, j.ErrorMessage, sqlitetime.FormatPtr(j.StartedAt), sqlitetime.FormatPtr(j.FinishedAt),
		j.TotalChunks, j.CompletedChunks)
	if err != nil {
		return "", fmt.Errorf("upsert ingest job: %w", err)
	}
	return j.ID, nil
}

func (s *SQLite) GetJob(ctx context.Context, id string) (Job, bool, error) {
	if s == nil || s.DB == nil {
		return Job{}, false, nil
	}
	var (
		j                 Job
		started, finished sql.NullTime
	)
	err := s.DB.QueryRowContext(ctx, `
SELECT id, request_json, status, COALESCE(error_message, ''), created_at,
       started_at, finished_at, total_chunks, completed_chunks
FROM ingest_jobs
WHERE id = ?
`, strings.TrimSpace(id)).Scan(&j.ID, &j.RequestJSON, &j.Status, &j.ErrorMessage, &j.CreatedAt,
		&started, &finished, &j.TotalChunks, &j.CompletedChunks)
	if err == sql.ErrNoRows {
		return Job{}, false, nil
	}
	if err != nil {
		return Job{}, false, fmt.Errorf("get ingest job: %w", err)
	}
	if started.Valid {
		t := started.Time.UTC()
		j.StartedAt = &t
	}
	if finished.Valid {
		t := finished.Time.UTC()
		j.FinishedAt = &t
	}
	return j, true, nil
}
