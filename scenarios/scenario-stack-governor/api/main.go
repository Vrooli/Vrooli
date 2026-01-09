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
	healthHandler := health.Handler()
	s.router.Handle("/health", cors(http.HandlerFunc(healthHandler))).Methods("GET", "OPTIONS")
	s.router.Handle("/api/v1/health", cors(http.HandlerFunc(healthHandler))).Methods("GET", "OPTIONS")

	s.router.Handle("/api/v1/rules", cors(http.HandlerFunc(s.handleListRules))).Methods("GET", "OPTIONS")
	s.router.Handle("/api/v1/config", cors(http.HandlerFunc(s.handleGetConfig))).Methods("GET", "OPTIONS")
	s.router.Handle("/api/v1/config", cors(http.HandlerFunc(s.handlePutConfig))).Methods("PUT", "OPTIONS")
	s.router.Handle("/api/v1/run", cors(http.HandlerFunc(s.handleRunRules))).Methods("POST", "OPTIONS")
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "scenario-stack-governor",
	}) {
		return // Process was re-exec'd after rebuild
	}

	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Run(server.Config{
		Handler: srv.Router(),
	}); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
