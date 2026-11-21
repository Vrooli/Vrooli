package conversationsearch

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
)

func catalogTestRecipe(t *testing.T, model string, dimensions int) (string, string) {
	t.Helper()
	recipe, err := aisearch.NewEmbeddingRecipe(aisearch.EmbeddingRecipeInput{
		SchemaVersion: "embedding-recipe-v1", ModelID: model, ModelRevision: "revision-1",
		ArtifactSHA256: strings.Repeat("a", 64), TokenizerRevision: "tokenizer-1", Runtime: "resource-ollama",
		RuntimeBuild: "build-1", Precision: "fp32", QueryInstruction: "none", DocumentInstruction: "none",
		Pooling: "mean", Normalization: "l2", Dimensions: dimensions, OutputDType: "float32",
		Truncation: "512", ChunkerVersion: "identity-v1", ComposerVersion: "conversation-search-v3",
	})
	require.NoError(t, err)
	return recipe.ID(), string(recipe.Canonical())
}

func TestSQLiteQdrantCatalogPersistsLifecycleAndCleanupReceipt(t *testing.T) {
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	now := time.Date(2026, 9, 9, 7, 0, 0, 0, time.UTC)
	recipeID, recipeCanonical := catalogTestRecipe(t, "granite-embedding:latest", 384)
	record := aisearch.GenerationRecord{
		Metadata: aisearch.GenerationMetadata{
			ID: "gen-1", CreatedAt: now, Owner: "agent-manager.runs", Namespace: "conversation-search", Alias: "conversation-search-v1", ContentIdentity: "digest-1",
			Model: "granite-embedding:latest", RecipeID: recipeID, RecipeCanonical: recipeCanonical, ChunkPolicy: "conversation-search-v3", EmbeddingRole: "embedding.default",
			EmbeddingDimensions: 384, PolicySchemaVersion: "policy-v4", EmbeddingTaskPrefix: true,
			MigrationPhase: "ready", MigrationSources: 12, MigrationPoints: 14,
		},
		CollectionName: "conversation-search-v1--generation--gen-1", State: aisearch.GenerationStateRetired,
		Lease: aisearch.GenerationLease{ID: "lease-1", Holder: "indexer", ExpiresAt: now.Add(time.Hour)}, Points: 42, Bytes: 1234, UpdatedAt: now,
	}
	require.NoError(t, catalog.SaveGenerationRecord(context.Background(), record))
	records, err := catalog.ListGenerationRecords(context.Background(), "conversation-search", "conversation-search-v1")
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, record.CollectionName, records[0].CollectionName)
	require.Equal(t, record.Lease.ID, records[0].Lease.ID)
	require.Equal(t, record.Metadata.Model, records[0].Metadata.Model)
	require.Equal(t, record.Metadata.RecipeID, records[0].Metadata.RecipeID)
	require.JSONEq(t, record.Metadata.RecipeCanonical, records[0].Metadata.RecipeCanonical)
	require.Equal(t, record.Metadata.ChunkPolicy, records[0].Metadata.ChunkPolicy)
	require.Equal(t, record.Metadata.EmbeddingRole, records[0].Metadata.EmbeddingRole)
	require.Equal(t, record.Metadata.EmbeddingDimensions, records[0].Metadata.EmbeddingDimensions)
	require.Equal(t, record.Metadata.PolicySchemaVersion, records[0].Metadata.PolicySchemaVersion)
	require.Equal(t, record.Metadata.EmbeddingTaskPrefix, records[0].Metadata.EmbeddingTaskPrefix)
	require.Equal(t, record.Metadata.MigrationPhase, records[0].Metadata.MigrationPhase)
	require.Equal(t, record.Metadata.MigrationSources, records[0].Metadata.MigrationSources)
	require.Equal(t, record.Metadata.MigrationPoints, records[0].Metadata.MigrationPoints)
	// Qdrant's lifecycle owner may update state using a minimal record. That
	// transition must not erase the recipe needed for restart census/restore.
	minimalRetirement := aisearch.GenerationRecord{
		Metadata:       aisearch.GenerationMetadata{ID: record.Metadata.ID, CreatedAt: now, Owner: record.Metadata.Owner, Namespace: record.Metadata.Namespace, Alias: record.Metadata.Alias},
		CollectionName: record.CollectionName, State: aisearch.GenerationStateRetired, UpdatedAt: now.Add(time.Minute),
	}
	require.NoError(t, catalog.SaveGenerationRecord(context.Background(), minimalRetirement))
	records, err = catalog.ListGenerationRecords(context.Background(), "conversation-search", "conversation-search-v1")
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, record.Metadata.Model, records[0].Metadata.Model)
	require.Equal(t, record.Metadata.RecipeID, records[0].Metadata.RecipeID)
	require.Equal(t, record.Metadata.EmbeddingDimensions, records[0].Metadata.EmbeddingDimensions)
	require.Equal(t, record.Metadata.MigrationPhase, records[0].Metadata.MigrationPhase)

	receipt := aisearch.GenerationCleanupReceipt{PlanIdentity: "plan-1", IdempotencyKey: "key-1", PolicyRevision: "policy-1", CompletedAt: now}
	require.NoError(t, catalog.SaveGenerationCleanupReceipt(context.Background(), receipt))
	loaded, found, err := catalog.LoadGenerationCleanupReceipt(context.Background(), "key-1")
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, receipt, loaded)
	_, found, err = catalog.LoadGenerationCleanupReceipt(context.Background(), "missing")
	require.NoError(t, err)
	require.False(t, found)
}

func TestStartupUsesExactDurableActiveEmbeddingRecipe(t *testing.T) {
	ctx := context.Background()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	now := time.Now().UTC()
	recipeID, recipeCanonical := catalogTestRecipe(t, "candidate-embed:latest", 384)
	metadata := aisearch.GenerationMetadata{
		ID: "active-alt", CreatedAt: now, Owner: ConversationSearchProviderID,
		Namespace: conversationCollectionDomain, Alias: "conversation-search-v9",
		Model: "candidate-embed:latest", RecipeID: recipeID, RecipeCanonical: recipeCanonical, ChunkPolicy: DefaultRecipeVersion,
		EmbeddingRole: "embedding.default", EmbeddingDimensions: 384,
		PolicySchemaVersion: "schema-v7", EmbeddingTaskPrefix: true,
	}
	record := aisearch.GenerationRecord{
		Metadata: metadata, CollectionName: "conversation-search-v9--generation--active-alt",
		State: aisearch.GenerationStateActive, UpdatedAt: now,
	}
	require.NoError(t, catalog.SaveGenerationRecord(ctx, record))

	active, err := loadActiveRecipeRecord(ctx, catalog, metadata.Alias)
	require.NoError(t, err)
	require.NotNil(t, active)
	require.Equal(t, metadata.Model, active.Metadata.Model)
	require.NoError(t, verifyGenerationRecipe(active.Metadata))

	tuning := aisearch.TuningConfig{Engine: aisearch.EngineHybrid}
	engine := engineFromGenerationRecipe(tuning, aisearch.EngineDeps{Collection: metadata.Alias}, active.Metadata)
	require.Equal(t, metadata.Model, engine.Spec.Model)
	require.Equal(t, metadata.EmbeddingRole, engine.Spec.Role)
	require.Equal(t, metadata.EmbeddingDimensions, engine.Spec.DenseSize)
	require.Equal(t, metadata.PolicySchemaVersion, engine.Spec.PolicySchemaVersion)
	require.True(t, engine.Tuning.EmbedTaskPrefix)
	active.Metadata.Model = "mismatched-model"
	require.Error(t, verifyGenerationRecipe(active.Metadata))
}

func TestSQLiteQdrantAliasBoundarySerializesReopenAndDisjointOwners(t *testing.T) {
	ctx := context.Background()
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	agentOperation := aisearch.FleetAliasOperation{
		OwnerID: "agent-manager.runs", EndpointID: "http://qdrant:6333", Alias: "agent-manager_conversation-search",
		OperationID: "legacy:promote:gen-2:agent-manager_conversation-search--generation--gen-2",
		Kind:        aisearch.FleetPromote, ArmID: "gen-2", Target: "agent-manager_conversation-search--generation--gen-2",
	}
	fence, err := catalog.AcquireFleetAliasMutation(ctx, agentOperation)
	require.NoError(t, err)

	// The durable row remains held across a newly constructed catalog adapter,
	// and even an identical deterministic operation cannot enter concurrently.
	reopened := NewSQLiteQdrantCatalog(db)
	_, err = reopened.AcquireFleetAliasMutation(ctx, agentOperation)
	require.ErrorIs(t, err, aisearch.ErrFleetStateConflict)
	foreign := agentOperation
	foreign.OperationID = "other-operation"
	foreign.Target = "agent-manager_conversation-search--generation--foreign"
	_, err = reopened.AcquireFleetAliasMutation(ctx, foreign)
	require.ErrorIs(t, err, aisearch.ErrFleetStateConflict)

	// The production owners derive different alias roots from their lifecycle
	// namespaces. Even on one Qdrant endpoint their operations do not contend or
	// address the same globally named alias.
	searchHubOperation := aisearch.FleetAliasOperation{
		OwnerID: "search-hub.embedding-fleet", EndpointID: agentOperation.EndpointID, Alias: "search-hub_embedding-fleet",
		OperationID: "fleet:promote:arm-1", Kind: aisearch.FleetPromote, ArmID: "arm-1", Target: "search-hub_embedding-fleet--fleet--arm-1",
	}
	searchHubFence, err := reopened.AcquireFleetAliasMutation(ctx, searchHubOperation)
	require.NoError(t, err)
	require.NoError(t, searchHubFence.Release(ctx))

	require.NoError(t, fence.Release(ctx))
	_, err = reopened.AcquireFleetAliasMutation(ctx, foreign)
	require.NoError(t, err)
}
