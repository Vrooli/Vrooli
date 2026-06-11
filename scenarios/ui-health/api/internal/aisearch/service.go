package aisearch

import (
	"context"
	"sort"
	"strings"

	pkg "github.com/vrooli/ai-go/search"
)

// DefaultCollection is the Qdrant collection ui-health indexes its surfaces
// into. Single named-dense vector layout (the shared engine's CollectionSpec).
const DefaultCollection = "ui-health-surface"

// DefaultSearchThreshold drops dense hits whose cosine score is below ~0.55.
// Empirically, scores from nomic-embed-text + the ui-surface corpus settle in
// the 0.45–0.65 band; matches under 0.55 are effectively random and crowd out
// the "(no matches)" signal a user needs when asking about something the corpus
// does not contain. It is the dense-leg ScoreThreshold (operational wiring), not
// a tuning factor — the SSOT search.json owns engine/rerank/floor.
const DefaultSearchThreshold = 0.55

// SearchMode tags how Search should retrieve results.
type SearchMode string

const (
	ModeAuto SearchMode = "auto"
	ModeAI   SearchMode = "ai"
	ModeText SearchMode = "text"
)

// Options carries the non-tuning wiring NewSearchService needs: the surface
// discovery source, the reconcile knobs, the dense ScoreThreshold, and the
// Qdrant endpoints. These are deployment facts, not search factors (those live
// in search.json), so they stay outside the TuningConfig.
type Options struct {
	Discovery        DiscoverySource
	Parallelism      int
	MaxEmbedsPerTick int
	Threshold        float64
	EngineDeps       pkg.EngineDeps
}

// Service is ui-health's thin command-domain wrapper over the shared read-path
// engine (pkg.Service). It owns only surface-domain shaping (typed-hit
// projection, discovery text fallback, ai/text mode vocab); the engine owns
// embedding, the vector store, reconciliation, and the read pipeline.
type Service struct {
	svc         *pkg.Service
	vectorStore pkg.VectorStore
	spec        pkg.CollectionSpec
}

// NewSearchService assembles the Service FROM a TuningConfig (engine shape,
// embed recipe, rerank policy, floor band) via the shared pkg.NewServiceForTuning
// — the engine shape is chosen by data, not a code literal. ui-health's surfaces
// are a dense single-chunk corpus; the dense ScoreThreshold (opts.Threshold)
// trims sub-0.55 noise. The reranker is off by default for this corpus.
func NewSearchService(tuning pkg.TuningConfig, opts Options) *Service {
	if opts.EngineDeps.Collection == "" {
		opts.EngineDeps.Collection = DefaultCollection
	}
	te := pkg.NewServiceForTuning(tuning, opts.EngineDeps)
	base := te.ServiceOptions()

	source := newSurfaceSource(opts.Discovery)
	binding := pkg.NewDenseBinding(surfaceKind, idPrefix, base.VectorStore, source)
	rec := pkg.NewReconciler(base.Embedder, []pkg.SourceBinding{binding}, opts.Parallelism)
	rec.MaxEmbedsPerTick = opts.MaxEmbedsPerTick

	base.Reconciler = rec
	base.Threshold = opts.Threshold
	base.TextFallback = surfaceTextFallback(opts.Discovery)
	svc := pkg.NewService(base)

	return &Service{svc: svc, vectorStore: base.VectorStore, spec: te.Spec}
}

// Reconciler exposes the engine's reconciler (the sync loop resolves it each
// tick via this).
func (s *Service) Reconciler() *pkg.Reconciler { return s.svc.Reconciler() }

// EnsureCollection is called once at startup; idempotent.
func (s *Service) EnsureCollection(ctx context.Context) error {
	return s.vectorStore.EnsureCollection(ctx, s.spec)
}

// Search performs retrieval with AI-first, text-fallback semantics, projecting
// the shared engine's results back into ui-health's typed surface hits.
func (s *Service) Search(ctx context.Context, query string, limit int, mode SearchMode) (*SearchResponse, error) {
	if strings.TrimSpace(query) == "" {
		return &SearchResponse{Query: query, Method: "text"}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	// SearchTyped projects each finished result into ui-health's typed surface hit
	// at the engine boundary (the shared generic), so this surface owns no
	// projection loop and cannot drift from the engine's result ordering.
	hits, presp, err := pkg.SearchTyped(ctx, s.svc, pkg.SearchQuery{Query: query, Limit: limit, Mode: serviceMode(mode)},
		func(r pkg.SearchResult) SearchHit {
			return payloadToHit(r.ID, r.Score, r.Payload)
		})
	if err != nil {
		return nil, err
	}
	method := presp.Method
	if method == "dense" {
		method = "ai" // ui-health's wire vocab
	}
	return &SearchResponse{
		Results: hits,
		Total:   len(hits),
		Query:   query,
		Method:  method,
	}, nil
}

// serviceMode maps ui-health's ai/text/auto vocab to the engine's mode set.
// ui-health is dense-only, so "ai" is the dense leg.
func serviceMode(m SearchMode) pkg.SearchMode {
	switch m {
	case ModeAI:
		return pkg.ModeDense
	case ModeText:
		return pkg.ModeText
	default:
		return pkg.ModeAuto
	}
}

// Status reports backend availability, mapping the engine's StatusReport into
// ui-health's wire shape. ui-health's "available" requires BOTH ollama and qdrant
// AND a non-empty corpus — an empty index returns zero results regardless of the
// question, which is not a usable state even when the dependencies are reachable.
func (s *Service) Status(ctx context.Context) StatusReport {
	r := s.svc.Status(ctx)
	return StatusReport{
		Available:            r.Ollama && r.Qdrant && r.IndexedCount > 0,
		Ollama:               r.Ollama,
		Qdrant:               r.Qdrant,
		IndexedCount:         r.IndexedCount,
		LastReconcileAt:      r.LastReconcileAt,
		LastReconcileOutcome: r.LastReconcileOutcome,
	}
}

// Reindex / ReindexStatus / ReindexCancel / JobExport forward to the shared
// engine's read-path service (the reindex handler's Reindexer adapter calls
// these).
func (s *Service) Reindex(ctx context.Context, scenario string, dryRun bool) (*pkg.ReindexJob, error) {
	return s.svc.Reindex(ctx, scenario, dryRun)
}

func (s *Service) ReindexStatus(jobID string) (*pkg.ReindexJob, bool) {
	return s.svc.ReindexStatus(jobID)
}

func (s *Service) ReindexCancel(jobID string) bool { return s.svc.ReindexCancel(jobID) }

func (s *Service) JobExport(job *pkg.ReindexJob) map[string]any { return s.svc.JobExport(job) }

// surfaceTextFallback is the offline-safe keyword leg over freshly discovered
// surfaces (substring scoring). It returns pkg.SearchResult with the surface
// fields in the payload so the shared engine's pipeline (and the typed
// projection in Search) treat it identically to a vector hit. Used when ollama
// or qdrant is unavailable.
func surfaceTextFallback(discovery DiscoverySource) pkg.TextFallbackFunc {
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
					res:   pkg.SearchResult{ID: PointIDForSurface(r.Scenario, r.FilePath), Score: score, Payload: surfaceMeta(r)},
					score: score,
					path:  r.FilePath,
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

func scoreRecord(r SurfaceRecord, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	name := strings.ToLower(r.Scenario + " " + r.DisplayName + " " + r.Slot)
	desc := strings.ToLower(r.Description)
	file := strings.ToLower(r.FilePath)
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
		if strings.Contains(file, t) {
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
