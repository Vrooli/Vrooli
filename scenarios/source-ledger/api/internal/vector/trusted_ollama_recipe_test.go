package vector_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"source-ledger/internal/forest"
	"source-ledger/internal/journal"
	"source-ledger/internal/vector"

	"github.com/stretchr/testify/require"
	ollamapolicy "github.com/vrooli/ai-go/ollama/policy"
	apidb "github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
)

type sourceLedgerPolicyFake struct {
	role  ollamapolicy.ResolvedRole
	model ollamapolicy.ResolvedRole
	err   error
}

func (f sourceLedgerPolicyFake) ResolveRole(context.Context, string) (ollamapolicy.ResolvedRole, error) {
	return f.role, f.err
}

func (f sourceLedgerPolicyFake) ResolveModel(context.Context, string) (ollamapolicy.ResolvedRole, error) {
	return f.model, f.err
}

type sourceLedgerTaskEmbedder struct{ calls int }

func (e *sourceLedgerTaskEmbedder) EmbedDocument(_ context.Context, text string) ([]float64, error) {
	e.calls++
	if !strings.Contains(text, "source document") {
		return nil, errors.New("unexpected document text")
	}
	vector := make([]float64, 768)
	vector[0], vector[1] = 3, 4
	return vector, nil
}

func (e *sourceLedgerTaskEmbedder) EmbedQuery(_ context.Context, text string) ([]float64, error) {
	e.calls++
	if !strings.Contains(text, "source query") {
		return nil, errors.New("unexpected query text")
	}
	vector := make([]float64, 768)
	vector[0], vector[1] = 3, 4
	return vector, nil
}

func TestTrustedSourceLedgerRuntimeBindsCurrentRecipeAndExactDocumentEncoder(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(forest.Schema), apidb.SchemaProviderFunc(vector.Schema)))
	resolved := trustedSourceLedgerRole()
	fakeEmbedder := &sourceLedgerTaskEmbedder{}
	runtime, err := vector.NewTrustedSourceLedgerOllamaRuntime(db, sourceLedgerPolicyFake{role: resolved, model: resolved}, fakeEmbedder)
	require.NoError(t, err)

	recipe, err := runtime.ResolveCurrentRecipe(ctx)
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(recipe.ID(), "sha256:"))
	input, err := recipe.Input()
	require.NoError(t, err)
	require.Equal(t, "nomic-embed-text:latest", input.ModelID)
	require.Equal(t, 768, input.Dimensions)
	require.Equal(t, "search_document: ", input.DocumentInstruction)

	vectorOut, err := runtime.EmbedDocument(ctx, recipe.ID(), "source document")
	require.NoError(t, err)
	require.Len(t, vectorOut, 768)
	require.InDelta(t, .6, vectorOut[0], 1e-12)
	require.InDelta(t, .8, vectorOut[1], 1e-12)
	require.Equal(t, 1, fakeEmbedder.calls)
	queryVector, err := runtime.EmbedQueryForRecipe(ctx, recipe.ID(), "source query")
	require.NoError(t, err)
	require.Len(t, queryVector, 768)
	require.InDelta(t, .6, queryVector[0], 1e-12)
	require.InDelta(t, .8, queryVector[1], 1e-12)
	require.Equal(t, 2, fakeEmbedder.calls)

	_, err = runtime.EmbedDocument(ctx, recipe.ID(), strings.Repeat("é", 1801))
	require.Error(t, err)
	_, err = runtime.EmbedQueryForRecipe(ctx, recipe.ID(), strings.Repeat("é", 1801))
	require.Error(t, err)
	require.Equal(t, 2, fakeEmbedder.calls, "oversized inputs must be rejected before provider invocation")
}

func TestTrustedSourceLedgerRuntimeRejectsRightsAndRuntimeDrift(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(forest.Schema), apidb.SchemaProviderFunc(vector.Schema)))
	resolved := trustedSourceLedgerRole()
	resolved.RuntimeIdentity.Digest = strings.Repeat("a", 64)
	runtime, err := vector.NewTrustedSourceLedgerOllamaRuntime(db, sourceLedgerPolicyFake{role: resolved}, &sourceLedgerTaskEmbedder{})
	require.NoError(t, err)
	_, err = runtime.ResolveCurrentRecipe(ctx)
	require.Error(t, err)

	resolved = trustedSourceLedgerRole()
	resolved.Rights.LicenseID = "unknown"
	runtime, err = vector.NewTrustedSourceLedgerOllamaRuntime(db, sourceLedgerPolicyFake{role: resolved}, &sourceLedgerTaskEmbedder{})
	require.NoError(t, err)
	_, err = runtime.ResolveCurrentRecipe(ctx)
	require.Error(t, err)
}

func TestTrustedSourceLedgerRecipeBindsEveryIdentityField(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(forest.Schema), apidb.SchemaProviderFunc(vector.Schema)))
	cases := []struct {
		name   string
		mutate func(*ollamapolicy.ResolvedRole)
		bound  bool
	}{
		{"exact owner resolution", func(*ollamapolicy.ResolvedRole) {}, true},
		{"observation times and obligations are not identity", func(r *ollamapolicy.ResolvedRole) {
			r.Rights.Evidence.ObservedAt, r.RuntimeIdentity.ObservedAt = "2026-11-13T00:00:00Z", "2026-11-13T00:00:00Z"
			r.Rights.Obligations = []string{"attribution"}
		}, true},
		{"another role", func(r *ollamapolicy.ResolvedRole) { r.Role = "model" }, false},
		{"another context window", func(r *ollamapolicy.ResolvedRole) { r.ContextWindowTokens = 4096 }, false},
		{"missing rights", func(r *ollamapolicy.ResolvedRole) { r.Rights = nil }, false},
		{"missing runtime identity", func(r *ollamapolicy.ResolvedRole) { r.RuntimeIdentity = nil }, false},
		{"policy digest is not sha256", func(r *ollamapolicy.ResolvedRole) { r.PolicySHA256 = "B" + strings.Repeat("b", 63) }, false},
		{"restricted classification", func(r *ollamapolicy.ResolvedRole) { r.Rights.Classification = "restricted" }, false},
		{"license evidence from another page", func(r *ollamapolicy.ResolvedRole) { r.Rights.Evidence.URL = "https://example.com" }, false},
		{"license evidence with another digest", func(r *ollamapolicy.ResolvedRole) { r.Rights.Evidence.SHA256 = strings.Repeat("c", 64) }, false},
		{"runtime never observed", func(r *ollamapolicy.ResolvedRole) { r.RuntimeIdentity.ObservedAt = "" }, false},
		{"runtime of another revision", func(r *ollamapolicy.ResolvedRole) { r.RuntimeIdentity.ArtifactRevision = "other" }, false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			resolved := trustedSourceLedgerRole()
			tc.mutate(&resolved)
			runtime, err := vector.NewTrustedSourceLedgerOllamaRuntime(db, sourceLedgerPolicyFake{role: resolved}, &sourceLedgerTaskEmbedder{})
			require.NoError(t, err)
			_, err = runtime.ResolveCurrentRecipe(ctx)
			require.Equal(t, tc.bound, err == nil, "err=%v", err)
		})
	}
}

func TestTrustedSourceLedgerRuntimeReResolvesPersistedRecipe(t *testing.T) {
	ctx := context.Background()
	db := dbtest.NewSQLite(t)
	require.NoError(t, apidb.EnsureSchemas(ctx, db,
		apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(forest.Schema), apidb.SchemaProviderFunc(vector.Schema)))
	seedGenerationSources(t, db, "facet-a", "source document alpha", "summary-a", "source document summary")
	modelLevel := trustedSourceLedgerRole()
	modelLevel.Role = "model"
	runtime, err := vector.NewTrustedSourceLedgerOllamaRuntime(db, sourceLedgerPolicyFake{role: trustedSourceLedgerRole(), model: modelLevel}, &sourceLedgerTaskEmbedder{})
	require.NoError(t, err)
	backend, err := vector.NewSQLiteGenerationBackend(db, runtime, runtime, newGenerationDiskAdmission(), vector.SQLiteGenerationPeakEstimate{}, vector.SQLiteGenerationOptions{})
	require.NoError(t, err)
	staged, err := backend.Stage(ctx, "trusted-stage")
	require.NoError(t, err)
	require.Equal(t, "candidate", staged.State)

	// The current role no longer resolves, but the persisted recipe still binds
	// through a model-level resolution of the same pinned artifact.
	retired := trustedSourceLedgerRole()
	retired.Role = "chat.default"
	runtime, err = vector.NewTrustedSourceLedgerOllamaRuntime(db, sourceLedgerPolicyFake{role: retired, model: modelLevel}, &sourceLedgerTaskEmbedder{})
	require.NoError(t, err)
	recipe, err := runtime.ResolveRecipe(ctx, staged.RecipeID)
	require.NoError(t, err)
	require.Equal(t, staged.RecipeID, recipe.ID())
	_, err = runtime.ResolveRecipe(ctx, "sha256:"+strings.Repeat("0", 64))
	require.Error(t, err, "an unregistered recipe that is not current must not resolve")

	drifted := modelLevel
	drifted.RuntimeIdentity = &ollamapolicy.RuntimeArtifactIdentity{}
	*drifted.RuntimeIdentity = *modelLevel.RuntimeIdentity
	drifted.RuntimeIdentity.Digest = strings.Repeat("a", 64)
	runtime, err = vector.NewTrustedSourceLedgerOllamaRuntime(db, sourceLedgerPolicyFake{role: trustedSourceLedgerRole(), model: drifted}, &sourceLedgerTaskEmbedder{})
	require.NoError(t, err)
	_, err = runtime.ResolveRecipe(ctx, staged.RecipeID)
	require.Error(t, err, "a persisted recipe must not resolve once the installed artifact drifts")
}

func trustedSourceLedgerRole() ollamapolicy.ResolvedRole {
	const manifest = "0a109f422b47e3a30ba2b10eca18548e944e8a23073ee3f3e947efcf3c45e59f"
	const license = "c71d239df91726fc519c6eb72d318ec65820627232b2f796219e87dcf35d0ab4"
	artifact := "registry.ollama.ai/library/nomic-embed-text:latest"
	revision := "ollama-manifest:sha256:" + manifest
	return ollamapolicy.ResolvedRole{
		Role: "embedding.default", Model: "nomic-embed-text:latest", EmbeddingDimensions: 768,
		ContextWindowTokens: 2048, PolicySHA256: strings.Repeat("b", 64),
		Rights:          &ollamapolicy.ArtifactRights{ArtifactID: artifact, Revision: revision, SHA256: manifest, LicenseID: "Apache-2.0", Classification: "unrestricted", Evidence: ollamapolicy.RightsEvidence{URL: "https://ollama.com/library/nomic-embed-text:latest", Revision: manifest, SHA256: license}},
		RuntimeIdentity: &ollamapolicy.RuntimeArtifactIdentity{Provider: "ollama", Model: "nomic-embed-text:latest", ArtifactID: artifact, ArtifactRevision: revision, DigestKind: "ollama_model_contents", Digest: manifest, Source: "ollama /api/tags", ObservedAt: "2026-11-12T00:00:00Z"},
	}
}
