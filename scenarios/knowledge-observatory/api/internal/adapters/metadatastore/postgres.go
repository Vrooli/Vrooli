package metadatastore

import (
	"context"
	"database/sql"
	"fmt"
	"math"
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

func (p *Postgres) InsertSearchHistory(ctx context.Context, row ports.SearchHistoryRow) error {
	if p == nil || p.DB == nil {
		return nil
	}
	row.Query = strings.TrimSpace(row.Query)
	if row.Query == "" {
		return fmt.Errorf("query is required")
	}
	row.Collection = strings.TrimSpace(row.Collection)
	row.UserSession = strings.TrimSpace(row.UserSession)

	var avgScore interface{} = nil
	if row.AvgScore != nil && !math.IsNaN(*row.AvgScore) && !math.IsInf(*row.AvgScore, 0) {
		avgScore = *row.AvgScore
	}

	_, err := p.DB.ExecContext(ctx, `
INSERT INTO knowledge_observatory.search_history
  (query, collection, result_count, avg_score, response_time_ms, user_session, created_at)
VALUES
  ($1, NULLIF($2, ''), $3, $4, $5, NULLIF($6, ''), NOW())
`, row.Query, row.Collection, row.ResultCount, avgScore, row.ResponseTimeMS, row.UserSession)
	if err != nil {
		return fmt.Errorf("search history insert failed: %w", err)
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

func (p *Postgres) UpsertExternalIDMapping(ctx context.Context, mapping ports.ExternalIDMapping) error {
	if p == nil || p.DB == nil {
		return nil
	}
	mapping.Namespace = strings.TrimSpace(mapping.Namespace)
	mapping.ExternalID = strings.TrimSpace(mapping.ExternalID)
	mapping.Kind = strings.TrimSpace(mapping.Kind)
	mapping.RecordID = strings.TrimSpace(mapping.RecordID)
	mapping.DocumentID = strings.TrimSpace(mapping.DocumentID)
	mapping.ContentHash = strings.TrimSpace(mapping.ContentHash)

	if mapping.Namespace == "" || mapping.ExternalID == "" {
		return fmt.Errorf("namespace and external_id are required")
	}
	if mapping.Kind != "record" && mapping.Kind != "document" {
		return fmt.Errorf("kind must be one of: record, document")
	}
	if mapping.Kind == "record" && mapping.RecordID == "" {
		return fmt.Errorf("record_id is required for kind=record")
	}
	if mapping.Kind == "document" && mapping.DocumentID == "" {
		return fmt.Errorf("document_id is required for kind=document")
	}

	_, err := p.DB.ExecContext(ctx, `
INSERT INTO knowledge_observatory.external_id_map
  (namespace, external_id, kind, record_id, document_id, content_hash, created_at, updated_at)
VALUES
  ($1, $2, $3, NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, ''), NOW(), NOW())
ON CONFLICT (namespace, external_id, kind) DO UPDATE
SET
  record_id = COALESCE(EXCLUDED.record_id, knowledge_observatory.external_id_map.record_id),
  document_id = COALESCE(EXCLUDED.document_id, knowledge_observatory.external_id_map.document_id),
  content_hash = COALESCE(EXCLUDED.content_hash, knowledge_observatory.external_id_map.content_hash),
  updated_at = NOW()
`, mapping.Namespace, mapping.ExternalID, mapping.Kind, mapping.RecordID, mapping.DocumentID, mapping.ContentHash)
	if err != nil {
		return fmt.Errorf("external id upsert failed: %w", err)
	}
	return nil
}

func (p *Postgres) LookupExternalIDMapping(ctx context.Context, namespace, externalID, kind string) (ports.ExternalIDMapping, bool, error) {
	if p == nil || p.DB == nil {
		return ports.ExternalIDMapping{}, false, nil
	}
	namespace = strings.TrimSpace(namespace)
	externalID = strings.TrimSpace(externalID)
	kind = strings.TrimSpace(kind)
	if namespace == "" || externalID == "" || kind == "" {
		return ports.ExternalIDMapping{}, false, fmt.Errorf("namespace, externalID, and kind are required")
	}

	var row ports.ExternalIDMapping
	err := p.DB.QueryRowContext(ctx, `
SELECT namespace, external_id, kind, COALESCE(record_id, ''), COALESCE(document_id, ''), COALESCE(content_hash, '')
FROM knowledge_observatory.external_id_map
WHERE namespace = $1 AND external_id = $2 AND kind = $3
`, namespace, externalID, kind).Scan(&row.Namespace, &row.ExternalID, &row.Kind, &row.RecordID, &row.DocumentID, &row.ContentHash)
	if err == sql.ErrNoRows {
		return ports.ExternalIDMapping{}, false, nil
	}
	if err != nil {
		return ports.ExternalIDMapping{}, false, fmt.Errorf("external id lookup failed: %w", err)
	}
	return row, true, nil
}

func (p *Postgres) UpsertQualityMetrics(ctx context.Context, row ports.QualityMetricsRow) error {
	if p == nil || p.DB == nil {
		return nil
	}
	row.CollectionName = strings.TrimSpace(row.CollectionName)
	if row.CollectionName == "" {
		return fmt.Errorf("collection_name is required")
	}

	_, err := p.DB.ExecContext(ctx, `
INSERT INTO knowledge_observatory.quality_metrics
  (collection_name, coherence_score, freshness_score, redundancy_score, coverage_score, total_entries, measured_at, created_at, updated_at)
VALUES
  ($1, $2, $3, $4, $5, $6, NOW(), NOW(), NOW())
`, row.CollectionName, row.Coherence, row.Freshness, row.Redundancy, row.Coverage, row.TotalEntries)
	if err != nil {
		return fmt.Errorf("quality metrics upsert failed: %w", err)
	}
	return nil
}

func (p *Postgres) UpsertCollectionStats(ctx context.Context, row ports.CollectionStatsRow) error {
	if p == nil || p.DB == nil {
		return nil
	}
	row.CollectionName = strings.TrimSpace(row.CollectionName)
	if row.CollectionName == "" {
		return fmt.Errorf("collection_name is required")
	}

	_, err := p.DB.ExecContext(ctx, `
INSERT INTO knowledge_observatory.collection_stats
  (collection_name, total_entries, last_updated, created_at)
VALUES
  ($1, $2, NOW(), NOW())
ON CONFLICT (collection_name) DO UPDATE
SET total_entries = EXCLUDED.total_entries,
    last_updated = NOW()
`, row.CollectionName, row.TotalEntries)
	if err != nil {
		return fmt.Errorf("collection stats upsert failed: %w", err)
	}
	return nil
}

func (p *Postgres) UpsertRelationshipEdges(ctx context.Context, edges []ports.RelationshipEdgeRow) error {
	if p == nil || p.DB == nil {
		return nil
	}
	if len(edges) == 0 {
		return nil
	}

	tx, err := p.DB.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin tx failed: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
INSERT INTO knowledge_observatory.knowledge_relationships
  (source_id, target_id, relationship_type, weight, discovered_at)
VALUES
  ($1, $2, $3, $4, NOW())
ON CONFLICT (source_id, target_id, relationship_type) DO UPDATE
SET weight = EXCLUDED.weight,
    discovered_at = NOW()
`)
	if err != nil {
		return fmt.Errorf("prepare failed: %w", err)
	}
	defer stmt.Close()

	for _, e := range edges {
		e.SourceID = strings.TrimSpace(e.SourceID)
		e.TargetID = strings.TrimSpace(e.TargetID)
		e.RelationshipType = strings.TrimSpace(e.RelationshipType)
		if e.SourceID == "" || e.TargetID == "" || e.SourceID == e.TargetID {
			continue
		}
		if e.RelationshipType == "" {
			e.RelationshipType = "semantic_similarity"
		}
		if _, err := stmt.ExecContext(ctx, e.SourceID, e.TargetID, e.RelationshipType, e.Weight); err != nil {
			return fmt.Errorf("edge insert failed: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit failed: %w", err)
	}
	return nil
}
