package aisearch

import (
	"context"
	"testing"
)

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
