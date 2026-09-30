package catalog_test

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"

	accessH "token-economy/handlers/access"
	catalogH "token-economy/handlers/catalog"
	internalaccess "token-economy/internal/access"
	"token-economy/internal/catalog"
	"token-economy/internal/mints"

	accessv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access"
	accessconnect "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/access/accessv1connect"
)

type validator struct{}

func (validator) Validate(_ context.Context, token string) (internalaccess.Identity, error) {
	switch token {
	case "holder":
		return internalaccess.Identity{Subject: "child:sam", Scopes: []string{internalaccess.ScopeHolder}}, nil
	case "minter":
		return internalaccess.Identity{Subject: "operator:alex", Scopes: []string{internalaccess.ScopeMinter}}, nil
	default:
		return internalaccess.Identity{}, internalaccess.ErrUnauthenticated
	}
}

type tokenReader struct{}

func (tokenReader) GetTokenType(_ context.Context, id string) (catalog.TokenTypeState, error) {
	return catalog.TokenTypeState{ID: id}, nil
}

func catalogHarness(t *testing.T) (catalog.Service, accessconnect.HolderServiceClient) {
	t.Helper()
	ctx := context.Background()
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(ctx, db,
		database.SchemaProviderFunc(mints.Schema),
		database.SchemaProviderFunc(catalog.Schema),
	))
	_, err := mints.NewSQLiteRepository(db).Create(ctx, mints.TokenType{
		ID: "chores", Name: "Chore tokens", Symbol: "CT", Color: "#2563eb",
		SupplyPolicy: mints.SupplyPolicyUnbounded,
		Authority:    mints.MinterAuthority{TokenTypeID: "chores", Subject: "operator:alex"},
		CreatedAt:    now,
	})
	require.NoError(t, err)
	service := catalog.NewService(catalog.NewSQLiteRepository(db), tokenReader{}, schedule.NewFake(now))
	module := accessH.Module(nil, nil, nil, nil, catalogH.NewConnectHandler(service, nil), nil, nil, validator{})
	router := mux.NewRouter()
	module.Mount(router)
	server := httptest.NewServer(router)
	t.Cleanup(server.Close)
	return service, accessconnect.NewHolderServiceClient(server.Client(), server.URL)
}

func createEntry(t *testing.T, service catalog.Service, id string, quantity int64, posture catalog.ApprovalPosture) {
	t.Helper()
	_, err := service.Create(context.Background(), catalog.Input{
		ID: id, TokenTypeID: "chores", Title: id, Description: id,
		CostAmount: 5, ApprovalPosture: posture,
		Availability: catalog.Availability{RemainingQuantity: &quantity},
	}, "create-"+id)
	require.NoError(t, err)
}

// [REQ:TKE-P0-008] A holder sees only server-available
// declarations and a direct request for an unavailable entry is refused before
// the redemption domain can run. Approval posture remains visible on entries.
func TestHolderCannotBypassCatalogAvailability(t *testing.T) {
	service, client := catalogHarness(t)
	createEntry(t, service, "available-trip", 1, catalog.ApprovalPostureRequiresApproval)
	createEntry(t, service, "sold-out-trip", 0, catalog.ApprovalPostureImmediate)

	browse := connect.NewRequest(&accessv1.BrowseCatalogRequest{})
	browse.Header().Set("Authorization", "Bearer holder")
	response, err := client.BrowseCatalog(context.Background(), browse)
	require.NoError(t, err)
	require.Len(t, response.Msg.Entries, 1)
	require.Equal(t, "available-trip", response.Msg.Entries[0].Id)
	require.Equal(t, accessv1.ApprovalPosture_APPROVAL_POSTURE_REQUIRES_APPROVAL, response.Msg.Entries[0].ApprovalPosture)

	direct := connect.NewRequest(&accessv1.RequestRedemptionRequest{
		Redemption:     &accessv1.Redemption{CatalogEntryId: "sold-out-trip"},
		IdempotencyKey: "redeem-sold-out",
	})
	direct.Header().Set("Authorization", "Bearer holder")
	_, err = client.RequestRedemption(context.Background(), direct)
	require.Equal(t, connect.CodeFailedPrecondition, connect.CodeOf(err))
	require.Contains(t, err.Error(), "out of stock")
}
