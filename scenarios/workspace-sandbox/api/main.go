package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "modernc.org/sqlite"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	apicoreserver "github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"

	"workspace-sandbox/internal/audit"
	"workspace-sandbox/internal/blobstore"
	"workspace-sandbox/internal/clock"
	"workspace-sandbox/internal/config"
	"workspace-sandbox/internal/driver"
	"workspace-sandbox/internal/fsmount"
	"workspace-sandbox/internal/gc"
	"workspace-sandbox/internal/handlers"
	"workspace-sandbox/internal/logging"
	"workspace-sandbox/internal/metrics"
	"workspace-sandbox/internal/namespace"
	"workspace-sandbox/internal/policy"
	"workspace-sandbox/internal/process"
	"workspace-sandbox/internal/repository"
	"workspace-sandbox/internal/runtime"
	"workspace-sandbox/internal/sandbox"
	"workspace-sandbox/internal/server"
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
	clock            clock.Clock      // Wall-clock seam (Round 4 Phase 2). Threads through middleware.
	processTracker   *process.Tracker // OT-P0-008: Process/Session Tracking
	gcService        *gc.Service      // OT-P1-003: GC/Prune Operations
	lifecycleRecon   *sandbox.Runner
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

	// Wall-clock seam (Round 4 Phase 2). Production wires
	// clock.System{} once and threads it through every constructor that
	// needs time. Tests construct equivalents with a FakeClock.
	clk := clock.System{}

	// Process exec seam (Round 4 Phase 7). OSExecStarter wraps os/exec
	// for every external-command invocation in driver, namespace,
	// fsmount, diff, policy hooks, and interactive handlers.
	starter := process.NewOSExecStarter()

	// Mount seam (Round 4 Phase 7). SystemMounter wraps syscall.Mount /
	// syscall.Unmount and the fuse-overlayfs subprocess; it depends on
	// starter for binary lookups and userspace fallbacks.
	mounter := fsmount.NewSystemMounter(starter)

	// driverDeps bundles the three seam dependencies every driver
	// constructor needs. Pass-through for SelectDriver, NewDriverFor,
	// and SwitchDriver so downstream code never sees raw nils.
	driverDeps := driver.Deps{Clock: clk, Mounter: mounter, Starter: starter}

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

	// Apply the embedded schema and run forward-only legacy migrations.
	// EnsureSchema is the single startup entry point for DDL: it applies
	// the idempotent CREATE TABLE statements, runs the driver_id rename
	// and home_overlay_state column-add migrations, and stamps the
	// schema_version row. Refuses to start if the persisted version
	// drifts from repository.ExpectedSchemaVersion (Round 4 Phase 9).
	if err := repository.EnsureSchema(context.Background(), db, clk); err != nil {
		return nil, fmt.Errorf("failed to ensure schema: %w", err)
	}

	// Initialize driver with automatic selection and fallback
	// Respects saved preference if available, otherwise:
	// Priority: native overlayfs (in user namespace) > fuse-overlayfs > copy driver
	driverCfg := driver.Config{
		BaseDir:            cfg.Driver.BaseDir,
		HomeOverlayBaseDir: cfg.Driver.HomeOverlayBaseDir,
		MaxSandboxes:       cfg.Limits.MaxSandboxes,
		MaxSizeMB:          cfg.Limits.MaxSandboxSizeMB,
	}
	initialDriver, selectionReport, err := driver.SelectDriverWithPreference(context.Background(), driverCfg, driverDeps)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize driver: %w", err)
	}
	driver.LogSelectionReport(selectionReport)
	// Boot-time self-check: kernel overlayfs requires the launcher to place
	// the API inside a user namespace before main runs. Without that launch
	// shape, Mount() would fail at runtime; failing fatally at boot makes the
	// deployment-shape contract explicit.
	if initialDriver.ID() == driver.DriverOverlayfsUserNS && !driver.InUserNamespace() {
		log.Fatalf("driver overlayfs-userns selected but API is not running inside a user namespace; start through workspace-sandbox-launcher and ensure the workspace_sandbox_userns safeguard is satisfied")
	}
	// Hold the driver in an atomic.Pointer-backed slot so /api/v1/driver/select
	// can hot-swap without locking every Driver() call.
	driverSlot := driver.NewSlot(initialDriver)
	log.Printf("driver selected | id=%s version=%s", initialDriver.ID(), initialDriver.Version())

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
		validationPolicy = policy.NewHookValidationPolicy(starter, hooks,
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
		teardownPolicy = policy.NewHookTeardownPolicy(starter, hooks,
			policy.WithTeardownGlobalTimeout(cfg.Policy.TeardownTimeout),
		)
		log.Printf("teardown hooks enabled | hooks=%d timeout=%v", len(hooks), cfg.Policy.TeardownTimeout)
	} else {
		teardownPolicy = policy.NewNoOpTeardownPolicy()
	}

	// Initialize repository and service
	repo := repository.NewSandboxRepository(db, clk)
	archiveRepo := repository.NewArchiveRepository(db, clk)

	// Diff-archive blob store. Resolves under api-core/storage's
	// ClassData root, sharing the same data tree as the SQLite database
	// so archive metadata + content travel together for backup/restore.
	blobsResolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create blobstore resolver: %w", err)
	}
	blobs, err := blobstore.New(blobsResolver)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize blobstore: %w", err)
	}

	// Audit emitter — single seam for sandbox audit-log writes (Round
	// 4 Phase 6). Wraps repo.LogAuditEvent and stamps EventTime via the
	// shared clock so reconcilers, GC, and the Service all share a
	// deterministic timestamp source.
	auditEmitter := audit.NewRepoEmitter(repo.LogAuditEvent, clk)
	svcCfg := sandbox.ServiceConfig{
		DefaultProjectRoot:      cfg.Driver.ProjectRoot,
		MaxSandboxes:            cfg.Limits.MaxSandboxes,
		DefaultTTL:              cfg.Lifecycle.DefaultTTL,
		DefaultNoLock:           cfg.Policy.DefaultNoLock,
		AgentManagerURL:         cfg.Integration.AgentManagerURL,
		AgentManagerSyncEnabled: cfg.Integration.AgentManagerSyncEnabled,
		AgentManagerSyncTimeout: cfg.Integration.AgentManagerSyncTimeout,
	}
	// Metrics collector is constructed here so the Service can record
	// daemon-reaped events via WithMetrics; handlers reuse the same
	// instance below, exposing it on /metrics.
	metricsCollector := metrics.NewCollector()

	svc := sandbox.NewService(repo, driverSlot, svcCfg, clk, auditEmitter, starter,
		sandbox.WithAttributionPolicy(attributionPolicy),
		sandbox.WithValidationPolicy(validationPolicy),
		sandbox.WithTeardownPolicy(teardownPolicy),
		sandbox.WithMetrics(metricsCollector),
		sandbox.WithArchive(archiveRepo, blobs),
	)
	healCfg := sandbox.HealConfig{
		IdleGracePeriod:        cfg.Lifecycle.AutoHealIdleGrace,
		MaxConsecutiveFailures: cfg.Lifecycle.AutoHealMaxRetries,
		BaseBackoff:            cfg.Lifecycle.AutoHealBaseBackoff,
	}

	// Diff-archive retention store. Seeds from the env-derived defaults
	// so first-boot retention matches Default()/LoadFromEnv; subsequent
	// runtime PUTs to /config/retention persist to a JSON file under
	// ClassConfig that takes over as the source of truth on next boot.
	retentionStore, err := config.NewFileRetentionStore(cfg.Retention)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize retention store: %w", err)
	}
	retentionProvider := func() sandbox.RetentionPolicy {
		rc := retentionStore.Get()
		return sandbox.RetentionPolicy{
			MaxArchiveAgeDays:     rc.MaxArchiveAgeDays,
			MaxArchiveSizeBytes:   rc.MaxArchiveSizeBytes,
			MaxArchivesPerProject: rc.MaxArchivesPerProject,
		}
	}

	lifecycleRecon := sandbox.DefaultRunner(svc, cfg.Lifecycle.GCInterval, cfg.Lifecycle.ManualReviewTTL, healCfg, retentionProvider)

	// Initialize process tracker (OT-P0-008)
	processTracker := process.NewTrackerWithConfig(process.TrackerConfig{
		GracePeriod: cfg.Lifecycle.ProcessGracePeriod,
		KillWait:    cfg.Lifecycle.ProcessKillWait,
	}, clk)

	// Initialize process logger (Phase 2)
	processLogger := process.NewLogger(process.DefaultLogConfig(cfg.Driver.BaseDir), clk)

	// Initialize profile store for isolation profiles.
	scenarioDir, err := resolveWorkspaceSandboxScenarioDir()
	if err != nil {
		return nil, err
	}
	profileStore, err := config.NewFileProfileStore(scenarioDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize profile store: %w", err)
	}
	// Snapshot the profile registry once at startup so the request-path
	// resolver is detached from the underlying file (Round 4 Phase 9).
	// Admin Save/Delete handlers refresh the snapshot via
	// Handlers.RefreshProfileSnapshot — file-system mutations to
	// profiles.json are intentionally ignored after boot.
	profileSnapshot, err := runtime.LoadProfiles(profileStore)
	if err != nil {
		return nil, fmt.Errorf("failed to load profile snapshot: %w", err)
	}

	// Initialize GC service (OT-P1-003)
	gcCfg := gc.Config{
		DefaultMaxAge:        cfg.Lifecycle.DefaultTTL,
		DefaultIdleTimeout:   cfg.Lifecycle.IdleTimeout,
		DefaultTerminalDelay: cfg.Lifecycle.TerminalCleanupDelay,
		DefaultLimit:         100,
		MaxTotalSizeBytes:    cfg.Limits.MaxTotalSizeMB * 1024 * 1024,
	}
	gcService := gc.NewService(repo, driverSlot, gcCfg, clk, auditEmitter)

	// Check if we're in a user namespace via /proc/self/uid_map.
	// driver.InUserNamespace is the canonical probe; it agrees with the
	// boot-time self-check above so the handlers and the driver layer
	// see the same answer.
	inUserNS := driver.InUserNamespace()

	// Create handlers with injected dependencies
	h := &handlers.Handlers{
		Service:         svc,
		DriverSlot:      driverSlot,
		DB:              db,
		Config:          cfg,
		StatsGetter:     repo, // Repository implements StatsGetter
		ProcessTracker:  processTracker,
		ProcessLogger:   processLogger,
		GCService:       gcService,
		ProfileStore:    profileStore,
		InUserNamespace: inUserNS,
		Reconcilers:     lifecycleRecon,
		RetentionStore:  retentionStore,
		Clock:           clk,
		Mounter:         mounter,
		Starter:         starter,
	}
	h.SetProfileSnapshot(profileSnapshot)

	// Initialize structured logger
	logger := logging.New("workspace-sandbox-api", logging.WithClock(clk))

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
		Driver:         driverSlot,
		ProcessTracker: processTracker,
		ProcessLogger:  processLogger,
		ProfileStore:   profileStore,
		ExecConfig:     cfg.Execution,
		Starter:        starter,
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
		driver:           driverSlot,
		handlers:         h,
		logger:           logger,
		clock:            clk,
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
		"driver":       driverSlot.ID(),
		"maxSandboxes": cfg.Limits.MaxSandboxes,
	})

	return srv, nil
}

func (s *Server) setupRoutes() {
	// Apply cross-cutting middleware. Round 4 Phase 3 extracted these
	// into internal/server so the live-HTTP test harness exercises the
	// exact same wrappers — closing the gap that let the 2026-04-28 SSE
	// flusher bug ship.
	server.Middleware{
		Logger:             s.logger,
		Clock:              s.clock,
		CORSAllowedOrigins: s.config.Server.CORSAllowedOrigins,
	}.Apply(s.router)

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

	// User namespace + driver selection are decoupled from main's process
	// model. The portable launcher is responsible for placing the API inside
	// a user namespace when the saved/default driver requires it. NewServer's
	// boot self-check fails fatally if that launch shape is misconfigured, so
	// main never tries to re-exec itself.
	bootStarter := process.NewOSExecStarter()
	if driver.InUserNamespace() {
		log.Printf("running in user namespace | kernel=%s", namespace.Check(bootStarter).KernelVersion)
	} else {
		log.Printf("running in host namespace | kernel=%s", namespace.Check(bootStarter).KernelVersion)
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
	if err := apicoreserver.Run(apicoreserver.Config{
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
