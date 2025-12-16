package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
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
	dbURL, err := resolveDatabaseURL()
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		Port:        requireEnv("API_PORT"),
		DatabaseURL: dbURL,
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize database handle: %w", err)
	}

	srv := &Server{
		config: cfg,
		db:     db,
		router: mux.NewRouter(),
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")
	s.router.HandleFunc("/api/v1/health", s.handleHealth).Methods("GET")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "git-control-tower-api",
		"port":    s.config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      handlers.RecoveryHandler()(s.router),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			s.log("server startup failed", map[string]interface{}{"error": err.Error()})
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	s.log("server stopped", nil)
	return nil
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	checks := NewHealthChecks(HealthCheckDeps{
		DB:      s.db,
		GitPath: "git",
		RepoDir: resolveRepoDir(),
	})
	result := checks.Run(r.Context())

	w.Header().Set("Content-Type", "application/json")
	if !result.Readiness {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(result)
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

func requireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

func resolveDatabaseURL() (string, error) {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw, nil
	}

	host := strings.TrimSpace(os.Getenv("POSTGRES_HOST"))
	port := strings.TrimSpace(os.Getenv("POSTGRES_PORT"))
	user := strings.TrimSpace(os.Getenv("POSTGRES_USER"))
	password := strings.TrimSpace(os.Getenv("POSTGRES_PASSWORD"))
	name := strings.TrimSpace(os.Getenv("POSTGRES_DB"))

	if host == "" || port == "" || user == "" || password == "" || name == "" {
		return "", fmt.Errorf("DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(user, password),
		Host:   fmt.Sprintf("%s:%s", host, port),
		Path:   name,
	}
	values := pgURL.Query()
	values.Set("sslmode", "disable")
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start git-control-tower

💡 The lifecycle system provides environment variables, port allocation,
   and dependency management automatically. Direct execution is not supported.
`)
		os.Exit(1)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}

func resolveRepoDir() string {
	if root := strings.TrimSpace(os.Getenv("VROOLI_ROOT")); root != "" {
		return root
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out))
	}
	return ""
}
