package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/policy"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/sandbox"
)

// Server wires the HTTP router, database, and services.
type Server struct {
	config   config.Config
	db       *sql.DB
	router   *mux.Router
	driver   driver.Driver
	handlers *handlers.Handlers
}

// NewServer initializes configuration, database, and routes.
func NewServer() (*Server, error) {
	// Load unified configuration from environment
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Build database URL
	dbURL, err := resolveDatabaseURL(cfg.Database)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Initialize driver from config
	driverCfg := driver.Config{
		BaseDir:          cfg.Driver.BaseDir,
		MaxSandboxes:     cfg.Limits.MaxSandboxes,
		MaxSizeMB:        cfg.Limits.MaxSandboxSizeMB,
		UseFuseOverlayfs: cfg.Driver.UseFuseOverlayfs,
	}
	drv := driver.NewOverlayfsDriver(driverCfg)

	// Initialize policies
	approvalPolicy := policy.NewDefaultApprovalPolicy(cfg.Policy)
	attributionPolicy, err := policy.NewDefaultAttributionPolicy(cfg.Policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create attribution policy: %w", err)
	}
	validationPolicy := policy.NewNoOpValidationPolicy()

	// Initialize repository and service
	repo := repository.NewSandboxRepository(db)
	svcCfg := sandbox.ServiceConfig{
		DefaultProjectRoot: cfg.Driver.ProjectRoot,
		MaxSandboxes:       cfg.Limits.MaxSandboxes,
		DefaultTTL:         cfg.Lifecycle.DefaultTTL,
	}
	svc := sandbox.NewService(repo, drv, svcCfg,
		sandbox.WithApprovalPolicy(approvalPolicy),
		sandbox.WithAttributionPolicy(attributionPolicy),
		sandbox.WithValidationPolicy(validationPolicy),
	)

	// Create handlers with injected dependencies
	h := &handlers.Handlers{
		Service: svc,
		Driver:  drv,
		DB:      db,
		Config:  cfg,
	}

	srv := &Server{
		config:   cfg,
		db:       db,
		router:   mux.NewRouter(),
		driver:   drv,
		handlers: h,
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	s.router.Use(corsMiddleware)

	h := s.handlers

	// Health endpoints
	s.router.HandleFunc("/health", h.Health).Methods("GET")
	s.router.HandleFunc("/api/v1/health", h.Health).Methods("GET")

	// Sandbox CRUD
	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/sandboxes", h.CreateSandbox).Methods("POST")
	api.HandleFunc("/sandboxes", h.ListSandboxes).Methods("GET")
	api.HandleFunc("/sandboxes/{id}", h.GetSandbox).Methods("GET")
	api.HandleFunc("/sandboxes/{id}", h.DeleteSandbox).Methods("DELETE")
	api.HandleFunc("/sandboxes/{id}/stop", h.StopSandbox).Methods("POST")

	// Diff and approval
	api.HandleFunc("/sandboxes/{id}/diff", h.GetDiff).Methods("GET")
	api.HandleFunc("/sandboxes/{id}/approve", h.Approve).Methods("POST")
	api.HandleFunc("/sandboxes/{id}/reject", h.Reject).Methods("POST")

	// Workspace path helper
	api.HandleFunc("/sandboxes/{id}/workspace", h.GetWorkspace).Methods("GET")

	// Driver info
	api.HandleFunc("/driver/info", h.DriverInfo).Methods("GET")
}

// Start launches the HTTP server with graceful shutdown.
func (s *Server) Start() error {
	log.Printf("starting server | service=workspace-sandbox-api port=%s", s.config.Server.Port)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%s", s.config.Server.Port),
		Handler:      gorillahandlers.RecoveryHandler()(s.router),
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	go func() {
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("server startup failed | error=%s", err.Error())
			log.Fatal(err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), s.config.Server.ShutdownTimeout)
	defer cancel()

	if err := httpServer.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown failed: %w", err)
	}

	log.Println("server stopped")
	return nil
}

// loggingMiddleware logs HTTP requests.
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// corsMiddleware adds CORS headers.
func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// resolveDatabaseURL builds database URL from config.
func resolveDatabaseURL(cfg config.DatabaseConfig) (string, error) {
	if cfg.URL != "" {
		return cfg.URL, nil
	}

	if cfg.Host == "" || cfg.Port == "" || cfg.User == "" || cfg.Password == "" || cfg.Name == "" {
		return "", fmt.Errorf("DATABASE_URL or POSTGRES_HOST/PORT/USER/PASSWORD/DB must be set by the lifecycle system")
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(cfg.User, cfg.Password),
		Host:   fmt.Sprintf("%s:%s", cfg.Host, cfg.Port),
		Path:   cfg.Name,
	}
	values := pgURL.Query()
	sslMode := cfg.SSLMode
	if sslMode == "" {
		sslMode = "disable"
	}
	values.Set("sslmode", sslMode)
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

func main() {
	if os.Getenv("VROOLI_LIFECYCLE_MANAGED") != "true" {
		fmt.Fprintf(os.Stderr, `This binary must be run through the Vrooli lifecycle system.

Instead, use:
   vrooli scenario start workspace-sandbox

The lifecycle system provides environment variables, port allocation,
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
