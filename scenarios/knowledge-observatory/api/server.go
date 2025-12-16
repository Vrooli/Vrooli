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
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

// Config holds minimal runtime configuration
type Config struct {
	Port                  string
	DatabaseURL            string
	QdrantURL              string
	OllamaURL              string
	OllamaEmbeddingModel   string
	ResourceQdrantCLI      string
	ResourceCommandTimeout time.Duration
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
		Port:                  requireEnv("API_PORT"),
		DatabaseURL:            dbURL,
		QdrantURL:              strings.TrimSpace(os.Getenv("QDRANT_URL")),
		OllamaURL:              strings.TrimSpace(os.Getenv("OLLAMA_URL")),
		OllamaEmbeddingModel:   strings.TrimSpace(os.Getenv("OLLAMA_EMBEDDING_MODEL")),
		ResourceQdrantCLI:      strings.TrimSpace(os.Getenv("RESOURCE_QDRANT_CLI")),
		ResourceCommandTimeout: 5 * time.Second,
	}

	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
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
	// Health endpoint (infrastructure health check)
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	// Semantic search endpoint [REQ:KO-SS-001]
	s.router.HandleFunc("/api/v1/knowledge/search", s.handleSearch).Methods("POST")

	// Knowledge health metrics endpoint [REQ:KO-QM-004]
	s.router.HandleFunc("/api/v1/knowledge/health", s.handleHealthEndpoint).Methods("GET")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "knowledge-observatory-api",
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
	status := "healthy"
	dbStatus := "connected"

	if s.db == nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	} else if err := s.db.PingContext(r.Context()); err != nil {
		status = "unhealthy"
		dbStatus = "disconnected"
	}

	response := map[string]interface{}{
		"status":    status,
		"service":   "Knowledge Observatory API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
		"dependencies": map[string]string{
			"database": dbStatus,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func (s *Server) respondError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(map[string]string{"error": message})
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
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario start <name>' so lifecycle exports it.", key)
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

func (s *Server) qdrantURL() string {
	if s != nil && s.config != nil {
		if value := strings.TrimSpace(s.config.QdrantURL); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(os.Getenv("QDRANT_URL")); value != "" {
		return value
	}
	return "http://localhost:6333"
}

func (s *Server) ollamaURL() string {
	if s != nil && s.config != nil {
		if value := strings.TrimSpace(s.config.OllamaURL); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(os.Getenv("OLLAMA_URL")); value != "" {
		return value
	}
	return "http://localhost:11434"
}

func (s *Server) ollamaEmbeddingModel() string {
	if s != nil && s.config != nil {
		if value := strings.TrimSpace(s.config.OllamaEmbeddingModel); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(os.Getenv("OLLAMA_EMBEDDING_MODEL")); value != "" {
		return value
	}
	return "nomic-embed-text"
}

func (s *Server) resourceQdrantCLI() string {
	if s != nil && s.config != nil {
		if value := strings.TrimSpace(s.config.ResourceQdrantCLI); value != "" {
			return value
		}
	}
	if value := strings.TrimSpace(os.Getenv("RESOURCE_QDRANT_CLI")); value != "" {
		return value
	}
	return "resource-qdrant"
}

func (s *Server) resourceCommandTimeout() time.Duration {
	if s == nil || s.config == nil || s.config.ResourceCommandTimeout <= 0 {
		return 5 * time.Second
	}
	return s.config.ResourceCommandTimeout
}

func (s *Server) execResourceQdrant(ctx context.Context, args ...string) ([]byte, error) {
	timeout := s.resourceCommandTimeout()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	cmd := exec.CommandContext(ctx, s.resourceQdrantCLI(), args...)
	return cmd.Output()
}
