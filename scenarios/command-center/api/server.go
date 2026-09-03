package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	capabilityregistry "github.com/vrooli/vrooli/packages/capability-registry-go"

	"command-center/upstream"
)

// Server is the command-center HTTP aggregator.
type Server struct {
	router              *mux.Router
	registry            *Registry
	cache               *Cache
	stats               *StatsBuffer
	integrationRegistry *capabilityregistry.Registry
	actionService       capabilityregistry.LifecycleActionService
	instrumentProvider  InstrumentProvider
	objectiveProvider   ObjectiveProvider
	promptManager       *promptManagerInstrumentProvider

	swarm  upstream.Client
	vrooli upstream.Client
	lpbs   upstream.Client
}

// NewServer wires the router, cache, upstream clients, and registry.
func NewServer(reg *Registry) *Server {
	s := &Server{
		router:   mux.NewRouter(),
		registry: reg,
		cache:    NewCache(),
		stats:    NewStatsBuffer(1024, time.Hour),

		swarm:  upstream.NewSwarmTypedResolved(resolveScenarioBaseURL("swarm-manager", "SWARM_MANAGER_BASE_URL", "SWARM_MANAGER_API_PORT")),
		vrooli: upstream.NewVrooliTypedResolved(resolveVrooliBaseURL),
		lpbs:   upstream.NewLPBSTypedResolved(resolveScenarioBaseURL("landing-page-business-suite", "LPBS_BASE_URL", "LPBS_API_PORT"), os.Getenv("LPBS_SERVICE_TOKEN")),
	}
	transmitter := newPromptManagerInstrumentProvider(resolveScenarioBaseURL("prompt-manager", "PROMPT_MANAGER_BASE_URL", "PROMPT_MANAGER_API_PORT"))
	s.instrumentProvider = transmitter
	s.objectiveProvider = transmitter
	s.promptManager = transmitter
	s.integrationRegistry = commandCenterIntegrationRegistry(s)
	if len(reg.Metrics) > 0 {
		if err := validateOutcomeBindings(s.integrationRegistry.Definitions(), reg.Metrics); err != nil {
			panic("command-center outcome binding validation failed: " + err.Error())
		}
	}
	s.actionService = capabilityregistry.LifecycleActionService{Defs: s.integrationRegistry.Definitions(), CLIPath: os.Getenv("VROOLI_CLI_PATH"), Timeout: 180 * time.Second}
	s.setupRoutes()
	return s
}

// Handler returns the HTTP handler wrapped with recovery middleware.
func (s *Server) Handler() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// Shutdown releases any resources held by the server.
func (s *Server) Shutdown(_ context.Context) error {
	return nil
}

// loggingMiddleware emits a compact structured log line per request.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		slog.Info("request",
			"method", r.Method,
			"uri", r.RequestURI,
			"duration_ms", time.Since(start).Milliseconds(),
		)
	})
}

func resolveVrooliBaseURL() string {
	if v := os.Getenv("VROOLI_CORE_BASE_URL"); v != "" {
		return v
	}
	return ""
}
