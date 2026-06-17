package aisearch

import (
	"context"
	"sort"
	"strings"

	pkg "github.com/vrooli/ai-go/search"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
)

// Service is SDA's dependency-governance search surface over the shared
// packages/ai-go/search engine. It indexes governance records as single-chunk
// dense sources and exposes RankIDs — a primitive ordering of record IDs the
// registry maps back to full records, applying facet filters and the governed
// projection. When Qdrant/Ollama are down the engine degrades to the keyword
// text fallback supplied here, so search never hard-fails.
type Service struct {
	svc         *pkg.Service
	vectorStore pkg.VectorStore
	spec        pkg.CollectionSpec
	provider    RecordProvider
}

// New assembles the dense governance-records search service. Boot resolves the
// EngineDeps embedding policy (ResolveEngineDepsEmbedding) before calling this.
// tuning chooses the engine recipe (dense for v1).
func New(provider RecordProvider, tuning pkg.TuningConfig, deps pkg.EngineDeps, parallelism, maxEmbedsPerTick int) *Service {
	deps.Collection = DefaultCollection
	te := pkg.NewServiceForTuning(tuning, deps)

	source := &recordSource{provider: provider}
	binding := pkg.NewDenseBinding(recordKind, idPrefix, te.VectorStore, source)
	rec := pkg.NewReconciler(te.Embedder, []pkg.SourceBinding{binding}, parallelism)
	rec.MaxEmbedsPerTick = maxEmbedsPerTick

	base := te.ServiceOptions()
	base.Reconciler = rec
	base.RerankText = func(r pkg.SearchResult) string { return candidateText(r.Payload) }
	base.TextFallback = textFallback(provider)
	// OverridePolicy left nil (DenyOverrides): SDA exposes no per-request tuning
	// override channel in v1.

	return &Service{
		svc:         pkg.NewService(base),
		vectorStore: te.VectorStore,
		spec:        te.Spec,
		provider:    provider,
	}
}

// Reconciler exposes the engine's reconciler for the sync loop.
func (s *Service) Reconciler() *pkg.Reconciler { return s.svc.Reconciler() }

// EnsureCollection is called once at startup; idempotent. Best-effort at boot —
// a down Qdrant degrades search to the text fallback rather than failing boot.
func (s *Service) EnsureCollection(ctx context.Context) error {
	return s.vectorStore.EnsureCollection(ctx, s.spec)
}

// Status reports backend availability. "available" means AI search works
// (Ollama AND Qdrant); the text fallback is a degradation, not "available".
func (s *Service) Status(ctx context.Context) (available, ollama, qdrant bool, indexed int) {
	r := s.svc.Status(ctx)
	return r.Ollama && r.Qdrant, r.Ollama, r.Qdrant, r.IndexedCount
}

// RankIDs runs the AI-first, text-fallback retrieval and returns the ordered
// record IDs (ecosystem/package) plus their relevance scores. available is false
// only when the query is empty or the engine errors — the caller then keeps its
// own keyword ordering. This is the primitive surface dependencygovernance
// consumes via the SemanticRanker interface (no import of this package).
func (s *Service) RankIDs(ctx context.Context, query string, limit int) (ids []string, scores map[string]float64, available bool, err error) {
	if strings.TrimSpace(query) == "" {
		return nil, nil, false, nil
	}
	if limit <= 0 {
		limit = 20
	}
	resp, err := s.svc.Search(ctx, pkg.SearchQuery{Query: query, Limit: limit, Mode: pkg.ModeAuto})
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

// sourceIDFromResult recovers the natural record ID (ecosystem/package) from a
// result payload — the engine stores it under source_id; fall back to composing
// it from the ecosystem/package_name meta fields.
func sourceIDFromResult(r pkg.SearchResult) string {
	if r.SourceID != "" {
		return strings.ToLower(strings.TrimSpace(r.SourceID))
	}
	if v, ok := r.Payload["source_id"].(string); ok && strings.TrimSpace(v) != "" {
		return strings.ToLower(strings.TrimSpace(v))
	}
	eco, _ := r.Payload["ecosystem"].(string)
	pkgName, _ := r.Payload["package_name"].(string)
	if strings.TrimSpace(eco) == "" || strings.TrimSpace(pkgName) == "" {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(eco) + "/" + strings.TrimSpace(pkgName))
}

// candidateText is the passage handed to the reranker for one hit.
func candidateText(payload map[string]any) string {
	if body, _ := payload["body"].(string); strings.TrimSpace(body) != "" {
		return body
	}
	eco, _ := payload["ecosystem"].(string)
	pkgName, _ := payload["package_name"].(string)
	return strings.TrimSpace(eco + "/" + pkgName)
}

// textFallback is the offline-safe keyword leg over the live governance records,
// used when Ollama/Qdrant are unavailable. It returns pkg.SearchResult with the
// record's natural ID under source_id so RankIDs treats it identically to a
// vector hit.
func textFallback(provider RecordProvider) pkg.TextFallbackFunc {
	return func(ctx context.Context, q pkg.SearchQuery) ([]pkg.SearchResult, error) {
		terms := tokenize(q.Query)
		if len(terms) == 0 {
			return nil, nil
		}
		records, err := provider.AllRecords(ctx)
		if err != nil {
			return nil, err
		}
		type scored struct {
			res   pkg.SearchResult
			score float64
			id    string
		}
		hits := make([]scored, 0, len(records))
		for _, r := range records {
			score := scoreRecord(r, terms)
			if score <= 0 {
				continue
			}
			id := recordID(r)
			payload := recordMeta(r)
			payload["source_id"] = id
			hits = append(hits, scored{
				res:   pkg.SearchResult{ID: pointIDForRecord(id), Score: score, Payload: payload, SourceID: id},
				score: score,
				id:    id,
			})
		}
		sort.Slice(hits, func(i, j int) bool {
			if hits[i].score != hits[j].score {
				return hits[i].score > hits[j].score
			}
			return hits[i].id < hits[j].id
		})
		out := make([]pkg.SearchResult, 0, len(hits))
		for _, h := range hits {
			out = append(out, h.res)
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

// scoreRecord is a weighted substring match: identity > keywords/use-cases >
// rationale/security. Normalized to (0,1].
func scoreRecord(r *governancev1.ApprovedDependencyRecord, terms []string) float64 {
	if len(terms) == 0 {
		return 0
	}
	identity := strings.ToLower(r.GetEcosystem() + " " + r.GetPackageName())
	vocab := strings.ToLower(strings.Join(r.GetKeywords(), " ") + " " + strings.Join(r.GetUseCases(), " "))
	prose := strings.ToLower(r.GetRationale() + " " + r.GetSecurityNotes())
	var score float64
	for _, t := range terms {
		hit := false
		if strings.Contains(identity, t) {
			score += 1.0
			hit = true
		}
		if strings.Contains(vocab, t) {
			score += 0.5
			hit = true
		}
		if strings.Contains(prose, t) {
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
	max := float64(len(terms)) * 1.8
	if max <= 0 {
		return 0
	}
	n := score / max
	if n > 1 {
		n = 1
	}
	return n
}
