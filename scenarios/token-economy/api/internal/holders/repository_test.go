package holders_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	"token-economy/internal/holders"
)

func newRepository(t *testing.T) holders.Repository {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(holders.Schema)))
	return holders.NewSQLiteRepository(db)
}

// [REQ:TKE-P0-006] Repository scoping, not a handler check, prevents one
// authenticated subject from reading another holder record.
func TestRepositoryRefusesCrossHolderReadWithoutExistenceDisclosure(t *testing.T) {
	repository := newRepository(t)
	created := holders.Holder{
		ID: "holder-sam", DisplayName: "Sam", AuthenticatorSubject: "auth:sam",
		CreatedAt: time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
	}
	_, err := repository.Create(context.Background(), created)
	require.NoError(t, err)

	owned, found, err := repository.GetScoped(context.Background(), holders.Scope{
		HolderID: created.ID, AuthenticatorSubject: created.AuthenticatorSubject,
	})
	require.NoError(t, err)
	require.True(t, found)
	require.Equal(t, created, owned)

	foreign, foreignFound, foreignErr := repository.GetScoped(context.Background(), holders.Scope{
		HolderID: created.ID, AuthenticatorSubject: "auth:lee",
	})
	missing, missingFound, missingErr := repository.GetScoped(context.Background(), holders.Scope{
		HolderID: "holder-absent", AuthenticatorSubject: "auth:lee",
	})
	require.NoError(t, foreignErr)
	require.NoError(t, missingErr)
	require.False(t, foreignFound)
	require.False(t, missingFound)
	require.Equal(t, holders.Holder{}, foreign)
	require.Equal(t, missing, foreign, "foreign and absent reads must have the same observable shape")
}

func TestRepositoryStoresSubjectBindingWithoutCredentials(t *testing.T) {
	repository := newRepository(t)
	want := holders.Holder{
		ID: "holder-lee", DisplayName: "Lee", AuthenticatorSubject: "auth:lee",
		CreatedAt: time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
	}
	_, err := repository.Create(context.Background(), want)
	require.NoError(t, err)
	got, err := repository.GetBySubject(context.Background(), want.AuthenticatorSubject)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

func TestCreateIdempotentAndListReturnTheOriginalHolderOnce(t *testing.T) {
	repository := newRepository(t)
	ctx := context.Background()
	createdAt := time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC)
	first := holders.Holder{ID: "holder-sam", DisplayName: "Sam", AuthenticatorSubject: "auth:sam", CreatedAt: createdAt}
	created, err := repository.CreateIdempotent(ctx, first, "holder:sam")
	require.NoError(t, err)
	require.Equal(t, first, created)

	retry := holders.Holder{ID: "holder-retry", DisplayName: "Changed", AuthenticatorSubject: "auth:changed", CreatedAt: createdAt.Add(time.Hour)}
	replayed, err := repository.CreateIdempotent(ctx, retry, "holder:sam")
	require.NoError(t, err)
	require.Equal(t, first, replayed)

	listed, err := repository.List(ctx)
	require.NoError(t, err)
	require.Equal(t, []holders.Holder{first}, listed)
	got, err := repository.Get(ctx, first.ID)
	require.NoError(t, err)
	require.Equal(t, first, got)
}
