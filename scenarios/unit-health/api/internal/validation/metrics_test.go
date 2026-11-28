package validation

import (
	"context"
	"testing"

	"github.com/vrooli/api-core/metrics"
)

func TestWithMetricsPreservesContextAndSupportsNil(t *testing.T) {
	ctx := context.Background()
	if WithMetrics(ctx, nil) != ctx {
		t.Fatal("nil collector changed context")
	}
	collector := metrics.Start()
	with := WithMetrics(ctx, collector)
	if metricsFrom(with) != collector || metricsFrom(ctx) != nil {
		t.Fatal("metrics collector was not attached to context")
	}
}
