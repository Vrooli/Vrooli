package checks

import (
	"context"

	"github.com/vrooli/api-core/metrics"
)

// metricsCtxKey threads the ExecutionMetrics collector through
// ValidateScenario without changing its signature (proto-health's
// metrics-seam recipe). A nil *metrics.Collector is a safe no-op for
// Stage/Gauge/Stop, so callers without a collector pay nothing.
type metricsCtxKey struct{}

// WithMetrics attaches a collector to the context for one validation run.
func WithMetrics(ctx context.Context, c *metrics.Collector) context.Context {
	return context.WithValue(ctx, metricsCtxKey{}, c)
}

func metricsFrom(ctx context.Context) *metrics.Collector {
	c, _ := ctx.Value(metricsCtxKey{}).(*metrics.Collector)
	return c
}
