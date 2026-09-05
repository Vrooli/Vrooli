package mints_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/database"
	dbtest "github.com/vrooli/api-core/databasetest"

	"token-economy/internal/mints"
)

func newRepository(t *testing.T) mints.Repository {
	t.Helper()
	db := dbtest.NewSQLite(t)
	require.NoError(t, database.EnsureSchemas(context.Background(), db, database.SchemaProviderFunc(mints.Schema)))
	return mints.NewSQLiteRepository(db)
}

// [REQ:TKE-P0-001] Storage preserves the declaration and its sole minter authority.
func TestRepositoryCreateAndGetTokenType(t *testing.T) {
	repo := newRepository(t)
	want := mints.TokenType{
		ID: "chores", Name: "Chore Points", Symbol: "CP", Color: "#2563eb",
		SupplyPolicy: mints.SupplyPolicyCapped, CapAmount: 10,
		Authority: mints.MinterAuthority{TokenTypeID: "chores", Subject: "parent:alex"},
		CreatedAt: time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC),
	}

	_, err := repo.Create(context.Background(), want)
	require.NoError(t, err)
	got, err := repo.Get(context.Background(), want.ID)
	require.NoError(t, err)
	require.Equal(t, want, got)
}

// [REQ:TKE-P0-001] Storage independently refuses a mint that would cross the declared cap.
func TestRepositoryMintEnforcesCap(t *testing.T) {
	repo := newRepository(t)
	value := mints.TokenType{
		ID: "chores", Name: "Chore Points", Symbol: "CP", Color: "#2563eb",
		SupplyPolicy: mints.SupplyPolicyCapped, CapAmount: 10,
		Authority: mints.MinterAuthority{TokenTypeID: "chores", Subject: "parent:alex"},
		CreatedAt: time.Now().UTC(),
	}
	_, err := repo.Create(context.Background(), value)
	require.NoError(t, err)
	got, err := repo.Mint(context.Background(), value.ID, 10)
	require.NoError(t, err)
	require.Equal(t, int64(10), got.MintedAmount)

	_, err = repo.Mint(context.Background(), value.ID, 1)
	var exceeded *mints.SupplyCapExceededError
	require.ErrorAs(t, err, &exceeded)
	require.Equal(t, int64(10), exceeded.Cap)
	require.Equal(t, int64(1), exceeded.AttemptedAmount)
}

// [REQ:TKE-P0-001] Retirement is retained, readable, and prevents any later mint.
func TestRepositoryRetirePreservesTokenType(t *testing.T) {
	repo := newRepository(t)
	value := mints.TokenType{
		ID: "chores", Name: "Chore Points", Symbol: "CP", Color: "#2563eb",
		SupplyPolicy: mints.SupplyPolicyUnbounded,
		Authority:    mints.MinterAuthority{TokenTypeID: "chores", Subject: "parent:alex"},
		CreatedAt:    time.Now().UTC(),
	}
	_, err := repo.Create(context.Background(), value)
	require.NoError(t, err)
	retiredAt := time.Date(2026, 8, 20, 8, 0, 0, 0, time.UTC)
	retired, err := repo.Retire(context.Background(), value.ID, retiredAt)
	require.NoError(t, err)
	require.True(t, retired.Retired)
	require.Equal(t, retiredAt, *retired.RetiredAt)

	readBack, err := repo.Get(context.Background(), value.ID)
	require.NoError(t, err)
	require.True(t, readBack.Retired)
	_, err = repo.Mint(context.Background(), value.ID, 1)
	require.True(t, errors.Is(err, mints.ErrTokenTypeRetired))
}
