package metadatastore

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"knowledge-observatory/internal/ports"
)

type Postgres struct {
	DB *sql.DB
}

func (p *Postgres) UpsertKnowledgeMetadata(ctx context.Context, vectorID, collectionName, contentHash, sourceScenario, sourceType string) error {
	if p == nil || p.DB == nil {
		return nil
	}
	vectorID = strings.TrimSpace(vectorID)
	if vectorID == "" {
		return fmt.Errorf("vectorID is required")
	}
	collectionName = strings.TrimSpace(collectionName)
	if collectionName == "" {
		return fmt.Errorf("collectionName is required")
	}
	sourceType = strings.TrimSpace(sourceType)
	if sourceType == "" {
		sourceType = "unknown"
	}

	_, err := p.DB.ExecContext(ctx, `
INSERT INTO knowledge_observatory.knowledge_metadata
  (vector_id, collection_name, content_hash, source_scenario, source_type, created_at, updated_at)
VALUES
  ($1, $2, $3, $4, $5, NOW(), NOW())
ON CONFLICT (vector_id) DO UPDATE
SET
  collection_name = EXCLUDED.collection_name,
  content_hash = EXCLUDED.content_hash,
  source_scenario = EXCLUDED.source_scenario,
  source_type = EXCLUDED.source_type,
  updated_at = NOW()
`, vectorID, collectionName, contentHash, sourceScenario, sourceType)
	if err != nil {
		return fmt.Errorf("metadata upsert failed: %w", err)
	}
	return nil
}

func (p *Postgres) InsertIngestHistory(ctx context.Context, row ports.IngestHistoryRow) error {
	if p == nil || p.DB == nil {
		return nil
	}
	_, err := p.DB.ExecContext(ctx, `
INSERT INTO knowledge_observatory.ingest_history
  (record_id, namespace, collection_name, content_hash, visibility, source, source_type, status, error_message, took_ms)
VALUES
  ($1, $2, $3, $4, $5, NULLIF($6, ''), NULLIF($7, ''), $8, NULLIF($9, ''), $10)
`, row.RecordID, row.Namespace, row.Collection, row.ContentHash, row.Visibility, row.Source, row.SourceType, row.Status, row.ErrorMessage, row.TookMS)
	if err != nil {
		return fmt.Errorf("ingest history insert failed: %w", err)
	}
	return nil
}

func (p *Postgres) LookupCollectionForVectorID(ctx context.Context, vectorID string) (string, bool, error) {
	if p == nil || p.DB == nil {
		return "", false, nil
	}
	vectorID = strings.TrimSpace(vectorID)
	if vectorID == "" {
		return "", false, fmt.Errorf("vectorID is required")
	}

	var collection string
	err := p.DB.QueryRowContext(ctx, `
SELECT collection_name
FROM knowledge_observatory.knowledge_metadata
WHERE vector_id = $1
`, vectorID).Scan(&collection)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("lookup failed: %w", err)
	}
	return collection, true, nil
}

