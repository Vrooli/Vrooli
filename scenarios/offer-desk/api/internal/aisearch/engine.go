package aisearch

import (
	"context"
	"fmt"
	"strings"

	pkg "github.com/vrooli/ai-go/search"
)

// Shared-engine adoption for Offer Desk. The engine ranks the same bounded
// records the lexical Service projects; it never invents a second catalog. The
// lexical Service remains the offline/degraded read path (it is wired as the
// engine's TextFallback), so an unavailable Ollama or Qdrant degrades retrieval
// quality without taking the provider down.

const (
	// ProviderID is Offer Desk's provider identity in .vrooli/search.json and the
	// search-hub registry. It is the SSOT both the boot tuning read and the
	// self-registration push resolve.
	ProviderID = "offer-desk.catalog"
	// Collection is Offer Desk's dedicated Qdrant collection. The collection
	// name appears once; the shared engine derives the rest of the layout.
	Collection = "offer-desk-catalog"
	// idPrefix namespaces Offer Desk's point IDs inside the shared engine.
	idPrefix = "offer-desk:"
	// catalogKind is the single logical index kind. The finer node/relationship
	// kind rides in each record's own kind and payload, so one collection holds
	// every catalog record class without a second router.
	catalogKind = "catalog"
)

// CatalogSource adapts the read-only catalog projection to the shared engine's
// Source contract: one SourceDoc per bounded catalog record.
type CatalogSource struct {
	source *StoreSource
}

// NewCatalogSource adapts a StoreSource to the engine Source contract.
func NewCatalogSource(source *StoreSource) *CatalogSource { return &CatalogSource{source: source} }

// LoadAll projects the current catalog snapshot to indexable documents.
func (s *CatalogSource) LoadAll(ctx context.Context) ([]pkg.SourceDoc, error) {
	snapshot, err := s.source.Load(ctx)
	if err != nil {
		return nil, fmt.Errorf("catalog source: %w", err)
	}
	out := make([]pkg.SourceDoc, 0, len(snapshot.Records))
	for _, r := range snapshot.Records {
		out = append(out, recordToSourceDoc(r))
	}
	return out, nil
}

// recordToSourceDoc maps one bounded record to an indexable document.
// ContentHash is the record revision — the same content hash the lexical
// snapshot generation uses — so the source-level drift gate skips an unchanged
// record wholesale.
func recordToSourceDoc(r Record) pkg.SourceDoc {
	return pkg.SourceDoc{
		ID:          r.ID,
		Kind:        catalogKind,
		ContentHash: r.Revision,
		Body:        embeddingTextFor(r),
		Meta:        sourceMetaFor(r),
	}
}

// embeddingTextFor is the passage embedded for a record: the human title and
// kind first, then snippet and full body, with explicit freshness/history
// markers so a stale or superseded record is retrievable by those properties.
func embeddingTextFor(r Record) string {
	parts := make([]string, 0, 6)
	if t := strings.TrimSpace(r.Title); t != "" {
		parts = append(parts, t)
	}
	if k := strings.TrimSpace(r.Kind); k != "" {
		parts = append(parts, k)
	}
	if s := strings.TrimSpace(r.Snippet); s != "" {
		parts = append(parts, s)
	}
	if b := strings.TrimSpace(r.Body); b != "" {
		parts = append(parts, b)
	}
	if r.Freshness != "" {
		parts = append(parts, "freshness: "+r.Freshness)
	}
	if r.Historical {
		parts = append(parts, "historical record")
	}
	return strings.Join(parts, "\n\n")
}

// sourceMetaFor carries the projection a result reader needs without a second
// owner round-trip. The shared engine appends body/source_id/payload_hash.
func sourceMetaFor(r Record) map[string]any {
	meta := map[string]any{
		"id":         r.ID,
		"kind":       r.Kind,
		"title":      r.Title,
		"snippet":    r.Snippet,
		"follow_up":  r.FollowUp,
		"freshness":  r.Freshness,
		"historical": r.Historical,
		"visibility": r.Visibility,
	}
	if len(r.Metadata) > 0 {
		meta["record"] = r.Metadata
	}
	return meta
}

// EngineComponents is the assembled retrieval bundle an Engine serves from.
// Production obtains it from NewTunedEngine; tests inject searchtest fakes.
type EngineComponents struct {
	Embedder      pkg.Embedder
	VectorStore   pkg.VectorStore
	SparseEncoder pkg.SparseEncoder
	Reranker      *pkg.RerankerChain
	Spec          pkg.CollectionSpec
}

// Engine is Offer Desk's production-capable search provider: the shared
// read-path Service bound to the catalog source, plus the collection identity
// and reconciler the sync loop drives.
type Engine struct {
	svc        *pkg.Service
	store      pkg.VectorStore
	spec       pkg.CollectionSpec
	reconciler *pkg.Reconciler
}

// NewEngineFromComponents assembles the engine from already-built components.
// It is the single place the Offer Desk source binding and read-path seams are
// wired, so the production and test paths cannot drift.
func NewEngineFromComponents(c EngineComponents, source pkg.Source, tuning pkg.TuningConfig, textFallback pkg.TextFallbackFunc, parallelism, maxEmbedsPerTick int) *Engine {
	tuning = tuning.WithDefaults()
	var binding pkg.SourceBinding
	if c.SparseEncoder != nil {
		// Single-chunk hybrid: the record body is already the embedding text
		// (identity composer) plus the local BM25 sparse leg.
		binding = pkg.NewHybridBinding(catalogKind, idPrefix, c.VectorStore, source, pkg.NewIdentityChunker(), pkg.NewIdentityComposer(), c.SparseEncoder)
	} else {
		binding = pkg.NewDenseBinding(catalogKind, idPrefix, c.VectorStore, source)
	}
	rec := pkg.NewReconciler(c.Embedder, []pkg.SourceBinding{binding}, parallelism)
	rec.MaxEmbedsPerTick = maxEmbedsPerTick

	opts := pkg.ServiceOptions{
		Embedder:      c.Embedder,
		SparseEncoder: c.SparseEncoder,
		VectorStore:   c.VectorStore,
		Reranker:      c.Reranker,
		Reconciler:    rec,
		RerankEnabled: tuning.RerankEnabled,
		RerankBlend:   tuning.RerankBlend,
		Shortlist:     tuning.RerankShortlist,
		HybridFusion:  tuning.HybridFusion,
		ApplyFloor:    true,
		Floor:         tuning.Floor.Config(),
		Project:       projectFederation,
		TextFallback:  textFallback,
	}
	return &Engine{svc: pkg.NewService(opts), store: c.VectorStore, spec: c.Spec, reconciler: rec}
}

// NewTunedEngine builds the engine from resolved tuning + deps. The embedding
// policy in deps must already be resolved by the caller (for example via
// ResolveEngineDepsEmbedding).
func NewTunedEngine(tuning pkg.TuningConfig, deps pkg.EngineDeps, source pkg.Source, textFallback pkg.TextFallbackFunc, parallelism, maxEmbedsPerTick int) *Engine {
	te := pkg.NewServiceForTuning(tuning, deps)
	return NewEngineFromComponents(EngineComponents{
		Embedder:      te.Embedder,
		VectorStore:   te.VectorStore,
		SparseEncoder: te.SparseEncoder,
		Reranker:      te.Reranker,
		Spec:          te.Spec,
	}, source, tuning, textFallback, parallelism, maxEmbedsPerTick)
}

// Reconciler exposes the engine's reconciler so a sync loop can drive periodic
// convergence.
func (e *Engine) Reconciler() *pkg.Reconciler { return e.reconciler }

// Spec exposes the resolved collection layout (including its embedding model).
func (e *Engine) Spec() pkg.CollectionSpec { return e.spec }

// EnsureCollection is idempotent. A failure means degraded search, never a boot
// failure; the caller decides whether to continue.
func (e *Engine) EnsureCollection(ctx context.Context) error {
	return e.store.EnsureCollection(ctx, e.spec)
}

// Status reports engine availability and indexed count. It is backend state,
// not corpus identity; the corpus generation stays source-derived.
func (e *Engine) Status(ctx context.Context) pkg.StatusReport { return e.svc.Status(ctx) }

// Search ranks the indexed corpus and projects each hit into Offer Desk's typed
// search shape. It returns the retrieval method the engine used.
func (e *Engine) Search(ctx context.Context, query string, limit int) ([]SearchHit, string, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	resp, err := e.svc.Search(ctx, pkg.SearchQuery{Query: query, Limit: limit})
	if err != nil {
		return nil, "", err
	}
	hits := make([]SearchHit, 0, len(resp.Results))
	for _, r := range resp.Results {
		hits = append(hits, hitFromEngineResult(r))
	}
	return hits, resp.Method, nil
}

// Reindex starts a shared reconcile job (the control-plane seam).
func (e *Engine) Reindex(ctx context.Context, scenario string, dryRun bool) (*pkg.ReindexJob, error) {
	return e.svc.Reindex(ctx, scenario, dryRun)
}

// projectFederation fills the shared federation projection fields from the
// engine payload so the canonical follow-up reference and snippet survive the
// read path even when a caller does not use the proto handler.
func projectFederation(r pkg.SearchResult) pkg.SearchResult {
	if r.Payload == nil {
		return r
	}
	r.SourceID = stringMeta(r.Payload, "id", r.SourceID)
	r.RelativePath = stringMeta(r.Payload, "follow_up", r.RelativePath)
	r.Path = r.RelativePath
	if r.Snippet == "" {
		r.Snippet = stringMeta(r.Payload, "snippet", "")
	}
	return r
}

func hitFromEngineResult(r pkg.SearchResult) SearchHit {
	return SearchHit{
		ID:         stringMeta(r.Payload, "id", r.ID),
		Kind:       stringMeta(r.Payload, "kind", ""),
		Title:      stringMeta(r.Payload, "title", ""),
		Snippet:    stringMeta(r.Payload, "snippet", r.Snippet),
		FollowUp:   stringMeta(r.Payload, "follow_up", r.RelativePath),
		Score:      r.Score,
		Freshness:  stringMeta(r.Payload, "freshness", ""),
		Historical: boolMeta(r.Payload, "historical"),
		Metadata:   r.Payload,
	}
}

func stringMeta(payload map[string]any, key, fallback string) string {
	if v, ok := payload[key].(string); ok && strings.TrimSpace(v) != "" {
		return v
	}
	return fallback
}

func boolMeta(payload map[string]any, key string) bool {
	v, _ := payload[key].(bool)
	return v
}
