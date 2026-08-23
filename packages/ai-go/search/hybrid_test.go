package aisearch

import (
	"context"
	"errors"
	"testing"
)

type lexicalSearchFunc func(context.Context, SearchQuery) ([]SearchResult, error)

func (f lexicalSearchFunc) SearchLexical(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	return f(ctx, query)
}

type semanticSearchFunc func(context.Context, SearchQuery) ([]SearchResult, error)

func (f semanticSearchFunc) SearchSemantic(ctx context.Context, query SearchQuery) ([]SearchResult, error) {
	return f(ctx, query)
}

func TestNewHybridEngineAlwaysDeclaresSparse(t *testing.T) {
	cfg := Config{
		QdrantURL:  "http://q",
		EmbedModel: fixtureEmbeddingModel,
	}
	engine := NewHybridEngine(cfg, "vrooli-docs")

	if engine.Embedder == nil || engine.VectorStore == nil || engine.Reranker == nil {
		t.Fatal("embedder, store, and reranker chain must all be wired")
	}
	if engine.SparseEncoder == nil {
		t.Fatal("hybrid engine must wire a sparse encoder")
	}
	// The whole point of the hybrid assembler: the sparse axis cannot be silently
	// forgotten.
	if !engine.Spec.Sparse {
		t.Error("Spec.Sparse must be true for a hybrid engine")
	}
	if engine.Spec.SparseModifier != DefaultSparseModifier {
		t.Errorf("Spec.SparseModifier = %q, want %q", engine.Spec.SparseModifier, DefaultSparseModifier)
	}
	if engine.Spec.Name != "vrooli-docs" {
		t.Errorf("Spec.Name = %q, want vrooli-docs", engine.Spec.Name)
	}
	if engine.Spec.DenseSize != DefaultVectorSize || engine.Spec.DenseDistance != DefaultDenseDistance {
		t.Errorf("dense leg must keep defaults: size=%d distance=%q", engine.Spec.DenseSize, engine.Spec.DenseDistance)
	}
	if engine.Spec.Model != cfg.EmbedModel {
		t.Errorf("Spec.Model = %q, want %q", engine.Spec.Model, cfg.EmbedModel)
	}
}

func TestNewHybridBindingWiresSparse(t *testing.T) {
	store := NewVectorStore("http://q", "", "vrooli-docs")
	src := nilSource{}
	b := NewHybridBinding("doc", "ko:", store, src, NewIdentityChunker(), NewIdentityComposer(), NewBM25SparseEncoder())

	if b.Kind != "doc" || b.IDPrefix != "ko:" {
		t.Errorf("kind/prefix not threaded: %q/%q", b.Kind, b.IDPrefix)
	}
	if b.Sparse == nil {
		t.Fatal("hybrid binding must carry a sparse encoder")
	}
	if b.Chunker == nil || b.Composer == nil {
		t.Fatal("hybrid binding must carry the chunker and composer")
	}
}

func TestAssertCollectionMatchesBindingCatchesSilentCliff(t *testing.T) {
	store := NewVectorStore("http://q", "", "vrooli-docs")
	hybrid := NewHybridBinding("doc", "ko:", store, nilSource{}, NewIdentityChunker(), NewIdentityComposer(), NewBM25SparseEncoder())

	// A hybrid binding against a dense-only spec is the cliff — must be a loud error.
	denseSpec := CollectionSpec{Name: "vrooli-docs", DenseSize: DefaultVectorSize, DenseDistance: DefaultDenseDistance}
	if err := AssertCollectionMatchesBinding(hybrid, denseSpec); err == nil {
		t.Fatal("expected a loud error when a hybrid binding targets a dense-only spec")
	}

	// The matching hybrid spec (what NewHybridEngine produces) must pass.
	hybridSpec := NewHybridEngine(Config{EmbedModel: fixtureEmbeddingModel}, "vrooli-docs").Spec
	if err := AssertCollectionMatchesBinding(hybrid, hybridSpec); err != nil {
		t.Fatalf("hybrid binding against a sparse spec must pass, got %v", err)
	}

	// A dense binding never trips the guard.
	dense := NewDenseBinding("command", "cli:", store, nilSource{})
	if err := AssertCollectionMatchesBinding(dense, denseSpec); err != nil {
		t.Fatalf("dense binding against a dense spec must pass, got %v", err)
	}
}

func TestEnsureCollectionForBindingRejectsBeforeQdrant(t *testing.T) {
	// The store URL is bogus; if the assertion did NOT fire first, EnsureCollection
	// would attempt a network call. The hybrid/dense mismatch must short-circuit.
	store := NewVectorStore("http://127.0.0.1:0", "", "vrooli-docs")
	hybrid := NewHybridBinding("doc", "ko:", store, nilSource{}, NewIdentityChunker(), NewIdentityComposer(), NewBM25SparseEncoder())
	denseSpec := CollectionSpec{Name: "vrooli-docs", DenseSize: DefaultVectorSize, DenseDistance: DefaultDenseDistance}
	if err := EnsureCollectionForBinding(context.Background(), store, hybrid, denseSpec); err == nil {
		t.Fatal("EnsureCollectionForBinding must reject the mismatch before touching Qdrant")
	}
}

func TestConcurrentFusionRunsLegsTogetherAndExplainsRanks(t *testing.T) {
	started := make(chan string, 2)
	release := make(chan struct{})
	fusion := ConcurrentFusion{
		Lexical: lexicalSearchFunc(func(context.Context, SearchQuery) ([]SearchResult, error) {
			started <- "lexical"
			<-release
			return []SearchResult{{ID: "shared", Score: 0.8}, {ID: "lexical-only", Score: 0.7}}, nil
		}),
		Semantic: semanticSearchFunc(func(context.Context, SearchQuery) ([]SearchResult, error) {
			started <- "semantic"
			<-release
			return []SearchResult{{ID: "semantic-only", Score: 0.9}, {ID: "shared", Score: 0.6}}, nil
		}),
	}
	type outcome struct {
		response FusionResponse
		err      error
	}
	done := make(chan outcome, 1)
	go func() {
		response, err := fusion.Search(context.Background(), SearchQuery{Query: "hybrid", Limit: 3})
		done <- outcome{response: response, err: err}
	}()
	seen := map[string]bool{<-started: true, <-started: true}
	if !seen["lexical"] || !seen["semantic"] {
		t.Fatalf("both legs must start before either is released: %v", seen)
	}
	close(release)
	got := <-done
	if got.err != nil {
		t.Fatal(got.err)
	}
	if len(got.response.Results) != 3 || got.response.Results[0].Result.ID != "shared" {
		t.Fatalf("shared result should win reciprocal-rank fusion: %+v", got.response.Results)
	}
	if len(got.response.Results[0].Evidence) != 2 {
		t.Fatalf("fused result must explain both ranks: %+v", got.response.Results[0].Evidence)
	}
}

func TestConcurrentFusionDegradesToHealthyLeg(t *testing.T) {
	fusion := ConcurrentFusion{
		Lexical: lexicalSearchFunc(func(context.Context, SearchQuery) ([]SearchResult, error) {
			return nil, errors.New("lexical unavailable")
		}),
		Semantic: semanticSearchFunc(func(context.Context, SearchQuery) ([]SearchResult, error) {
			return []SearchResult{{ID: "semantic", Score: 0.9}}, nil
		}),
	}
	response, err := fusion.Search(context.Background(), SearchQuery{Query: "fallback"})
	if err != nil {
		t.Fatal(err)
	}
	if len(response.Results) != 1 || len(response.Degraded) != 1 || response.Degraded[0] != "lexical" {
		t.Fatalf("expected truthful semantic-only degradation, got %+v", response)
	}
}

func TestWeightedAdmissionHonorsCancellation(t *testing.T) {
	admission := NewWeightedAdmission(1)
	release, err := admission.Acquire(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := admission.Acquire(ctx, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("queued admission must return context cancellation, got %v", err)
	}
	release()
}
