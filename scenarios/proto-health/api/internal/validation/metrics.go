package validation

import (
	"context"
	"strings"

	"github.com/vrooli/api-core/metrics"
)

// metricsCtxKey carries an optional execution-metrics collector through the
// validation call so ValidateScenario can open profiling stages without changing
// its signature (and the Validator interface / its many test mocks). When no
// collector is present, every stage call is a nil no-op.
type metricsCtxKey struct{}
type scenarioPathCtxKey struct{}

// WithMetrics attaches a metrics collector to ctx. The connect handler creates
// the collector, threads it here, and reads the assembled metrics back after the
// validation returns.
func WithMetrics(ctx context.Context, collector *metrics.Collector) context.Context {
	if collector == nil {
		return ctx
	}
	return context.WithValue(ctx, metricsCtxKey{}, collector)
}

// metricsFrom returns the collector attached to ctx, or nil. A nil *Collector is
// safe to call Stage/Gauge/Stop on (they are no-ops), so callers need not branch.
func metricsFrom(ctx context.Context) *metrics.Collector {
	collector, _ := ctx.Value(metricsCtxKey{}).(*metrics.Collector)
	return collector
}

// WithScenarioPath attaches the caller-resolved physical scenario directory to
// validation. Providers normally resolve by scenario name, but Test Genie can
// validate an isolated generated scenario outside the repository tree.
func WithScenarioPath(ctx context.Context, path string) context.Context {
	if path = strings.TrimSpace(path); path == "" {
		return ctx
	}
	return context.WithValue(ctx, scenarioPathCtxKey{}, path)
}

// ScenarioPathFrom returns the optional physical scenario directory supplied
// by the caller.
func ScenarioPathFrom(ctx context.Context) string {
	path, _ := ctx.Value(scenarioPathCtxKey{}).(string)
	return path
}
