package aisearch

import (
	"context"
	"fmt"
	"log/slog"

	"swarm-manager/internal/records"
)

// recordPointID returns the deterministic UUIDv5 Qdrant point ID for a record.
// The namespace prefix "swarm-manager:record/" is distinct from backlog/
// initiative prefixes so collisions across collections are impossible by
// construction; see vectorstore.go:qdrantNamespace.
func recordPointID(id string) string {
	if id == "" {
		id = "unknown"
	}
	return uuidV5(qdrantNamespace, "swarm-manager:record/"+id)
}

// composeRecordText builds the embedding input text for a record. The
// narrative fields (trigger / approach / ruled_out) dominate; metadata is
// folded in so semantic queries can be steered by domain/kind/outcome.
func composeRecordText(r records.Record) string {
	return r.EmbeddingText()
}

// buildRecordPayload returns the Qdrant payload map for a record. Keys here
// must stay in sync with RecordPayload in models.go.
//
// payloadHash, if non-empty, is added under "payload_hash"; same skip-if-
// unchanged convention as buildBacklogPayload.
func buildRecordPayload(r records.Record, payloadHash string) map[string]interface{} {
	files := r.FilesChanged
	if files == nil {
		files = []string{}
	}
	out := map[string]interface{}{
		"entity_type":   "record",
		"record_id":     r.ID,
		"kind":          string(r.Kind),
		"scenario":      r.Scenario,
		"backlog_ref":   r.BacklogRef,
		"initiative_id": r.InitiativeID,
		"supersedes":    r.Supersedes,
		"superseded_by": r.SupersededBy,
		"outcome":       string(r.Outcome),
		"commit":        r.Commit,
		"files_changed": files,
		"stub":          r.Stub,
		"title":         r.Trigger,
	}
	if payloadHash != "" {
		out["payload_hash"] = payloadHash
	}
	return out
}

// IndexRecord embeds and upserts a record's vector. Stub records (those with
// empty narrative) are skipped — an empty embedding would pollute results and
// stubs are filterable via the store's IncludeStubs flag anyway. Safe to call
// from write-through hooks; the records.Service does this synchronously today.
func (s *Service) IndexRecord(ctx context.Context, r records.Record) error {
	if s.embedder == nil || s.recordStore == nil {
		return fmt.Errorf("aisearch not configured for record indexing")
	}
	if r.Stub {
		// Indexing an empty-narrative stub would write a zero-content vector;
		// the contract is "only filled records are searchable."
		slog.Debug("[aisearch] skip indexing stub record", "id", r.ID)
		return nil
	}

	text := composeRecordText(r)
	if text == "" {
		return fmt.Errorf("record %s has no embedding text", r.ID)
	}

	payload := buildRecordPayload(r, "")
	payload["payload_hash"] = composePayloadHash(text, payload)

	vector, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embed record %s: %w", r.ID, err)
	}

	id := recordPointID(r.ID)
	// Records have NO reconciler coverage — the aisearch Reconciler ensures only
	// the backlog + initiative collections (reconciler.go), so this write-through
	// is the sole path that populates the records collection. Ensure it exists
	// before the upsert; otherwise the first record upsert hits a non-existent
	// collection (404) and, because records.Service swallows index errors, records
	// search silently 500s forever (the collection is never born). EnsureCollection
	// is idempotent and cheap — a GET when the collection already exists.
	if err := s.recordStore.EnsureCollection(ctx); err != nil {
		return fmt.Errorf("ensure records collection for %s: %w", r.ID, err)
	}
	if err := s.recordStore.Upsert(ctx, id, vector, payload); err != nil {
		return fmt.Errorf("upsert record %s: %w", r.ID, err)
	}

	slog.Debug("[aisearch] indexed record", "id", r.ID, "point", id, "kind", r.Kind, "scenario", r.Scenario)
	return nil
}

// DeleteRecord removes the vector for a record identified by record id.
func (s *Service) DeleteRecord(ctx context.Context, id string) error {
	if s.recordStore == nil {
		return fmt.Errorf("aisearch not configured for record indexing")
	}
	pointID := recordPointID(id)
	if err := s.recordStore.Delete(ctx, pointID); err != nil {
		return fmt.Errorf("delete record %s: %w", id, err)
	}
	slog.Debug("[aisearch] deleted record from index", "id", id, "point", pointID)
	return nil
}

// RecordIndexerAdapter implements records.Indexer by delegating to the
// aisearch.Service. Production wiring constructs one of these and hands it to
// records.NewService; tests substitute a fake records.Indexer instead.
type RecordIndexerAdapter struct {
	svc *Service
}

// NewRecordIndexerAdapter wraps an aisearch.Service for use as a records.Indexer.
func NewRecordIndexerAdapter(svc *Service) *RecordIndexerAdapter {
	return &RecordIndexerAdapter{svc: svc}
}

// IndexRecord satisfies records.Indexer.
func (a *RecordIndexerAdapter) IndexRecord(ctx context.Context, r records.Record) error {
	if a == nil || a.svc == nil {
		return nil
	}
	return a.svc.IndexRecord(ctx, r)
}

// SearchRecordHit is the typed shape returned by SearchRecords. Distinct from
// AISearchResult because the records handler wants the rehydrated Record, not
// a payload-map.
type SearchRecordHit struct {
	Record records.Record
	Score  float64
}

// SearchRecords performs a semantic search constrained to the records
// collection and rehydrates each hit by fetching the underlying record from
// the supplied store. Hits whose record cannot be fetched (deleted between
// search and rehydrate) are silently skipped — Qdrant convergence will retire
// the orphan on the next reconcile pass.
func (s *Service) SearchRecords(ctx context.Context, query string, filter records.SearchFilter, store records.Store) ([]SearchRecordHit, error) {
	if s.embedder == nil || s.recordStore == nil {
		return nil, fmt.Errorf("aisearch not configured for record search")
	}
	if store == nil {
		return nil, fmt.Errorf("records store is required for rehydrate")
	}
	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	vec, err := s.embedder.Embed(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("embed record query: %w", err)
	}
	raw, err := s.recordStore.Search(ctx, vec, limit, s.threshold)
	if err != nil {
		return nil, fmt.Errorf("vector search records: %w", err)
	}
	out := make([]SearchRecordHit, 0, len(raw))
	for _, r := range raw {
		// Cheap payload-side filter before paying the disk read.
		if filter.Kind != "" {
			if k, _ := r.Payload["kind"].(string); k != string(filter.Kind) {
				continue
			}
		}
		if filter.Scenario != "" {
			if sc, _ := r.Payload["scenario"].(string); sc != filter.Scenario {
				continue
			}
		}
		recID, _ := r.Payload["record_id"].(string)
		if recID == "" {
			continue
		}
		rec, err := store.Get(recID)
		if err != nil {
			continue
		}
		out = append(out, SearchRecordHit{Record: rec, Score: r.Score})
	}
	return out, nil
}

// RecordSearcherAdapter implements records.Searcher by delegating to the
// aisearch.Service. main.go constructs one and wires it onto the records
// handler via SetSearcher.
type RecordSearcherAdapter struct {
	svc   *Service
	store records.Store
}

// NewRecordSearcherAdapter wraps an aisearch.Service + records.Store. Both are
// required.
func NewRecordSearcherAdapter(svc *Service, store records.Store) *RecordSearcherAdapter {
	return &RecordSearcherAdapter{svc: svc, store: store}
}

// SearchRecords satisfies records.Searcher.
func (a *RecordSearcherAdapter) SearchRecords(query string, filter records.SearchFilter) ([]records.SearchHit, error) {
	if a == nil || a.svc == nil || a.store == nil {
		return nil, records.ErrSearchUnavailable
	}
	hits, err := a.svc.SearchRecords(context.Background(), query, filter, a.store)
	if err != nil {
		return nil, err
	}
	out := make([]records.SearchHit, 0, len(hits))
	for _, h := range hits {
		out = append(out, records.SearchHit{Record: h.Record, Score: h.Score})
	}
	return out, nil
}
