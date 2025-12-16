package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
)

// Config holds runtime configuration.
type Config struct {
	Port string
}

// Server wires the HTTP router and rule/config handlers.
type Server struct {
	config      *Config
	router      *mux.Router
	scenarioRoot string
	configStore *ConfigStore
}

// NewServer initializes configuration and routes.
func NewServer() (*Server, error) {
	cfg := &Config{
		Port: requireEnv("API_PORT"),
	}

	scenarioRoot, err := scenarioRootFromCWD()
	if err != nil {
		return nil, err
	}

	srv := &Server{
		config:      cfg,
		router:      mux.NewRouter(),
		scenarioRoot: scenarioRoot,
		configStore: NewConfigStore(configPathForScenario(scenarioRoot)),
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)

	// CORS for local UI development / serving.
	cors := handlers.CORS(
		handlers.AllowedOrigins([]string{"*"}),
		handlers.AllowedMethods([]string{"GET", "POST", "PUT", "OPTIONS"}),
		handlers.AllowedHeaders([]string{"Content-Type", "Authorization"}),
	)

	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	s.router.Handle("/health", cors(http.HandlerFunc(s.handleHealth))).Methods("GET", "OPTIONS")
	s.router.Handle("/api/v1/health", cors(http.HandlerFunc(s.handleHealth))).Methods("GET", "OPTIONS")

	s.router.Handle("/api/v1/rules", cors(http.HandlerFunc(s.handleListRules))).Methods("GET", "OPTIONS")
	s.router.Handle("/api/v1/config", cors(http.HandlerFunc(s.handleGetConfig))).Methods("GET", "OPTIONS")
	s.router.Handle("/api/v1/config", cors(http.HandlerFunc(s.handlePutConfig))).Methods("PUT", "OPTIONS")
	s.router.Handle("/api/v1/run", cors(http.HandlerFunc(s.handleRunRules))).Methods("POST", "OPTIONS")
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "scenario-stack-governor-api",
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

	response := map[string]interface{}{
		"status":    status,
		"service":   "Scenario Stack Governor API",
		"version":   "1.0.0",
		"readiness": status == "healthy",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
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
	value := trimEnv(key)
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `❌ This binary must be run through the Vrooli lifecycle system.

🚀 Instead, use:
   vrooli scenario start scenario-stack-governor

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
