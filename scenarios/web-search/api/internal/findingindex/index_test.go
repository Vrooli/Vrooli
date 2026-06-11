package findingindex

import (
	"context"
	"fmt"
	"hash/fnv"
	"math"
	"sort"
	"strings"
	"sync"
	"testing"

	"web-search/internal/clock"
	localdb "web-search/internal/database"
	"web-search/internal/findings"
	testdb "web-search/internal/testutil/db"

	"github.com/stretchr/testify/require"
	pkg "github.com/vrooli/ai-go/search"
	apidb "github.com/vrooli/api-core/database"
)

// fakeDims is the dense vector size the hermetic fake embedder produces.
const fakeDims = 64

// fakeEmbedder is a deterministic bag-of-words embedder: token overlap between
// two texts yields cosine similarity, so semantic recall is exercised without
// a live ollama. It is symmetric (no task prefixes), matching the default
// embedder contract.
type fakeEmbedder struct{}

func (fakeEmbedder) Embed(_ context.Context, text string) ([]float64, error) {
	vec := make([]float64, fakeDims)
	for _, tok := range strings.Fields(strings.ToLower(text)) {
		tok = strings.Trim(tok, ".,;:()\"'")
		if tok == "" {
			continue
		}
		h := fnv.New32a()
		_, _ = h.Write([]byte(tok))
		vec[int(h.Sum32())%fakeDims]++
	}
	var norm float64
	for _, v := range vec {
		norm += v * v
	}
	if norm > 0 {
		norm = math.Sqrt(norm)
		for i := range vec {
			vec[i] /= norm
		}
	}
	return vec, nil
}

func (fakeEmbedder) Available(context.Context) bool { return true }

// memStore is an in-memory pkg.VectorStore honoring the same payload contract
// as qdrant (ScrollIDs projects payload_hash / source_id / source_hash /
// chunk_total) so the reconciler's drift + ghost-deletion logic runs for real.
// Query ranks by dot product over the (normalized) dense vectors — cosine.
type memStore struct {
	mu     sync.Mutex
	points map[string]pkg.Point
}

func newMemStore() *memStore { return &memStore{points: map[string]pkg.Point{}} }

func (m *memStore) EnsureCollection(context.Context, pkg.CollectionSpec) error { return nil }

func (m *memStore) Upsert(_ context.Context, p pkg.Point) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.points[p.ID] = p
	return nil
}

func (m *memStore) SetPayload(_ context.Context, id string, payload map[string]any) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if p, ok := m.points[id]; ok {
		p.Payload = payload
		m.points[id] = p
	}
	return nil
}

func (m *memStore) BatchDelete(_ context.Context, ids []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, id := range ids {
		delete(m.points, id)
	}
	return nil
}

func (m *memStore) Query(_ context.Context, q pkg.HybridQuery) ([]pkg.SearchResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]pkg.SearchResult, 0, len(m.points))
	for id, p := range m.points {
		var dot float64
		for i := range q.Dense {
			if i < len(p.Dense) {
				dot += q.Dense[i] * p.Dense[i]
			}
		}
		out = append(out, pkg.SearchResult{ID: id, Score: dot, Payload: p.Payload})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Score > out[j].Score })
	if q.Limit > 0 && len(out) > q.Limit {
		out = out[:q.Limit]
	}
	return out, nil
}

func (m *memStore) CountPoints(context.Context) (int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.points), nil
}

func (m *memStore) ScrollIDs(context.Context) (map[string]pkg.ScrollItem, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make(map[string]pkg.ScrollItem, len(m.points))
	for id, p := range m.points {
		hash, _ := p.Payload["payload_hash"].(string)
		srcID, _ := p.Payload["source_id"].(string)
		srcHash, _ := p.Payload["source_hash"].(string)
		total, _ := p.Payload["chunk_total"].(int)
		out[id] = pkg.ScrollItem{PayloadHash: hash, SourceID: srcID, SourceHash: srcHash, ChunkTotal: total}
	}
	return out, nil
}

func (m *memStore) Available(context.Context) bool { return true }

var _ pkg.VectorStore = (*memStore)(nil)

// newTestIndex assembles the findingindex Service with the same source /
// binding / reconciler / read-path shape New builds, but over the fake
// embedder + in-memory store, so the full write -> reconcile -> search
// lifecycle runs hermetically.
func newTestIndex(loader Loader) (*Service, *memStore) {
	store := newMemStore()
	emb := fakeEmbedder{}
	source := &findingSource{load: loader}
	binding := pkg.NewDenseBinding(findingKind, idPrefix, store, source)
	rec := pkg.NewReconciler(emb, []pkg.SourceBinding{binding}, 1)
	svc := pkg.NewService(pkg.ServiceOptions{
		Embedder:    emb,
		VectorStore: store,
		Reconciler:  rec,
		RerankText:  func(r pkg.SearchResult) string { return claimText(r.Payload) },
	})
	return &Service{
		svc:         svc,
		vectorStore: store,
		spec:        pkg.CollectionSpec{Name: DefaultCollection, DenseSize: fakeDims},
		reconciler:  rec,
	}, store
}

// newFindingsRepo builds a findings repository over a fresh schema-complete
// SQLite database.
func newFindingsRepo(t *testing.T) findings.Repository {
	t.Helper()
	d := testdb.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(findings.Schema),
	))
	return findings.NewSQLiteRepository(d, clock.System{})
}

// TestWriteReindexSemanticRecall is the OT-P0-005 store/index round trip:
// write a finding into the web-search SQLite store, run the index reconcile,
// issue a semantic query, and verify the finding is recalled at the top of
// the results — and that the same finding is also retrievable by exact id.
func TestWriteReindexSemanticRecall(t *testing.T) {
	ctx := context.Background()
	repo := newFindingsRepo(t)

	target, err := repo.Add(ctx, findings.NewFinding{
		Claim:     "Anthropic released the Claude Opus 4.8 model in May 2026",
		Source:    findings.SourceL3,
		Citations: []findings.NewCitation{{URL: "https://anthropic.example", Title: "Anthropic announcement"}},
	}, "agent")
	require.NoError(t, err)
	for i := 0; i < 5; i++ {
		_, err := repo.Add(ctx, findings.NewFinding{
			Claim:  fmt.Sprintf("unrelated distractor fact number %d about gardening soil acidity", i),
			Source: findings.SourceManual,
		}, "operator")
		require.NoError(t, err)
	}

	idx, store := newTestIndex(repo.LoadIndexable)
	require.NoError(t, idx.EnsureCollection(ctx))
	_, _, err = idx.Reconciler().RunOnce(ctx)
	require.NoError(t, err)
	n, err := store.CountPoints(ctx)
	require.NoError(t, err)
	require.Equal(t, 6, n, "every active finding is indexed")

	hits, method, err := idx.Search(ctx, "claude opus model release", 3)
	require.NoError(t, err)
	require.Equal(t, "dense", method)
	require.NotEmpty(t, hits)
	require.Equal(t, target.ID, hits[0].FindingID, "the semantically nearest finding is recalled first")

	// The same finding is retrievable by exact id from the store.
	got, err := repo.Get(ctx, target.ID)
	require.NoError(t, err)
	require.Equal(t, target.Claim, got.Claim)
	require.Equal(t, findings.SourceL3, got.Source)
}

// TestReconcileLifecycleMirrorsCliHealthPattern pins the aisearch-go adoption
// contract (the cli-health Source/Service pattern): only active + disputed
// findings are registered in the index; an edit drifts the content hash and
// re-embeds; a supersede drops the finding from the index on the next
// reconcile (ghost deletion) so the default search can never surface it.
func TestReconcileLifecycleMirrorsCliHealthPattern(t *testing.T) {
	ctx := context.Background()
	repo := newFindingsRepo(t)

	active, err := repo.Add(ctx, findings.NewFinding{Claim: "active retrievable claim"}, "operator")
	require.NoError(t, err)
	disputed, err := repo.Add(ctx, findings.NewFinding{Claim: "disputed but indexed claim"}, "operator")
	require.NoError(t, err)
	_, err = repo.Flag(ctx, disputed.ID, "contested", "operator")
	require.NoError(t, err)
	gone, err := repo.Add(ctx, findings.NewFinding{Claim: "doomed ephemeral assertion"}, "operator")
	require.NoError(t, err)

	idx, store := newTestIndex(repo.LoadIndexable)
	require.NoError(t, idx.EnsureCollection(ctx))
	_, _, err = idx.Reconciler().RunOnce(ctx)
	require.NoError(t, err)
	n, err := store.CountPoints(ctx)
	require.NoError(t, err)
	require.Equal(t, 3, n, "active + disputed are indexed; nothing else exists yet")

	// Supersede -> next reconcile drops the point (ghost deletion).
	_, err = repo.Supersede(ctx, gone.ID, active.ID, "outdated", "operator")
	require.NoError(t, err)
	_, _, err = idx.Reconciler().RunOnce(ctx)
	require.NoError(t, err)
	n, err = store.CountPoints(ctx)
	require.NoError(t, err)
	require.Equal(t, 2, n, "the superseded finding drops out of the index")
	hits, _, err := idx.Search(ctx, "doomed ephemeral assertion", 5)
	require.NoError(t, err)
	for _, h := range hits {
		require.NotEqual(t, gone.ID, h.FindingID, "a superseded finding is never recalled")
	}

	// Edit -> content hash drifts -> the chunk is re-embedded under new text.
	_, err = repo.Edit(ctx, active.ID, findings.EditInput{Claim: "active claim now about volcanic geology", Confidence: 0.7}, "operator")
	require.NoError(t, err)
	_, _, err = idx.Reconciler().RunOnce(ctx)
	require.NoError(t, err)
	hits, _, err = idx.Search(ctx, "volcanic geology", 1)
	require.NoError(t, err)
	require.NotEmpty(t, hits)
	require.Equal(t, active.ID, hits[0].FindingID, "the re-embedded claim is retrievable under its new text")
}

// BenchmarkSemanticSearch10k measures the semantic read path over a
// 10,000-finding index (hermetic fake embedder + in-memory store, so the
// figure is the in-process budget excluding qdrant/ollama network time). It
// guards the OT-P0-005 top-10 recall latency budget against read-path
// complexity regressions. Run: go test -bench Semantic ./internal/findingindex
func BenchmarkSemanticSearch10k(b *testing.B) {
	ctx := context.Background()
	store := newMemStore()
	emb := fakeEmbedder{}
	for i := 0; i < 10000; i++ {
		text := fmt.Sprintf("finding %d about topic-%d and subject-%d", i, i%97, i%31)
		vec, err := emb.Embed(ctx, text)
		if err != nil {
			b.Fatal(err)
		}
		if err := store.Upsert(ctx, pkg.Point{
			ID:    fmt.Sprintf("web-search-findings:%05d:0", i),
			Dense: vec,
			Payload: map[string]any{
				"finding_id":   fmt.Sprintf("f-%05d", i),
				"claim":        text,
				"payload_hash": "h",
				"source_id":    fmt.Sprintf("f-%05d", i),
				"chunk_total":  1,
			},
		}); err != nil {
			b.Fatal(err)
		}
	}
	svc := pkg.NewService(pkg.ServiceOptions{Embedder: emb, VectorStore: store})
	idx := &Service{svc: svc, vectorStore: store, spec: pkg.CollectionSpec{Name: DefaultCollection, DenseSize: fakeDims}}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hits, _, err := idx.Search(ctx, fmt.Sprintf("findings about topic-%d", i%97), 10)
		if err != nil {
			b.Fatal(err)
		}
		if len(hits) == 0 {
			b.Fatal("expected top-10 hits over the 10k index")
		}
	}
}
