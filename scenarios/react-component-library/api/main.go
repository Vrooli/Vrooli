package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
)

// Config holds minimal runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
}

// Server wires the HTTP router and database connection
type Server struct {
	config *Config
	db     *sql.DB
	router *mux.Router
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	// Connect to database with automatic retry and backoff.
	// Reads POSTGRES_* environment variables set by the lifecycle system.
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	srv := &Server{
		config: &Config{},
		db:     db,
		router: mux.NewRouter(),
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	// Health endpoints
	healthHandler := health.New().Version("1.0.0").Check(health.DB(s.db), health.Critical).Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Component endpoints
	s.router.HandleFunc("/api/v1/components", s.handleGetComponents).Methods("GET")
	s.router.HandleFunc("/api/v1/components", s.handleCreateComponent).Methods("POST")
	s.router.HandleFunc("/api/v1/components/search", s.handleSearchComponents).Methods("GET")
	s.router.HandleFunc("/api/v1/components/{id}", s.handleGetComponent).Methods("GET")
	s.router.HandleFunc("/api/v1/components/{id}", s.handleUpdateComponent).Methods("PUT")
	s.router.HandleFunc("/api/v1/components/{id}/content", s.handleGetComponentContent).Methods("GET")
	s.router.HandleFunc("/api/v1/components/{id}/content", s.handleUpdateComponentContent).Methods("PUT")
	s.router.HandleFunc("/api/v1/components/{id}/versions", s.handleGetComponentVersions).Methods("GET")

	// Adoption endpoints
	s.router.HandleFunc("/api/v1/adoptions", s.handleGetAdoptions).Methods("GET")
	s.router.HandleFunc("/api/v1/adoptions", s.handleCreateAdoption).Methods("POST")

	// AI endpoints
	s.router.HandleFunc("/api/v1/ai/chat", s.handleAIChat).Methods("POST")
	s.router.HandleFunc("/api/v1/ai/refactor", s.handleAIRefactor).Methods("POST")
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return handlers.RecoveryHandler()(s.router)
}

// Cleanup releases resources when the server shuts down
func (s *Server) Cleanup() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Println(msg)
		return
	}
	log.Printf("%s | %v", msg, fields)
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "react-component-library",
	}) {
		return // Process was re-exec'd after rebuild
	}

	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Run(server.Config{
		Handler: srv.Router(),
		Cleanup: func(ctx context.Context) error {
			return srv.Cleanup()
		},
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
