package aisearch

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	pkg "github.com/vrooli/ai-go/search"
)

// readEngine is one corpus's assembled read path: the shared-engine Service
// bound to that corpus's own Qdrant collection (store/qspec), plus the corpus
// identity. The reconciler that indexes it is shared across all corpora (see
// Service.rec), so this struct holds only the read-side bundle. The read Service
// is held behind a mutex + rebuild closure so ApplyTuning can swap query-time
// factors live (rerank/floor) without a process restart.
type readEngine struct {
	spec  corpusSpec
	store pkg.VectorStore
	qspec pkg.CollectionSpec

	mu      sync.RWMutex
	svc     *pkg.Service
	tuning  pkg.TuningConfig
	rebuild func(pkg.TuningConfig) *pkg.Service
}

// service returns the live read Service under the read lock.
func (e *readEngine) service() *pkg.Service {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.svc
}

// Service is SDA's multi-corpus search surface over the shared
// packages/ai-go/search engine. It hosts N corpora — each its own Qdrant
// collection, point-id namespace, and federated leaf — on ONE Reconciler / sync
// loop / embedder (the engine's native []SourceBinding model). Per corpus it
// exposes a primitive ordered-results surface the Connect handlers project back
// to wire types; the .dependencies corpus additionally backs the SemanticRanker
// (RankIDs) dependencygovernance consumes. When Qdrant/Ollama are down each
// corpus degrades to its own keyword text fallback, so search never hard-fails.
type Service struct {
	engines map[CorpusID]*readEngine
	order   []CorpusID
	rec     *pkg.Reconciler
}

// New assembles the multi-corpus search service: one read-path engine per corpus
// spec (each its own collection) plus a SINGLE Reconciler over all bindings and a
// SINGLE shared embedder, so one sync loop converges every collection. deps
// carries the resolved embedding policy + Qdrant/reranker endpoints (Collection
// is overridden per corpus). Boot resolves deps via ResolveEngineDepsEmbedding
// before calling this.
//
// All corpora share ONE embedder (the first corpus's), enforcing a uniform embed
// space — the corpusSpec INVARIANT. Per-corpus read tuning (rerank/floor) is
// honored independently because each corpus keeps its own read Service.
func New(specs []corpusSpec, deps pkg.EngineDeps, parallelism, maxEmbedsPerTick int) *Service {
	type built struct {
		spec corpusSpec
		te   pkg.TunedEngine
	}
	builts := make([]built, 0, len(specs))
	bindings := make([]pkg.SourceBinding, 0, len(specs))
	var shared pkg.Embedder

	for _, spec := range specs {
		d := deps
		d.Collection = spec.collection
		te := pkg.NewServiceForTuning(spec.tuning, d)
		if shared == nil {
			shared = te.Embedder
		}
		bindings = append(bindings, pkg.NewDenseBinding(spec.kind, spec.idPrefix, te.VectorStore, spec.source))
		builts = append(builts, built{spec: spec, te: te})
	}

	rec := pkg.NewReconciler(shared, bindings, parallelism)
	rec.MaxEmbedsPerTick = maxEmbedsPerTick

	engines := make(map[CorpusID]*readEngine, len(builts))
	order := make([]CorpusID, 0, len(builts))
	for _, b := range builts {
		b := b
		// rebuild assembles a fresh read Service for a query-time tuning, reusing
		// the SHARED embedder + this corpus's store/reranker/reconciler. It changes
		// only the query-time factors (rerank/floor/shortlist/blend) — the embed
		// recipe stays uniform across corpora (the corpusSpec INVARIANT), so a live
		// swap never re-embeds in a divergent space.
		rebuild := func(t pkg.TuningConfig) *pkg.Service {
			t = t.WithDefaults()
			base := pkg.ServiceOptions{
				Embedder:      shared,
				VectorStore:   b.te.VectorStore,
				Reranker:      b.te.Reranker,
				Reconciler:    rec,
				RerankEnabled: t.RerankEnabled,
				RerankBlend:   t.RerankBlend,
				Shortlist:     t.RerankShortlist,
				ApplyFloor:    true,
				Floor:         t.Floor.Config(),
				RerankText:    b.spec.rerankText,
				TextFallback:  b.spec.textFallback,
				// OverridePolicy left nil (DenyOverrides): SDA exposes no per-request
				// query-time override channel; tuning changes flow through search.json
				// + the searchcontrol config endpoint, not per-call overrides.
			}
			return pkg.NewService(base)
		}
		engines[b.spec.id] = &readEngine{
			spec:    b.spec,
			store:   b.te.VectorStore,
			qspec:   b.te.Spec,
			svc:     rebuild(b.spec.tuning),
			tuning:  b.spec.tuning.WithDefaults(),
			rebuild: rebuild,
		}
		order = append(order, b.spec.id)
	}
	return &Service{engines: engines, order: order, rec: rec}
}

// Reconciler exposes the single shared reconciler for the sync loop and reindex.
func (s *Service) Reconciler() *pkg.Reconciler { return s.rec }

// Corpora returns the corpus IDs this service hosts, in deterministic order.
func (s *Service) Corpora() []CorpusID { return s.order }

// EnsureCollections ensures every corpus's Qdrant collection. Best-effort at
// boot — a down Qdrant degrades search to the text fallback rather than failing
// boot; the first error is returned for logging but all collections are still
// attempted.
func (s *Service) EnsureCollections(ctx context.Context) error {
	var firstErr error
	for _, id := range s.order {
		e := s.engines[id]
		if err := e.store.EnsureCollection(ctx, e.qspec); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// CorpusResult is one ranked hit from a corpus search: the natural source ID, the
// relevance score, and the stored payload. The per-leaf Connect handlers project
// Payload into their wire response shape.
type CorpusResult struct {
	SourceID string
	Score    float64
	Payload  map[string]any
}

// SearchCorpus runs AI-first, text-fallback retrieval against one corpus and
// returns its ordered hits. It is the primitive the per-leaf Search RPC handlers
// (.scenarios, .resources) consume. An unknown corpus or empty query returns no
// results without error.
func (s *Service) SearchCorpus(ctx context.Context, corpus CorpusID, query string, limit int) ([]CorpusResult, error) {
	e := s.engines[corpus]
	if e == nil || strings.TrimSpace(query) == "" {
		return nil, nil
	}
	if limit <= 0 {
		limit = 20
	}
	resp, err := e.service().Search(ctx, pkg.SearchQuery{Query: query, Limit: limit, Mode: pkg.ModeAuto})
	if err != nil {
		return nil, err
	}
	out := make([]CorpusResult, 0, len(resp.Results))
	seen := make(map[string]struct{}, len(resp.Results))
	for _, r := range resp.Results {
		id := corpusSourceID(r)
		if id == "" {
			continue
		}
		if _, dup := seen[id]; dup {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, CorpusResult{SourceID: id, Score: r.Score, Payload: r.Payload})
	}
	return out, nil
}

// corpusSourceID recovers a hit's natural ID generically: the engine's SourceID,
// then the payload source_id key.
func corpusSourceID(r pkg.SearchResult) string {
	if strings.TrimSpace(r.SourceID) != "" {
		return strings.TrimSpace(r.SourceID)
	}
	if v, ok := r.Payload["source_id"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.TrimSpace(v)
	}
	return ""
}

// RankIDs runs the AI-first, text-fallback retrieval over the .dependencies
// corpus and returns the ordered record IDs (ecosystem/package) plus their
// relevance scores. available is false only when the query is empty or the
// engine errors — the caller then keeps its own keyword ordering. This is the
// primitive surface dependencygovernance consumes via the SemanticRanker
// interface (no import of this package); its signature is fixed by that contract.
func (s *Service) RankIDs(ctx context.Context, query string, limit int) (ids []string, scores map[string]float64, available bool, err error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil, false, nil
	}
	e := s.engines[CorpusDependencies]
	if e == nil {
		return nil, nil, false, nil
	}
	if limit <= 0 {
		limit = 20
	}
	resp, err := e.service().Search(ctx, pkg.SearchQuery{Query: query, Limit: limit, Mode: pkg.ModeAuto})
	if err != nil {
		return nil, nil, false, err
	}
	ids = make([]string, 0, len(resp.Results))
	scores = make(map[string]float64, len(resp.Results))
	for _, r := range resp.Results {
		id := sourceIDFromResult(r)
		if id == "" {
			continue
		}
		if _, seen := scores[id]; seen {
			continue
		}
		ids = append(ids, id)
		scores[id] = r.Score
	}
	return ids, scores, true, nil
}

// Status reports a corpus's backend availability. "available" means AI search
// works (Ollama AND Qdrant); the text fallback is a degradation, not "available".
// An unknown corpus reports all-false.
func (s *Service) Status(ctx context.Context, corpus CorpusID) (available, ollama, qdrant bool, indexed int) {
	e := s.engines[corpus]
	if e == nil {
		return false, false, false, 0
	}
	r := e.service().Status(ctx)
	return r.Ollama && r.Qdrant, r.Ollama, r.Qdrant, r.IndexedCount
}

// jobService is the read Service that owns reindex job tracking. Reindex via the
// shared reconciler reconciles EVERY corpus's binding (scenario-scoped), so any
// one corpus's Service can drive it; the first corpus's Service is the canonical
// home so a job started by Reindex is found by ReindexStatus/Cancel.
func (s *Service) jobService() *pkg.Service {
	if len(s.order) == 0 {
		return nil
	}
	return s.engines[s.order[0]].service()
}

// Reindex queues a reconcile job over the shared reconciler — scenario-scoped, so
// it reconciles ALL corpus bindings regardless of scope. Satisfies the
// searchcontrol Reindexer seam.
func (s *Service) Reindex(ctx context.Context, scope string, dryRun bool) (*pkg.ReindexJob, error) {
	js := s.jobService()
	if js == nil {
		return nil, errors.New("aisearch: no corpora configured")
	}
	return js.Reindex(ctx, scope, dryRun)
}

// ReindexStatus / ReindexCancel / JobExport forward to the canonical job service.
func (s *Service) ReindexStatus(jobID string) (*pkg.ReindexJob, bool) {
	js := s.jobService()
	if js == nil {
		return nil, false
	}
	return js.ReindexStatus(jobID)
}

func (s *Service) ReindexCancel(jobID string) bool {
	js := s.jobService()
	if js == nil {
		return false
	}
	return js.ReindexCancel(jobID)
}

func (s *Service) JobExport(job *pkg.ReindexJob) map[string]any {
	js := s.jobService()
	if js == nil {
		return nil
	}
	return js.JobExport(job)
}

// ApplyTuning swaps a corpus's read Service to a new tuning live (no restart),
// honoring the new query-time factors (rerank/floor/shortlist/blend). It REJECTS
// an embed-recipe change (embed_model / embed_task_prefix) because every SDA
// corpus shares ONE embedder/Reconciler — a per-corpus embed change would index
// in a space the shared query embedder cannot match (the corpusSpec INVARIANT).
// An index-time change is therefore applied by restart (search.json is the SSOT),
// not live. Returns the corpus the provider maps to as unknown if absent.
func (s *Service) ApplyTuning(_ context.Context, corpus CorpusID, tuning pkg.TuningConfig) error {
	e := s.engines[corpus]
	if e == nil {
		return fmt.Errorf("aisearch: unknown corpus %q", corpus)
	}
	next := tuning.WithDefaults()
	e.mu.RLock()
	cur := e.tuning
	e.mu.RUnlock()
	if next.EmbedModel != cur.EmbedModel || next.EmbedTaskPrefix != cur.EmbedTaskPrefix || next.Engine != cur.Engine {
		return fmt.Errorf("aisearch: embed recipe change for %q (model/task_prefix/engine) requires a restart — SDA corpora share one embedder", corpus)
	}
	svc := e.rebuild(next)
	e.mu.Lock()
	e.svc = svc
	e.tuning = next
	e.mu.Unlock()
	return nil
}
