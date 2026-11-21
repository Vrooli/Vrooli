// Package metrics is the search-hub metrics domain's API surface: the generated
// MetricsService Connect-RPC handler that aggregates per-query telemetry into
// federation-health insights (Phase 7). It sits beside the registry and routing
// domains and reads both the telemetry store and the provider registry (to mark
// registered-but-never-routed leaves as under-utilized).
package metrics

import (
	"context"
	"log"
	"net/http"

	"search-hub/internal/module"

	"github.com/vrooli/api-core/schedule"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/connectx"
	"github.com/vrooli/api-core/database"

	measuresconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/measures/measures_v1connect"
	metricsconnect "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/metrics/metrics_v1connect"

	internalmetrics "search-hub/internal/metrics"
	internalregistry "search-hub/internal/registry"
)

// Module returns the metrics domain's contribution to the API: the generated
// MetricsService Connect handler backed by the SQLite telemetry store and the
// provider registry store.
func Module(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) module.Module {
	store := internalmetrics.NewSQLiteStore(db, clk)
	serveHandler, err := MeasuresHandler(store, clk.Now)
	if err != nil {
		if logger == nil {
			logger = log.Default()
		}
		logger.Printf("metrics measures registry disabled: %v", err)
	}
	connectPath, connectHandler := metricsconnect.NewMetricsServiceHandler(NewConnectHandler(Deps{
		Insights:      store,
		RangeInsights: store,
		Lister:        internalregistry.NewSQLiteStore(db, clk),
		Logger:        logger,
	}))
	measuresPath, measuresConnectHandler := measuresconnect.NewMeasuresServiceHandler(NewMeasuresConnectHandler(store, clk.Now))
	return module.Module{
		Name: "metrics",
		Mount: func(r *mux.Router) {
			connectx.RegisterServices(r, connectx.ServiceMount{Path: connectPath, Handler: connectHandler})
			connectx.RegisterServices(r, connectx.ServiceMount{Path: measuresPath, Handler: measuresConnectHandler})
			if serveHandler != nil {
				r.PathPrefix("/measures/").Handler(http.StripPrefix("/measures", serveHandler))
			}
		},
		Endpoints: Endpoints,
	}
}

// Schema re-exports internalmetrics.Schema so the modules registry can collect
// both endpoint descriptors and schema from one symbol per handler package.
func Schema() string { return internalmetrics.Schema() }

// Migrate applies metrics-owned guarded migrations that must run before
// EnsureSchemas performs its declared-column drift check.
func Migrate(ctx context.Context, db *database.RoutedDB) error {
	return internalmetrics.Migrate(ctx, db.Primary())
}

// Recorder constructs the telemetry write bridge the routing domain uses. It
// adapts the metrics SQLite store to routing.TelemetryRecorder at the wiring
// edge, so internal/metrics never imports internal/routing. The bridge lives in
// recorder.go.
func Recorder(db *database.RoutedDB, clk schedule.Clock, logger *log.Logger) *TelemetryBridge {
	return NewTelemetryBridge(internalmetrics.NewSQLiteStore(db, clk), logger)
}
