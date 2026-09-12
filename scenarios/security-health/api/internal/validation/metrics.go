package validation

import (
	"context"

	"github.com/vrooli/api-core/metrics"
)

// metricsCtxKey carries an optional execution-metrics collector through the
// validation call so ValidateScenario can open profiling stages without
// changing its signature (and the Validator interface / its many test mocks).
// When no collector is present, every stage call is a nil no-op.
type metricsCtxKey struct{}

// WithMetrics attaches a metrics collector to ctx. The connect handler creates
// the collector, threads it here, and reads the assembled metrics back after
// the validation returns.
func WithMetrics(ctx context.Context, collector *metrics.Collector) context.Context {
	if collector == nil {
		return ctx
	}
	ctx = collector.Context()
	return context.WithValue(ctx, metricsCtxKey{}, collector)
}

// metricsFrom returns the collector attached to ctx, or nil. A nil *Collector
// is safe to call Stage/Gauge/Stop on (they are no-ops), so callers need not
// branch.
func metricsFrom(ctx context.Context) *metrics.Collector {
	collector, _ := ctx.Value(metricsCtxKey{}).(*metrics.Collector)
	return collector
}
