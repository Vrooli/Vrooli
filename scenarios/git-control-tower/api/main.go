package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"git-control-tower/internal/config"
	"git-control-tower/ssh"

	gorillahandlers "github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	"github.com/vrooli/api-core/storage"
	repocontract "github.com/vrooli/repo-contract-go"
	_ "modernc.org/sqlite" // Pure-Go SQLite driver (CGO-free, enables static builds)
)

// Config holds minimal runtime configuration
type Config struct {
	Port string
}

// Server wires the HTTP router and database connection
type Server struct {
	config               *Config
	policy               config.Config
	db                   *sql.DB
	router               *mux.Router
	git                  GitRunner
	repoLock             *RepoLock
	audit                AuditLogger
	sandbox              WorkspaceSandboxAPI
	capabilities         *CapabilityRegistry
	sshDeps              ssh.SSHDeps
	repos                *RepoService
	precommit            *PrecommitService
	commitChecks         *CommitCheckStore
	credStore            *CredentialsStore
	storageResolver      *storage.Resolver
	basClient            *BrowserAutomationClient
	visualCaptureStorage *VisualCaptureStorage
	periodicCapture      *PeriodicCapture
	testGenieClient      *TestGenieClient
	testGenieEligibility *TestGenieEligibilityClient
	isolationCache       *IsolationCache
	tidinessClient       *TidinessManagerClient
	agentManagerClient   *AgentManagerClient
	auditorClient        *AuditorClient
	scenarioLocator      *ScenarioLocator
	envelopeCache        *EnvelopeCache
	reviewJobStore       *ReviewJobStore
	configCache          *GitConfigCache
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	cfg := &Config{
		Port: requireEnv("API_PORT"),
	}

	db, auditLogger, err := initDatabase()
	if err != nil {
		return nil, err
	}

	// Agent-access policy config (from <scenarioDir>/.vrooli/config.json
	// `policy` block). Missing file / missing key falls back to
	// DefaultConfig (confirm + broad + standard override flag).
	policyCfg, err := loadPolicyConfig()
	if err != nil {
		return nil, fmt.Errorf("load policy config: %w", err)
	}

	srv := &Server{
		config:       cfg,
		policy:       policyCfg,
		db:           db,
		router:       mux.NewRouter(),
		git:          &ExecGitRunner{GitPath: "git"},
		repoLock:     NewRepoLock(),
		audit:        auditLogger,
		sandbox:      NewWorkspaceSandboxClient(5 * time.Second),
		capabilities: NewCapabilityRegistry(knownCapabilities, newStatusCheckers(), 30*time.Second),
		sshDeps:      ssh.SSHDeps{Platform: ssh.DefaultPlatform()},
	}
	srv.repos = NewRepoService(NewSQLiteRepoStore(db), srv.git)
	srv.precommit = NewPrecommitService(db)
	srv.commitChecks = NewCommitCheckStore(db)
	srv.configCache = NewGitConfigCache(60 * time.Second)

	if err := srv.initClients(); err != nil {
		return nil, err
	}

	srv.initServices()
	srv.setupRoutes()
	return srv, nil
}

func initDatabase() (*sql.DB, AuditLogger, error) {
	dsn, err := sqliteDSN()
	if err != nil {
		return nil, nil, fmt.Errorf("sqlite configuration failed: %w", err)
	}

	db, err := database.Connect(context.Background(), database.Config{
		Driver:       "sqlite",
		DSN:          dsn,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, nil, fmt.Errorf("database connection failed: %w", err)
	}

	if err := ensureAuditSchema(db); err != nil {
		return nil, nil, fmt.Errorf("audit schema initialization failed: %w", err)
	}
	if err := ensureRepoSchema(db); err != nil {
		return nil, nil, fmt.Errorf("repo schema initialization failed: %w", err)
	}

	var auditLogger AuditLogger
	if db != nil {
		auditLogger = NewSQLiteAuditLogger(db)
	}
	if auditLogger == nil {
		auditLogger = &NoOpAuditLogger{}
	}

	return db, auditLogger, nil
}

func newStatusCheckers() map[string]StatusChecker {
	slugs := []string{
		"workspace-sandbox",
		"browser-automation-studio",
		"test-genie",
		"tidiness-manager",
		"agent-manager",
		"scenario-auditor",
	}
	checkers := make(map[string]StatusChecker, len(slugs))
	for _, slug := range slugs {
		checkers[slug] = &ScenarioChecker{
			Slug:   slug,
			Client: &http.Client{Timeout: 3 * time.Second},
		}
	}
	return checkers
}

func (s *Server) initClients() error {
	credStore, err := NewCredentialsStore("")
	if err != nil {
		log.Printf("WARNING: credentials store initialization failed: %v (SSH/HTTPS auth for git operations will be unavailable)", err)
	} else {
		s.credStore = credStore
	}

	resolver, err := storage.NewResolver(storage.ResolverConfig{})
	if err != nil {
		return fmt.Errorf("storage resolver init failed: %w", err)
	}
	s.storageResolver = resolver

	s.basClient = NewBrowserAutomationClient(30 * time.Second)
	s.testGenieClient = NewTestGenieClient(600 * time.Second)
	s.testGenieEligibility = NewTestGenieEligibilityClient(15 * time.Second)
	s.isolationCache = NewIsolationCache(30 * time.Second)
	s.tidinessClient = NewTidinessManagerClient(30 * time.Second)
	s.agentManagerClient = NewAgentManagerClient(120 * time.Second)
	s.auditorClient = NewAuditorClient(120 * time.Second)
	return nil
}

func (s *Server) initServices() {
	// Best-effort: ensure the default agent profile exists once agent-manager is reachable.
	go func() {
		for i := 0; i < 10; i++ {
			time.Sleep(time.Duration(i*5+5) * time.Second)
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			if s.capabilities.IsAvailable(ctx, "agent-manager") {
				if _, err := s.agentManagerClient.EnsureDefaultProfile(ctx); err != nil {
					log.Printf("warn: ensure default agent profile: %v", err)
				} else {
					cancel()
					return
				}
			}
			cancel()
		}
	}()

	s.reviewJobStore = NewReviewJobStore()
	s.reviewJobStore.StartCleanup(10 * time.Minute)
	s.scenarioLocator = NewScenarioLocator(30 * time.Second)
	s.envelopeCache = NewEnvelopeCache(60 * time.Second)
	s.visualCaptureStorage = NewVisualCaptureStorage(s.storageResolver, OSFileIO{})
	s.periodicCapture = NewPeriodicCapture(PeriodicCaptureConfig{
		Interval: 1 * time.Hour, MaxSnapshots: 10,
	}, s.capabilities, s.basClient, s.visualCaptureStorage, s.repos, s.git)
	s.periodicCapture.Start()
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return gorillahandlers.RecoveryHandler()(s.router)
}

// NOTE: The old handleHealth with custom HealthChecks has been replaced by
// api-core/health for standardized responses. The DB check is now handled
// by the health package with Critical priority.

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// loadPolicyConfig resolves the GCT scenario directory and loads its
// `.vrooli/config.json` `policy` block. Greenfield: missing file or
// missing key resolves to DefaultConfig; only malformed JSON or invalid
// values bubble up.
func loadPolicyConfig() (config.Config, error) {
	repoRoot, err := repocontract.FindRepoRootFromEnvOrCWD()
	if err != nil {
		// Without a repo root we can't find the scenario dir; fall
		// back to defaults rather than failing boot.
		return config.DefaultConfig(), nil
	}
	scenarioDir, err := repocontract.ResolveScenarioPath(repoRoot, "git-control-tower")
	if err != nil {
		return config.DefaultConfig(), nil
	}
	return config.Load(scenarioDir)
}

func requireEnv(key string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "git-control-tower",
	}) {
		return // Process was re-exec'd after rebuild
	}

	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	if err := server.Run(server.Config{
		Handler:      srv.Router(),
		WriteTimeout: 5 * time.Minute, // workflow captures poll BAS and can take several minutes
		Cleanup: func(ctx context.Context) error {
			srv.periodicCapture.Stop()
			return srv.db.Close()
		},
	}); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}
