// Package mocks contains test doubles owned by the mints domain.
package mocks

import (
	"context"
	"time"

	"token-economy/internal/mints"
)

type FakeRepository struct {
	CreateFunc func(context.Context, mints.TokenType) (mints.TokenType, error)
	GetFunc    func(context.Context, string) (mints.TokenType, error)
	ListFunc   func(context.Context, bool) ([]mints.TokenType, error)
	RetireFunc func(context.Context, string, time.Time) (mints.TokenType, error)
	MintFunc   func(context.Context, string, int64) (mints.TokenType, error)
}

func (f *FakeRepository) Create(ctx context.Context, tokenType mints.TokenType) (mints.TokenType, error) {
	return f.CreateFunc(ctx, tokenType)
}

func (f *FakeRepository) Get(ctx context.Context, id string) (mints.TokenType, error) {
	return f.GetFunc(ctx, id)
}

func (f *FakeRepository) List(ctx context.Context, includeRetired bool) ([]mints.TokenType, error) {
	return f.ListFunc(ctx, includeRetired)
}

func (f *FakeRepository) Retire(ctx context.Context, id string, at time.Time) (mints.TokenType, error) {
	return f.RetireFunc(ctx, id, at)
}

func (f *FakeRepository) Mint(ctx context.Context, id string, amount int64) (mints.TokenType, error) {
	return f.MintFunc(ctx, id, amount)
}
