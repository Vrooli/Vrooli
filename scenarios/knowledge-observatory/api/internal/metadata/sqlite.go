package metadata

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

func (s *SQLite) UpsertEntry(ctx context.Context, e Entry) error {
	if s == nil || s.DB == nil {
		return nil
	}
	e.VectorID = strings.TrimSpace(e.VectorID)
	if e.VectorID == "" {
		return fmt.Errorf("vector_id is required")
	}
	e.CollectionName = strings.TrimSpace(e.CollectionName)
	if e.CollectionName == "" {
		return fmt.Errorf("collection_name is required")
	}
	if strings.TrimSpace(e.SourceType) == "" {
		e.SourceType = "unknown"
	}
	if e.ID == "" {
		e.ID = uuid.NewString()
	}

	_, err := s.DB.ExecContext(ctx, `
INSERT INTO knowledge_metadata
  (id, vector_id, collection_name, content_hash, source_scenario, source_type,
   quality_score, access_count, last_accessed, created_at, updated_at)
VALUES (?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), ?, ?, ?, ?, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(vector_id) DO UPDATE SET
  collection_name = excluded.collection_name,
  content_hash = excluded.content_hash,
  source_scenario = excluded.source_scenario,
  source_type = excluded.source_type,
  quality_score = COALESCE(excluded.quality_score, knowledge_metadata.quality_score),
  updated_at = CURRENT_TIMESTAMP
`, e.ID, e.VectorID, e.CollectionName, strings.TrimSpace(e.ContentHash),
		strings.TrimSpace(e.SourceScenario), strings.TrimSpace(e.SourceType),
		e.QualityScore, e.AccessCount, sqlitetime.FormatPtr(e.LastAccessed))
	if err != nil {
		return fmt.Errorf("upsert knowledge metadata: %w", err)
	}
	return nil
}

func (s *SQLite) GetEntry(ctx context.Context, vectorID string) (Entry, bool, error) {
	if s == nil || s.DB == nil {
		return Entry{}, false, nil
	}
	vectorID = strings.TrimSpace(vectorID)
	if vectorID == "" {
		return Entry{}, false, fmt.Errorf("vector_id is required")
	}

	var (
		e            Entry
		lastAccessed sql.NullTime
	)
	err := s.DB.QueryRowContext(ctx, `
SELECT id, vector_id, collection_name, COALESCE(content_hash, ''), COALESCE(source_scenario, ''),
       COALESCE(source_type, ''), quality_score, access_count, last_accessed, created_at, updated_at
FROM knowledge_metadata
WHERE vector_id = ?
`, vectorID).Scan(&e.ID, &e.VectorID, &e.CollectionName, &e.ContentHash, &e.SourceScenario,
		&e.SourceType, &e.QualityScore, &e.AccessCount, &lastAccessed, &e.CreatedAt, &e.UpdatedAt)
	if err == sql.ErrNoRows {
		return Entry{}, false, nil
	}
	if err != nil {
		return Entry{}, false, fmt.Errorf("get knowledge metadata: %w", err)
	}
	if lastAccessed.Valid {
		t := lastAccessed.Time.UTC()
		e.LastAccessed = &t
	}
	return e, true, nil
}

func (s *SQLite) LookupCollectionForVectorID(ctx context.Context, vectorID string) (string, bool, error) {
	if s == nil || s.DB == nil {
		return "", false, nil
	}
	vectorID = strings.TrimSpace(vectorID)
	if vectorID == "" {
		return "", false, fmt.Errorf("vector_id is required")
	}

	var collection string
	err := s.DB.QueryRowContext(ctx,
		`SELECT collection_name FROM knowledge_metadata WHERE vector_id = ?`, vectorID).Scan(&collection)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("lookup collection for vector: %w", err)
	}
	return collection, true, nil
}

func (s *SQLite) CountByCollection(ctx context.Context, collection string) (int, error) {
	if s == nil || s.DB == nil {
		return 0, nil
	}
	var n int
	err := s.DB.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM knowledge_metadata WHERE collection_name = ?`,
		strings.TrimSpace(collection)).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("count knowledge metadata: %w", err)
	}
	return n, nil
}

func (s *SQLite) DeleteByCollection(ctx context.Context, collection string) (int64, error) {
	if s == nil || s.DB == nil {
		return 0, nil
	}
	res, err := s.DB.ExecContext(ctx,
		`DELETE FROM knowledge_metadata WHERE collection_name = ?`, strings.TrimSpace(collection))
	if err != nil {
		return 0, fmt.Errorf("delete knowledge metadata: %w", err)
	}
	return res.RowsAffected()
}

func (s *SQLite) UpsertExternalIDMapping(ctx context.Context, m ExternalIDMapping) error {
	if s == nil || s.DB == nil {
		return nil
	}
	m.Namespace = strings.TrimSpace(m.Namespace)
	m.ExternalID = strings.TrimSpace(m.ExternalID)
	m.Kind = strings.TrimSpace(m.Kind)
	m.RecordID = strings.TrimSpace(m.RecordID)
	m.DocumentID = strings.TrimSpace(m.DocumentID)
	m.ContentHash = strings.TrimSpace(m.ContentHash)

	if m.Namespace == "" || m.ExternalID == "" {
		return fmt.Errorf("namespace and external_id are required")
	}
	if m.Kind != "record" && m.Kind != "document" {
		return fmt.Errorf("kind must be one of: record, document")
	}
	if m.Kind == "record" && m.RecordID == "" {
		return fmt.Errorf("record_id is required for kind=record")
	}
	if m.Kind == "document" && m.DocumentID == "" {
		return fmt.Errorf("document_id is required for kind=document")
	}
	if m.ID == "" {
		m.ID = uuid.NewString()
	}

	_, err := s.DB.ExecContext(ctx, `
INSERT INTO external_id_map
  (id, namespace, external_id, kind, record_id, document_id, content_hash, created_at, updated_at)
VALUES (?, ?, ?, ?, NULLIF(?, ''), NULLIF(?, ''), NULLIF(?, ''), CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
ON CONFLICT(namespace, external_id, kind) DO UPDATE SET
  record_id = COALESCE(excluded.record_id, external_id_map.record_id),
  document_id = COALESCE(excluded.document_id, external_id_map.document_id),
  content_hash = COALESCE(excluded.content_hash, external_id_map.content_hash),
  updated_at = CURRENT_TIMESTAMP
`, m.ID, m.Namespace, m.ExternalID, m.Kind, m.RecordID, m.DocumentID, m.ContentHash)
	if err != nil {
		return fmt.Errorf("upsert external id mapping: %w", err)
	}
	return nil
}

func (s *SQLite) LookupExternalIDMapping(ctx context.Context, namespace, externalID, kind string) (ExternalIDMapping, bool, error) {
	if s == nil || s.DB == nil {
		return ExternalIDMapping{}, false, nil
	}
	namespace = strings.TrimSpace(namespace)
	externalID = strings.TrimSpace(externalID)
	kind = strings.TrimSpace(kind)
	if namespace == "" || externalID == "" || kind == "" {
		return ExternalIDMapping{}, false, fmt.Errorf("namespace, externalID, and kind are required")
	}

	var m ExternalIDMapping
	err := s.DB.QueryRowContext(ctx, `
SELECT id, namespace, external_id, kind, COALESCE(record_id, ''), COALESCE(document_id, ''),
       COALESCE(content_hash, ''), created_at, updated_at
FROM external_id_map
WHERE namespace = ? AND external_id = ? AND kind = ?
`, namespace, externalID, kind).Scan(&m.ID, &m.Namespace, &m.ExternalID, &m.Kind,
		&m.RecordID, &m.DocumentID, &m.ContentHash, &m.CreatedAt, &m.UpdatedAt)
	if err == sql.ErrNoRows {
		return ExternalIDMapping{}, false, nil
	}
	if err != nil {
		return ExternalIDMapping{}, false, fmt.Errorf("lookup external id mapping: %w", err)
	}
	return m, true, nil
}
