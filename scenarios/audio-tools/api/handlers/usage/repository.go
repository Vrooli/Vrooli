package usage

import (
	"context"
	"time"

	"audio-tools/internal/store"
)

// seam: UsageRepository is the usage-handler persistence seam (SEAMS.md
// row "store.UsageRepository"). Production wires *store.UsageStore;
// tests wire handlers/usage/mocks::FakeRepository.
//
// Repository declares only the methods the handler actually calls. The
// concrete store can grow more methods without forcing handler tests to
// implement them.
type Repository interface {
	ListRecent(ctx context.Context, since time.Time, limit int, capability string, providerTier string) ([]store.UsageRow, error)
	Summary(ctx context.Context, since time.Time, capability string) (store.UsageSummary, error)
}

// Compile-time guarantee that *store.UsageStore satisfies Repository.
// Adding a method to the interface that the store doesn't implement
// fails the build at this line.
var _ Repository = (*store.UsageStore)(nil)
