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

// SearchMode is the proto-facing mode vocabulary (ai/text/auto), mapped to
// the engine's dense/hybrid/text legs below.
type SearchMode string

const (
	ModeAuto SearchMode = "auto"
	ModeAI   SearchMode = "ai"
	ModeText SearchMode = "text"
)

// engine is the assembled, immutable retrieval bundle (shared read-path
// service + collection identity). ApplyTuning swaps a fresh one in place.
type engine struct {
	svc         *pkg.Service
	vectorStore pkg.VectorStore
	spec        pkg.CollectionSpec
	hybrid      bool
}

// Service is business-health's intent-search surface: the typed-hit
// projection, the fleet text fallback, and the ai/text mode vocabulary over
// the shared read-path engine. The engine sits behind an RWMutex so
// ApplyTuning can swap it live.
type Service struct {
	mu  sync.RWMutex
	eng *engine

	rebuild func(pkg.TuningConfig) *engine
}

func (s *Service) current() *engine {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.eng
}

// Options configure NewService (tests/experiments; production uses
// NewTunedService).
type Options struct {
	Embedder         pkg.Embedder
	VectorStore      pkg.VectorStore
	Source           IntentSource
	Parallelism      int
	MaxEmbedsPerTick int
	Threshold        float64
	Floor            pkg.FloorConfig
	RerankEnabled    bool
	Reranker         *pkg.RerankerChain
	RerankShortlist  int
	RerankBlend      bool
	Sparse           pkg.SparseEncoder
	Collection       string
	Embedding        pkg.EmbeddingPolicy
	Compose          func(IntentRecord) string
	DisableFloor     bool
}

// NewService builds a Service from explicit components (not
// tuning-rebuildable; tests and recall experiments only).
func NewService(opts Options) *Service {
	return &Service{eng: buildEngine(opts)}
}

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

// buildEngineFromBase overlays the intent-domain wiring (source, binding,
// reconciler, rerank/fallback text) on a read-path base whose engine
// components and tuning factors are already resolved — the production
// caller forwards zero factor fields by hand.
func buildEngineFromBase(base pkg.ServiceOptions, opts Options) *engine {
	collection := opts.Collection
	if collection == "" {
		collection = DefaultCollection
	}
	source := newIntentSource(opts.Source, opts.Compose)

	hybrid := base.SparseEncoder != nil
	var binding pkg.SourceBinding
	if hybrid {
		binding = pkg.NewHybridBinding(intentKind, idPrefix, base.VectorStore, source,
			pkg.NewIdentityChunker(), pkg.NewIdentityComposer(), base.SparseEncoder)
	} else {
		binding = pkg.NewDenseBinding(intentKind, idPrefix, base.VectorStore, source)
	}
	rec := pkg.NewReconciler(base.Embedder, []pkg.SourceBinding{binding}, opts.Parallelism)
	rec.MaxEmbedsPerTick = opts.MaxEmbedsPerTick

	base.Reconciler = rec
	base.RerankText = func(r pkg.SearchResult) string { return candidateText(r.Payload) }
	base.TextFallback = intentTextFallback(opts.Source)
	// Project natural identity: results leave the engine keyed by the corpus
	// record id ("<scenario>/<OT|REQ|prd>"), not the derived point UUID —
	// this is what the eval grader (GradeSuite) and the wire hits match on.
	base.Project = func(r pkg.SearchResult) pkg.SearchResult {
		if id, _ := r.Payload["record_id"].(string); id != "" {
			r.ID = id
		}
		return r
	}
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
	return &engine{svc: svc, vectorStore: base.VectorStore, spec: spec, hybrid: hybrid}
}

// TunedOptions carries the non-tuning wiring (deployment facts, not search
// factors).
type TunedOptions struct {
	Source           IntentSource
	Parallelism      int
	MaxEmbedsPerTick int
	Compose          func(IntentRecord) string
	Collection       string
	EngineDeps       pkg.EngineDeps
}

// NewTunedService builds the production Service from a TuningConfig (the
// search.json SSOT) via the shared pkg.NewServiceForTuning — engine shape
// chosen by data, live-rebuildable via ApplyTuning.
func NewTunedService(tuning pkg.TuningConfig, opts TunedOptions) *Service {
	if opts.Collection == "" {
		opts.Collection = DefaultCollection
	}
	opts.EngineDeps.Collection = opts.Collection
	build := func(t pkg.TuningConfig) *engine {
		te := pkg.NewServiceForTuning(t, opts.EngineDeps)
		eng := buildEngineFromBase(te.ServiceOptions(), Options{
			Source:           opts.Source,
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

// Reconciler exposes the current engine's reconciler (sync loop resolves it
// each tick, so ApplyTuning re-points the loop).
func (s *Service) Reconciler() *pkg.Reconciler { return s.current().svc.Reconciler() }

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

// ApplyTuning rebuilds the engine for tuning and swaps it in place (see
// cli-health's worked example for the swap-safety notes: EnsureCollection
// BEFORE the swap; never auto-drop a collection).
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

// Search performs retrieval with AI-first, text-fallback semantics.
func (s *Service) Search(ctx context.Context, query string, limit int, mode SearchMode, opts ...pkg.SearchOption) (*SearchResponse, error) {
	if strings.TrimSpace(query) == "" {
		return &SearchResponse{Query: query, Method: "text"}, nil
	}
	e := s.current()
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
		method = "ai"
	}
	return &SearchResponse{
		Results:  hits,
		Total:    len(hits),
		Query:    query,
		Method:   method,
		Reranker: presp.Reranker,
	}, nil
}

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

// Status reports backend availability. "Available" means the AI leg works
// (ollama AND qdrant); the text fallback is a degradation, not available.
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

// Collection reports the live collection name (substrate declaration +
// status rendering).
func (s *Service) Collection() string { return s.current().spec.Name }

func candidateText(payload map[string]any) string {
	if body, _ := payload["body"].(string); strings.TrimSpace(body) != "" {
		return body
	}
	title, _ := payload["title"].(string)
	snippet, _ := payload["snippet"].(string)
	return strings.TrimSpace(title + " " + snippet)
}

// intentTextFallback is the offline-safe keyword leg over freshly loaded
// records — used when ollama or qdrant is unavailable.
func intentTextFallback(source IntentSource) pkg.TextFallbackFunc {
	return func(ctx context.Context, q pkg.SearchQuery) ([]pkg.SearchResult, error) {
		terms := tokenize(q.Query)
		if len(terms) == 0 {
			return nil, nil
		}
		records, err := source.LoadRecords(ctx)
		if err != nil {
			return nil, err
		}
		type scored struct {
			res   pkg.SearchResult
			score float64
			id    string
		}
		scoredHits := make([]scored, 0, 64)
		for _, r := range records {
			score := scoreRecord(r, terms)
			if score <= 0 {
				continue
			}
			scoredHits = append(scoredHits, scored{
				res:   pkg.SearchResult{ID: pointIDForIntent(r.ID), Score: score, Payload: intentMeta(r)},
				score: score,
				id:    r.ID,
			})
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

func scoreRecord(r IntentRecord, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	name := strings.ToLower(r.Scenario + " " + r.Title + " " + r.ID)
	desc := strings.ToLower(r.Snippet)
	purpose := strings.ToLower(r.ScenarioPurpose)
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
		if strings.Contains(purpose, t) {
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
	max := float64(len(terms)) * 1.7
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
