// Package mocks contains test doubles owned by the journal domain.
package mocks

import (
	"context"

	"token-economy/internal/journal"
)

type FakeRepository struct {
	AppendFunc    func(context.Context, journal.Event) (journal.Event, error)
	ReadFunc      func(context.Context, string, string) ([]journal.Event, error)
	BalanceAtFunc func(context.Context, string, string) (journal.Balance, error)
	RebuildFunc   func(context.Context) error
}

func (r *FakeRepository) Append(ctx context.Context, event journal.Event) (journal.Event, error) {
	return r.AppendFunc(ctx, event)
}

func (r *FakeRepository) Read(ctx context.Context, holderID, tokenTypeID string) ([]journal.Event, error) {
	return r.ReadFunc(ctx, holderID, tokenTypeID)
}

func (r *FakeRepository) BalanceAt(ctx context.Context, holderID, tokenTypeID string) (journal.Balance, error) {
	return r.BalanceAtFunc(ctx, holderID, tokenTypeID)
}

func (r *FakeRepository) Rebuild(ctx context.Context) error { return r.RebuildFunc(ctx) }
