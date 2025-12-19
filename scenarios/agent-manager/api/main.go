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
	"strings"
	"syscall"
	"time"

	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/handlers"
	"agent-manager/internal/orchestration"
	"agent-manager/internal/repository"

	gorillaHandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/preflight"
)

// Config holds runtime configuration
type Config struct {
	Port        string
	DatabaseURL string
	UseInMemory bool
}

// Server wires the HTTP router, database, and orchestration service
type Server struct {
	config       *Config
	db           *sql.DB
	router       *mux.Router
	orchestrator orchestration.Service
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	cfg := &Config{
		Port:        requireEnv("API_PORT"),
		UseInMemory: strings.ToLower(os.Getenv("USE_IN_MEMORY")) == "true",
	}

	var db *sql.DB
	var err error

	// Database connection (optional for in-memory mode)
	if !cfg.UseInMemory {
		cfg.DatabaseURL, err = resolveDatabaseURL()
		if err != nil {
			log.Printf("Database URL not configured, using in-memory storage: %v", err)
			cfg.UseInMemory = true
		} else {
			db, err = sql.Open("postgres", cfg.DatabaseURL)
			if err != nil {
				log.Printf("Failed to connect to database, using in-memory: %v", err)
				cfg.UseInMemory = true
			} else if err := db.Ping(); err != nil {
				log.Printf("Failed to ping database, using in-memory: %v", err)
				db.Close()
				db = nil
				cfg.UseInMemory = true
			}
		}
	}

	// Create the orchestrator with appropriate repositories
	orch := createOrchestrator(cfg.UseInMemory)

	srv := &Server{
		config:       cfg,
		db:           db,
		router:       mux.NewRouter(),
		orchestrator: orch,
	}

	srv.setupRoutes()
	return srv, nil
}

// createOrchestrator creates the orchestration service with all dependencies
func createOrchestrator(useInMemory bool) orchestration.Service {
	// Create repositories
	var (
		profileRepo     repository.ProfileRepository
		taskRepo        repository.TaskRepository
		runRepo         repository.RunRepository
		checkpointRepo  repository.CheckpointRepository
		idempotencyRepo repository.IdempotencyRepository
	)

	// Always use in-memory for now (PostgreSQL implementations would go here)
	profileRepo = repository.NewMemoryProfileRepository()
	taskRepo = repository.NewMemoryTaskRepository()
	runRepo = repository.NewMemoryRunRepository()
	checkpointRepo = repository.NewMemoryCheckpointRepository()
	idempotencyRepo = repository.NewMemoryIdempotencyRepository()

	// Create event store
	eventStore := event.NewMemoryStore()

	// Create runner registry
	runnerRegistry := runner.NewRegistry()

	// Register Claude Code runner (real implementation)
	claudeRunner, err := runner.NewClaudeCodeRunner()
	if err != nil {
		log.Printf("Warning: Failed to create Claude Code runner: %v", err)
		runnerRegistry.Register(runner.NewStubRunner(
			domain.RunnerTypeClaudeCode,
			fmt.Sprintf("claude-code runner failed to initialize: %v", err),
		))
	} else {
		runnerRegistry.Register(claudeRunner)
	}

	// Register stub runners for other types (real implementations would go here)
	runnerRegistry.Register(runner.NewStubRunner(
		domain.RunnerTypeCodex,
		"codex runner not configured",
	))
	runnerRegistry.Register(runner.NewStubRunner(
		domain.RunnerTypeOpenCode,
		"opencode runner not configured",
	))

	// Create workspace-sandbox provider
	sandboxURL := os.Getenv("WORKSPACE_SANDBOX_URL")
	if sandboxURL == "" {
		// Try to get from port allocation
		port := os.Getenv("WORKSPACE_SANDBOX_API_PORT")
		if port == "" {
			port = "15427" // Default workspace-sandbox port
		}
		sandboxURL = fmt.Sprintf("http://localhost:%s", port)
	}
	sandboxProvider := sandbox.NewWorkspaceSandboxProvider(sandboxURL)

	// Build orchestrator with all dependencies
	orch := orchestration.New(
		profileRepo,
		taskRepo,
		runRepo,
		orchestration.WithConfig(orchestration.OrchestratorConfig{
			DefaultTimeout:          30 * time.Minute,
			MaxConcurrentRuns:       10,
			RequireSandboxByDefault: true,
		}),
		orchestration.WithEvents(eventStore),
		orchestration.WithRunners(runnerRegistry),
		orchestration.WithSandbox(sandboxProvider),
		orchestration.WithCheckpoints(checkpointRepo),
		orchestration.WithIdempotency(idempotencyRepo),
	)

	log.Printf("Orchestrator initialized (in-memory: %v, sandbox: %s)", useInMemory, sandboxURL)
	return orch
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	// Register all API routes via the handlers package
	handler := handlers.New(s.orchestrator)
	handler.RegisterRoutes(s.router)
}

// Start launches the HTTP server with graceful shutdown
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": "agent-manager-api",
		"port":    s.config.Port,
	})

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Port),
		Handler:      gorillaHandlers.RecoveryHandler()(s.router),
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

	// Clean up database connection
	if s.db != nil {
		s.db.Close()
	}

	s.log("server stopped", nil)
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

// corsMiddleware adds CORS headers for development
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if len(fields) == 0 {
		log.Println(msg)
		return
	}
	data, _ := json.Marshal(fields)
	log.Printf("%s | %s", msg, string(data))
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
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "agent-manager",
	}) {
		return // Process was re-exec'd after rebuild
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
