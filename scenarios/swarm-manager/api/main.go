// Package main provides the Swarm Manager API server.
//
// DOC: docs/concepts/ARCHITECTURE.md#logical-architecture
// DOC: docs/internal/INTENT.md#api-components
//
// # Purpose
//
// This API serves as the backend for the Swarm Manager UI, providing endpoints
// for managing the scenario ecosystem: backlog, scenario lifecycle, and
// recommendations.
//
// # Current Status: Backlog/Scenarios/Settings/Recommendations Implemented (File-Based)
//
// The API provides:
//   - Health check endpoints at / and /api/v1/health
//   - Backlog CRUD endpoints at /api/v1/backlog (GET, POST, PUT, DELETE)
//   - Backlog queue + research endpoints at /api/v1/backlog/{kind}/{name}/queue and /api/v1/backlog/{kind}/{name}/research
//   - Scenario catalog endpoints at /api/v1/scenarios
//   - Settings persistence at /api/v1/settings
//   - Recommendations engine at /api/v1/recommendations
//   - Agent-manager status at /api/v1/agent-manager/status
//
// # Architecture
//
// The server uses:
//   - gorilla/mux for HTTP routing
//   - api-core/health for standardized health responses
//   - api-core/server for graceful shutdown
//   - File-system based storage for backlog items (git-tracked in scenarios/swarm-manager/{ideas,research,fix,execute}/)
//
// # Implemented Endpoints (P0)
//
//	GET    /api/v1/backlog                   - List all backlog items
//	POST   /api/v1/backlog                   - Create new backlog item
//	GET    /api/v1/backlog/{kind}/{name}     - Get backlog item by name
//	PUT    /api/v1/backlog/{kind}/{name}     - Update backlog item
//	DELETE /api/v1/backlog/{kind}/{name}     - Delete backlog item
//	POST   /api/v1/backlog/{kind}/{name}/queue    - Queue backlog item for processing
//	POST   /api/v1/backlog/{kind}/{name}/research - Spawn research agent
//	POST   /api/v1/backlog/{kind}/{name}/convert  - Convert backlog item to another kind
//
// # Scenario Endpoints (P0)
//
//	GET    /api/v1/scenarios      - List all scenarios (supports search, filter, sort)
//	GET    /api/v1/scenarios/{name} - Get scenario details
//	POST   /api/v1/scenarios/{name}/start - Start scenario via CLI
//	POST   /api/v1/scenarios/{name}/stop - Stop scenario via CLI
//	POST   /api/v1/scenarios/{name}/restart - Restart scenario via CLI
//
// # Settings Endpoints (P1)
//
//	GET    /api/v1/settings       - Fetch settings
//	PUT    /api/v1/settings       - Update settings (partial)
//
// # Recommendations Endpoints (P1)
//
//	GET    /api/v1/recommendations          - List recommendations
//	POST   /api/v1/recommendations          - Create manual recommendation
//	POST   /api/v1/recommendations/refresh  - Refresh via engine
//	PATCH  /api/v1/recommendations/{id}     - Update recommendation status
//	POST   /api/v1/recommendations/{id}/start - Spawn agent run for recommendation
//
// # Agent-manager Endpoints
//
//	GET /api/v1/agent-manager/status - Agent-manager availability and profile status
//
// # Queue Endpoints (Local)
//
//	GET    /api/v1/queue          - List queue items
//	POST   /api/v1/queue          - Enqueue item
//	DELETE /api/v1/queue/{id}     - Remove item (idempotent)
//
// Related PRD targets: OT-P0-002, OT-P0-005, OT-P0-006
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/backlog"
	"swarm-manager/internal/queue"
	"swarm-manager/internal/recommendations"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
)

type Server struct {
	router   *mux.Router
	agentSvc *agentmanager.AgentService
}

// NewServer initializes routes. Database connection is optional.
func NewServer() *Server {
	agentEnabled := strings.ToLower(strings.TrimSpace(os.Getenv("AGENT_MANAGER_ENABLED"))) != "false"
	agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
		ProfileName: getEnvDefault("AGENT_MANAGER_PROFILE_NAME", "swarm-manager"),
		ProfileKey:  getEnvDefault("AGENT_MANAGER_PROFILE_KEY", "swarm-manager"),
		Timeout:     30 * time.Second,
		Enabled:     agentEnabled,
	})

	srv := &Server{
		router:   mux.NewRouter(),
		agentSvc: agentSvc,
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	// Uses api-core/health for standardized response format
	healthHandler := health.New().Version("1.0.0").
		Check(health.CheckerFunc(func(_ context.Context) health.CheckResult {
			return health.CheckResult{Name: "database", Connected: true}
		}), health.Optional).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Backlog endpoints
	// [REQ:REQ-P0-002] Backlog management
	backlogHandler := backlog.NewHandlerWithClients("", s.agentSvc)
	backlogHandler.RegisterRoutes(s.router)

	// Scenarios catalog endpoints
	// [REQ:REQ-P0-006] Scenario catalog with priority, search, and filter
	scenariosHandler := scenarios.NewHandler("")
	scenariosHandler.RegisterRoutes(s.router)

	// Settings persistence endpoints
	settingsHandler := settings.NewHandler("")
	settingsHandler.RegisterRoutes(s.router)

	// Recommendations endpoints (filesystem-backed engine)
	recommendationsHandler := recommendations.NewHandlerWithServices(
		recommendations.NewStore(""),
		recommendations.NewEngine(""),
		settings.NewStore(""),
		s.agentSvc,
	)
	recommendationsHandler.RegisterRoutes(s.router)

	// Agent-manager status endpoint
	agentManagerHandler := agentmanager.NewHandler(s.agentSvc)
	agentManagerHandler.RegisterRoutes(s.router)

	// Local queue endpoints (filesystem-backed)
	queueHandler := queue.NewHandler("")
	queueHandler.RegisterRoutes(s.router)
}

// Handler returns the HTTP handler with recovery middleware
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "swarm-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	log.Printf("Running in filesystem-only mode")
	srv := NewServer()

	if srv.agentSvc != nil && srv.agentSvc.IsEnabled() {
		initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := srv.agentSvc.Initialize(initCtx, agentmanager.DefaultProfileConfig()); err != nil {
			log.Printf("[agent-manager] Warning: failed to initialize profile: %v", err)
		}
		cancel()
	}

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func getEnvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
