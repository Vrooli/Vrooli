package catalog_test

import (
	"context"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"
	"github.com/vrooli/api-core/schedule"
	"google.golang.org/protobuf/reflect/protoreflect"

	"token-economy/internal/catalog"
	"token-economy/internal/mints"

	catalogv1 "github.com/vrooli/vrooli/packages/proto/gen/go/token-economy/v1/catalog"
)

type tokenReader struct{}

func (tokenReader) GetTokenType(_ context.Context, id string) (catalog.TokenTypeState, error) {
	return catalog.TokenTypeState{ID: id}, nil
}

func newCatalogService(t *testing.T, now time.Time) catalog.Service {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db,
		database.SchemaProviderFunc(mints.Schema),
		database.SchemaProviderFunc(catalog.Schema),
	))
	_, err := mints.NewSQLiteRepository(db).Create(context.Background(), mints.TokenType{
		ID: "chores", Name: "Chore tokens", Symbol: "CT", Color: "#2563eb",
		SupplyPolicy: mints.SupplyPolicyUnbounded,
		Authority:    mints.MinterAuthority{TokenTypeID: "chores", Subject: "operator:alex"},
		CreatedAt:    now,
	})
	require.NoError(t, err)
	service := catalog.NewService(catalog.NewSQLiteRepository(db), tokenReader{}, schedule.NewFake(now))
	return service
}

func quantity(value int64) *int64 { return &value }

func entryInput(id, title string, availability catalog.Availability, posture catalog.ApprovalPosture) catalog.Input {
	return catalog.Input{
		ID: id, TokenTypeID: "chores", Title: title, Description: title + " description",
		CostAmount: 5, Availability: availability, ApprovalPosture: posture,
	}
}

// [REQ:TKE-P0-008] Availability is enforced by the service for direct reads,
// independently of any UI filtering.
func TestAvailabilityWindowAndQuantityAreServerEnforced(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	service := newCatalogService(t, now)
	past := now.Add(-time.Hour)
	future := now.Add(time.Hour)
	later := now.Add(2 * time.Hour)
	earlier := now.Add(-2 * time.Hour)
	tests := []struct {
		id           string
		availability catalog.Availability
		available    bool
		reason       string
	}{
		{id: "available", availability: catalog.Availability{AvailableFrom: &past, AvailableUntil: &future, RemainingQuantity: quantity(2)}, available: true},
		{id: "future", availability: catalog.Availability{AvailableFrom: &future, AvailableUntil: &later}, reason: "not started"},
		{id: "expired", availability: catalog.Availability{AvailableFrom: &earlier, AvailableUntil: &past}, reason: "ended"},
		{id: "out-of-stock", availability: catalog.Availability{RemainingQuantity: quantity(0)}, reason: "out of stock"},
	}
	for _, test := range tests {
		_, err := service.Create(context.Background(), entryInput(test.id, test.id, test.availability, catalog.ApprovalPostureImmediate), "create-"+test.id)
		require.NoError(t, err, test.id)
		entry, err := service.RequireAvailable(context.Background(), test.id)
		if test.available {
			require.NoError(t, err)
			require.Equal(t, test.id, entry.ID)
		} else {
			require.ErrorIs(t, err, catalog.ErrEntryUnavailable)
			require.Contains(t, err.Error(), test.reason)
		}
	}
	available, err := service.ListAvailable(context.Background())
	require.NoError(t, err)
	require.Len(t, available, 1)
	require.Equal(t, "available", available[0].ID)
}

// [REQ:TKE-P0-008] The product seeds no built-in redeemables, and retirement
// keeps the declared row readable while removing it from available results.
func TestCatalogStartsEmptyAndRetireRetainsDeclaration(t *testing.T) {
	now := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	service := newCatalogService(t, now)
	entries, err := service.List(context.Background(), true)
	require.NoError(t, err)
	require.Empty(t, entries)
	created, err := service.Create(context.Background(), entryInput("trip", "Day trip", catalog.Availability{}, catalog.ApprovalPostureRequiresApproval), "create-trip")
	require.NoError(t, err)
	replayed, err := service.Create(context.Background(), entryInput("different-id", "Changed retry", catalog.Availability{}, catalog.ApprovalPostureImmediate), "create-trip")
	require.NoError(t, err)
	require.Equal(t, created.ID, replayed.ID)

	retired, err := service.Retire(context.Background(), created.ID, "retire-trip")
	require.NoError(t, err)
	require.True(t, retired.Retired)
	stored, err := service.Get(context.Background(), created.ID)
	require.NoError(t, err)
	require.True(t, stored.Retired)
	available, err := service.ListAvailable(context.Background())
	require.NoError(t, err)
	require.Empty(t, available)
}

// [REQ:TKE-P0-008] Approval posture is part of every catalog declaration,
// while monetary capability is structurally absent.
func TestCatalogContractDeclaresApprovalWithoutMonetaryFields(t *testing.T) {
	message := catalogv1.File_token_economy_v1_catalog_catalog_proto.Messages().ByName("CatalogEntry")
	require.NotNil(t, message)
	fields := make([]string, 0, message.Fields().Len())
	for i := 0; i < message.Fields().Len(); i++ {
		field := message.Fields().Get(i)
		fields = append(fields, string(field.Name()))
		assertNoMonetaryName(t, field)
	}
	require.Contains(t, fields, "approval_posture")
	require.Contains(t, fields, "cost_amount")
	require.NotContains(t, fields, "price")
	require.NotContains(t, fields, "currency")

	posture := catalogv1.ApprovalPosture_APPROVAL_POSTURE_REQUIRES_APPROVAL
	require.Equal(t, "APPROVAL_POSTURE_REQUIRES_APPROVAL", posture.String())
	require.Equal(t, protoreflect.EnumNumber(2), posture.Number())
	require.NotEqual(t, reflect.ValueOf(catalogv1.ApprovalPosture_APPROVAL_POSTURE_IMMEDIATE).Int(), reflect.ValueOf(posture).Int())
}

func assertNoMonetaryName(t *testing.T, field protoreflect.FieldDescriptor) {
	t.Helper()
	name := strings.ToLower(string(field.Name()))
	for _, forbidden := range []string{"price", "currency", "monetary", "payout"} {
		require.NotContains(t, name, forbidden)
	}
}
