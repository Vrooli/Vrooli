package main

// DOC: docs/internal/STORAGE_AUDIT.md

import (
	"context"
	"time"

	"knowledge-observatory/internal/quality"
)

// Retention bounds the growth of quality_metrics.
//
// The table is append-only and the materializer writes one sample per
// collection every few minutes: roughly 10,000 rows a day, which reached 1.23M
// rows and 252 MB before anything reclaimed them (docs/internal/STORAGE_AUDIT.md
// §2). Migrating that data without also bounding it would only reset the clock.
//
// The policy is the one the operator approved on 2026-08-01:
//
//	within FullResolutionWindow -> keep every sample
//	older                       -> keep one sample per collection per day
//
// Downsampling rather than deleting is deliberate: every trend line survives at
// daily granularity, and only redundant intra-day resolution is discarded.
type Retention struct {
	Repo quality.Repository

	// FullResolutionWindow is how far back every sample is kept.
	FullResolutionWindow time.Duration

	// Interval is how often the policy runs.
	Interval time.Duration

	// Now and Sleep exist so tests can drive the loop deterministically.
	Now   func() time.Time
	Sleep func(time.Duration)

	// Log receives a line per pass. Optional.
	Log func(msg string, fields map[string]interface{})
}

const (
	defaultFullResolutionWindow = 30 * 24 * time.Hour
	defaultRetentionInterval    = 6 * time.Hour
)

func (r *Retention) window() time.Duration {
	if r.FullResolutionWindow > 0 {
		return r.FullResolutionWindow
	}
	return defaultFullResolutionWindow
}

func (r *Retention) interval() time.Duration {
	if r.Interval > 0 {
		return r.Interval
	}
	return defaultRetentionInterval
}

func (r *Retention) now() time.Time {
	if r.Now != nil {
		return r.Now()
	}
	return time.Now().UTC()
}

// Run applies the policy until ctx is cancelled.
func (r *Retention) Run(ctx context.Context) {
	if r == nil || r.Repo == nil {
		return
	}
	sleep := r.Sleep
	if sleep == nil {
		sleep = time.Sleep
	}
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		_ = r.ApplyOnce(ctx)
		sleep(r.interval())
	}
}

// ApplyOnce collapses everything older than the full-resolution window to one
// sample per collection per day. It is safe to call repeatedly: a range that is
// already collapsed has nothing left to delete.
func (r *Retention) ApplyOnce(ctx context.Context) error {
	if r == nil || r.Repo == nil {
		return nil
	}

	cutoff := r.now().Add(-r.window())
	deleted, err := r.Repo.DownsampleMetricsOlderThan(ctx, cutoff)
	if err != nil {
		if r.Log != nil {
			r.Log("quality metric retention failed", map[string]interface{}{"error": err.Error()})
		}
		return err
	}
	if deleted > 0 && r.Log != nil {
		remaining, _ := r.Repo.CountMetrics(ctx)
		r.Log("quality metric retention applied", map[string]interface{}{
			"downsampled_rows": deleted,
			"cutoff":           cutoff.Format(time.RFC3339),
			"remaining_rows":   remaining,
		})
	}
	return nil
}
