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

// SearchMode tags how Search should retrieve results. It is the proto-facing
// enum the Connect handler maps to and is intentionally kept local rather than
// adopting pkg.SearchMode (which has no "ai" member): cli-health's wire vocab is
// ai/text, mapped to the engine's dense/text below.
type SearchMode string

const (
	ModeAuto SearchMode = "auto" // prefer ai, fall back to text
	ModeAI   SearchMode = "ai"   // ai only; error if unavailable
	ModeText SearchMode = "text" // text only; never embed
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

// Service is cli-health's command-search surface. It adds the command-domain
// shaping (typed-hit projection, discovery text fallback, ai/text mode vocab)
// over the shared read-path engine. The engine is held behind an RWMutex rather
// than embedded so ApplyTuning can swap it in place (live index-time tuning
// apply) while concurrent Search/Reindex/sync-loop calls read a consistent
// bundle. Every access goes through current() under the read lock — embedding the
// engine and swapping the pointer would race on that field.
type Service struct {
	mu  sync.RWMutex
	eng *engine

	// rebuild, when non-nil (a Service built via NewTunedService), assembles a
	// fresh engine for a tuning — the seam ApplyTuning re-runs. A Service built
	// from explicit components (NewService, used by the recall experiments and
	// tests) has no builder and is not tuning-rebuildable.
	rebuild func(pkg.TuningConfig) *engine
}

// current returns the live engine under the read lock.
func (s *Service) current() *engine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.eng
}

// Options configure NewService.
type Options struct {
	Embedder         pkg.Embedder
	VectorStore      pkg.VectorStore
	Discovery        DiscoverySource
	Parallelism      int
	MaxEmbedsPerTick int
	Threshold        float64
	// Floor tunes the relative relevance cutoff applied to AI results (the
	// operator override; FloorForMethodLeg supplies the regime default).
	Floor pkg.FloorConfig
	// RerankEnabled gates the reranker pass; Reranker is the degradation chain
	// (cross-encoder -> llm -> fused order). When disabled or nil, results keep
	// their dense order.
	RerankEnabled bool
	Reranker      *pkg.RerankerChain
	// RerankShortlist is the over-fetch depth handed to the reranker (the
	// configured lever, <PREFIX>_RERANK_SHORTLIST). Non-positive falls back to the
	// engine default.
	RerankShortlist int
	// RerankBlend fuses the reranker order with the retrieval order (RRF) instead
	// of letting the reranker reorder outright — it keeps the cross-encoder's
	// gibberish rejection while not burying strongly-retrieved canonical commands.
	RerankBlend bool
	// Sparse, when non-nil, switches the index+query to the hybrid (dense+sparse
	// RRF) leg — the lexical half that nails exact-token command queries a terse
	// dense vector fumbles. nil keeps the dense-only common case.
	Sparse pkg.SparseEncoder
	// Collection overrides the Qdrant collection name (default DefaultCollection).
	// Used by the recall experiment to index alternative strategies into scratch
	// collections without disturbing the live index.
	Collection string
	// Embedding carries resolved Ollama policy metadata for scratch/test engine
	// assembly. Production NewTunedService obtains this through EngineDeps.
	Embedding pkg.EmbeddingPolicy
	// Compose overrides the per-command embedding-text strategy (nil =>
	// composeCommandEmbeddingText, the measured-best production strategy). The
	// seam is retained for measurement; an enriched variant was tried and removed
	// (see graduation-retrospective.md).
	Compose func(CommandRecord) string
	// DisableFloor turns OFF ApplyRelevanceFloor (default: floor ON). The floor
	// trims the low-relevance tail but, in the cosine regime, can also drop a
	// genuinely-relevant-but-mid-score canonical command — a measured recall cost.
	DisableFloor bool
}

// NewService builds a Service over the shared engine from explicit components.
// cli-health indexes commands as single-chunk, dense-only sources (the
// NewDenseBinding common case); the shared read path applies the relevance floor
// (commands are cosine/cross-encoder scored, not RRF-fused) and reranks when a
// chain is supplied. A Service built this way is NOT tuning-rebuildable (it has
// no factor builder) — it is used by the recall experiments and tests that index
// alternative strategies into scratch collections. Production uses
// NewTunedService.
func NewService(opts Options) *Service {
	return &Service{eng: buildEngine(opts)}
}

// buildEngine assembles one immutable engine bundle from the flat Options. This
// is the test/experiment entry point (callers that index scratch strategies set
// the engine components + tuning factors directly); it derives the shared
// read-path base from Options and delegates to buildEngineFromBase. Production
// (NewTunedService) bypasses it and feeds the base from the shared helper.
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
// components + tuning-derived query-time factors — e.g.
// pkg.TunedEngine.ServiceOptions()) overlaid with cli-health's command-domain
// seams: the command source/binding/reconciler, the rerank/fallback text, and
// the per-request override policy. The Embedder/VectorStore/Sparse/Reranker and
// every tuning factor come from base, so the production caller forwards ZERO
// factor fields by hand (forget one and the Service silently runs misconfigured).
// opts supplies only the structural wiring: the discovery source, the compose
// strategy, the collection name, and the reconcile knobs.
func buildEngineFromBase(base pkg.ServiceOptions, opts Options) *engine {
	collection := opts.Collection
	if collection == "" {
		collection = DefaultCollection
	}
	source := newCommandSource(opts.Discovery, opts.Compose)

	hybrid := base.SparseEncoder != nil
	var binding pkg.SourceBinding
	if hybrid {
		// Single-chunk hybrid: identity chunk/compose (the Body is the pre-composed
		// embedding text) plus the local sparse encoder for the BM25 leg.
		binding = pkg.NewHybridBinding(commandKind, idPrefix, base.VectorStore, source,
			pkg.NewIdentityChunker(), pkg.NewIdentityComposer(), base.SparseEncoder)
	} else {
		binding = pkg.NewDenseBinding(commandKind, idPrefix, base.VectorStore, source)
	}
	rec := pkg.NewReconciler(base.Embedder, []pkg.SourceBinding{binding}, opts.Parallelism)
	rec.MaxEmbedsPerTick = opts.MaxEmbedsPerTick

	base.Reconciler = rec
	base.RerankText = func(r pkg.SearchResult) string { return candidateText(r.Payload) }
	base.TextFallback = commandTextFallback(opts.Discovery)
	if hybrid {
		// The sparse leg supplies exact-token candidates, while this bounded
		// owner-side seam lets an exact command leaf survive the cross-encoder's
		// absolute floor. Search Hub remains agnostic about command vocabulary.
		base.PreFloorDecorate = commandLexicalRescue
	}
	// Allow the engine to honor per-request query-time overrides. This is the
	// INNER layer only (it still clamps every factor to its taxonomy range); the
	// OUTER token + experiment-flag gate lives in the search handler, so a public,
	// tokenless request never reaches an applied override.
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

// TunedOptions carries the non-tuning wiring NewTunedService needs to (re)assemble
// the engine for a tuning: the command discovery source, the reconcile knobs, the
// embed-text composer, the collection name, and the reranker resource endpoints.
// These are deployment facts, not search factors, so they stay outside the
// TuningConfig the sweep moves.
type TunedOptions struct {
	Discovery        DiscoverySource
	Parallelism      int
	MaxEmbedsPerTick int
	Compose          func(CommandRecord) string
	Collection       string
	EngineDeps       pkg.EngineDeps
}

// NewTunedService builds the production Service: its engine is assembled FROM a
// TuningConfig (engine shape, embed recipe, rerank policy, floor band) via the
// shared pkg.NewServiceForTuning, so the engine shape is chosen by data, not a
// code literal. It retains the builder so ApplyTuning can re-assemble the engine
// for a swept tuning in place — an index-time change re-embeds without a restart.
func NewTunedService(tuning pkg.TuningConfig, opts TunedOptions) *Service {
	if opts.Collection == "" {
		opts.Collection = DefaultCollection
	}
	// The store collection name lives in EngineDeps (NewServiceForTuning builds the
	// vector store) and must agree with the spec buildEngine records. Keep the
	// caller's deployment wiring immutable: structural engine changes select a
	// versioned sibling collection below.
	baseDeps := opts.EngineDeps
	build := func(t pkg.TuningConfig) *engine {
		collection := collectionForTuning(opts.Collection, t)
		deps := baseDeps
		deps.Collection = collection
		te := pkg.NewServiceForTuning(t, deps)
		// te.ServiceOptions() carries the engine components AND every query-time
		// tuning factor (RerankEnabled/RerankBlend/Shortlist/Floor) derived from the
		// resolved tuning — no factor is forwarded by hand here. opts supplies only
		// the structural command-domain wiring.
		eng := buildEngineFromBase(te.ServiceOptions(), Options{
			Discovery:        opts.Discovery,
			Parallelism:      opts.Parallelism,
			MaxEmbedsPerTick: opts.MaxEmbedsPerTick,
			Compose:          opts.Compose,
			Collection:       collection,
		})
		eng.spec = te.Spec
		return eng
	}
	return &Service{eng: build(tuning), rebuild: build}
}

// Reconciler exposes the current engine's reconciler (the sync loop resolves it
// each tick via this, so a live ApplyTuning swap re-points the loop) and is also
// used by the recall experiments to drive Plan/Apply directly.
func (s *Service) Reconciler() *pkg.Reconciler { return s.current().svc.Reconciler() }

// Reindex / ReindexStatus / ReindexCancel / JobExport forward to the current
// engine's read-path service under the read lock (the searchcontrol Reindexer
// seam calls these). They are explicit forwarders rather than promoted methods so
// an ApplyTuning swap of the engine pointer can never race a concurrent caller.
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
// index-time apply that avoids a process restart. It returns the reindex job id +
// planned drift counts (the same shape Reindex returns) so the caller can poll
// ReindexStatus to terminal.
//
// Ordering is swap-safe: the new engine's collection is ensured BEFORE the swap.
// Structural dense↔hybrid changes use a versioned sibling collection, so the
// schema guard never needs to mutate or delete the incumbent collection.
// Recipe changes (embed_task_prefix, an in-dimension embed_model swap) keep the
// collection layout and re-embed live — the case the boot-recipe reindex could
// not apply.
func (s *Service) ApplyTuning(ctx context.Context, tuning pkg.TuningConfig) (jobID string, plannedUpserts, plannedDeletes int, err error) {
	s.mu.RLock()
	build := s.rebuild
	s.mu.RUnlock()
	if build == nil {
		return "", 0, 0, errors.New("aisearch: service is not tuning-rebuildable (built without NewTunedService)")
	}

	next := build(tuning)
	if err := next.vectorStore.EnsureCollection(ctx, next.spec); err != nil {
		// Structural mismatch (or qdrant unreachable): do not swap, do not drop.
		return "", 0, 0, fmt.Errorf("apply tuning: ensure collection: %w", err)
	}

	s.mu.Lock()
	s.eng = next
	s.mu.Unlock()

	// Re-embed the stored vectors with the new recipe. The recipe-aware drift hash
	// marks every point drifted, so this re-upserts the whole corpus.
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
// the shared engine's results back into cli-health's typed command hits. Optional
// pkg.SearchOptions (pkg.WithOverrides) vary the query-time factors for this one
// call — the search handler passes them only after the token + experiment-flag
// gate; an ordinary request passes none.
func (s *Service) Search(ctx context.Context, query string, limit int, mode SearchMode, opts ...pkg.SearchOption) (*SearchResponse, error) {
	if strings.TrimSpace(query) == "" {
		return &SearchResponse{Query: query, Method: "text"}, nil
	}
	e := s.current()
	// SearchTyped projects each finished result into cli-health's typed command hit
	// at the engine boundary (the shared generic), so this surface owns no
	// projection loop and cannot drift from the engine's result ordering.
	hits, presp, err := pkg.SearchTyped(ctx, e.svc, pkg.SearchQuery{Query: query, Limit: limit, Mode: serviceMode(e.hybrid, mode)},
		func(r pkg.SearchResult) SearchHit {
			hit := payloadToHit(r.ID, r.Score, r.Payload)
			hit.Weak = r.Weak
			return hit
		}, opts...)
	if err != nil {
		return nil, err
	}
	method := presp.Method
	if method == "dense" || method == "hybrid" {
		method = "ai" // cli-health's wire vocab
	}
	return &SearchResponse{
		Results:  hits,
		Total:    len(hits),
		Query:    query,
		Method:   method,
		Reranker: presp.Reranker,
	}, nil
}

// serviceMode maps cli-health's ai/text/auto vocab to the engine's mode set.
// "ai" is the hybrid leg when the current engine is hybrid, else the dense leg —
// so a live ApplyTuning that flips engine shape routes correctly.
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

// Status reports backend availability, mapping the engine's StatusReport into
// cli-health's wire shape. cli-health's "available" means AI search works, so it
// requires BOTH ollama and qdrant (the engine's default "qdrant OR text" is KO's
// looser doc-search semantics); the text fallback is a degradation, not
// "available".
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
// retrievable body, falling back to the command's path + description.
func candidateText(payload map[string]any) string {
	if body, _ := payload["body"].(string); strings.TrimSpace(body) != "" {
		return body
	}
	full, _ := payload["full_path"].(string)
	desc, _ := payload["description"].(string)
	return strings.TrimSpace(full + " " + desc)
}

// commandLexicalRescue is the command-owner's second lexical leg. Hybrid
// retrieval can surface an exact leaf in the reranker shortlist while the
// cross-encoder gives it a low absolute score because the query is phrased as
// an operation rather than a command title. A leading query term matching the
// command leaf is strong lexical evidence; a joined leading phrase handles
// natural wording such as "set up" -> "setup". The bounded boost is scaled to
// the score regime: RRF/DBSF scores are roughly 0.03, while cross-encoder scores
// are 0..1 and need enough lift to clear the shared 0.35 garbage floor.
func commandLexicalRescue(hits []pkg.SearchResult, q pkg.SearchQuery) {
	terms := tokenize(q.Query)
	if len(terms) == 0 {
		return
	}
	first := terms[0]
	joinedLeading := first
	if len(terms) > 1 {
		joinedLeading += terms[1]
	}
	maxScore := 0.0
	for _, hit := range hits {
		if hit.Score > maxScore {
			maxScore = hit.Score
		}
	}
	for i := range hits {
		name, _ := hits[i].Payload["name"].(string)
		nameTokens := tokenize(name)
		// Match the complete leaf, not an arbitrary token inside a compound
		// command such as api-source-set. The latter would turn the phrase
		// "set up" into a boost for unrelated *-set commands.
		matchesLeading := commandLeafMatches(nameTokens, first)
		if !matchesLeading && len(terms) > 1 {
			matchesLeading = commandLeafMatches(nameTokens, joinedLeading)
		}
		if !matchesLeading {
			continue
		}
		if maxScore <= 0.2 {
			hits[i].Score += 0.36
		} else {
			hits[i].Score += 0.01
		}
	}
}

func commandLeafMatches(tokens []string, want string) bool {
	return len(tokens) == 1 && tokens[0] == want
}

// commandTextFallback is the offline-safe keyword leg over freshly discovered
// records (BM25-ish substring scoring). It returns pkg.SearchResult with the
// command fields in the payload so the shared engine's pipeline (and the typed
// projection in Search) treat it identically to a vector hit. Slow (re-scans on
// every call) but correct — used when ollama or qdrant is unavailable. Only the
// scenario CLIs are scanned (matching the vector index's scope at query time).
func commandTextFallback(discovery DiscoverySource) pkg.TextFallbackFunc {
	return func(ctx context.Context, q pkg.SearchQuery) ([]pkg.SearchResult, error) {
		terms := tokenize(q.Query)
		if len(terms) == 0 {
			return nil, nil
		}
		scenarios, err := discovery.ListScenarios(ctx)
		if err != nil {
			return nil, err
		}
		type scored struct {
			res   pkg.SearchResult
			score float64
			path  string
		}
		scoredHits := make([]scored, 0, 64)
		for _, scenario := range scenarios {
			records, err := discovery.Discover(ctx, scenario)
			if err != nil {
				continue
			}
			for _, r := range records {
				score := scoreRecord(r, terms)
				if score <= 0 {
					continue
				}
				scoredHits = append(scoredHits, scored{
					res:   pkg.SearchResult{ID: pointIDForCommand(r.FullPath), Score: score, Payload: commandMeta(r)},
					score: score,
					path:  r.FullPath,
				})
			}
		}
		sort.Slice(scoredHits, func(i, j int) bool {
			if scoredHits[i].score != scoredHits[j].score {
				return scoredHits[i].score > scoredHits[j].score
			}
			return scoredHits[i].path < scoredHits[j].path
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

func scoreRecord(r CommandRecord, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	// Weighted fields: full path > name > group > description > tags.
	name := strings.ToLower(r.FullPath + " " + r.Name + " " + r.Group)
	desc := strings.ToLower(r.Description)
	tags := strings.ToLower(strings.Join(r.Tags, " "))
	bind := strings.ToLower(r.Binding)
	var score float64
	for _, t := range terms {
		hit := false
		if strings.Contains(name, t) {
			score += 1.0
			hit = true
		}
		if strings.Contains(desc, t) {
			score += 0.4
			hit = true
		}
		if strings.Contains(tags, t) {
			score += 0.3
			hit = true
		}
		if strings.Contains(bind, t) {
			score += 0.2
			hit = true
		}
		if !hit {
			// Missing term shouldn't tank an otherwise good match.
			score -= 0.05
		}
	}
	if score <= 0 {
		return 0
	}
	// Normalize to (0, 1] for ScorePercent — divide by max possible.
	max := float64(len(terms)) * 1.9
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
