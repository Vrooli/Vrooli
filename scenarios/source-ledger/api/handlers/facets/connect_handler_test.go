package facets

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	localdb "source-ledger/internal/database"
	internalfacets "source-ledger/internal/facets"
	"source-ledger/internal/journal"
	"source-ledger/internal/policy"

	facetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets"
)

func newHandler(t *testing.T) (*connectHandler, *journal.SQLiteRepository) {
	t.Helper()
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:facets-handler?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(internalfacets.Schema)))
	repo := internalfacets.NewSQLiteRepository(db.Primary())
	require.NoError(t, repo.Seed(context.Background()))
	registry := policy.NewRegistry(db.Primary())
	require.NoError(t, registry.Ensure(context.Background(), policy.BuiltInDefaults()))
	return NewConnectHandler(internalfacets.NewService(repo), nil, registry), journal.NewSQLiteRepository(db.Primary())
}

func TestListFacetsRejectsUnregisteredScope(t *testing.T) {
	h, _ := newHandler(t)
	_, err := h.ListFacets(context.Background(), connect.NewRequest(&facetsv1.ListFacetsRequest{Scope: "no-such-scope-xyz"}))
	require.Equal(t, connect.CodeNotFound, connect.CodeOf(err))
}

func TestAssignFacetRejectsUnknownFacetAndSetsPin(t *testing.T) { // [REQ:VMEM-P1-006] [REQ:VMEM-P1-010]
	h, journalRepo := newHandler(t)
	ctx := context.Background()
	entry, err := journalRepo.Append(ctx, journal.Entry{Body: "operator rule", FacetID: "standing-rule"}, nil)
	require.NoError(t, err)

	_, err = h.AssignFacet(ctx, connect.NewRequest(&facetsv1.AssignFacetRequest{EntryId: entry.ID, FacetId: "unknown"}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))

	_, err = h.AssignFacet(ctx, connect.NewRequest(&facetsv1.AssignFacetRequest{EntryId: entry.ID, FacetId: "standing-rule"}))
	require.NoError(t, err)
	_, err = h.SetPin(ctx, connect.NewRequest(&facetsv1.SetPinRequest{EntryId: entry.ID, Pinned: true}))
	require.NoError(t, err)
}

func TestSetFacetPolicyPersistsScopedRetentionAndResidency(t *testing.T) { // [REQ:SL-P1-001]
	h, _ := newHandler(t)
	response, err := h.SetFacetPolicy(context.Background(), connect.NewRequest(&facetsv1.SetFacetPolicyRequest{
		Scope: "agent-memory", FacetId: "episode", RetentionPolicy: "compact", CompactionEligible: true, ResidentBudget: 6,
	}))
	require.NoError(t, err)
	require.Equal(t, "episode", response.Msg.GetFacet().GetId())
	require.Equal(t, "compact", response.Msg.GetFacet().GetRetentionPolicy())
	require.True(t, response.Msg.GetFacet().GetCompactionEligible())
	require.Equal(t, int32(6), response.Msg.GetFacet().GetResidentBudget())
}

func TestCountUnassignedReportsEntriesWithoutFacetAssignments(t *testing.T) { // [REQ:SL-P1-001]
	h, journalRepo := newHandler(t)
	entry, err := journalRepo.Append(context.Background(), journal.Entry{Body: "unclassified"}, nil)
	require.NoError(t, err)

	response, err := h.CountUnassigned(context.Background(), connect.NewRequest(&facetsv1.CountUnassignedRequest{Scope: "agent-memory"}))
	require.NoError(t, err)
	require.Equal(t, "agent-memory", response.Msg.GetScope())
	require.Equal(t, int32(1), response.Msg.GetCount())
	require.NotEmpty(t, entry.ID)
}

func TestEnsureFacetCreatesDefinitionAndPreservesExistingPolicy(t *testing.T) {
	h, journalRepo := newHandler(t)
	ctx := context.Background()

	created, err := h.EnsureFacet(ctx, connect.NewRequest(&facetsv1.EnsureFacetRequest{
		Scope: "agent-memory",
		Facet: &facetsv1.Facet{Id: "migration-facet", Label: "Migration facet", Guidance: "A migration facet.", RetentionPolicy: "compact", CompactionEligible: true, ResidentBudget: 3},
	}))
	require.NoError(t, err)
	require.Equal(t, "migration-facet", created.Msg.GetFacet().GetId())
	require.Equal(t, int32(3), created.Msg.GetFacet().GetResidentBudget())

	updated, err := h.SetFacetPolicy(ctx, connect.NewRequest(&facetsv1.SetFacetPolicyRequest{Scope: "agent-memory", FacetId: "migration-facet", RetentionPolicy: "retain", ResidentBudget: 1}))
	require.NoError(t, err)
	require.Equal(t, int32(1), updated.Msg.GetFacet().GetResidentBudget())

	repeated, err := h.EnsureFacet(ctx, connect.NewRequest(&facetsv1.EnsureFacetRequest{
		Scope: "agent-memory",
		Facet: &facetsv1.Facet{Id: "migration-facet", Label: "Updated label", Guidance: "Updated guidance.", RetentionPolicy: "compact", CompactionEligible: true, ResidentBudget: 99},
	}))
	require.NoError(t, err)
	require.Equal(t, "Updated label", repeated.Msg.GetFacet().GetLabel())
	require.Equal(t, "retain", repeated.Msg.GetFacet().GetRetentionPolicy())
	require.Equal(t, int32(1), repeated.Msg.GetFacet().GetResidentBudget())

	entry, err := journalRepo.Append(ctx, journal.Entry{Body: "legacy assignment"}, nil)
	require.NoError(t, err)
	_, err = h.AssignFacet(ctx, connect.NewRequest(&facetsv1.AssignFacetRequest{Scope: "agent-memory", EntryId: entry.ID, FacetId: "migration-facet"}))
	require.NoError(t, err)
	_, err = h.AssignFacet(ctx, connect.NewRequest(&facetsv1.AssignFacetRequest{Scope: "agent-memory", EntryId: entry.ID, FacetId: "episode"}))
	require.NoError(t, err)
	_, err = h.DeleteFacet(ctx, connect.NewRequest(&facetsv1.DeleteFacetRequest{Scope: "agent-memory", FacetId: "migration-facet"}))
	require.NoError(t, err)
}
