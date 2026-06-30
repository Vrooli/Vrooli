package aisearch

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	pkg "github.com/vrooli/ai-go/search"
)

// engine is the assembled, immutable retrieval bundle a Service serves from: the
// shared read-path engine (*pkg.Service — query/floor/rerank/fallback pipeline +
// reindex job control) plus the collection identity. ApplyTuning builds a FRESH
// engine for a new tuning and swaps the Service's pointer to it under the write
// lock, so an index-time tuning change (embed recipe) re-embeds live without a
// process restart.
type engine struct {
	svc         *pkg.Service
	vectorStore pkg.VectorStore
	spec        pkg.CollectionSpec
	hybrid      bool
}

// Service is the cartographer's domain-search surface. It adds the domain-domain
// shaping (typed-hit projection, in-process text fallback, ai/text mode vocab)
// over the shared read-path engine. The engine is held behind an RWMutex so
// ApplyTuning can swap it in place (live index-time tuning apply) while
// concurrent Search/Reindex/sync-loop calls read a consistent bundle.
type Service struct {
	mu  sync.RWMutex
	eng *engine

	// rebuild, when non-nil (a Service built via NewTunedService), assembles a
	// fresh engine for a tuning — the seam ApplyTuning re-runs. A Service built
	// from explicit components (NewService, used by tests) has no builder and is
	// not tuning-rebuildable.
	rebuild func(pkg.TuningConfig) *engine
}

// current returns the live engine under the read lock.
func (s *Service) current() *engine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.eng
}

// Options configure NewService (the test/experiment entry point).
type Options struct {
	Embedder         pkg.Embedder
	VectorStore      pkg.VectorStore
	Provider         DomainProvider
	Lister           ScenarioLister
	Parallelism      int
	MaxEmbedsPerTick int
	Threshold        float64
	Floor            pkg.FloorConfig
	RerankEnabled    bool
	Reranker         *pkg.RerankerChain
	RerankShortlist  int
	RerankBlend      bool
	// Sparse, when non-nil, switches the index+query to the hybrid (dense+sparse
	// RRF) leg. nil keeps the dense-only common case (the small domain corpus).
	Sparse pkg.SparseEncoder
	// Collection overrides the Qdrant collection name (default DefaultCollection).
	Collection string
	// Embedding carries resolved Ollama policy metadata for scratch/test engine
	// assembly. Production NewTunedService obtains this through EngineDeps.
	Embedding pkg.EmbeddingPolicy
	// Compose overrides the per-domain embedding-text strategy (nil =>
	// composeDomainEmbeddingText). Retained as a measurable seam.
	Compose func(DomainRecord) string
	// DisableFloor turns OFF ApplyRelevanceFloor (default: floor ON). The floor
	// keeps an honest "no match" (a gibberish query returns nothing above floor).
	DisableFloor bool
}

// NewService builds a Service over the shared engine from explicit components.
// Domains are indexed as single-chunk, dense-only sources (the NewDenseBinding
// common case). A Service built this way is NOT tuning-rebuildable — it is used
// by tests. Production uses NewTunedService.
func NewService(opts Options) *Service {
	return &Service{eng: buildEngine(opts)}
}

// buildEngine assembles one immutable engine bundle from the flat Options. This
// is the test/experiment entry point; production (NewTunedService) bypasses it
// and feeds the base from the shared helper.
func buildEngine(opts Options) *engine {
	base := pkg.ServiceOptions{
		Embedder:      opts.Embedder,
		SparseEncoder: opts.Sparse,
		VectorStore:   opts.VectorStore,
		Reranker:      opts.Reranker,
		RerankEnabled: opts.RerankEnabled,
		RerankBlend:   opts.RerankBlend,
		Shortlist:     opts.RerankShortlist,
		ApplyFloor:    !opts.DisableFloor,
		Floor:         opts.Floor,
		Threshold:     opts.Threshold,
	}
	return buildEngineFromBase(base, opts)
}

// buildEngineFromBase assembles the engine from a read-path base (engine
// components + tuning-derived query-time factors) overlaid with the
// cartographer's domain-domain seams: the domain source/binding/reconciler, the
// rerank/fallback text, and the per-request override policy. The
// Embedder/VectorStore/Sparse/Reranker and every tuning factor come from base,
// so the production caller forwards ZERO factor fields by hand.
func buildEngineFromBase(base pkg.ServiceOptions, opts Options) *engine {
	collection := opts.Collection
	if collection == "" {
		collection = DefaultCollection
	}
	source := newDomainSource(opts.Provider, opts.Lister, opts.Compose)

	hybrid := base.SparseEncoder != nil
	var binding pkg.SourceBinding
	if hybrid {
		binding = pkg.NewHybridBinding(domainKind, idPrefix, base.VectorStore, source,
			pkg.NewIdentityChunker(), pkg.NewIdentityComposer(), base.SparseEncoder)
	} else {
		binding = pkg.NewDenseBinding(domainKind, idPrefix, base.VectorStore, source)
	}
	rec := pkg.NewReconciler(base.Embedder, []pkg.SourceBinding{binding}, opts.Parallelism)
	rec.MaxEmbedsPerTick = opts.MaxEmbedsPerTick

	base.Reconciler = rec
	base.RerankText = func(r pkg.SearchResult) string { return candidateText(r.Payload) }
	base.TextFallback = domainTextFallback(opts.Provider, opts.Lister)
	// Allow the engine to honor token-gated per-request query-time overrides (the
	// inner clamping layer; the outer token gate lives in the search handler).
	base.OverridePolicy = pkg.AllowOverrides()
	svc := pkg.NewService(base)

	spec := pkg.CollectionSpec{
		Name:          collection,
		DenseSize:     opts.Embedding.Dimensions,
		DenseDistance: pkg.DefaultDenseDistance,
		Model:         opts.Embedding.Model,
	}
	if hybrid {
		spec.Sparse = true
		spec.SparseModifier = pkg.DefaultSparseModifier
	}
	return &engine{
		svc:         svc,
		vectorStore: base.VectorStore,
		spec:        spec,
		hybrid:      hybrid,
	}
}

// TunedOptions carries the non-tuning wiring NewTunedService needs to
// (re)assemble the engine for a tuning: the domain provider + scenario lister,
// the reconcile knobs, the embed-text composer, the collection name, and the
// embedding/qdrant endpoints. Deployment facts, not search factors.
type TunedOptions struct {
	Provider         DomainProvider
	Lister           ScenarioLister
	Parallelism      int
	MaxEmbedsPerTick int
	Compose          func(DomainRecord) string
	Collection       string
	EngineDeps       pkg.EngineDeps
}

// NewTunedService builds the production Service: its engine is assembled FROM a
// TuningConfig (engine shape, embed recipe, rerank policy, floor band) via the
// shared pkg.NewServiceForTuning, so the engine shape is chosen by data, not a
// code literal. It retains the builder so ApplyTuning can re-assemble the engine
// for a swept tuning in place.
func NewTunedService(tuning pkg.TuningConfig, opts TunedOptions) *Service {
	if opts.Collection == "" {
		opts.Collection = DefaultCollection
	}
	opts.EngineDeps.Collection = opts.Collection
	build := func(t pkg.TuningConfig) *engine {
		te := pkg.NewServiceForTuning(t, opts.EngineDeps)
		eng := buildEngineFromBase(te.ServiceOptions(), Options{
			Provider:         opts.Provider,
			Lister:           opts.Lister,
			Parallelism:      opts.Parallelism,
			MaxEmbedsPerTick: opts.MaxEmbedsPerTick,
			Compose:          opts.Compose,
			Collection:       opts.Collection,
		})
		eng.spec = te.Spec
		return eng
	}
	return &Service{eng: build(tuning), rebuild: build}
}

// Reconciler exposes the current engine's reconciler (the sync loop resolves it
// each tick via this, so a live ApplyTuning swap re-points the loop).
func (s *Service) Reconciler() *pkg.Reconciler { return s.current().svc.Reconciler() }

// Reindex / ReindexStatus / ReindexCancel / JobExport forward to the current
// engine's read-path service under the read lock (the searchcontrol Reindexer
// seam calls these).
func (s *Service) Reindex(ctx context.Context, scenario string, dryRun bool) (*pkg.ReindexJob, error) {
	return s.current().svc.Reindex(ctx, scenario, dryRun)
}

func (s *Service) ReindexStatus(jobID string) (*pkg.ReindexJob, bool) {
	return s.current().svc.ReindexStatus(jobID)
}

func (s *Service) ReindexCancel(jobID string) bool { return s.current().svc.ReindexCancel(jobID) }

func (s *Service) JobExport(job *pkg.ReindexJob) map[string]any {
	return s.current().svc.JobExport(job)
}

// ApplyTuning rebuilds the engine for tuning and swaps it in place, then kicks an
// async re-embed so the stored vectors converge to the new recipe — the live
// index-time apply that avoids a process restart. The new engine's collection is
// ensured BEFORE the swap, so a structural change the schema guard rejects
// (dense↔hybrid) returns an error and leaves the live engine untouched (no
// auto-drop).
func (s *Service) ApplyTuning(ctx context.Context, tuning pkg.TuningConfig) (jobID string, plannedUpserts, plannedDeletes int, err error) {
	s.mu.RLock()
	build := s.rebuild
	s.mu.RUnlock()
	if build == nil {
		return "", 0, 0, errors.New("aisearch: service is not tuning-rebuildable (built without NewTunedService)")
	}

	next := build(tuning)
	if err := next.vectorStore.EnsureCollection(ctx, next.spec); err != nil {
		return "", 0, 0, fmt.Errorf("apply tuning: ensure collection: %w", err)
	}

	s.mu.Lock()
	s.eng = next
	s.mu.Unlock()

	job, err := next.svc.Reindex(ctx, "", false)
	if err != nil {
		return "", 0, 0, fmt.Errorf("apply tuning: reindex: %w", err)
	}
	exp := next.svc.JobExport(job)
	up, _ := exp["planned_upserts"].(int)
	del, _ := exp["planned_deletes"].(int)
	return job.ID, up, del, nil
}

// EnsureCollection is called once at startup; idempotent.
func (s *Service) EnsureCollection(ctx context.Context) error {
	e := s.current()
	return e.vectorStore.EnsureCollection(ctx, e.spec)
}

// Search performs retrieval with AI-first, text-fallback semantics, projecting
// the shared engine's results back into the cartographer's typed domain hits.
func (s *Service) Search(ctx context.Context, query string, limit int, mode SearchMode, opts ...pkg.SearchOption) (*SearchResponse, error) {
	if strings.TrimSpace(query) == "" {
		return &SearchResponse{Query: query, Method: "text"}, nil
	}
	e := s.current()
	hits, presp, err := pkg.SearchTyped(ctx, e.svc, pkg.SearchQuery{Query: query, Limit: limit, Mode: serviceMode(e.hybrid, mode)},
		func(r pkg.SearchResult) DomainHit {
			hit := payloadToHit(r.ID, r.Score, r.Payload)
			hit.Weak = r.Weak
			return hit
		}, opts...)
	if err != nil {
		return nil, err
	}
	method := presp.Method
	if method == "dense" || method == "hybrid" {
		method = "ai" // wire vocab
	}
	// The engine resolves the scoring regime once, authoritatively, from the
	// actual (method, leg) it ran — read it off the response rather than
	// re-deriving from presp.Reranker, whose blended leg name ("blend:…") is
	// observability-only and does not round-trip through RegimeForMethod.
	for i := range hits {
		hits[i].Regime = presp.Regime
	}
	return &SearchResponse{
		Results:  hits,
		Total:    len(hits),
		Query:    query,
		Method:   method,
		Reranker: presp.Reranker,
	}, nil
}

// serviceMode maps the ai/text/auto vocab to the engine's mode set.
func serviceMode(hybrid bool, m SearchMode) pkg.SearchMode {
	switch m {
	case ModeAI:
		if hybrid {
			return pkg.ModeHybrid
		}
		return pkg.ModeDense
	case ModeText:
		return pkg.ModeText
	default:
		return pkg.ModeAuto
	}
}

// Status reports backend availability. "available" means AI search works, so it
// requires BOTH ollama and qdrant; the text fallback is a degradation.
func (s *Service) Status(ctx context.Context) StatusReport {
	r := s.current().svc.Status(ctx)
	return StatusReport{
		Available:            r.Ollama && r.Qdrant,
		Ollama:               r.Ollama,
		Qdrant:               r.Qdrant,
		IndexedCount:         r.IndexedCount,
		LastReconcileAt:      r.LastReconcileAt,
		LastReconcileOutcome: r.LastReconcileOutcome,
		Reranker:             r.Reranker,
	}
}

// candidateText is the passage handed to the reranker for one hit: the stored
// retrievable body, falling back to the domain identity + responsibility.
func candidateText(payload map[string]any) string {
	if body, _ := payload["body"].(string); strings.TrimSpace(body) != "" {
		return body
	}
	scenario, _ := payload["scenario"].(string)
	name, _ := payload["name"].(string)
	resp, _ := payload["responsibility"].(string)
	return strings.TrimSpace(scenario + " " + name + " " + resp)
}

// domainTextFallback is the offline-safe keyword leg over freshly derived
// domains (BM25-ish substring scoring). It returns pkg.SearchResult with the
// domain fields in the payload so the shared pipeline (and the typed projection
// in Search) treat it identically to a vector hit. Used when ollama or qdrant is
// unavailable.
func domainTextFallback(provider DomainProvider, lister ScenarioLister) pkg.TextFallbackFunc {
	return func(ctx context.Context, q pkg.SearchQuery) ([]pkg.SearchResult, error) {
		terms := tokenize(q.Query)
		if len(terms) == 0 {
			return nil, nil
		}
		scenarios, err := lister.List(ctx)
		if err != nil {
			return nil, err
		}
		type scored struct {
			res   pkg.SearchResult
			score float64
			id    string
		}
		scoredHits := make([]scored, 0, 64)
		for _, scenario := range scenarios {
			m, err := provider.ExtractDomains(ctx, scenario)
			if err != nil {
				continue
			}
			for _, d := range m.Domains {
				r := toDomainRecord(scenario, m, d)
				score := scoreDomain(r, terms)
				if score <= 0 {
					continue
				}
				scoredHits = append(scoredHits, scored{
					res:   pkg.SearchResult{ID: pointIDForDomain(r.ID), Score: score, Payload: domainMeta(r)},
					score: score,
					id:    r.ID,
				})
			}
		}
		sort.Slice(scoredHits, func(i, j int) bool {
			if scoredHits[i].score != scoredHits[j].score {
				return scoredHits[i].score > scoredHits[j].score
			}
			return scoredHits[i].id < scoredHits[j].id
		})
		out := make([]pkg.SearchResult, 0, len(scoredHits))
		for _, sh := range scoredHits {
			out = append(out, sh.res)
		}
		return out, nil
	}
}

func tokenize(q string) []string {
	q = strings.ToLower(q)
	fields := strings.FieldsFunc(q, func(r rune) bool {
		switch r {
		case ' ', '\t', '\n', '\r', ',', ';', ':', '/', '\\', '-', '_', '.', '"', '\'', '(', ')', '[', ']':
			return true
		}
		return false
	})
	out := make([]string, 0, len(fields))
	seen := make(map[string]struct{}, len(fields))
	for _, f := range fields {
		f = strings.TrimSpace(f)
		if len(f) < 2 {
			continue
		}
		if _, ok := seen[f]; ok {
			continue
		}
		seen[f] = struct{}{}
		out = append(out, f)
	}
	return out
}

// scoreDomain is the keyword-leg score for one domain against the query terms.
// Weighted fields: scenario+name > responsibility > purpose > glossary.
func scoreDomain(r DomainRecord, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	ident := strings.ToLower(r.Scenario + " " + r.Name)
	resp := strings.ToLower(r.Responsibility)
	purpose := strings.ToLower(r.Purpose)
	gloss := strings.ToLower(strings.Join(r.Glossary, " "))
	var score float64
	for _, t := range terms {
		hit := false
		if strings.Contains(ident, t) {
			score += 1.0
			hit = true
		}
		if strings.Contains(resp, t) {
			score += 0.5
			hit = true
		}
		if strings.Contains(purpose, t) {
			score += 0.3
			hit = true
		}
		if strings.Contains(gloss, t) {
			score += 0.3
			hit = true
		}
		if !hit {
			score -= 0.05
		}
	}
	if score <= 0 {
		return 0
	}
	max := float64(len(terms)) * 2.1
	if max <= 0 {
		return 0
	}
	n := score / max
	if n > 1 {
		n = 1
	}
	if n < 0 {
		n = 0
	}
	return n
}
