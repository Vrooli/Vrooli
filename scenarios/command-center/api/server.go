package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	"command-center/upstream"
)

// Server is the command-center HTTP aggregator.
type Server struct {
	router   *mux.Router
	registry *Registry
	cache    *Cache
	stats    *StatsBuffer

	swarm  upstream.Client
	vrooli upstream.Client
	lpbs   upstream.Client
}

// NewServer wires the router, cache, upstream clients, and registry.
func NewServer(reg *Registry) *Server {
	directory := newPortDirectory(resolveVrooliBaseURL())
	s := &Server{
		router:   mux.NewRouter(),
		registry: reg,
		cache:    NewCache(),
		stats:    NewStatsBuffer(1024, time.Hour),

		swarm:  upstream.NewSwarmResolved(directory.resolver("swarm-manager", "SWARM_MANAGER_BASE_URL", "SWARM_MANAGER_API_PORT")),
		vrooli: upstream.NewVrooli(resolveVrooliBaseURL()),
		lpbs:   upstream.NewLPBSResolved(directory.resolver("landing-page-business-suite", "LPBS_BASE_URL", "LPBS_API_PORT"), os.Getenv("LPBS_SERVICE_TOKEN")),
	}
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
	return "http://localhost:8092"
}
