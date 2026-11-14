package conversationsearch

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	aisearch "github.com/vrooli/ai-go/search"
)

func TestSQLiteQdrantCatalogPersistsLifecycleAndCleanupReceipt(t *testing.T) {
	db := openProjectionTestDB(t)
	applyProjectionSchema(t, db)
	catalog := NewSQLiteQdrantCatalog(db)
	now := time.Date(2026, 9, 9, 7, 0, 0, 0, time.UTC)
	record := aisearch.GenerationRecord{
		Metadata:       aisearch.GenerationMetadata{ID: "gen-1", CreatedAt: now, Owner: "agent-manager.runs", Namespace: "conversation-search", Alias: "conversation-search-v1", ContentIdentity: "digest-1"},
		CollectionName: "conversation-search-v1--generation--gen-1", State: aisearch.GenerationStateRetired,
		Lease: aisearch.GenerationLease{ID: "lease-1", Holder: "indexer", ExpiresAt: now.Add(time.Hour)}, Points: 42, Bytes: 1234, UpdatedAt: now,
	}
	require.NoError(t, catalog.SaveGenerationRecord(context.Background(), record))
	records, err := catalog.ListGenerationRecords(context.Background(), "conversation-search", "conversation-search-v1")
	require.NoError(t, err)
	require.Len(t, records, 1)
	require.Equal(t, record.CollectionName, records[0].CollectionName)
	require.Equal(t, record.Lease.ID, records[0].Lease.ID)

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
	// and conflicting alias effects cannot enter while recovery is pending.
	reopened := NewSQLiteQdrantCatalog(db)
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
