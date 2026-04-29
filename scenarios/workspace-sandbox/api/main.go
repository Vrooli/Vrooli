package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"

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
	"workspace-sandbox/internal/toolexecution"
	"workspace-sandbox/internal/toolregistry"
)

// Server wires the HTTP router, database, and services.
type Server struct {
	config           config.Config
	db               *sql.DB
	router           *mux.Router
	driver           driver.Driver
	handlers         *handlers.Handlers
	logger           *logging.Logger
	processTracker   *process.Tracker // OT-P0-008: Process/Session Tracking
	gcService        *gc.Service      // OT-P1-003: GC/Prune Operations
	lifecycleRecon   *sandbox.LifecycleReconciler
	metricsCollector *metrics.Collector // OT-P1-008: Metrics/Observability

	// Tool Discovery Protocol support
	toolRegistry *toolregistry.Registry
	toolHandler  *toolexecution.Handler
}

// NewServer initializes configuration, database, and routes.
func NewServer() (*Server, error) {
	// Load unified configuration from environment
	cfg, err := config.LoadFromEnv()
	if err != nil {
		return nil, fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve the embedded SQLite database path. Honors SQLITE_PATH for
	// explicit overrides; otherwise places the file under the cross-platform
	// data directory provided by api-core/storage.
	dsn, err := resolveSQLiteDSN()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve SQLite path: %w", err)
	}
	db, err := database.Connect(context.Background(), database.Config{
		Driver: database.DriverSQLite,
		DSN:    dsn,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	// SQLite serializes writes; cap the pool to a single connection so the
	// pragmas applied by the DSN govern every transaction.
	db.SetMaxOpenConns(1)

	// Apply the embedded schema. The file is idempotent (CREATE ... IF NOT
	// EXISTS) so this runs safely on every startup.
	if _, err := db.ExecContext(context.Background(), repository.SchemaSQL); err != nil {
		return nil, fmt.Errorf("failed to apply SQLite schema: %w", err)
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
	initialDriver, err := driver.SelectDriverWithPreference(context.Background(), driverCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize driver: %w", err)
	}
	// Wrap in Manager for hot-swap support
	driverManager := driver.NewManager(initialDriver, driverCfg)
	log.Printf("driver selected | type=%s version=%s", driverManager.Type(), driverManager.Version())

	// Initialize policies
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

	// Initialize teardown policy
	// If teardown hooks are configured, use HookTeardownPolicy to run
	// pre-teardown hooks before sandbox unmount/delete. This allows external
	// systems to evacuate processes from the sandbox's merged directory.
	var teardownPolicy policy.TeardownPolicy
	if len(cfg.Policy.TeardownHooks) > 0 {
		hooks := make([]policy.TeardownHook, len(cfg.Policy.TeardownHooks))
		for i, h := range cfg.Policy.TeardownHooks {
			hooks[i] = policy.TeardownHook{
				Name:        h.Name,
				Description: h.Description,
				Command:     h.Command,
				Args:        h.Args,
				Timeout:     h.Timeout,
			}
		}
		teardownPolicy = policy.NewHookTeardownPolicy(hooks,
			policy.WithTeardownGlobalTimeout(cfg.Policy.TeardownTimeout),
		)
		log.Printf("teardown hooks enabled | hooks=%d timeout=%v", len(hooks), cfg.Policy.TeardownTimeout)
	} else {
		teardownPolicy = policy.NewNoOpTeardownPolicy()
	}

	// Initialize repository and service
	repo := repository.NewSandboxRepository(db)
	svcCfg := sandbox.ServiceConfig{
		DefaultProjectRoot:      cfg.Driver.ProjectRoot,
		MaxSandboxes:            cfg.Limits.MaxSandboxes,
		DefaultTTL:              cfg.Lifecycle.DefaultTTL,
		DefaultNoLock:           cfg.Policy.DefaultNoLock,
		AgentManagerURL:         cfg.Integration.AgentManagerURL,
		AgentManagerSyncEnabled: cfg.Integration.AgentManagerSyncEnabled,
		AgentManagerSyncTimeout: cfg.Integration.AgentManagerSyncTimeout,
	}
	svc := sandbox.NewService(repo, driverManager, svcCfg,
		sandbox.WithAttributionPolicy(attributionPolicy),
		sandbox.WithValidationPolicy(validationPolicy),
		sandbox.WithTeardownPolicy(teardownPolicy),
	)
	healCfg := sandbox.HealConfig{
		IdleGracePeriod:        cfg.Lifecycle.AutoHealIdleGrace,
		MaxConsecutiveFailures: cfg.Lifecycle.AutoHealMaxRetries,
		BaseBackoff:            cfg.Lifecycle.AutoHealBaseBackoff,
	}
	lifecycleRecon := sandbox.NewLifecycleReconciler(svc, cfg.Lifecycle.GCInterval, healCfg).
		WithManualReviewTTL(cfg.Lifecycle.ManualReviewTTL)

	// Initialize process tracker (OT-P0-008)
	processTracker := process.NewTrackerWithConfig(process.TrackerConfig{
		GracePeriod: cfg.Lifecycle.ProcessGracePeriod,
		KillWait:    cfg.Lifecycle.ProcessKillWait,
	})

	// Initialize process logger (Phase 2)
	processLogger := process.NewLogger(process.DefaultLogConfig(cfg.Driver.BaseDir))

	// Initialize profile store for isolation profiles.
	scenarioDir, err := resolveWorkspaceSandboxScenarioDir()
	if err != nil {
		return nil, err
	}
	profileStore, err := config.NewFileProfileStore(scenarioDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize profile store: %w", err)
	}

	// Initialize GC service (OT-P1-003)
	gcCfg := gc.Config{
		DefaultMaxAge:        cfg.Lifecycle.DefaultTTL,
		DefaultIdleTimeout:   cfg.Lifecycle.IdleTimeout,
		DefaultTerminalDelay: cfg.Lifecycle.TerminalCleanupDelay,
		DefaultLimit:         100,
		MaxTotalSizeBytes:    cfg.Limits.MaxTotalSizeMB * 1024 * 1024,
	}
	gcService := gc.NewService(repo, driverManager, gcCfg)

	// Check if we're in a user namespace
	inUserNS := namespace.Check().InUserNamespace

	// Create handlers with injected dependencies
	h := &handlers.Handlers{
		Service:         svc,
		DriverManager:   driverManager,
		DB:              db,
		Config:          cfg,
		StatsGetter:     repo, // Repository implements StatsGetter
		ProcessTracker:  processTracker,
		ProcessLogger:   processLogger,
		GCService:       gcService,
		ProfileStore:    profileStore,
		InUserNamespace: inUserNS,
	}

	// Initialize structured logger
	logger := logging.New("workspace-sandbox-api")

	// Initialize metrics collector [OT-P1-008]
	metricsCollector := metrics.NewCollector()

	// --- Tool Discovery Protocol support ---
	// Initialize tool registry with all providers
	toolReg := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "workspace-sandbox",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Isolated workspace management with CoW filesystems for safe agent development",
	})

	// Register all tool providers (4 tiers)
	toolReg.RegisterProvider(toolregistry.NewSandboxToolProvider())   // Tier 1: Sandbox lifecycle
	toolReg.RegisterProvider(toolregistry.NewExecutionToolProvider()) // Tier 2: Command execution
	toolReg.RegisterProvider(toolregistry.NewFileToolProvider())      // Tier 3: File operations
	toolReg.RegisterProvider(toolregistry.NewDiffToolProvider())      // Tier 4: Diff/approval

	// Create adapters for tool execution
	processExecutor := toolexecution.NewProcessExecutorAdapter(toolexecution.ProcessExecutorConfig{
		SandboxService: svc,
		Driver:         driverManager,
		ProcessTracker: processTracker,
		ProcessLogger:  processLogger,
		ProfileStore:   profileStore,
		ExecConfig:     cfg.Execution,
	})
	fileOperator := toolexecution.NewFileOperatorAdapter(svc)

	// Create tool executor and handler
	toolExecutor := toolexecution.NewServerExecutor(toolexecution.ServerExecutorConfig{
		SandboxService:  svc,
		ProcessExecutor: processExecutor,
		FileOperator:    fileOperator,
	})
	toolHandler := toolexecution.NewHandler(toolExecutor)

	log.Printf("tool discovery protocol enabled | providers=%d", toolReg.ProviderCount())

	srv := &Server{
		config:           cfg,
		db:               db,
		router:           mux.NewRouter(),
		driver:           driverManager,
		handlers:         h,
		logger:           logger,
		processTracker:   processTracker,
		gcService:        gcService,
		lifecycleRecon:   lifecycleRecon,
		metricsCollector: metricsCollector,
		toolRegistry:     toolReg,
		toolHandler:      toolHandler,
	}

	srv.setupRoutes()

	logger.Info("server.initialized", "Server initialized successfully", map[string]interface{}{
		"port":         cfg.Server.Port,
		"driver":       driverManager.Type(),
		"maxSandboxes": cfg.Limits.MaxSandboxes,
	})

	return srv, nil
}

func (s *Server) setupRoutes() {
	// Apply middleware
	s.router.Use(s.structuredLoggingMiddleware)
	s.router.Use(s.corsMiddleware)

	// Health endpoint using api-core/health for standardized response format
	healthHandler := health.New().
		Version("1.0.0").
		Check(health.DB(s.db), health.Critical).
		Handler()
	s.router.HandleFunc("/health", healthHandler).Methods("GET")
	s.router.HandleFunc("/api/v1/health", healthHandler).Methods("GET")

	// Tool Discovery Protocol routes (agent-inbox integration)
	// GET /api/v1/tools - Get tool manifest with all available tools
	// GET /api/v1/tools/{name} - Get a specific tool definition
	// POST /api/v1/tools/execute - Execute a tool
	s.router.HandleFunc("/api/v1/tools", s.toolRegistry.HandleGetManifest).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/tools/{name}", s.toolRegistry.HandleGetTool).Methods("GET", "OPTIONS")
	s.router.HandleFunc("/api/v1/tools/execute", s.toolHandler.Execute).Methods("POST", "OPTIONS")

	// Delegate remaining route registration to handlers package
	// This centralizes route knowledge with the handlers and makes the API surface explicit
	s.handlers.RegisterRoutes(s.router, s.metricsCollector)
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return gorillahandlers.RecoveryHandler()(s.router)
}

// StartServices starts background services (call before server.Run)
func (s *Server) StartServices() {
	if s.lifecycleRecon != nil {
		s.lifecycleRecon.Start()
	}
}

// Cleanup releases resources when the server shuts down
func (s *Server) Cleanup() error {
	if s.lifecycleRecon != nil {
		s.lifecycleRecon.Stop()
	}
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

// responseWriter wraps http.ResponseWriter to capture status code.
//
// Embedding http.ResponseWriter does NOT propagate the writer's other
// optional interfaces (Flusher, Hijacker, Pusher) to type assertions on
// *responseWriter — Go interface satisfaction looks at the wrapper's own
// method set. SSE handlers in this service do `w.(http.Flusher)`; without
// explicit pass-through methods every SSE response would 500 with
// "streaming not supported", silently breaking the agent-manager log
// stream consumer (see ErrSandboxNoExitInfo). Each method below delegates
// to the underlying writer when it actually supports the interface, and
// no-ops otherwise so middleware composition stays robust.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if f, ok := rw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (rw *responseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	if h, ok := rw.ResponseWriter.(http.Hijacker); ok {
		return h.Hijack()
	}
	return nil, nil, http.ErrNotSupported
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

// resolveSQLiteDSN returns the modernc.org/sqlite DSN for the embedded
// store. It honors SQLITE_PATH for explicit overrides and otherwise resolves
// a cross-platform data path through api-core/storage. The DSN appends the
// pragmas every connection needs (WAL, busy_timeout, foreign_keys) and sets
// _txlock=immediate so BeginTx acquires the SQLite reserved lock up front
// for write-ordered Create + CheckScopeOverlap flows.
func resolveSQLiteDSN() (string, error) {
	path := strings.TrimSpace(os.Getenv("SQLITE_PATH"))
	if path == "" {
		resolver, err := storage.NewResolver(storage.ResolverConfig{
			AppID:   "vrooli",
			Profile: storage.ProfileAuto,
		})
		if err != nil {
			return "", fmt.Errorf("create storage resolver: %w", err)
		}
		dataDir, err := storage.EnsureClassDir(resolver,
			storage.Options{ScenarioID: "workspace-sandbox"},
			storage.ClassData,
			0o755,
		)
		if err != nil {
			return "", fmt.Errorf("ensure data dir: %w", err)
		}
		path = filepath.Join(dataDir, "workspace-sandbox.db")
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("create db parent dir: %w", err)
	}

	dsn := path +
		"?_pragma=journal_mode(WAL)" +
		"&_pragma=foreign_keys(1)" +
		"&_pragma=busy_timeout(5000)" +
		"&_pragma=synchronous(NORMAL)" +
		"&_txlock=immediate"

	log.Printf("workspace-sandbox: using SQLite database at %s", path)
	return dsn, nil
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "workspace-sandbox",
	}) {
		return // Process was re-exec'd after rebuild
	}

	// Decide whether to enter user namespace based on driver strategy.
	//
	// Key insight: fuse-overlayfs is already unprivileged (uses FUSE, not kernel mount).
	// If we enter a user namespace with private mount propagation, the fuse-overlayfs
	// mount becomes invisible to processes outside the namespace (like agent shells).
	//
	// Default behavior (optimized for agent integration):
	// - If fuse-overlayfs is available → stay in host namespace (mounts visible)
	// - If fuse-overlayfs unavailable → enter user namespace for native overlayfs
	//
	// Override with WORKSPACE_SANDBOX_PREFER_NATIVE_OVERLAYFS=true to force user
	// namespace even when fuse-overlayfs is available (better performance, isolated mounts).
	preferNativeOverlayfs := os.Getenv("WORKSPACE_SANDBOX_PREFER_NATIVE_OVERLAYFS") == "true" ||
		os.Getenv("WORKSPACE_SANDBOX_PREFER_NATIVE_OVERLAYFS") == "1"
	fuseAvailable, _, _ := driver.IsFuseOverlayfsAvailable()

	nsStatus := namespace.Check()

	// Decision logic:
	// 1. If already in namespace → continue (re-exec completed)
	// 2. If fuse available AND not preferring native → stay in host namespace
	// 3. If can create namespace AND (prefer native OR fuse unavailable) → enter namespace
	// 4. Otherwise → use fallback (copy driver)
	if nsStatus.InUserNamespace {
		log.Printf("running in user namespace | kernel=%s overlayfs=%v",
			nsStatus.KernelVersion, nsStatus.CanMountOverlayfs)
	} else if fuseAvailable && !preferNativeOverlayfs {
		// Best for agent integration: fuse-overlayfs in host namespace
		// Mounts are visible to all processes (agents, shells, file managers)
		log.Printf("using fuse-overlayfs in host namespace | mounts visible to all processes | kernel=%s",
			nsStatus.KernelVersion)
	} else if nsStatus.CanCreateUserNamespace {
		// Enter user namespace for native overlayfs (better performance, isolated mounts)
		log.Printf("entering user namespace for native overlayfs | kernel=%s | preferNative=%v fuseAvailable=%v",
			nsStatus.KernelVersion, preferNativeOverlayfs, fuseAvailable)
		if err := namespace.EnterUserNamespace(); err != nil {
			// EnterUserNamespace only returns on error; success replaces the process
			log.Printf("warning: failed to enter user namespace: %v (will use fallback driver)", err)
		}
	} else {
		log.Printf("no overlayfs available | kernel=%s reason=%s (will use copy driver)",
			nsStatus.KernelVersion, nsStatus.Reason)
	}

	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	// Start background services before HTTP server
	srv.StartServices()

	// Forward server.* timeouts from local config so the SSE log-stream
	// endpoint isn't capped by api-core's 30s WriteTimeout default.
	// WriteTimeout is intentionally 0 (disabled) for this service —
	// see config.Default for the rationale.
	if err := server.Run(server.Config{
		Handler:         srv.Router(),
		Port:            srv.config.Server.Port,
		ReadTimeout:     srv.config.Server.ReadTimeout,
		WriteTimeout:    srv.config.Server.WriteTimeout,
		IdleTimeout:     srv.config.Server.IdleTimeout,
		ShutdownTimeout: srv.config.Server.ShutdownTimeout,
		Cleanup: func(ctx context.Context) error {
			return srv.Cleanup()
		},
	}); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
