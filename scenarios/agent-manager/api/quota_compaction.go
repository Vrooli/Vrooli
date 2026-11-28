package main

import (
	"context"
	"time"

	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/pricing"
)

// startQuotaCompaction removes, once per start, the repeated provider quota
// readings stored before write-time deduplication. It runs in the background
// in small batches, so readiness never waits on history size; on a compacted
// table it scans and deletes nothing. Freed pages return to the file through
// incremental auto-vacuum.
func startQuotaCompaction(ctx context.Context, repository pricing.Repository) {
	compactor, ok := repository.(pricing.QuotaObservationCompactor)
	if !ok {
		return
	}
	go func() {
		started := time.Now()
		result, err := compactor.CompactQuotaObservations(ctx)
		obs.Component("quota-observation").Info("quota observation compaction", "scanned", result.Scanned, "deleted", result.Deleted, "duration", time.Since(started).String(), "error", err)
	}()
}
