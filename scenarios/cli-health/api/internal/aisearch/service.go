package aisearch

import (
	"context"
	"sort"
	"strings"

	pkg "github.com/vrooli/aisearch-go"
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

// Service is cli-health's command-search surface. It embeds the shared read-path
// engine (*pkg.Service) — which owns the query/floor/rerank/fallback pipeline and
// the reindex job control — and adds only the command-domain shaping: the
// typed-hit projection, the discovery text fallback, and the ai/text mode vocab.
type Service struct {
	*pkg.Service
	vectorStore pkg.VectorStore
	spec        pkg.CollectionSpec
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
	// operator override; FloorForLeg supplies the regime default).
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
}

// NewService builds a Service over the shared engine. cli-health indexes commands
// as single-chunk, dense-only sources (the NewDenseBinding common case); the
// shared read path applies the relevance floor (commands are cosine/cross-encoder
// scored, not RRF-fused) and reranks when a chain is supplied.
func NewService(opts Options) *Service {
	binding := pkg.NewDenseBinding(commandKind, idPrefix, opts.VectorStore, newCommandSource(opts.Discovery))
	rec := pkg.NewReconciler(opts.Embedder, []pkg.SourceBinding{binding}, opts.Parallelism)
	rec.MaxEmbedsPerTick = opts.MaxEmbedsPerTick

	engine := pkg.NewService(pkg.ServiceOptions{
		Embedder:      opts.Embedder,
		VectorStore:   opts.VectorStore,
		Reconciler:    rec,
		RerankEnabled: opts.RerankEnabled,
		Reranker:      opts.Reranker,
		ApplyFloor:    true,
		Floor:         opts.Floor,
		Threshold:     opts.Threshold,
		Shortlist:     opts.RerankShortlist,
		RerankText:    func(r pkg.SearchResult) string { return candidateText(r.Payload) },
		TextFallback:  commandTextFallback(opts.Discovery),
	})
	return &Service{
		Service:     engine,
		vectorStore: opts.VectorStore,
		spec: pkg.CollectionSpec{
			Name:          DefaultCollection,
			DenseSize:     pkg.DefaultVectorSize,
			DenseDistance: pkg.DefaultDenseDistance,
			Model:         pkg.DefaultEmbedModel,
		},
	}
}

// EnsureCollection is called once at startup; idempotent.
func (s *Service) EnsureCollection(ctx context.Context) error {
	return s.vectorStore.EnsureCollection(ctx, s.spec)
}

// Search performs retrieval with AI-first, text-fallback semantics, projecting
// the shared engine's results back into cli-health's typed command hits.
func (s *Service) Search(ctx context.Context, query string, limit int, mode SearchMode) (*SearchResponse, error) {
	if strings.TrimSpace(query) == "" {
		return &SearchResponse{Query: query, Method: "text"}, nil
	}
	// SearchTyped projects each finished result into cli-health's typed command hit
	// at the engine boundary (the shared generic), so this surface owns no
	// projection loop and cannot drift from the engine's result ordering.
	hits, presp, err := pkg.SearchTyped(ctx, s.Service, pkg.SearchQuery{Query: query, Limit: limit, Mode: serviceMode(mode)},
		func(r pkg.SearchResult) SearchHit {
			hit := payloadToHit(r.ID, r.Score, r.Payload)
			hit.Weak = r.Weak
			return hit
		})
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
// "ai" is the dense vector leg (cli-health has no sparse vector).
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
// cli-health's wire shape. cli-health's "available" means AI search works, so it
// requires BOTH ollama and qdrant (the engine's default "qdrant OR text" is KO's
// looser doc-search semantics); the text fallback is a degradation, not
// "available".
func (s *Service) Status(ctx context.Context) StatusReport {
	r := s.Service.Status(ctx)
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
