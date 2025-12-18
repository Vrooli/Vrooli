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
	"strings"
	"syscall"
	"time"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/preflight"

	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/gc"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/logging"
	"workspace-sandbox/internal/metrics"
	"workspace-sandbox/internal/namespace"
	"workspace-sandbox/internal/policy"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/sandbox"
)

// Server wires the HTTP router, database, and services.
type Server struct {
	config           config.Config
	db               *sql.DB
	router           *mux.Router
	driver           driver.Driver
	handlers         *handlers.Handlers
	logger           *logging.Logger
	processTracker   *process.Tracker   // OT-P0-008: Process/Session Tracking
	gcService        *gc.Service        // OT-P1-003: GC/Prune Operations
	metricsCollector *metrics.Collector // OT-P1-008: Metrics/Observability
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

	// Initialize driver with automatic selection and fallback
	// Respects saved preference if available, otherwise:
	// Priority: native overlayfs (in user namespace) > fuse-overlayfs > copy driver
	driverCfg := driver.Config{
		BaseDir:          cfg.Driver.BaseDir,
		MaxSandboxes:     cfg.Limits.MaxSandboxes,
		MaxSizeMB:        cfg.Limits.MaxSandboxSizeMB,
		UseFuseOverlayfs: cfg.Driver.UseFuseOverlayfs,
	}
	drv, err := driver.SelectDriverWithPreference(context.Background(), driverCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize driver: %w", err)
	}
	log.Printf("driver selected | type=%s version=%s", drv.Type(), drv.Version())

	// Initialize policies
	approvalPolicy := policy.NewDefaultApprovalPolicy(cfg.Policy)
	attributionPolicy, err := policy.NewDefaultAttributionPolicy(cfg.Policy)
	if err != nil {
		return nil, fmt.Errorf("failed to create attribution policy: %w", err)
	}

	// Initialize validation policy [OT-P1-005]
	// If validation hooks are configured, use HookValidationPolicy
	var validationPolicy policy.ValidationPolicy
	if len(cfg.Policy.ValidationHooks) > 0 {
		hooks := make([]policy.ValidationHook, len(cfg.Policy.ValidationHooks))
		for i, h := range cfg.Policy.ValidationHooks {
			hooks[i] = policy.ValidationHook{
				Name:        h.Name,
				Description: h.Description,
				Command:     h.Command,
				Args:        h.Args,
				Required:    h.Required,
			}
		}
		validationPolicy = policy.NewHookValidationPolicy(hooks,
			policy.WithGlobalTimeout(cfg.Policy.ValidationTimeout),
		)
		log.Printf("validation hooks enabled | hooks=%d timeout=%v", len(hooks), cfg.Policy.ValidationTimeout)
	} else {
		validationPolicy = policy.NewNoOpValidationPolicy()
	}

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

	// Initialize process tracker (OT-P0-008)
	processTracker := process.NewTrackerWithConfig(process.TrackerConfig{
		GracePeriod: cfg.Lifecycle.ProcessGracePeriod,
		KillWait:    cfg.Lifecycle.ProcessKillWait,
	})

	// Initialize GC service (OT-P1-003)
	gcCfg := gc.Config{
		DefaultMaxAge:        cfg.Lifecycle.DefaultTTL,
		DefaultIdleTimeout:   cfg.Lifecycle.IdleTimeout,
		DefaultTerminalDelay: cfg.Lifecycle.TerminalCleanupDelay,
		DefaultLimit:         100,
		MaxTotalSizeBytes:    cfg.Limits.MaxTotalSizeMB * 1024 * 1024,
	}
	gcService := gc.NewService(repo, drv, gcCfg)

	// Check if we're in a user namespace
	inUserNS := namespace.Check().InUserNamespace

	// Create handlers with injected dependencies
	h := &handlers.Handlers{
		Service:         svc,
		Driver:          drv,
		DB:              db,
		Config:          cfg,
		StatsGetter:     repo, // Repository implements StatsGetter
		ProcessTracker:  processTracker,
		GCService:       gcService,
		InUserNamespace: inUserNS,
	}

	// Initialize structured logger
	logger := logging.New("workspace-sandbox-api")

	// Initialize metrics collector [OT-P1-008]
	metricsCollector := metrics.NewCollector()

	srv := &Server{
		config:           cfg,
		db:               db,
		router:           mux.NewRouter(),
		driver:           drv,
		handlers:         h,
		logger:           logger,
		processTracker:   processTracker,
		gcService:        gcService,
		metricsCollector: metricsCollector,
	}

	srv.setupRoutes()

	logger.Info("server.initialized", "Server initialized successfully", map[string]interface{}{
		"port":         cfg.Server.Port,
		"driver":       drv.Type(),
		"maxSandboxes": cfg.Limits.MaxSandboxes,
	})

	return srv, nil
}

func (s *Server) setupRoutes() {
	// Apply middleware
	s.router.Use(s.structuredLoggingMiddleware)
	s.router.Use(s.corsMiddleware)

	// Delegate route registration to handlers package
	// This centralizes route knowledge with the handlers and makes the API surface explicit
	s.handlers.RegisterRoutes(s.router, s.metricsCollector)
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

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// structuredLoggingMiddleware logs HTTP requests with structured JSON output.
func (s *Server) structuredLoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Wrap response writer to capture status
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		// Add logger to context for handlers
		ctx := logging.WithLogger(r.Context(), s.logger)
		r = r.WithContext(ctx)

		next.ServeHTTP(wrapped, r)

		duration := time.Since(start)
		s.logger.APIRequest(r.Method, r.RequestURI, wrapped.statusCode, float64(duration.Milliseconds()))
	})
}

// corsMiddleware returns a handler that adds CORS headers based on config.
// If CORSAllowedOrigins is empty, it allows the UI port origin for local dev.
// Otherwise, it checks the Origin header against the allowed list.
func (s *Server) corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")

		allowedOrigins := s.config.Server.CORSAllowedOrigins
		if len(allowedOrigins) == 0 {
			// Default: allow local UI port for development
			// This is more secure than "*" while still supporting local dev
			uiPort := os.Getenv("UI_PORT")
			if uiPort != "" {
				allowedOrigins = []string{
					"http://localhost:" + uiPort,
					"http://127.0.0.1:" + uiPort,
				}
			}
		}

		// Check if origin is allowed
		originAllowed := false
		for _, allowed := range allowedOrigins {
			if origin == allowed {
				originAllowed = true
				break
			}
		}

		if originAllowed && origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
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
		// Append search_path if schema is set and not already in URL
		if cfg.Schema != "" && !strings.Contains(cfg.URL, "search_path") {
			sep := "?"
			if strings.Contains(cfg.URL, "?") {
				sep = "&"
			}
			return cfg.URL + sep + "search_path=" + url.QueryEscape(cfg.Schema) + ",public", nil
		}
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

	// Set search_path to use the schema
	if cfg.Schema != "" {
		values.Set("search_path", cfg.Schema+",public")
	}
	pgURL.RawQuery = values.Encode()

	return pgURL.String(), nil
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "workspace-sandbox",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Enter user namespace for unprivileged overlayfs support (Linux 5.11+)
	// This re-execs the process inside a user namespace where we appear as root
	// and can mount overlayfs without actual root privileges.
	//
	// If user namespaces aren't available (older kernel, container restrictions),
	// we continue without them and fall back to copy driver or fuse-overlayfs.
	nsStatus := namespace.Check()
	if !nsStatus.InUserNamespace && nsStatus.CanCreateUserNamespace {
		log.Printf("entering user namespace for unprivileged overlayfs | kernel=%s", nsStatus.KernelVersion)
		if err := namespace.EnterUserNamespace(); err != nil {
			// EnterUserNamespace only returns on error; success replaces the process
			log.Printf("warning: failed to enter user namespace: %v (will use fallback driver)", err)
		}
	} else if !nsStatus.InUserNamespace {
		log.Printf("user namespace not available: %s (will use fallback driver)", nsStatus.Reason)
	} else {
		log.Printf("running in user namespace | kernel=%s overlayfs=%v",
			nsStatus.KernelVersion, nsStatus.CanMountOverlayfs)
	}

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
