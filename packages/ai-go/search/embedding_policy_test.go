package aisearch

import (
	"context"
	"testing"

	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
)

const (
	fixtureEmbeddingModel      = "fixture-embed-model:latest"
	fixtureEmbeddingDimensions = 1234
)

type embeddingPolicyRunner struct {
	t    *testing.T
	want []string
}

func (r embeddingPolicyRunner) Run(_ context.Context, name string, args ...string) ([]byte, error) {
	got := append([]string{name}, args...)
	if len(got) != len(r.want) {
		r.t.Fatalf("argv length = %d, want %d: %#v", len(got), len(r.want), got)
	}
	for i := range got {
		if got[i] != r.want[i] {
			r.t.Fatalf("argv[%d] = %q, want %q; argv=%#v", i, got[i], r.want[i], got)
		}
	}
	return []byte(`{"role":"embedding.default","source":"role","model":"fixture-embed-model:latest","capabilities":["embedding"],"embedding_dimensions":1234}`), nil
}

func TestConfigWithResolvedEmbedding(t *testing.T) {
	cfg, err := ConfigWithResolvedEmbedding(Config{EmbedModel: "old", EmbedRole: "old"}, ollamapolicy.ResolvedRole{
		Role:                "embedding.default",
		Model:               fixtureEmbeddingModel,
		EmbeddingDimensions: fixtureEmbeddingDimensions,
	})
	if err != nil {
		t.Fatalf("ConfigWithResolvedEmbedding returned error: %v", err)
	}
	if cfg.EmbedRole != "embedding.default" || cfg.EmbedModel != fixtureEmbeddingModel || cfg.EmbedDimensions != fixtureEmbeddingDimensions {
		t.Fatalf("cfg = %#v", cfg)
	}
}

func TestConfigWithResolvedEmbeddingRequiresDimensions(t *testing.T) {
	_, err := ConfigWithResolvedEmbedding(Config{}, ollamapolicy.ResolvedRole{Model: fixtureEmbeddingModel})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveEmbeddingConfigUsesRole(t *testing.T) {
	cfg, err := ResolveEmbeddingConfigWithResolver(context.Background(), Config{EmbedRole: "embedding.default"}, ollamapolicy.Resolver{
		Run: embeddingPolicyRunner{t: t, want: []string{"resource-ollama", "policy", "resolve", "--role", "embedding.default", "--json"}},
	})
	if err != nil {
		t.Fatalf("ResolveEmbeddingConfigWithResolver returned error: %v", err)
	}
	if cfg.EmbedRole != "embedding.default" || cfg.EmbedModel != fixtureEmbeddingModel || cfg.EmbedDimensions != fixtureEmbeddingDimensions {
		t.Fatalf("cfg = %#v", cfg)
	}
}

func TestNewServiceForTuningResolvedUsesRoleOnlyDeps(t *testing.T) {
	engine, err := NewServiceForTuningResolvedWithResolver(context.Background(), TuningConfig{Engine: EngineDense}, EngineDeps{
		QdrantURL:  "http://localhost:6333",
		Collection: "test-coll",
		EmbedRole:  "embedding.default",
	}, ollamapolicy.Resolver{
		Run: embeddingPolicyRunner{t: t, want: []string{"resource-ollama", "policy", "resolve", "--role", "embedding.default", "--json"}},
	})
	if err != nil {
		t.Fatalf("NewServiceForTuningResolvedWithResolver returned error: %v", err)
	}
	if engine.Spec.Model != fixtureEmbeddingModel || engine.Spec.DenseSize != fixtureEmbeddingDimensions {
		t.Fatalf("spec = %#v", engine.Spec)
	}
	if engine.Tuning.EmbedModel != fixtureEmbeddingModel {
		t.Fatalf("tuning embed model = %q", engine.Tuning.EmbedModel)
	}
}

func TestEnginesUseConfigEmbeddingDimensions(t *testing.T) {
	cfg := Config{EmbedModel: fixtureEmbeddingModel, EmbedDimensions: fixtureEmbeddingDimensions}
	dense := NewDenseEngine(cfg, "dense")
	if dense.Spec.DenseSize != fixtureEmbeddingDimensions {
		t.Fatalf("dense size = %d", dense.Spec.DenseSize)
	}
	hybrid := NewHybridEngine(cfg, "hybrid")
	if hybrid.Spec.DenseSize != fixtureEmbeddingDimensions {
		t.Fatalf("hybrid size = %d", hybrid.Spec.DenseSize)
	}
}
