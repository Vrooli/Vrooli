package aisearch

import (
	"bytes"
	"strings"
	"testing"
)

func TestEmbeddingRecipeCanonicalIdentityIsStableAndImmutable(t *testing.T) {
	input := validEmbeddingRecipeInput()
	first, err := NewEmbeddingRecipe(input)
	if err != nil {
		t.Fatal(err)
	}
	second, err := NewEmbeddingRecipe(validEmbeddingRecipeInput())
	if err != nil {
		t.Fatal(err)
	}
	if first.ID() != second.ID() || !bytes.Equal(first.Canonical(), second.Canonical()) {
		t.Fatalf("identical recipe inputs produced different identities: %s != %s", first.ID(), second.ID())
	}

	input.ModelRevision = "mutated-caller-input"
	copyOfCanonical := first.Canonical()
	copyOfCanonical[0] = '!'
	if err := first.Verify(); err != nil {
		t.Fatalf("caller mutation changed immutable recipe: %v", err)
	}
	fields, err := first.Input()
	if err != nil {
		t.Fatal(err)
	}
	if fields.ModelRevision != "commit-17" {
		t.Fatalf("stored recipe changed via caller copy: %q", fields.ModelRevision)
	}
	fields.ModelRevision = "mutated-detached-copy"
	if err := first.Verify(); err != nil {
		t.Fatalf("detached input copy changed recipe: %v", err)
	}
}

func TestEmbeddingRecipeRejectsMalformedAndChangedPersistedBytes(t *testing.T) {
	recipe, err := NewEmbeddingRecipe(validEmbeddingRecipeInput())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := EmbeddingRecipeFromCanonical(recipe.Canonical(), strings.Repeat("0", 64)); err == nil {
		t.Fatal("accepted recipe bytes with a different claimed digest")
	}
	if _, err := EmbeddingRecipeFromCanonical([]byte(`{"model_id":"only"}`), ""); err == nil {
		t.Fatal("accepted incomplete recipe")
	}
	changed := bytes.Replace(recipe.Canonical(), []byte("commit-17"), []byte("commit-18"), 1)
	if _, err := EmbeddingRecipeFromCanonical(changed, recipe.Digest()); err == nil {
		t.Fatal("accepted modified recipe with stale digest")
	}
	noncanonical := append([]byte(" "), recipe.Canonical()...)
	if _, err := EmbeddingRecipeFromCanonical(noncanonical, recipe.Digest()); err == nil {
		t.Fatal("accepted non-canonical recipe encoding")
	}
}

func TestEmbeddingRecipeRejectsMissingRecipeIdentityFields(t *testing.T) {
	input := validEmbeddingRecipeInput()
	input.ArtifactSHA256 = "not-a-digest"
	if _, err := NewEmbeddingRecipe(input); err == nil || !strings.Contains(err.Error(), "artifact_sha256") {
		t.Fatalf("expected artifact digest validation error, got %v", err)
	}
	input = validEmbeddingRecipeInput()
	input.Dimensions = 0
	if _, err := NewEmbeddingRecipe(input); err == nil || !strings.Contains(err.Error(), "dimensions") {
		t.Fatalf("expected dimensions validation error, got %v", err)
	}
}

func validEmbeddingRecipeInput() EmbeddingRecipeInput {
	return EmbeddingRecipeInput{
		SchemaVersion: "embedding-recipe-v1", ModelID: "local/e5-small", ModelRevision: "commit-17", ArtifactSHA256: strings.Repeat("a", 64),
		TokenizerRevision: "tokenizer-4", Runtime: "onnx", RuntimeBuild: "onnxruntime-1.24",
		Precision: "FP32", QueryInstruction: "query: ", DocumentInstruction: "passage: ",
		Pooling: "mean", Normalization: "l2", Dimensions: 384, Projection: "none",
		OutputDType: "float32", OutputQuantization: "none", Truncation: "512-tokens",
		ChunkerVersion: "chunker-v2", ComposerVersion: "composer-v3",
	}
}
