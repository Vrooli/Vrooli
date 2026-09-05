package audit

import (
	"context"

	"github.com/vrooli/api-core/metrics"
)

// metricsCtxKey carries an optional execution-metrics collector through the
// ValidateScenario call so it can open profiling stages without changing its
// signature or the audit.Service interface.
// A nil *Collector is safe to call Stage/Gauge/Stop on (they are no-ops),
// so callers need not branch.
type metricsCtxKey struct{}

// WithMetrics attaches a metrics collector to ctx. The handler creates the
// collector, threads it here, and reads the assembled metrics back after the
// audit returns.
func WithMetrics(ctx context.Context, collector *metrics.Collector) context.Context {
	if collector == nil {
		return ctx
	}
	return context.WithValue(ctx, metricsCtxKey{}, collector)
}
