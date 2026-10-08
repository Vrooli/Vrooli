package mocks

import (
	"context"

	"token-economy/internal/journal"
)

type FakeAppender struct {
	AppendFunc func(context.Context, journal.Event) (journal.Event, error)
}

func (a *FakeAppender) Append(ctx context.Context, event journal.Event) (journal.Event, error) {
	return a.AppendFunc(ctx, event)
}
