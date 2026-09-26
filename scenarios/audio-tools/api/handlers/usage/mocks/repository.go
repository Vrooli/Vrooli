// Package mocks holds the hand-written fakes for the usage handler's
// consumer-side seams.
package mocks

import (
	"context"
	"sync/atomic"
	"time"

	usageH "audio-tools/handlers/usage"
	"audio-tools/internal/store"
)

// FakeRepository satisfies handlers/usage.Repository for tests.
type FakeRepository struct {
	Rows         []store.UsageRow
	SummaryValue store.UsageSummary
	ListErr      error
	SumErr       error
	ListCalls    atomic.Int64
	SumCalls     atomic.Int64
	LastSince    time.Time
}

func (f *FakeRepository) ListRecent(_ context.Context, since time.Time, _ int, _, _ string) ([]store.UsageRow, error) {
	f.ListCalls.Add(1)
	f.LastSince = since
	if f.ListErr != nil {
		return nil, f.ListErr
	}
	return f.Rows, nil
}

func (f *FakeRepository) Summary(_ context.Context, _ time.Time, _ string) (store.UsageSummary, error) {
	f.SumCalls.Add(1)
	if f.SumErr != nil {
		return store.UsageSummary{}, f.SumErr
	}
	return f.SummaryValue, nil
}

var _ usageH.Repository = (*FakeRepository)(nil)
