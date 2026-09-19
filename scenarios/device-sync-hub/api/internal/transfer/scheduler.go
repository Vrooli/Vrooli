package transfer

import (
	"context"
	"log"
	"time"
)

// DefaultPurgeInterval is how often the retention sweep runs. Held items live
// 24h and Live items drain on delivery, so a few-minute cadence keeps purges
// timely without busy-looping. Overridable by the caller.
const DefaultPurgeInterval = 5 * time.Minute

// RunPurgeLoop runs the retention sweep on a ticker until ctx is cancelled. It
// is the scenario's scheduled purge of expired Held items and delivered Live
// items (retention is the scenario's job, not the storage module's). main.go
// launches it in a goroutine at boot; cancelling ctx (server shutdown) stops it.
//
// One sweep runs immediately on start so a process that was down across an
// expiry boundary reclaims promptly rather than waiting a full interval.
func RunPurgeLoop(ctx context.Context, svc Service, interval time.Duration, logger *log.Logger) {
	if interval <= 0 {
		interval = DefaultPurgeInterval
	}
	if logger == nil {
		logger = log.Default()
	}
	sweep := func() {
		n, err := svc.Purge(ctx)
		if err != nil {
			logger.Printf("transfer.purge sweep: %v", err)
			return
		}
		if n > 0 {
			logger.Printf("transfer.purge: removed %d expired item(s)", n)
		}
	}
	sweep()

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			sweep()
		}
	}
}
