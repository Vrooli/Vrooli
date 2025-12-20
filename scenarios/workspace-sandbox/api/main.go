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
	"path/filepath"
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

	// Run automatic migrations
	if err := ensureSchema(db); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
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
	svc := sandbox.NewService(repo, driverManager, svcCfg,
		sandbox.WithApprovalPolicy(approvalPolicy),
		sandbox.WithAttributionPolicy(attributionPolicy),
		sandbox.WithValidationPolicy(validationPolicy),
	)

	// Initialize process tracker (OT-P0-008)
	processTracker := process.NewTrackerWithConfig(process.TrackerConfig{
		GracePeriod: cfg.Lifecycle.ProcessGracePeriod,
		KillWait:    cfg.Lifecycle.ProcessKillWait,
	})

	// Initialize process logger (Phase 2)
	processLogger := process.NewLogger(process.DefaultLogConfig(cfg.Driver.BaseDir))

	// Initialize profile store for isolation profiles
	// Determine scenario base directory from VROOLI_ROOT
	scenarioDir := os.Getenv("VROOLI_ROOT")
	if scenarioDir != "" {
		scenarioDir = filepath.Join(scenarioDir, "scenarios", "workspace-sandbox")
	} else {
		// Fallback to current directory
		scenarioDir, _ = os.Getwd()
	}
	profileStore := config.NewFileProfileStore(scenarioDir)

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

	srv := &Server{
		config:           cfg,
		db:               db,
		router:           mux.NewRouter(),
		driver:           driverManager,
		handlers:         h,
		logger:           logger,
		processTracker:   processTracker,
		gcService:        gcService,
		metricsCollector: metricsCollector,
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

// ensureSchema runs automatic migrations to ensure required tables exist.
// This is idempotent and safe to run on every startup.
func ensureSchema(db *sql.DB) error {
	// Ensure core schema exists (sandboxes table).
	var sandboxesExists bool
	err := db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'sandboxes'
		)
	`).Scan(&sandboxesExists)
	if err != nil {
		return fmt.Errorf("failed to check sandboxes table: %w", err)
	}

	if !sandboxesExists {
		log.Println("running migration: initializing workspace-sandbox schema")
		schemaSQL, err := loadSchemaSQL()
		if err != nil {
			return fmt.Errorf("failed to load schema.sql: %w", err)
		}
		if _, err := db.Exec(schemaSQL); err != nil {
			return fmt.Errorf("failed to apply schema.sql: %w", err)
		}
		log.Println("migration complete: schema.sql applied")
	}

	// Check if applied_changes table exists
	var exists bool
	err = db.QueryRow(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = 'applied_changes'
		)
	`).Scan(&exists)
	if err != nil {
		return fmt.Errorf("failed to check applied_changes table: %w", err)
	}

	if !exists {
		log.Println("running migration: creating applied_changes table")
		_, err = db.Exec(`
			CREATE TABLE applied_changes (
				id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
				sandbox_id UUID REFERENCES sandboxes(id) ON DELETE SET NULL,
				sandbox_owner TEXT,
				sandbox_owner_type TEXT,
				file_path TEXT NOT NULL,
				project_root TEXT NOT NULL,
				change_type TEXT NOT NULL,
				file_size BIGINT DEFAULT 0,
				applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
				committed_at TIMESTAMPTZ,
				commit_hash TEXT,
				commit_message TEXT,
				CONSTRAINT valid_applied_change_type CHECK (change_type IN ('added', 'modified', 'deleted'))
			);
			CREATE INDEX idx_applied_changes_sandbox_id ON applied_changes(sandbox_id);
			CREATE INDEX idx_applied_changes_file_path ON applied_changes(file_path);
			CREATE INDEX idx_applied_changes_project_root ON applied_changes(project_root);
			CREATE INDEX idx_applied_changes_pending ON applied_changes(committed_at) WHERE committed_at IS NULL;
		`)
		if err != nil {
			return fmt.Errorf("failed to create applied_changes table: %w", err)
		}
		log.Println("migration complete: applied_changes table created")
	}

	// --- reserved_path support (soft safety reserved directory) ---
	// Add column if missing (idempotent).
	if _, err := db.Exec(`ALTER TABLE sandboxes ADD COLUMN IF NOT EXISTS reserved_path TEXT`); err != nil {
		return fmt.Errorf("failed to add reserved_path column: %w", err)
	}

	// Add reserved_paths array for multi-reserve support (idempotent).
	if _, err := db.Exec(`ALTER TABLE sandboxes ADD COLUMN IF NOT EXISTS reserved_paths TEXT[] DEFAULT '{}'`); err != nil {
		return fmt.Errorf("failed to add reserved_paths column: %w", err)
	}

	// Backfill reserved_path for existing rows to preserve legacy behavior.
	if _, err := db.Exec(`UPDATE sandboxes SET reserved_path = scope_path WHERE reserved_path IS NULL`); err != nil {
		return fmt.Errorf("failed to backfill reserved_path: %w", err)
	}

	// Backfill reserved_paths when empty or NULL to align with reserved_path/scope_path.
	if _, err := db.Exec(`
		UPDATE sandboxes
		SET reserved_paths = ARRAY[COALESCE(reserved_path, scope_path)]
		WHERE reserved_paths IS NULL OR array_length(reserved_paths, 1) IS NULL OR array_length(reserved_paths, 1) = 0
	`); err != nil {
		return fmt.Errorf("failed to backfill reserved_paths: %w", err)
	}

	// Index for reserved_path overlap queries and UI filtering.
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_sandboxes_reserved_path ON sandboxes(reserved_path)`); err != nil {
		return fmt.Errorf("failed to create idx_sandboxes_reserved_path: %w", err)
	}

	// Update overlap check function to use reserved_paths when present.
	// Note: We keep the function name/signature for backwards compatibility.
	if _, err := db.Exec(`
		CREATE OR REPLACE FUNCTION check_scope_overlap(
			new_scope TEXT,
			new_project TEXT,
			exclude_id UUID DEFAULT NULL
		) RETURNS TABLE(id UUID, scope_path TEXT, status sandbox_status) AS $$
		BEGIN
			RETURN QUERY
			SELECT s.id, existing_prefix, s.status
			FROM sandboxes s,
			     LATERAL unnest(
			        CASE
			            WHEN s.reserved_paths IS NOT NULL AND array_length(s.reserved_paths, 1) > 0 THEN s.reserved_paths
			            ELSE ARRAY[COALESCE(s.reserved_path, s.scope_path)]
			        END
			     ) AS existing_prefix
			WHERE s.project_root = new_project
			  AND s.status IN ('creating', 'active')
			  AND (exclude_id IS NULL OR s.id != exclude_id)
			  AND (
			      existing_prefix LIKE new_scope || '/%'
			      OR existing_prefix = new_scope
			      OR new_scope LIKE existing_prefix || '/%'
			  );
		END;
		$$ LANGUAGE plpgsql;
	`); err != nil {
		return fmt.Errorf("failed to update check_scope_overlap function: %w", err)
	}

	return nil
}

func loadSchemaSQL() (string, error) {
	candidates := []string{}

	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		candidates = append(candidates, filepath.Join(root, "scenarios", "workspace-sandbox", "initialization", "postgres", "schema.sql"))
	}

	if cwd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(cwd, "initialization", "postgres", "schema.sql"))
		candidates = append(candidates, filepath.Join(cwd, "scenarios", "workspace-sandbox", "initialization", "postgres", "schema.sql"))
	}

	for _, path := range candidates {
		if path == "" {
			continue
		}
		if _, err := os.Stat(path); err == nil {
			bytes, readErr := os.ReadFile(path)
			if readErr != nil {
				return "", readErr
			}
			return string(bytes), nil
		}
	}

	return "", fmt.Errorf("schema.sql not found (checked %s)", strings.Join(candidates, ", "))
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

	server, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Start(); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
