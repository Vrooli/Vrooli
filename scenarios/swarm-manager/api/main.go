// Package main provides the Swarm Manager API server.
//
// DOC: docs/concepts/ARCHITECTURE.md#logical-architecture
// DOC: docs/internal/INTENT.md#api-components
//
// # Purpose
//
// This API serves as the backend for the Swarm Manager UI, providing endpoints
// for managing the scenario ecosystem: ideas backlog, scenario lifecycle, and
// recommendations.
//
// # Current Status: Ideas/Scenarios/Settings/Recommendations Implemented (File-Based)
//
// The API provides:
//   - Health check endpoints at / and /api/v1/health
//   - Ideas CRUD endpoints at /api/v1/ideas (GET, POST, PUT, DELETE)
//   - Idea queue + research endpoints at /api/v1/ideas/{name}/queue and /api/v1/ideas/{name}/research
//   - Scenario catalog endpoints at /api/v1/scenarios
//   - Settings persistence at /api/v1/settings
//   - Recommendations engine at /api/v1/recommendations
//
// # Architecture
//
// The server uses:
//   - gorilla/mux for HTTP routing
//   - api-core/health for standardized health responses
//   - api-core/server for graceful shutdown
//   - File-system based storage for ideas (git-tracked in scenarios/swarm-manager/ideas/)
//
// # Implemented Endpoints (P0)
//
//	GET    /api/v1/ideas          - List all ideas
//	POST   /api/v1/ideas          - Create new idea
//	GET    /api/v1/ideas/{name}   - Get idea by name
//	PUT    /api/v1/ideas/{name}   - Update idea
//	DELETE /api/v1/ideas/{name}   - Delete idea
//	POST   /api/v1/ideas/{name}/queue    - Queue idea for processing
//	POST   /api/v1/ideas/{name}/research - Spawn research agent
//
// # Scenario Endpoints (P0)
//
//	GET    /api/v1/scenarios      - List all scenarios (supports search, filter, sort)
//	GET    /api/v1/scenarios/{name} - Get scenario details
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
	"log"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"swarm-manager/internal/ideas"
	"swarm-manager/internal/queue"
	"swarm-manager/internal/recommendations"
	"swarm-manager/internal/scenarios"
	"swarm-manager/internal/settings"
)

type Server struct {
	router *mux.Router
}

// NewServer initializes routes. Database connection is optional.
func NewServer() *Server {
	srv := &Server{
		router: mux.NewRouter(),
	}
	srv.setupRoutes()
	return srv
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	// Uses api-core/health for standardized response format
	healthHandler := health.New().Version("1.0.0").Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Ideas CRUD endpoints
	// [REQ:REQ-P0-002] Ideas backlog management
	ideasHandler := ideas.NewHandler("")
	ideasHandler.RegisterRoutes(s.router)

	// Scenarios catalog endpoints
	// [REQ:REQ-P0-006] Scenario catalog with priority, search, and filter
	scenariosHandler := scenarios.NewHandler("")
	scenariosHandler.RegisterRoutes(s.router)

	// Settings persistence endpoints
	settingsHandler := settings.NewHandler("")
	settingsHandler.RegisterRoutes(s.router)

	// Recommendations endpoints (filesystem-backed engine)
	recommendationsHandler := recommendations.NewHandler("")
	recommendationsHandler.RegisterRoutes(s.router)

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

	// Start server with graceful shutdown (port from API_PORT env var)
	if err := server.Run(server.Config{
		Handler: srv.Handler(),
	}); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}
