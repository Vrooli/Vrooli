package store

import (
	"context"
	"log"
	"time"
)

// PrunerConfig holds configuration for the background pruning goroutine.
type PrunerConfig struct {
	Interval time.Duration // how often to run (default 6h)
	Store    Store
	Logger   func(format string, args ...any)
}

// StartPruner launches a background pruning goroutine that runs at the configured interval.
// It returns when the context is cancelled.
func StartPruner(ctx context.Context, cfg PrunerConfig) {
	if cfg.Interval == 0 {
		cfg.Interval = 6 * time.Hour
	}
	if cfg.Logger == nil {
		cfg.Logger = log.Printf
	}

	ticker := time.NewTicker(cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := cfg.Store.Prune(ctx)
			if err != nil {
				cfg.Logger("prune error: %v", err)
				continue
			}
			if result.TimeDeletedCount > 0 || result.SizeDeletedCount > 0 {
				cfg.Logger("pruned %d events (time: %d, size: %d)",
					result.TimeDeletedCount+result.SizeDeletedCount,
					result.TimeDeletedCount, result.SizeDeletedCount)
			}
		}
	}
}
