// Package findingindex adopts the shared aisearch-go engine to give the
// findings knowledge store semantic search. It mirrors cli-health's adoption
// pattern: a pkg.Source over the SQLite findings table (embedding text = claim
// + citation titles), a dense engine assembled from a TuningConfig, a reconcile
// loop, and a typed read path. Only active + disputed findings are indexed, so
// a superseded finding drops out of qdrant on the next reconcile and the
// default search never surfaces it.
package findingindex

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"

	"web-search/internal/findings"

	pkg "github.com/vrooli/ai-go/search"
)

const (
	// DefaultCollection is the qdrant collection backing the findings index.
	DefaultCollection = "web-search-findings"
	findingKind       = "finding"
	idPrefix          = "web-search-findings:"
)

// Loader returns the findings eligible for indexing (active + disputed).
type Loader func(ctx context.Context) ([]findings.Finding, error)

// Options carries the non-tuning wiring the index needs.
type Options struct {
	Loader           Loader
	EngineDeps       pkg.EngineDeps
	Parallelism      int
	MaxEmbedsPerTick int
}

// Service is the findings semantic-search read path plus its reconcile engine.
type Service struct {
	svc         *pkg.Service
	vectorStore pkg.VectorStore
	spec        pkg.CollectionSpec
	reconciler  *pkg.Reconciler
	embedder    pkg.Embedder
}

// Hit is one semantic-search result projected to the finding id + score.
type Hit struct {
	FindingID string
	Score     float64
	Weak      bool
}

// New assembles the dense findings index from a TuningConfig.
func New(tuning pkg.TuningConfig, opts Options) *Service {
	deps := opts.EngineDeps
	if deps.Collection == "" {
		deps.Collection = DefaultCollection
	}
	te := pkg.NewServiceForTuning(tuning, deps)

	source := &findingSource{load: opts.Loader}
	binding := pkg.NewDenseBinding(findingKind, idPrefix, te.VectorStore, source)
	rec := pkg.NewReconciler(te.Embedder, []pkg.SourceBinding{binding}, opts.Parallelism)
	if opts.MaxEmbedsPerTick > 0 {
		rec.MaxEmbedsPerTick = opts.MaxEmbedsPerTick
	}

	svc := pkg.NewService(pkg.ServiceOptions{
		Embedder:      te.Embedder,
		VectorStore:   te.VectorStore,
		Reconciler:    rec,
		RerankEnabled: te.Tuning.RerankEnabled,
		RerankBlend:   te.Tuning.RerankBlend,
		Shortlist:     te.Tuning.RerankShortlist,
		Reranker:      te.Reranker,
		ApplyFloor:    true,
		Floor:         te.Tuning.Floor.Config(),
		RerankText:    func(r pkg.SearchResult) string { return claimText(r.Payload) },
	})

	return &Service{svc: svc, vectorStore: te.VectorStore, spec: te.Spec, reconciler: rec, embedder: te.Embedder}
}

// Embedder exposes the tuned embedder (same model + task-prefix recipe as the
// index) for other query-relevance consumers — e.g. the L2 relevance-aware
// excerpter — so they score in the same vector space the corpus lives in.
func (s *Service) Embedder() pkg.Embedder { return s.embedder }

// EnsureCollection creates the qdrant collection if absent. Best-effort at boot.
func (s *Service) EnsureCollection(ctx context.Context) error {
	return s.vectorStore.EnsureCollection(ctx, s.spec)
}

// Reconciler exposes the reconcile engine for the sync loop + reindex.
func (s *Service) Reconciler() *pkg.Reconciler { return s.reconciler }

// Search runs the semantic read path and projects each result to a finding id.
func (s *Service) Search(ctx context.Context, query string, limit int) ([]Hit, string, error) {
	if strings.TrimSpace(query) == "" {
		return nil, "text", nil
	}
	hits, resp, err := pkg.SearchTyped(ctx, s.svc, pkg.SearchQuery{Query: query, Limit: limit},
		func(r pkg.SearchResult) Hit {
			id, _ := r.Payload["finding_id"].(string)
			return Hit{FindingID: id, Score: r.Score, Weak: r.Weak}
		})
	if err != nil {
		return nil, "", err
	}
	return hits, resp.Method, nil
}

// findingSource adapts the findings store to the engine's Source interface.
type findingSource struct {
	load Loader
}

func (s *findingSource) LoadAll(ctx context.Context) ([]pkg.SourceDoc, error) {
	items, err := s.load(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]pkg.SourceDoc, 0, len(items))
	for _, f := range items {
		out = append(out, pkg.SourceDoc{
			ID:          f.ID,
			Kind:        findingKind,
			ContentHash: contentHash(f),
			Body:        composeEmbeddingText(f),
			Meta: map[string]any{
				"finding_id": f.ID,
				"status":     f.Status,
				"claim":      f.Claim,
			},
		})
	}
	return out, nil
}

// composeEmbeddingText is the input handed to the embedder: the claim followed
// by the citation titles, which carry the real-world vocabulary a query uses.
func composeEmbeddingText(f findings.Finding) string {
	parts := []string{f.Claim}
	titles := citationTitles(f)
	if len(titles) > 0 {
		parts = append(parts, "Sources: "+strings.Join(titles, "; "))
	}
	return strings.Join(parts, "\n\n")
}

func citationTitles(f findings.Finding) []string {
	var titles []string
	for _, c := range f.Citations {
		t := strings.TrimSpace(c.Title)
		if t != "" {
			titles = append(titles, t)
		}
	}
	return titles
}

// contentHash gates the source-level drift skip: a stable hash of the fields
// that affect the embedding or the index membership.
func contentHash(f findings.Finding) string {
	var b strings.Builder
	b.WriteString(f.ID)
	b.WriteByte('\n')
	b.WriteString(f.Status)
	b.WriteByte('\n')
	b.WriteString(f.Claim)
	b.WriteByte('\n')
	b.WriteString(strings.Join(citationTitles(f), "|"))
	b.WriteByte('\n')
	b.WriteString(f.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000000000Z07:00"))
	sum := sha256.Sum256([]byte(b.String()))
	return "sha256:" + hex.EncodeToString(sum[:8])
}

func claimText(payload map[string]any) string {
	s, _ := payload["claim"].(string)
	return s
}
