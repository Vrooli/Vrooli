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
	s := &Server{
		router:   mux.NewRouter(),
		registry: reg,
		cache:    NewCache(),
		stats:    NewStatsBuffer(1024, time.Hour),

		swarm:  upstream.NewSwarm(resolveSwarmBaseURL()),
		vrooli: upstream.NewVrooli(resolveVrooliBaseURL()),
		lpbs:   upstream.NewLPBS(resolveLPBSBaseURL(), os.Getenv("LPBS_SERVICE_TOKEN")),
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

func resolveSwarmBaseURL() string {
	if v := os.Getenv("SWARM_MANAGER_BASE_URL"); v != "" {
		return v
	}
	port := os.Getenv("SWARM_MANAGER_API_PORT")
	if port == "" {
		port = "36234"
	}
	return "http://localhost:" + port
}

func resolveVrooliBaseURL() string {
	if v := os.Getenv("VROOLI_CORE_BASE_URL"); v != "" {
		return v
	}
	return "http://localhost:8092"
}

func resolveLPBSBaseURL() string {
	if v := os.Getenv("LPBS_BASE_URL"); v != "" {
		return v
	}
	port := os.Getenv("LPBS_API_PORT")
	if port == "" {
		return ""
	}
	return "http://localhost:" + port
}
