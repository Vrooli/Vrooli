package wiring

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"agent-manager/internal/adapters/database"
	"agent-manager/internal/conformance"
	"agent-manager/internal/eventlog"
	"agent-manager/internal/handlers"
	healthstore "agent-manager/internal/health"
	"agent-manager/internal/httpmw"
	"agent-manager/internal/invocationreadmodel"
	analyticsmeasures "agent-manager/internal/measures"
	"agent-manager/internal/metrics"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/permissionpolicy"
	"agent-manager/internal/pricing"
	"agent-manager/internal/repository"
	"agent-manager/internal/rolepolicy"
	"agent-manager/internal/stats"
	"agent-manager/internal/storage"

	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/health"
	measureconnect "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/measures/measures_v1connect"
	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
)

// RouteDependencies names every runtime capability used by HTTP route
// registration. Keeping this composition data in wiring prevents the entry
// point from acquiring presentation or business logic.
type RouteDependencies struct {
	DB                    *database.DB
	Orchestrator          *orchestration.Orchestrator
	StatsService          orchestration.StatsService
	StatsRepository       repository.StatsRepository
	PricingService        pricing.Service
	WebSocketHub          *handlers.WebSocketHub
	RolePolicyState       *rolepolicy.State
	PermissionPolicyState *permissionpolicy.State
	PermissionPolicy      *permissionpolicy.Service
	Storage               storage.Service
	StatsEngine           *stats.Engine
	HealthStore           *healthstore.Store
	EventRepository       eventlog.Repository
	InvocationReadModel   invocationreadmodel.Store
}

// SetupRoutes registers the complete HTTP surface. It intentionally owns only
// transport composition; every route's behavior remains in handlers.
func SetupRoutes(router *mux.Router, deps RouteDependencies) {
	router.Use(httpmw.Logging)
	router.Use(httpmw.SecurityHeaders)
	router.Use(httpmw.CORS)

	healthHandler := health.New().
		Version("1.0.0").
		Check(health.Func("database", func(ctx context.Context) error {
			if deps.DB == nil {
				return fmt.Errorf("database is not configured")
			}
			return deps.DB.PingContext(ctx)
		}), health.Critical).
		Check(RolePolicyHealthChecker(deps.RolePolicyState), health.Critical).
		Check(PermissionPolicyHealthChecker(deps.PermissionPolicyState, deps.PermissionPolicy), health.Critical).
		Handler()
	router.HandleFunc("/health", healthHandler).Methods("GET")

	eventsBaseURL := os.Getenv("VROOLI_EVENTS_API_BASE")
	if eventsBaseURL == "" {
		if resolved, err := discovery.ResolveScenarioURLDefault(context.Background(), "vrooli-events"); err == nil {
			eventsBaseURL = resolved
		}
	}
	handler := handlers.New(
		orchestration.NewHandlerServices(deps.Orchestrator),
		handlers.WithStorage(deps.Storage),
		handlers.WithRolePolicyState(deps.RolePolicyState),
		handlers.WithPermissionPolicy(deps.PermissionPolicyState, deps.PermissionPolicy),
		handlers.WithObservedReceipts(eventbus.Client{BaseURL: eventsBaseURL}),
	)
	handler.SetWebSocketHub(deps.WebSocketHub)
	router.HandleFunc("/api/v1/health", handler.Health).Methods("GET")

	repoRoot := os.Getenv("PROJECT_ROOT")
	if repoRoot == "" {
		repoRoot, _ = filepath.Abs(filepath.Join("..", "..", ".."))
	}
	conformancePath, conformanceHandler := scenariovalidationconnect.NewScenarioValidationServiceHandler(conformance.NewHandler(repoRoot, deps.PermissionPolicy))
	router.PathPrefix(strings.TrimRight(conformancePath, "/")).Handler(conformanceHandler)
	handler.RegisterRoutes(router)

	routesLog := obs.Component("routes")
	if deps.StatsService != nil {
		handlers.NewStatsHandler(deps.StatsService).RegisterRoutes(router)
		routesLog.Info("stats endpoints registered", "path", "/api/v1/stats/*")
	}
	if deps.StatsEngine != nil {
		handlers.NewOperationalStatsHandler(deps.StatsEngine).RegisterRoutes(router)
		routesLog.Info("operational stats endpoints registered", "path", "/api/v1/stats/operational, /api/v1/stats/fallback")
	}
	if deps.HealthStore != nil {
		handlers.NewHealthAuditHandler(deps.HealthStore).RegisterRoutes(router)
		routesLog.Info("health audit endpoints registered", "path", "/api/v1/health/{models,runners,audit}")
	}
	if deps.EventRepository != nil {
		handlers.NewEventsHandler(deps.EventRepository).RegisterRoutes(router)
		routesLog.Info("events endpoint registered", "path", "/api/v1/events")
	}
	if deps.InvocationReadModel != nil {
		measureHandler := analyticsmeasures.NewHandler(deps.InvocationReadModel, nil)
		if registryHandler, err := measureHandler.MeasuresHandler(); err != nil {
			routesLog.Error("measures registry unavailable", obs.KeyError, err.Error())
		} else {
			router.PathPrefix("/measures").Handler(http.StripPrefix("/measures", registryHandler))
			connectPath, connectHandler := measureconnect.NewMeasuresServiceHandler(measureHandler)
			router.PathPrefix(strings.TrimRight(connectPath, "/")).Handler(connectHandler)
			routesLog.Info("durable friction measures registered", "paths", "/measures, "+connectPath)
		}
	}
	if deps.PricingService != nil && deps.StatsRepository != nil {
		handlers.NewPricingHandler(deps.PricingService, deps.StatsRepository).RegisterRoutes(router)
		routesLog.Info("pricing endpoints registered", "path", "/api/v1/pricing/*")
	}
	router.Handle("/metrics", metrics.Handler()).Methods("GET")
	routesLog.Info("websocket endpoint registered", "path", "/api/v1/ws")
	routesLog.Info("metrics endpoint registered", "path", "/metrics")
}
