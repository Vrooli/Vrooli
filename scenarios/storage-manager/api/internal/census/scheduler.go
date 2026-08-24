package census

// The census and retention loops intentionally share one wait-from-end
// implementation. A filesystem census can outlive its nominal interval; a
// ticker would queue another walk and turn a slow scan into continuous host
// pressure. Keep this package as the domain-facing alias so existing census
// callers retain their API while the scheduling contract has one owner.

import (
	"context"
	"time"

	shared "github.com/vrooli/vrooli/packages/scheduler"
)

type (
	Cycle     = shared.Cycle
	Stats     = shared.Stats
	Scheduler = shared.Runner
)

func NewScheduler(interval time.Duration, run func(context.Context) error) *Scheduler {
	return shared.New(interval, run)
}
