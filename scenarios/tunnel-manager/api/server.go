package main

import (
	"database/sql"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"

	"tunnel-manager/handler"
	"tunnel-manager/service"
	"tunnel-manager/store"
)

// Server wires the HTTP router and database connection
type Server struct {
	db             *sql.DB
	router         *mux.Router
	routeSvc       *service.RouteService
	portAuditor    *service.PortAuditor
	tunnelHealth   *service.TunnelHealthChecker
	probeSvc       *service.ProbeService
	recoveryEngine *service.RecoveryEngine
	metricsStore   *store.MetricsStore
	probeStore     *store.ProbeStore
}

// NewServer initializes services and routes
func NewServer(db *sql.DB) *Server {
	routeStore := store.NewRouteStore(db)
	probeStore := store.NewProbeStore(db)
	metricsStore := store.NewMetricsStore(db)
	recoveryStore := store.NewRecoveryStore(db)

	routeSvc := service.NewRouteService(routeStore)
	scenariosRoot := detectScenariosRoot()

	tunnelHealth := service.NewTunnelHealthChecker()
	probeSvc := service.NewProbeService(routeSvc, probeStore)

	srv := &Server{
		db:             db,
		router:         mux.NewRouter(),
		routeSvc:       routeSvc,
		portAuditor:    service.NewPortAuditor(routeSvc, scenariosRoot),
		tunnelHealth:   tunnelHealth,
		probeSvc:       probeSvc,
		recoveryEngine: service.NewRecoveryEngine(recoveryStore, tunnelHealth),
		metricsStore:   metricsStore,
		probeStore:     probeStore,
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(s.db), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Route manifest CRUD
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/routes", handler.HandleListRoutes(s.routeSvc)).Methods("GET")
	api.HandleFunc("/routes", handler.HandleCreateRoute(s.routeSvc)).Methods("POST")
	api.HandleFunc("/routes/{id:[0-9]+}", handler.HandleGetRoute(s.routeSvc)).Methods("GET")
	api.HandleFunc("/routes/{id:[0-9]+}", handler.HandleUpdateRoute(s.routeSvc)).Methods("PUT", "PATCH")
	api.HandleFunc("/routes/{id:[0-9]+}", handler.HandleDeleteRoute(s.routeSvc)).Methods("DELETE")

	// Port compliance audit
	api.HandleFunc("/audit/ports", handler.HandlePortAudit(s.portAuditor)).Methods("GET")

	// Tunnel health
	api.HandleFunc("/tunnel/health", handler.HandleTunnelHealth(s.tunnelHealth)).Methods("GET")

	// Detailed health (cross-scenario consumption) [REQ:OBS-004]
	api.HandleFunc("/health/detailed", handler.HandleDetailedHealth(s.tunnelHealth, s.routeSvc, s.probeSvc)).Methods("GET")

	// Metrics history [REQ:OBS-001]
	api.HandleFunc("/metrics/history", handler.HandleMetricsHistory(s.metricsStore)).Methods("GET")
	api.HandleFunc("/metrics/latest", handler.HandleMetricsLatest(s.metricsStore)).Methods("GET")

	// Probe history [REQ:OBS-002]
	api.HandleFunc("/probes/history", handler.HandleProbeHistory(s.probeStore)).Methods("GET")

	// Liveness probes
	api.HandleFunc("/probes", handler.HandleRunProbes(s.probeSvc)).Methods("POST")

	// Recovery engine
	api.HandleFunc("/recovery/state", handler.HandleRecoveryState(s.recoveryEngine)).Methods("GET")
	api.HandleFunc("/recovery/trigger", handler.HandleRecoveryTrigger(s.recoveryEngine)).Methods("POST")
	api.HandleFunc("/recovery/events", handler.HandleRecoveryEvents(s.recoveryEngine)).Methods("GET")
	api.HandleFunc("/recovery/circuit/reset", handler.HandleCircuitReset(s.recoveryEngine)).Methods("POST")
}

// Handler returns the HTTP handler with recovery middleware
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// detectScenariosRoot finds the scenarios directory relative to the API binary.
func detectScenariosRoot() string {
	// Try environment variable first
	if root := os.Getenv("SCENARIOS_ROOT"); root != "" {
		return root
	}
	// Default: assume we're in scenarios/<name>/api/
	wd, err := os.Getwd()
	if err != nil {
		return "/home/matthalloran8/Vrooli/scenarios"
	}
	return filepath.Join(wd, "..", "..")
}
