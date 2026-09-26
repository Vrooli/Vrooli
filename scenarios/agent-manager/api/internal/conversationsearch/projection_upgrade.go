package conversationsearch

import (
	"context"
	"database/sql"
	"fmt"
)

// Before 2026-09-14 the catalog lived in conversation_search_documents, keyed
// by an implicit rowid that a VACUUM may renumber, and fed a standalone FTS
// table that stored a second copy of every chunk. Schema application creates
// the current objects beside it without touching data; this upgrade moves the
// rows across and removes the legacy objects.
const (
	legacyCatalogTable  = "conversation_search_documents"
	legacyLexicalTable  = "conversation_search_fts"
	legacyCopyBatchSize = 1000
)

var legacyCatalogIndexes = []string{
	"idx_conversation_search_source",
	"idx_conversation_search_visibility_time",
	"idx_conversation_search_role_time",
	"idx_conversation_search_harness_time",
	"idx_conversation_search_project_time",
	"idx_conversation_search_model_time",
	"idx_conversation_search_profile_time",
	"idx_conversation_search_status_time",
	"idx_conversation_search_content_hash",
}

// UpgradeLegacyProjection moves a legacy projection onto the current catalog
// and reports whether one was found. It is the index owner's domain
// operation: the owner loop runs it before recovery admits any generation,
// never schema application at startup. Copying keeps every document identity,
// so the serving generation and its semantic vectors stay valid and no full
// rebuild is needed. It is resumable: copied documents are skipped by
// identity, and legacy objects are dropped only after the copy completes.
func (r *SQLiteRepository) UpgradeLegacyProjection(ctx context.Context) (bool, error) {
	var legacy bool
	if err := r.db.GetContext(ctx, &legacy, `SELECT EXISTS(SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?)`, legacyCatalogTable); err != nil {
		return false, fmt.Errorf("detect legacy conversation projection: %w", err)
	}
	if !legacy {
		return false, nil
	}
	if err := r.copyLegacyCatalog(ctx); err != nil {
		return true, err
	}
	if err := r.dropLegacyProjection(ctx); err != nil {
		return true, err
	}
	r.invalidateCoverage()
	return true, nil
}

// copyLegacyCatalog copies in bounded rowid pages, yielding SQLite's single
// writer between them. A document with a pending canonical deletion is not
// revived; generation publication applies the same guard.
func (r *SQLiteRepository) copyLegacyCatalog(ctx context.Context) error {
	var after int64
	for {
		var last sql.NullInt64
		if err := r.db.GetContext(ctx, &last, `SELECT MAX(rowid) FROM (SELECT rowid FROM `+legacyCatalogTable+`
WHERE rowid > ? ORDER BY rowid LIMIT ?)`, after, legacyCopyBatchSize); err != nil {
			return fmt.Errorf("page legacy conversation catalog: %w", err)
		}
		if !last.Valid {
			return nil
		}
		if _, err := r.db.ExecContext(ctx, `INSERT OR IGNORE INTO conversation_search_catalog (`+projectionDocumentColumns+`)
SELECT `+projectionDocumentColumns+` FROM `+legacyCatalogTable+` legacy
WHERE legacy.rowid > ? AND legacy.rowid <= ?
AND NOT EXISTS (SELECT 1 FROM conversation_search_catalog serving WHERE serving.document_id = legacy.document_id)
AND NOT EXISTS (
  SELECT 1 FROM conversation_search_changes c
  WHERE c.processed_at IS NULL
    AND ((c.operation = 'delete_run' AND c.source_run_id = legacy.source_run_id)
      OR (c.operation = 'delete_event' AND c.source_run_id = legacy.source_run_id AND c.source_event_id = legacy.source_event_id))
)
ORDER BY legacy.rowid`, after, last.Int64); err != nil {
			return fmt.Errorf("copy legacy conversation catalog through rowid %d: %w", last.Int64, err)
		}
		after = last.Int64
		if err := yieldGenerationWriter(ctx); err != nil {
			return err
		}
	}
}

// dropLegacyProjection removes legacy objects one statement at a time,
// yielding the writer between them. Dropping the catalog also drops the
// triggers that fed the legacy FTS table.
func (r *SQLiteRepository) dropLegacyProjection(ctx context.Context) error {
	statements := make([]string, 0, len(legacyCatalogIndexes)+2)
	for _, index := range legacyCatalogIndexes {
		statements = append(statements, `DROP INDEX IF EXISTS `+index)
	}
	statements = append(statements, `DROP TABLE IF EXISTS `+legacyCatalogTable, `DROP TABLE IF EXISTS `+legacyLexicalTable)
	for _, statement := range statements {
		if _, err := r.db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("drop legacy conversation projection (%s): %w", statement, err)
		}
		if err := yieldGenerationWriter(ctx); err != nil {
			return err
		}
	}
	return nil
}
