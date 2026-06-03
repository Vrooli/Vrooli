package metrics

import (
	"context"
	"log"

	internalmetrics "search-hub/internal/metrics"
	internalrouting "search-hub/internal/routing"
)

// TelemetryBridge adapts the metrics SQLite store to the routing domain's
// TelemetryRecorder seam. It lives at the wiring edge precisely so
// internal/metrics never imports internal/routing (and vice versa): the router
// owns the TelemetrySample shape (consumer-defined seam), the store owns the
// Sample shape, and this bridge translates between them.
//
// Record is best-effort by the routing seam's contract — it logs and swallows
// the store error so a telemetry write failure never affects the live query.
type TelemetryBridge struct {
	store  internalmetrics.Store
	logger *log.Logger
}

// NewTelemetryBridge constructs the bridge. Logger defaults to log.Default()
// when nil.
func NewTelemetryBridge(store internalmetrics.Store, logger *log.Logger) *TelemetryBridge {
	if logger == nil {
		logger = log.Default()
	}
	return &TelemetryBridge{store: store, logger: logger}
}

// Compile-time guarantee the bridge satisfies the routing seam.
var _ internalrouting.TelemetryRecorder = (*TelemetryBridge)(nil)

// Record translates the router's TelemetrySample into a metrics.Sample and
// persists it, swallowing (logging) any store error.
func (b *TelemetryBridge) Record(ctx context.Context, s internalrouting.TelemetrySample) {
	if err := b.store.Record(ctx, internalmetrics.Sample{
		QueryHash:    s.QueryHash,
		RoutedTypes:  s.RoutedTypes,
		ProviderHits: s.ProviderHits,
		ResultCount:  s.ResultCount,
		Degraded:     s.Degraded,
		Reranked:     s.Reranked,
		LatencyMs:    s.LatencyMs,
	}); err != nil {
		b.logger.Printf("metrics.TelemetryBridge.Record: %v", err)
	}
}
