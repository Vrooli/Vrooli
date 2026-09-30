package validation

import (
	"context"

	"github.com/vrooli/api-core/metrics"
)

type metricsCtxKey struct{}

func WithMetrics(ctx context.Context, collector *metrics.Collector) context.Context {
	if collector == nil {
		return ctx
	}
	return context.WithValue(ctx, metricsCtxKey{}, collector)
}

func metricsFrom(ctx context.Context) *metrics.Collector {
	if collector, ok := ctx.Value(metricsCtxKey{}).(*metrics.Collector); ok && collector != nil {
		return collector
	}
	return nil
}
