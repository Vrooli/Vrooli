package mints_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/api-core/schedule"

	"token-economy/internal/mints"
	"token-economy/internal/mints/mocks"
)

// [REQ:TKE-P0-001] A token type is created only with display identity, supply policy, and one authority.
func TestServiceCreateRequiresCompleteDeclaredType(t *testing.T) {
	clock := schedule.NewFake(time.Date(2026, 8, 19, 12, 0, 0, 0, time.UTC))
	var persisted mints.TokenType
	repo := &mocks.FakeRepository{CreateFunc: func(_ context.Context, value mints.TokenType) (mints.TokenType, error) {
		persisted = value
		return value, nil
	}}
	service := mints.NewService(repo, clock)

	created, err := service.Create(context.Background(), mints.CreateInput{
		Name: "House Points", Symbol: "HP", Color: "#4f46e5",
		SupplyPolicy: mints.SupplyPolicyCapped, CapAmount: 100, MinterSubject: "parent:alex",
	})
	require.NoError(t, err)
	require.NotEmpty(t, created.ID)
	require.Equal(t, created.ID, created.Authority.TokenTypeID)
	require.Equal(t, "parent:alex", created.Authority.Subject)
	require.Equal(t, clock.Now(), persisted.CreatedAt)

	_, err = service.Create(context.Background(), mints.CreateInput{
		Name: "House Points", Symbol: "HP", Color: "#4f46e5",
		SupplyPolicy: mints.SupplyPolicyCapped, CapAmount: 100,
	})
	var invalid *mints.InvalidTokenTypeError
	require.ErrorAs(t, err, &invalid)
}

// [REQ:TKE-P0-001] A capped token accepts its boundary and rejects an over-cap mint with cap and attempted amount.
func TestServiceMintEnforcesCapBoundary(t *testing.T) {
	current := mints.TokenType{ID: "chores", SupplyPolicy: mints.SupplyPolicyCapped, CapAmount: 100, MintedAmount: 90}
	mintCalls := 0
	repo := &mocks.FakeRepository{
		GetFunc: func(_ context.Context, id string) (mints.TokenType, error) {
			require.Equal(t, "chores", id)
			return current, nil
		},
		MintFunc: func(_ context.Context, id string, amount int64) (mints.TokenType, error) {
			mintCalls++
			current.MintedAmount += amount
			return current, nil
		},
	}
	service := mints.NewService(repo, schedule.System())

	atCap, err := service.Mint(context.Background(), "chores", 10)
	require.NoError(t, err)
	require.Equal(t, int64(100), atCap.MintedAmount)

	current.MintedAmount = 90
	_, err = service.Mint(context.Background(), "chores", 11)
	var exceeded *mints.SupplyCapExceededError
	require.ErrorAs(t, err, &exceeded)
	require.Equal(t, int64(100), exceeded.Cap)
	require.Equal(t, int64(11), exceeded.AttemptedAmount)
	require.Contains(t, err.Error(), "cap 100")
	require.Contains(t, err.Error(), "amount 11")
	require.Equal(t, 1, mintCalls, "rejected supply must not reach persistence")
}

// [REQ:TKE-P0-001] An unknown token type is never created implicitly by a mint request.
func TestServiceMintUnknownTypeDoesNotCreate(t *testing.T) {
	createCalls := 0
	mintCalls := 0
	repo := &mocks.FakeRepository{
		GetFunc: func(context.Context, string) (mints.TokenType, error) {
			return mints.TokenType{}, mints.ErrTokenTypeNotFound
		},
		CreateFunc: func(context.Context, mints.TokenType) (mints.TokenType, error) {
			createCalls++
			return mints.TokenType{}, nil
		},
		MintFunc: func(context.Context, string, int64) (mints.TokenType, error) {
			mintCalls++
			return mints.TokenType{}, nil
		},
	}

	_, err := mints.NewService(repo, schedule.System()).Mint(context.Background(), "missing", 1)
	require.True(t, errors.Is(err, mints.ErrTokenTypeNotFound))
	require.Zero(t, createCalls)
	require.Zero(t, mintCalls)
}
