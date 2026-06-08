package aisearch

import "testing"

// TestApplyRerankRRFDoesNotBuryStrongRetrieval is the core property the blend
// exists for: a result the reranker scores LOW but retrieval ranked HIGH must
// not be buried to the bottom (the pure-reorder failure mode). Retrieval order
// is [A,B,C] (A best); the reranker prefers B>C>A.
func TestApplyRerankRRFDoesNotBuryStrongRetrieval(t *testing.T) {
	hits := []SearchResult{
		{ID: "A", Score: 0.9},
		{ID: "B", Score: 0.5},
		{ID: "C", Score: 0.4},
	}
	scores := []RerankScore{{ID: "A", Score: 0.10}, {ID: "B", Score: 0.99}, {ID: "C", Score: 0.50}}

	pure := ApplyRerank(cloneResults(hits), scores)
	if pure[len(pure)-1].ID != "A" {
		t.Fatalf("precondition: pure rerank should bury A last, got %v", blendIDs(pure))
	}

	blended := ApplyRerankRRF(cloneResults(hits), scores, DefaultRRFK)
	if blended[0].ID != "B" {
		t.Errorf("blend should keep the reranker's top (B) first, got %v", blendIDs(blended))
	}
	// A (retrieval rank 0, rerank rank 2) must outrank C (retrieval 2, rerank 1).
	posA, posC := indexOf(blended, "A"), indexOf(blended, "C")
	if posA >= posC {
		t.Errorf("blend should rescue strongly-retrieved A above C, got %v", blendIDs(blended))
	}
}

func TestApplyRerankRRFEmptyScoresKeepsOrder(t *testing.T) {
	hits := []SearchResult{{ID: "A"}, {ID: "B"}}
	out := ApplyRerankRRF(hits, nil, DefaultRRFK)
	if out[0].ID != "A" || out[1].ID != "B" {
		t.Errorf("no scores => retrieval order preserved, got %v", blendIDs(out))
	}
}

func TestBuildChunkPayloadRecipeAffectsHash(t *testing.T) {
	chunk := Chunk{SourceID: "s", Index: 0, Body: "vrooli scenario list", Meta: map[string]any{"k": "v"}}

	base := buildChunkPayload(chunk, "vrooli scenario list", "src", 1, "")
	again := buildChunkPayload(chunk, "vrooli scenario list", "src", 1, "")
	withRecipe := buildChunkPayload(chunk, "vrooli scenario list", "src", 1, "model=nomic|q=search_query: |d=search_document: ")

	if base[payloadHashKey] != again[payloadHashKey] {
		t.Fatal("empty-recipe hash must be stable across calls")
	}
	if base[payloadHashKey] == withRecipe[payloadHashKey] {
		t.Error("a non-empty recipe must change the drift hash (forces re-embed on prefix change)")
	}
	// Back-compat: empty recipe == the legacy composePayloadHash over the text.
	if base[payloadHashKey] != composePayloadHash("vrooli scenario list", base) {
		t.Error("empty recipe must reproduce the legacy hash exactly")
	}
}

func TestEmbedRecipeEmptyWhenSymmetric(t *testing.T) {
	sym := &cliEmbedder{model: "nomic-embed-text"} // no prefixes
	if got := embedderRecipe(sym); got != "" {
		t.Errorf("symmetric embedder recipe must be empty, got %q", got)
	}
	if r := embedderRecipe(NewEmbedder("nomic-embed-text")); r != "" {
		t.Errorf("default NewEmbedder is symmetric => empty recipe, got %q", r)
	}
	nomic := NewEmbedderForConfig(Config{EmbedModel: "nomic-embed-text", EmbedTaskPrefix: true}).(RecipeEmbedder)
	if nomic.EmbedRecipe() == "" {
		t.Error("prefixed nomic embedder must report a non-empty recipe")
	}
}

// helpers
func cloneResults(in []SearchResult) []SearchResult {
	out := make([]SearchResult, len(in))
	copy(out, in)
	return out
}

func blendIDs(rs []SearchResult) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.ID
	}
	return out
}

func indexOf(rs []SearchResult, id string) int {
	for i, r := range rs {
		if r.ID == id {
			return i
		}
	}
	return -1
}
