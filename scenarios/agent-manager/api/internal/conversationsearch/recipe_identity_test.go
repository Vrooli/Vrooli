package conversationsearch

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
	aisearch "github.com/vrooli/ai-go/search"
	"github.com/vrooli/ai-go/search/searchtest"
)

const (
	testManifestDigest = "0a109f422b47e3a30ba2b10eca18548e944e8a23073ee3f3e947efcf3c45e59f"
	testEmbeddingModel = "nomic-embed-text:latest"
)

func policyForTest() ollamapolicy.ResolvedRole {
	return ollamapolicy.ResolvedRole{
		SchemaVersion: "2026-06-10", Role: "embedding.default", Model: testEmbeddingModel,
		ContextWindowTokens: 8192, EmbeddingDimensions: 768,
		RuntimeIdentity: &ollamapolicy.RuntimeArtifactIdentity{
			Provider: "ollama", Model: testEmbeddingModel, ArtifactID: "registry.ollama.ai/library/" + testEmbeddingModel,
			ArtifactRevision: "ollama-manifest:sha256:" + testManifestDigest, DigestKind: "ollama_model_contents",
			Digest: testManifestDigest, Source: "ollama /api/tags",
		},
	}
}

func specForTest() aisearch.CollectionSpec {
	return aisearch.CollectionSpec{Model: testEmbeddingModel, Role: "embedding.default", DenseSize: 768, PolicySchemaVersion: "2026-06-10"}
}

func legacyServingMetadata() aisearch.GenerationMetadata {
	return aisearch.GenerationMetadata{
		ID: "gen-legacy", Owner: ConversationSearchProviderID, Namespace: conversationCollectionDomain, Alias: guardTestAlias,
		Model: testEmbeddingModel, ChunkPolicy: DefaultRecipeVersion, EmbeddingRole: "embedding.default",
		EmbeddingDimensions: 768, PolicySchemaVersion: "2026-06-10",
	}
}

func confirmingSentinel() servingSentinel {
	return servingSentinel{Model: testEmbeddingModel, Role: "embedding.default", DenseSize: 768, PolicySchema: "2026-06-10"}
}

func mintedForTest(t *testing.T) aisearch.EmbeddingRecipe {
	t.Helper()
	recipe, err := mintConversationRecipe(policyForTest(), specForTest(), false)
	require.NoError(t, err)
	return recipe
}

func TestMintConversationRecipeBindsTheArtifactTheOwnerPolicyObserved(t *testing.T) {
	t.Parallel()
	recipe := mintedForTest(t)
	require.NoError(t, recipe.Verify())
	input, err := recipe.Input()
	require.NoError(t, err)
	require.Equal(t, testEmbeddingModel, input.ModelID)
	require.Equal(t, testManifestDigest, input.ArtifactSHA256, "the recipe must carry the observed manifest digest, not a placeholder")
	require.Equal(t, "ollama-manifest:sha256:"+testManifestDigest, input.ModelRevision)
	require.Equal(t, 768, input.Dimensions)
	require.Equal(t, DefaultRecipeVersion, input.ChunkerVersion)
	require.Equal(t, conversationComposerVersion, input.ComposerVersion)
	require.Equal(t, "none", input.QueryInstruction)
	require.Equal(t, recipeUnobserved, input.Precision, "fields the policy does not record are marked, not guessed")

	again, err := mintConversationRecipe(policyForTest(), specForTest(), false)
	require.NoError(t, err)
	require.Equal(t, recipe.ID(), again.ID(), "the same policy must always mint the same recipe")

	prefixed, err := mintConversationRecipe(policyForTest(), specForTest(), true)
	require.NoError(t, err)
	require.NotEqual(t, recipe.ID(), prefixed.ID(), "a task prefix changes every vector and so the recipe")

	changed := policyForTest()
	changed.RuntimeIdentity.Digest = strings.Repeat("b", 64)
	changed.RuntimeIdentity.ArtifactRevision = "ollama-manifest:sha256:" + changed.RuntimeIdentity.Digest
	other, err := mintConversationRecipe(changed, specForTest(), false)
	require.NoError(t, err)
	require.NotEqual(t, recipe.ID(), other.ID(), "a different artifact under the same model name is a different recipe")
}

func TestMintConversationRecipeRefusesWhatItCannotAttest(t *testing.T) {
	t.Parallel()
	for name, mutate := range map[string]func(*ollamapolicy.ResolvedRole, *aisearch.CollectionSpec){
		"no runtime identity": func(r *ollamapolicy.ResolvedRole, _ *aisearch.CollectionSpec) { r.RuntimeIdentity = nil },
		"identity for a stranger": func(r *ollamapolicy.ResolvedRole, _ *aisearch.CollectionSpec) {
			r.RuntimeIdentity.Model = "other:latest"
		},
		"not the ollama runtime": func(r *ollamapolicy.ResolvedRole, _ *aisearch.CollectionSpec) { r.RuntimeIdentity.Provider = "vllm" },
		"model mismatch":         func(_ *ollamapolicy.ResolvedRole, s *aisearch.CollectionSpec) { s.Model = "other:latest" },
		"dimension mismatch":     func(_ *ollamapolicy.ResolvedRole, s *aisearch.CollectionSpec) { s.DenseSize = 384 },
		"no artifact revision": func(r *ollamapolicy.ResolvedRole, _ *aisearch.CollectionSpec) {
			r.RuntimeIdentity.ArtifactRevision = ""
		},
		"digest is not sha256": func(r *ollamapolicy.ResolvedRole, _ *aisearch.CollectionSpec) { r.RuntimeIdentity.Digest = "abc" },
	} {
		t.Run(name, func(t *testing.T) {
			resolved, spec := policyForTest(), specForTest()
			mutate(&resolved, &spec)
			_, err := mintConversationRecipe(resolved, spec, false)
			require.Error(t, err)
		})
	}
}

// The composer's output is part of every vector's identity. This pins it to the
// version string: change what Compose emits and this fails until
// conversationComposerVersion changes with it.
func TestComposerVersionPinsOutput(t *testing.T) {
	t.Parallel()
	got := (conversationEmbeddingComposer{}).Compose(aisearch.Chunk{
		Body: "the body",
		Meta: map[string]any{"role": "assistant", "run_label": "Admission review", "project_scope": "/workspace/alpha"},
	})
	require.Equal(t, "assistant · Admission review · /workspace/alpha\n\nthe body", got,
		"the composed embedding input changed; bump conversationComposerVersion (was %q)", conversationComposerVersion)
}

func TestLegacyAdoptionBlocker(t *testing.T) {
	t.Parallel()
	recipe := mintedForTest(t)
	for _, test := range []struct {
		name      string
		mutate    func(*aisearch.GenerationMetadata, *[]servingSentinel)
		wantBlock string
	}{
		{"confirmed legacy generation is adoptable", func(*aisearch.GenerationMetadata, *[]servingSentinel) {}, ""},
		{"untagged sentinel model is the latest tag", func(_ *aisearch.GenerationMetadata, s *[]servingSentinel) { (*s)[0].Model = "nomic-embed-text" }, ""},
		{"already has a recipe", func(m *aisearch.GenerationMetadata, _ *[]servingSentinel) { m.RecipeID = "sha256:x" }, "already carries"},
		{"catalog model differs", func(m *aisearch.GenerationMetadata, _ *[]servingSentinel) { m.Model = "granite-embedding:latest" }, "generation model"},
		{"catalog dimensions differ", func(m *aisearch.GenerationMetadata, _ *[]servingSentinel) { m.EmbeddingDimensions = 384 }, "generation dimensions"},
		{"chunk policy differs", func(m *aisearch.GenerationMetadata, _ *[]servingSentinel) { m.ChunkPolicy = "conversation-search-v2" }, "chunk policy"},
		{"no sentinel to confirm", func(_ *aisearch.GenerationMetadata, s *[]servingSentinel) { *s = nil }, "no meta sentinel"},
		{"collection built with another model", func(_ *aisearch.GenerationMetadata, s *[]servingSentinel) { (*s)[0].Model = "granite-embedding:latest" }, "built with model"},
		{"collection holds other dimensions", func(_ *aisearch.GenerationMetadata, s *[]servingSentinel) { (*s)[0].DenseSize = 384 }, "384-dimension"},
		{"collection recorded another policy schema", func(_ *aisearch.GenerationMetadata, s *[]servingSentinel) { (*s)[0].PolicySchema = "2025-01-01" }, "policy schema"},
		{"mixed collection: one ancestor differs", func(_ *aisearch.GenerationMetadata, s *[]servingSentinel) {
			*s = append(*s, servingSentinel{Model: "granite-embedding:latest", Role: "embedding.default", DenseSize: 768, PolicySchema: "2026-06-10"})
		}, "built with model"},
		{"collection already stamped with another recipe", func(_ *aisearch.GenerationMetadata, s *[]servingSentinel) { (*s)[0].RecipeID = "sha256:other" }, "different recipe"},
		{"collection already stamped with this recipe", func(_ *aisearch.GenerationMetadata, s *[]servingSentinel) { (*s)[0].RecipeID = recipe.ID() }, ""},
	} {
		t.Run(test.name, func(t *testing.T) {
			metadata, sentinels := legacyServingMetadata(), []servingSentinel{confirmingSentinel()}
			test.mutate(&metadata, &sentinels)
			got := legacyAdoptionBlocker(metadata, recipe, sentinels)
			if test.wantBlock == "" {
				require.Empty(t, got)
			} else {
				require.Contains(t, got, test.wantBlock)
			}
		})
	}
}

func TestReadServingSentinelsAsksTheCollectionItself(t *testing.T) {
	t.Parallel()
	var path, apiKey string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path, apiKey = r.URL.Path, r.Header.Get("api-key")
		_ = json.NewDecoder(r.Body).Decode(&body)
		_, _ = w.Write([]byte(`{"result":{"points":[
			{"payload":{"__aisearch_meta__":true,"model":"nomic-embed-text:latest","role":"embedding.default","dense_size":768,"policy_schema_version":"2026-06-10","embedding_recipe_id":""}},
			{"payload":{"__aisearch_meta__":true,"model":"nomic-embed-text:latest","role":"embedding.default","dense_size":768,"policy_schema_version":"2026-06-10"}}]}}`))
	}))
	defer server.Close()

	sentinels, err := readServingSentinels(t.Context(), server.URL+"/", "secret", guardTestAlias+"--generation--g1")
	require.NoError(t, err)
	require.Len(t, sentinels, 2)
	require.Equal(t, confirmingSentinel(), sentinels[0])
	require.Equal(t, "/collections/"+guardTestAlias+"--generation--g1/points/scroll", path)
	require.Equal(t, "secret", apiKey)
	require.Contains(t, body, "filter", "only meta sentinels may be read, never conversation points")
	require.Equal(t, false, body["with_vector"])
}

func TestReadServingSentinelsReportsAnUnreachableOrRefusingStore(t *testing.T) {
	t.Parallel()
	refusing := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { http.Error(w, "no", http.StatusNotFound) }))
	defer refusing.Close()
	_, err := readServingSentinels(t.Context(), refusing.URL, "", "missing")
	require.ErrorContains(t, err, "status 404")
	_, err = readServingSentinels(t.Context(), "", "", "missing")
	require.Error(t, err)
}

func TestAdoptLegacyRecipeStampsTheServingGenerationAndNothingElse(t *testing.T) {
	ctx := context.Background()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	base := time.Date(2026, 11, 20, 12, 0, 0, 0, time.UTC)
	row := legacyGenerationRow{ID: "gen-legacy", CreatedAt: base, Model: testEmbeddingModel}
	insertLegacyActiveGeneration(t, db, row)
	_, err := db.Exec(`UPDATE conversation_search_qdrant_generations SET embedding_dimensions=768, policy_schema_version='2026-06-10', recipe_id='', recipe_canonical='' WHERE generation_id='gen-legacy'`)
	require.NoError(t, err)

	active, err := loadActiveRecipeRecord(ctx, catalog, guardTestAlias)
	require.NoError(t, err)
	require.Equal(t, 1, generationServingRank(active.Metadata), "the fixture is the legacy shape")
	recipe := mintedForTest(t)

	reason, err := adoptLegacyRecipe(ctx, catalog, active, recipe, []servingSentinel{{Model: "granite-embedding:latest", Role: "embedding.default", DenseSize: 768, PolicySchema: "2026-06-10"}})
	require.NoError(t, err)
	require.Contains(t, reason, "built with model")
	stored, err := loadActiveRecipeRecord(ctx, catalog, guardTestAlias)
	require.NoError(t, err)
	require.Equal(t, 1, generationServingRank(stored.Metadata), "a declined adoption must not change the catalog")

	reason, err = adoptLegacyRecipe(ctx, catalog, active, recipe, []servingSentinel{confirmingSentinel()})
	require.NoError(t, err)
	require.Empty(t, reason)
	stored, err = loadActiveRecipeRecord(ctx, catalog, guardTestAlias)
	require.NoError(t, err)
	require.Equal(t, 2, generationServingRank(stored.Metadata), "the serving generation now carries a verified recipe")
	require.Equal(t, recipe.ID(), stored.Metadata.RecipeID)
	require.True(t, completeGenerationRecipe(stored.Metadata))
	require.Equal(t, "gen-legacy", stored.Metadata.ID)

	// The next process start reads the stamped record as a verified recipe.
	again := *stored
	reason, err = adoptLegacyRecipe(ctx, catalog, &again, recipe, []servingSentinel{confirmingSentinel()})
	require.NoError(t, err)
	require.Contains(t, reason, "already carries")
}

func TestHealthExplainsWhyALegacyGenerationWasNotAdopted(t *testing.T) {
	t.Parallel()
	legacy := &aisearch.GenerationRecord{Metadata: legacyServingMetadata(), CollectionName: guardTestAlias + "--generation--gen-legacy"}
	runtime := &SemanticRuntime{Collection: guardTestAlias, recipeAdoptionNote: "recipe adoption declined: the serving collection was built with model \"x\""}
	runtime.recordGenerationHealth(ActiveGenerationReconciliation{ActiveBefore: 1}, true, nil, legacy)
	line := runtime.GenerationHealthSnapshot().Degradation()
	require.Contains(t, line, "predates immutable recipe identity")
	require.Contains(t, line, "recipe adoption declined: the serving collection was built with model")

	// A stamped generation is healthy: the note is gone, not reworded.
	recipe := mintedForTest(t)
	stamped := *legacy
	stamped.Metadata.RecipeID, stamped.Metadata.RecipeCanonical = recipe.ID(), string(recipe.Canonical())
	runtime = &SemanticRuntime{Collection: guardTestAlias}
	runtime.recordGenerationHealth(ActiveGenerationReconciliation{ActiveBefore: 1}, true, nil, &stamped)
	require.Empty(t, runtime.GenerationHealthSnapshot().Degradation())
}

func TestHealthCoverageIsNeverPrintedAsZeroBeforeItIsObserved(t *testing.T) {
	t.Parallel()
	health := SemanticGenerationHealth{State: GenerationHealthRecovered, Detail: "serving gen", EligibleDocuments: 60_000}
	require.Contains(t, health.Degradation(), "vector coverage not yet observed")
	require.NotContains(t, health.Degradation(), "vectors 0/")
	health.EmbeddedObserved, health.EmbeddedDocuments = true, 59_000
	require.Contains(t, health.Degradation(), "vectors 59000/60000")
	health.EmbeddedDocuments = 0
	require.Contains(t, health.Degradation(), "vectors 0/60000", "an observed empty collection is real zero coverage")
}

// A reconciliation pass rewrites catalog health but must keep the measured
// counts, or status flips between coverage and "not observed".
func TestGenerationHealthRecordingKeepsObservedCoverage(t *testing.T) {
	t.Parallel()
	runtime := &SemanticRuntime{Collection: guardTestAlias}
	runtime.recordSemanticCoverage(t.Context(), 59_000, true)
	runtime.healthMu.Lock()
	runtime.health.EligibleDocuments = 60_000
	runtime.healthMu.Unlock()
	runtime.recordGenerationHealth(ActiveGenerationReconciliation{ActiveBefore: 1}, true, nil, &aisearch.GenerationRecord{Metadata: legacyServingMetadata()})
	health := runtime.GenerationHealthSnapshot()
	require.True(t, health.EmbeddedObserved)
	require.Equal(t, uint64(59_000), health.EmbeddedDocuments)
	require.Equal(t, uint64(60_000), health.EligibleDocuments)
}

// Every incremental generation is created with the serving recipe, so the
// catalog row (and the candidate collection's sentinel) never lose it.
type metadataRecordingStore struct {
	began []aisearch.GenerationMetadata
}

func (s *metadataRecordingStore) BeginGeneration(_ context.Context, metadata aisearch.GenerationMetadata) error {
	s.began = append(s.began, metadata)
	return nil
}
func (s *metadataRecordingStore) LookupActiveSources(context.Context, []string) (map[string]aisearch.StoredSourceState, error) {
	return map[string]aisearch.StoredSourceState{}, nil
}
func (s *metadataRecordingStore) StageSource(context.Context, string, aisearch.GenerationSourceWrite) error {
	return nil
}
func (s *metadataRecordingStore) StageDelete(context.Context, string, string) error { return nil }
func (s *metadataRecordingStore) ValidateGeneration(context.Context, string) (aisearch.GenerationValidation, error) {
	return aisearch.GenerationValidation{Valid: true}, nil
}
func (s *metadataRecordingStore) PromoteGeneration(context.Context, string) error  { return nil }
func (s *metadataRecordingStore) RollbackGeneration(context.Context, string) error { return nil }
func (s *metadataRecordingStore) CleanupGenerations(context.Context, int) error    { return nil }

func TestIncrementalGenerationIsCreatedWithTheServingRecipe(t *testing.T) {
	t.Parallel()
	recipe := mintedForTest(t)
	store := &metadataRecordingStore{}
	embedder := &searchtest.Embedder{Vector: []float64{1, 0, 0}, AvailableValue: true}
	runtime := &SemanticRuntime{
		Engine:              aisearch.TunedEngine{Spec: specForTest()},
		GenerationStore:     store,
		StreamingBinding:    aisearch.StreamingBinding{Kind: "conversation", Store: store, Source: searchtest.PagedSource{}, Chunker: aisearch.NewIdentityChunker(), Composer: conversationEmbeddingComposer{}, PageSize: 10},
		StreamingReconciler: aisearch.NewStreamingReconciler(embedder),
		EmbeddingRecipeID:   recipe.ID(), EmbeddingRecipeJSON: string(recipe.Canonical()),
	}
	require.NoError(t, runtime.RebuildChanges(t.Context(), "gen-next", nil, []string{"doc-removed"}, nil))
	require.Len(t, store.began, 1)
	require.Equal(t, recipe.ID(), store.began[0].RecipeID)
	require.Equal(t, string(recipe.Canonical()), store.began[0].RecipeCanonical)
	require.NoError(t, verifyGenerationRecipe(store.began[0]), "the recorded identity must verify against its own canonical bytes")
}
