package aisearch

import (
	"context"
	"testing"
)

func TestNewDenseEngineDerivesSpecFromDefaults(t *testing.T) {
	cfg := Config{
		QdrantURL:    "http://q",
		QdrantAPIKey: "",
		EmbedModel:   "nomic-embed-text",
		RerankRole:   "rerank.llm_fallback",
	}
	engine := NewDenseEngine(cfg, "my-collection")

	if engine.Embedder == nil {
		t.Fatal("Embedder must be wired")
	}
	if engine.VectorStore == nil {
		t.Fatal("VectorStore must be wired")
	}
	if engine.Reranker == nil {
		t.Fatal("Reranker chain must be wired")
	}

	// The spec is derived so it cannot drift from the store/embedder.
	if engine.Spec.Name != "my-collection" {
		t.Errorf("Spec.Name = %q, want %q", engine.Spec.Name, "my-collection")
	}
	if engine.Spec.DenseSize != DefaultVectorSize {
		t.Errorf("Spec.DenseSize = %d, want %d", engine.Spec.DenseSize, DefaultVectorSize)
	}
	if engine.Spec.DenseDistance != DefaultDenseDistance {
		t.Errorf("Spec.DenseDistance = %q, want %q", engine.Spec.DenseDistance, DefaultDenseDistance)
	}
	if engine.Spec.Model != cfg.EmbedModel {
		t.Errorf("Spec.Model = %q, want %q (must match the wired embedder)", engine.Spec.Model, cfg.EmbedModel)
	}
	if engine.Spec.Sparse {
		t.Error("dense-only engine must not declare a sparse vector")
	}
}

// TestNewDenseEngineSpecMatchesStore confirms the derived spec passes the WS2
// name cross-check against its own store (the foot-gun the helper closes).
func TestNewDenseEngineSpecMatchesStore(t *testing.T) {
	doer := &capturingDoer{respond: func(req capturedReq) (int, string) {
		if req.method == "GET" {
			return 404, "{}" // force creation; no schema-guard path
		}
		return 200, "{}"
	}}
	engine := NewDenseEngine(Config{EmbedModel: "nomic-embed-text"}, "cli-health-commands")
	// Swap in the capturing store so EnsureCollection is exercised offline while
	// still validating the derived spec.Name against the store collection.
	store := NewVectorStoreWithClient("http://q", "", "cli-health-commands", doer)
	if err := store.EnsureCollection(context.Background(), engine.Spec); err != nil {
		t.Fatalf("derived spec must pass EnsureCollection's name cross-check, got %v", err)
	}
}
