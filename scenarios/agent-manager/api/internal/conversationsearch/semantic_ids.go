package conversationsearch

import "context"

// SemanticDocumentIDs returns the serving catalog's document ids for a run, or
// for one event when eventID is set, that the semantic leg can have embedded.
// Only prose classes are embedded, so an incremental change must delete only
// these from the vector store. A shrinking tool payload drops chunk ids that
// no longer appear among the changed documents; counting those as semantic
// deletions made every compacted tool event copy the serving collection.
func (r *SQLiteRepository) SemanticDocumentIDs(ctx context.Context, runID, eventID string) ([]string, error) {
	query := `SELECT document_id FROM conversation_search_catalog
WHERE source_run_id = ? AND content_class IN (?, ?)`
	args := []any{runID, ContentClassProse, ContentClassQuotedProse}
	if eventID != "" {
		query += ` AND source_event_id = ?`
		args = append(args, eventID)
	}
	var ids []string
	err := r.db.SelectContext(ctx, &ids, query+` ORDER BY document_id`, args...)
	return ids, err
}
