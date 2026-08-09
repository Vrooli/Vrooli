package facets

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	apidb "github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"

	facetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/source-ledger/v1/facets"
	localdb "source-ledger/internal/database"
	internalfacets "source-ledger/internal/facets"
	"source-ledger/internal/journal"
)

func newHandler(t *testing.T) (*connectHandler, *journal.SQLiteRepository) {
	t.Helper()
	db, err := apidb.Open(context.Background(), apidb.Config{Driver: apidb.DriverSQLite, DSN: "file:facets-handler?mode=memory&cache=shared"})
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	require.NoError(t, apidb.EnsureSchemas(context.Background(), db.Primary(), apidb.SchemaProviderFunc(localdb.SystemSchema), apidb.SchemaProviderFunc(journal.Schema), apidb.SchemaProviderFunc(internalfacets.Schema)))
	repo := internalfacets.NewSQLiteRepository(db.Primary())
	require.NoError(t, repo.Seed(context.Background()))
	return NewConnectHandler(internalfacets.NewService(repo), nil), journal.NewSQLiteRepository(db.Primary())
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
