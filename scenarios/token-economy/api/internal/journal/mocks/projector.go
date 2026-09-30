package mocks

import (
	"context"

	"token-economy/internal/journal"
)

type FakeProjector struct {
	BalanceAtFunc func(context.Context, string, string) (journal.Balance, error)
	RebuildFunc   func(context.Context) error
}

func (p *FakeProjector) BalanceAt(ctx context.Context, holderID, tokenTypeID string) (journal.Balance, error) {
	return p.BalanceAtFunc(ctx, holderID, tokenTypeID)
}

func (p *FakeProjector) Rebuild(ctx context.Context) error { return p.RebuildFunc(ctx) }
