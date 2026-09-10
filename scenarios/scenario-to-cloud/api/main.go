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

	"scenario-to-cloud/agentmanager"
	"scenario-to-cloud/authz"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/credentials"
	"scenario-to-cloud/deployment"
	"scenario-to-cloud/deploymentsvc"
	"scenario-to-cloud/dns"
	stchealth "scenario-to-cloud/health"
	"scenario-to-cloud/identity"
	"scenario-to-cloud/instance"
	"scenario-to-cloud/investigation"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/operations"
	"scenario-to-cloud/operationsvc"
	"scenario-to-cloud/persistence"
	"scenario-to-cloud/reach"
	bridgereach "scenario-to-cloud/reach/bridge"
	"scenario-to-cloud/reach/sshadapter"
	"scenario-to-cloud/secrets"
	"scenario-to-cloud/tasks"
	"scenario-to-cloud/tlsinfo"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"
	_ "modernc.org/sqlite"

	vpspreflight "scenario-to-cloud/vps/preflight"
)

// ServerConfig holds minimal runtime configuration.
type ServerConfig struct {
	Port string
}

// Server wires the HTTP router.
// Fields marked with "// Seam:" are integration points that can be substituted
// for testing. If nil, defaults to the production implementation.
type Server struct {
	config           *ServerConfig
	router           *mux.Router
	db               *database.RoutedDB
	repo             *persistence.Repository
	progressHub      *deployment.Hub
	agentSvc         *agentmanager.AgentService
	investigationSvc *investigation.Service
	taskSvc          *tasks.Service
	historyRecorder  deployment.HistoryRecorder
	orchestrator     *deployment.Orchestrator
	// operations is the durable operation owner (P07): the worker pool,
	// reconciler and wait/cancel contract over cloud_operations.
	operations *operations.Service
	// authz is the management boundary; every route passes through it.
	authz          *authz.Enforcer
	authzInstalled bool
	// testProvider is set only by newTestEnforcer so authorization tests can
	// change the principal between admission and effect.
	testProvider *FakePrincipalProvider

	// Seam: SSH command execution for the bounded reach adapter (defaults to
	// sshadapter.ExecRunner)
	sshRunner sshadapter.Runner
	// Seam: SCP file transfer for the bounded reach adapter (defaults to
	// sshadapter.ExecSCPRunner)
	scpRunner sshadapter.SCPRunner
	// Seam: Secrets fetching (defaults to secrets.NewClient())
	secretsFetcher secrets.Fetcher
	// Seam: Secrets generation (defaults to secrets.NewGenerator())
	secretsGenerator secrets.GeneratorFunc
	// Seam: DNS services (defaults to dns.NewService(dns.NetResolver{}, dns.WithTimeout(...)))
	dnsService dns.Service
	// Seam: TLS probe service (defaults to tlsinfo.NewService(...))
	tlsService tlsinfo.Service
	// Seam: TLS ALPN runner (defaults to tlsinfo.DefaultALPNRunner)
	tlsALPNRunner tlsinfo.ALPNRunner
	// Seam: Deployment repository (defaults to persistence.Repository)
	deploymentRepo DeploymentRepository
	// Reach: typed target access selected by the deployment binding.
	reach reach.Reach
	// Seam: instance provider (defaults to the local disposable QEMU lane).
	instanceProvider instance.Provider
	// publicationDeps are the governance seams (governor, receipt signer,
	// target pointer reader, release resolver); production values resolve
	// lazily, tests substitute fakes.
	publicationDeps *publicationDeps
}

// devRoutingMux adapts gorilla/mux's fluent registration API to api-core's
// intentionally tiny mounting interface. Mount is important here: Connect
// RPC handlers own a service-prefix subtree rather than one exact path.
type devRoutingMux struct {
	router *mux.Router
}

func (m devRoutingMux) Handle(pattern string, handler http.Handler) {
	m.router.Handle(pattern, handler)
}

func (m devRoutingMux) Mount(pattern string, handler http.Handler) {
	m.router.PathPrefix(pattern).Handler(handler)
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	cfg := &ServerConfig{
		Port: requireEnv("API_PORT"),
	}

	// Connect to database
	db, err := database.Open(context.Background(), database.Config{
		Driver: database.DriverPostgres,
		// Test Genie provisions an isolated SQLite lease for routed workflow
		// runs; production traffic remains on PostgreSQL.
		TestDriver: database.DriverSQLite,
	})
	if err != nil {
		return nil, err
	}

	// Initialize repository and schema
	repo := persistence.NewRepository(db)
	// Keep the api-core schema contract active even though this scenario still
	// owns a migration-rich repository schema. This also makes the routed
	// primary/test-pool boundary explicit to storage-health.
	if err := database.EnsureSchemas(context.Background(), db.Primary()); err != nil {
		db.Close()
		return nil, err
	}
	if err := repo.InitSchema(context.Background()); err != nil {
		db.Close()
		return nil, err
	}
	db.SetTestPoolInitializer(func(ctx context.Context, pool *sql.DB) error {
		if err := database.EnsureSchemas(ctx, pool); err != nil {
			return err
		}
		return repo.InitSchemaOnDialect(ctx, pool, database.DriverSQLite)
	})

	progressHub := deployment.NewHub()

	// Initialize agent-manager integration
	agentEnabled := os.Getenv("AGENT_MANAGER_ENABLED") != "false"
	agentSvc := agentmanager.NewAgentService(agentmanager.AgentServiceConfig{
		ProfileName: getEnvDefault("AGENT_MANAGER_PROFILE_NAME", "scenario-to-cloud-investigator"),
		ProfileKey:  getEnvDefault("AGENT_MANAGER_PROFILE_KEY", "scenario-to-cloud-investigator"),
		Timeout:     30 * time.Second,
		Enabled:     agentEnabled,
	})

	// Initialize agent profile if enabled (non-blocking, log warnings)
	if agentEnabled {
		initCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := agentSvc.Initialize(initCtx, agentmanager.DefaultProfileConfig()); err != nil {
			log.Printf("[agent-manager] Warning: failed to initialize profile: %v", err)
		}
		cancel()
	}

	// Initialize seams with production implementations
	sshRunner := sshadapter.ExecRunner{}
	scpRunner := sshadapter.ExecSCPRunner{}
	secretsFetcher := secrets.NewClient()
	secretsGenerator := secrets.NewGenerator()
	dnsService := dns.NewService(dns.NetResolver{}, dns.WithTimeout(10*time.Second))
	verifyMode := strings.ToLower(strings.TrimSpace(os.Getenv("TLS_VERIFY_MODE")))
	verifyFull := verifyMode == "full" || verifyMode == "true"
	tlsService := tlsinfo.NewService(
		tlsinfo.WithTimeout(10*time.Second),
		tlsinfo.WithVerify(verifyFull),
	)
	tlsALPNRunner := tlsinfo.DefaultALPNRunner

	authConfig, err := authz.FromEnvironment(os.Getenv)
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("management authorization: %w", err)
	}

	srv := &Server{
		config:           cfg,
		router:           mux.NewRouter(),
		db:               db,
		repo:             repo,
		deploymentRepo:   repo,
		progressHub:      progressHub,
		historyRecorder:  repo,
		agentSvc:         agentSvc,
		investigationSvc: investigation.NewService(repo, agentSvc, progressHub),
		taskSvc:          tasks.NewService(repo, agentSvc, progressHub),
		sshRunner:        sshRunner,
		scpRunner:        scpRunner,
		secretsFetcher:   secretsFetcher,
		secretsGenerator: secretsGenerator,
		dnsService:       dnsService,
		tlsService:       tlsService,
		tlsALPNRunner:    tlsALPNRunner,
		instanceProvider: instance.LocalQEMUProvider{},
	}
	authConfig.Logger = srv.log
	enforcer, err := authz.New(authConfig, targetResolver{repo: repo})
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("management authorization: %w", err)
	}
	srv.authz = enforcer

	// Initialize manifest refresher for rebuild operations. The closure
	// service (P05) is the only dependency source; a failure to build it is a
	// startup error rather than a silent fallback.
	closureSvc, err := newDefaultClosureService()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("closure service: %w", err)
	}
	manifestRefresher := deployment.NewManifestRefresher(deployment.ManifestRefresherConfig{
		SecretsFetcher: secretsFetcher,
		Closure:        closureSvc,
		PortsFetcher:   &deployment.DefaultPortsFetcher{},
		Logger:         srv.log,
	})

	// Reach: the deployment's target binding selects the transport. Bridge
	// reach needs an operator token for the Bridge API; without one the
	// bridge path is a typed reach_unavailable, never an SSH fallback. Every
	// target effect of a plan (verbs and artifact delivery) goes through it.
	targetReach := &reach.Router{
		SSH: &sshadapter.Adapter{Runner: sshRunner, SCP: scpRunner, Config: srv.sshConfigForTarget},
	}
	if token := strings.TrimSpace(os.Getenv("VROOLI_BRIDGE_API_TOKEN")); token != "" {
		bridgeClient := nodereach.New(nodereach.Config{Token: token})
		targetReach.Bridge = &bridgereach.Adapter{Client: bridgeClient, Artifacts: bridgereach.NewArtifactClient(bridgeClient)}
	}
	srv.reach = targetReach

	// Initialize the deployment orchestrator with all dependencies
	srv.orchestrator = deployment.NewOrchestrator(deployment.OrchestratorConfig{
		Repo:              repo,
		ProgressHub:       progressHub,
		Reach:             targetReach,
		ReleaseBuilder:    srv.releaseBuilder(),
		Credentials:       srv.credentialBinder(),
		Backups:           srv.recoveryPointRecorder(),
		SecretsFetcher:    secretsFetcher,
		SecretsGenerator:  secretsGenerator,
		DNSService:        dnsService,
		HistoryRecorder:   repo,
		ManifestRefresher: manifestRefresher,
		Logger:            srv.log,
	})

	// Durable operation owner: the worker pool executes admitted plans
	// through the orchestrator; receipts are read back from the target's
	// native owner before any unknown outcome is replayed. Startup
	// reconciliation reacquires every unowned or expired record.
	opsCfg := operations.DefaultConfig()
	opsCfg.Logger = srv.log
	hostMax, budgetsSource := stchealth.LoadHostConcurrencyLimit()
	repo.SetHostConcurrencyLimit(hostMax)
	srv.log("operation admission bounds configured", map[string]interface{}{"effectful_operations_per_host_max": hostMax, "budgets": budgetsSource})
	srv.operations = operations.NewService(opsCfg, repo, srv.orchestrator, &deployment.ReachTargetReceipts{Reach: targetReach, Repo: repo}, srv.orchestrator)
	srv.operations.Start()
	reconcileCtx, cancelReconcile := context.WithTimeout(context.Background(), 30*time.Second)
	taken, rerr := srv.operations.Reconcile(reconcileCtx)
	cancelReconcile()
	if rerr != nil {
		srv.log("startup operation reconciliation failed", map[string]interface{}{"error": rerr.Error()})
	} else {
		srv.log("startup operation reconciliation complete", map[string]interface{}{"worker_id": srv.operations.WorkerID(), "acquired": len(taken), "operation_ids": taken})
	}

	srv.setupRoutes()
	return srv, nil
}

// DOC: docs/reference/api-endpoints.md — complete endpoint reference
func (s *Server) setupRoutes() {
	if s.authz == nil {
		panic("scenario-to-cloud: routes cannot be registered without the authorization boundary")
	}
	if !s.authzInstalled {
		s.router.Use(loggingMiddleware, s.authz.Middleware)
		s.authzInstalled = true
	}
	gate := s.authz.EffectGate
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	var healthDB *sql.DB
	if s.db != nil {
		healthDB = s.db.Primary()
	}
	healthHandler := health.Handler(health.DB(healthDB))
	s.router.HandleFunc("/health", healthHandler).Methods("GET")

	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", healthHandler).Methods("GET")
	api.HandleFunc("/authz/matrix", authz.MatrixHandler()).Methods("GET")
	api.HandleFunc("/scenarios", s.handleListScenarios).Methods("GET")
	api.HandleFunc("/scenarios/{id}/ports", s.handleScenarioPorts).Methods("GET")
	api.HandleFunc("/scenarios/{id}/dependencies", s.handleScenarioDependencies).Methods("GET")
	api.HandleFunc("/validate/reachability", s.handleReachabilityCheck).Methods("POST")
	api.HandleFunc("/manifest/schema", s.handleManifestSchema).Methods("GET")
	api.HandleFunc("/manifest/template", s.handleManifestTemplate).Methods("GET")
	api.HandleFunc("/manifest/init", s.handleManifestInit).Methods("POST")
	api.HandleFunc("/manifest/doctor", s.handleManifestDoctor).Methods("POST")
	api.HandleFunc("/manifest/fix", s.handleManifestFix).Methods("POST")
	api.HandleFunc("/manifest/validate", s.handleManifestValidate).Methods("POST")
	api.HandleFunc("/bundle/build", s.handleBundleBuild).Methods("POST")
	api.HandleFunc("/bundles", bundle.HandleListBundles()).Methods("GET")
	api.HandleFunc("/bundles/stats", bundle.HandleBundleStats()).Methods("GET")
	api.HandleFunc("/bundles/cleanup", gate(authz.EffectHost, bundle.HandleBundleCleanup())).Methods("POST")
	api.HandleFunc("/bundles/{sha256}", gate(authz.EffectHost, bundle.HandleDeleteBundle())).Methods("DELETE")

	// Deployment-scoped VPS bundle cache (recommended, selector-first via deployment resolve).
	api.HandleFunc("/deployments/{id}/bundles/vps", s.handleListDeploymentVPSBundles).Methods("GET")
	api.HandleFunc("/deployments/{id}/bundles/vps/gc", gate(authz.EffectHost, s.handleGCDeploymentVPSBundles)).Methods("POST")
	api.HandleFunc("/preflight", vpspreflight.HandlePreflight(vpspreflight.HandlerDeps{
		Reach:             s.targetReach(),
		DNSService:        s.dnsService,
		ValidateManifest:  manifest.ValidateAndNormalize,
		HasBlockingIssues: manifest.HasBlockingIssues,
	})).Methods("POST")
	api.HandleFunc("/preflight/requirements", vpspreflight.HandleRequirements()).Methods("GET")
	api.HandleFunc("/secrets/{scenario}", secrets.HandleGetSecrets(s.secretsFetcher)).Methods("GET")
	api.HandleFunc("/vps/setup/plan", s.handleVPSSetupPlan).Methods("POST")
	api.HandleFunc("/vps/setup/apply", gate(authz.EffectHost, s.handleVPSSetupApply)).Methods("POST")
	api.HandleFunc("/vps/deploy/plan", s.handleVPSDeployPlan).Methods("POST")
	api.HandleFunc("/vps/deploy/apply", gate(authz.EffectWorkloadMutation, s.handleVPSDeployApply)).Methods("POST")
	api.HandleFunc("/vps/inspect/plan", s.handleVPSInspectPlan).Methods("POST")
	api.HandleFunc("/vps/inspect/apply", s.handleVPSInspectApply).Methods("POST")
	api.HandleFunc("/instances/plan", s.handleInstancePlan).Methods("POST")
	api.HandleFunc("/instances/readiness", s.handleInstanceReadiness).Methods("GET")
	api.HandleFunc("/instances", gate(authz.EffectHost, s.handleInstanceCreate)).Methods("POST")
	api.HandleFunc("/instances/{id}/{action}", gate(authz.EffectHost, s.handleInstanceAction)).Methods("POST")

	// Deployment management. The selector route is registered before the
	// {id} route so "resolve" is never read as a deployment id.
	api.HandleFunc("/deployments/resolve", s.handleResolveDeployment).Methods("GET")
	api.HandleFunc("/deployments", s.handleListDeployments).Methods("GET")
	api.HandleFunc("/deployments", s.handleCreateDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}", s.handleGetDeployment).Methods("GET")
	api.HandleFunc("/deployments/{id}/receipt", s.handleGetDeploymentReceipt).Methods("GET")
	api.HandleFunc("/deployments/{id}/recovery", s.handleRecoverDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/recovery/{operation_id}", s.handleGetCloudRecoveryOperation).Methods("GET")
	api.HandleFunc("/deployments/{id}", s.handleDeleteDeployment).Methods("DELETE")
	api.HandleFunc("/deployments/{id}/execute", s.handleExecuteDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/progress", s.handleDeploymentProgress).Methods("GET")
	api.HandleFunc("/deployments/{id}/inspect", s.handleInspectDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/stop", s.handleStopDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/start", s.handleStartDeployment).Methods("POST")

	// Live state inspection (Ground Truth Redesign - Phase 1)
	api.HandleFunc("/deployments/{id}/live-state", s.handleGetLiveState).Methods("GET")
	api.HandleFunc("/deployments/{id}/metrics-debug", s.handleGetMetricsDebug).Methods("GET")
	api.HandleFunc("/deployments/{id}/files", s.handleGetFiles).Methods("GET")
	api.HandleFunc("/deployments/{id}/files/content", s.handleGetFileContent).Methods("GET")
	api.HandleFunc("/deployments/{id}/drift", s.handleGetDrift).Methods("GET")
	api.HandleFunc("/deployments/{id}/health", s.handleGetDeploymentHealth).Methods("GET")
	api.HandleFunc("/deployments/{id}/actions/kill", gate(authz.EffectHost, s.handleKillProcess)).Methods("POST")
	api.HandleFunc("/deployments/{id}/actions/restart", gate(authz.EffectHost, s.handleRestartProcess)).Methods("POST")
	api.HandleFunc("/deployments/{id}/actions/process", gate(authz.EffectHost, s.handleProcessControl)).Methods("POST")
	api.HandleFunc("/deployments/{id}/actions/vps", gate(authz.EffectHost, s.handleVPSAction)).Methods("POST")

	// History & Logs (Ground Truth Redesign - Phase 7)
	api.HandleFunc("/deployments/{id}/history", s.handleGetHistory).Methods("GET")
	api.HandleFunc("/deployments/{id}/history", s.handleAddHistoryEvent).Methods("POST")
	api.HandleFunc("/deployments/{id}/logs", s.handleGetLogs).Methods("GET")

	// VPS Secrets Management (Post-deployment secret CRUD)
	secretsMgmtDeps := secrets.ManagementDeps{Repo: s.repo, Reach: s.targetReach()}
	api.HandleFunc("/deployments/{id}/secrets", gate(authz.EffectSecret, secrets.HandleListVPSSecrets(secretsMgmtDeps))).Methods("GET")
	api.HandleFunc("/deployments/{id}/secrets", gate(authz.EffectSecret, secrets.HandleCreateVPSSecret(secretsMgmtDeps))).Methods("POST")
	api.HandleFunc("/deployments/{id}/secrets/{key}", gate(authz.EffectSecret, secrets.HandleGetVPSSecret(secretsMgmtDeps))).Methods("GET")
	api.HandleFunc("/deployments/{id}/secrets/{key}", gate(authz.EffectSecret, secrets.HandleUpdateVPSSecret(secretsMgmtDeps))).Methods("PUT")
	api.HandleFunc("/deployments/{id}/secrets/{key}", gate(authz.EffectSecret, secrets.HandleDeleteVPSSecret(secretsMgmtDeps))).Methods("DELETE")

	// Expected Secrets (secrets defined in scenario's service.json)
	api.HandleFunc("/deployments/{id}/expected-secrets", s.handleGetExpectedSecrets).Methods("GET")

	// Terminal (Ground Truth Redesign - Phase 8)
	api.HandleFunc("/deployments/{id}/terminal", s.handleTerminalWebSocket).Methods("GET")

	// Edge/TLS Management (Ground Truth Redesign - Enhancement)
	api.HandleFunc("/deployments/{id}/edge/dns-check", s.handleDNSCheck).Methods("GET")
	api.HandleFunc("/deployments/{id}/edge/dns-records", s.handleDNSRecords).Methods("GET")
	api.HandleFunc("/deployments/{id}/edge/caddy", gate(authz.EffectHost, s.handleCaddyControl)).Methods("POST")
	api.HandleFunc("/deployments/{id}/edge/tls", s.handleTLSInfo).Methods("GET")
	api.HandleFunc("/deployments/{id}/edge/tls/renew", gate(authz.EffectHost, s.handleTLSRenew)).Methods("POST")

	// Documentation
	api.HandleFunc("/docs/manifest", s.handleGetDocsManifest).Methods("GET")
	api.HandleFunc("/docs/content", s.handleGetDocContent).Methods("GET")

	// Preflight fix actions: every one is an owner operation through the
	// bounded reach adapter (privilege broker actions, lifecycle stops,
	// observation programs). Host key custody is the credential binding's
	// and enrollment is vrooli-bridge onboard; there is no /ssh surface.
	api.HandleFunc("/preflight/fix/firewall", gate(authz.EffectHost, vpspreflight.HandleOpenFirewallPorts(s.adHocReach))).Methods("POST")
	api.HandleFunc("/preflight/fix/stop-processes", gate(authz.EffectHost, vpspreflight.HandleStopScenarioProcesses(s.adHocReach))).Methods("POST")
	api.HandleFunc("/preflight/disk/usage", vpspreflight.HandleDiskUsage(s.adHocReach)).Methods("POST")
	api.HandleFunc("/preflight/disk/cleanup", gate(authz.EffectHost, vpspreflight.HandleDiskCleanup(s.adHocReach))).Methods("POST")

	// Investigation endpoints (agent-manager integration) - legacy, kept for backward compatibility
	api.HandleFunc("/deployments/{id}/investigate", s.handleInvestigateDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/investigations", s.handleListInvestigations).Methods("GET")
	api.HandleFunc("/deployments/{id}/investigations/{invId}", s.handleGetInvestigation).Methods("GET")
	api.HandleFunc("/deployments/{id}/investigations/{invId}/stop", s.handleStopInvestigation).Methods("POST")
	api.HandleFunc("/deployments/{id}/investigations/{invId}/apply-fixes", gate(authz.EffectHost, s.handleApplyFixes)).Methods("POST")
	api.HandleFunc("/agent-manager/status", s.handleCheckAgentManagerStatus).Methods("GET")

	// New unified task endpoints
	s.registerTaskRoutes(api)

	// Dependency closure (P05) and typed health (P16).
	s.registerClosureRoutes(api)
	s.registerHealthRoutes(api)
	s.registerReleaseRoutes(api)
	s.registerCredentialRoutes(api)
	s.registerEdgeRoutes(api)
	s.registerBackupRoutes(api)
	// Evidence, exact review binding and governed publication (P17).
	s.registerPublicationRoutes(api)

	// Executable plans (P06): REST preview/apply plus the Connect PlansService.
	s.registerPlanRoutes(api)
	if s.repo != nil {
		path, handler := s.PlansService().Handler()
		s.router.PathPrefix(path).Handler(handler)
	}

	// Durable operations (P07): REST wait/status/cancel plus the Connect
	// OperationsService.
	s.registerOperationRoutes(api)
	s.registerReconcileRoutes(api)
	if s.repo != nil {
		path, handler := operationsvc.New(s.operations, s.repo).Handler()
		s.router.PathPrefix(path).Handler(handler)
	}

	// Typed DeploymentsService (Connect) beside the REST routes. Later phases
	// migrate the remaining slices onto generated contracts.
	if s.repo != nil {
		path, handler := deploymentsvc.New(s.repo).Handler()
		s.router.PathPrefix(path).Handler(handler)
	}

	// Test-genie uses a leased shadow pool in development. The registration is
	// deliberately dev-only inside api-core and is never exposed in production.
	if s.db != nil {
		devrouting.Register(devRoutingMux{router: s.router}, s.db)
	}
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return apihttp.TestModeMiddleware(handlers.RecoveryHandler()(securityHeadersMiddleware(s.router)))
}

// securityHeadersMiddleware centralizes the baseline browser boundary for
// every API response. HSTS is included because the API is also deployed
// behind the HTTPS edge; local development remains functional because this
// header is only honored by browsers after a secure response.
func securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		next.ServeHTTP(w, r)
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
		log.Fatalf("environment variable %s is required. Run the scenario via 'vrooli scenario run <name>' so lifecycle exports it.", key)
	}
	return value
}

func getEnvDefault(key, defaultValue string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return defaultValue
	}
	return value
}

func main() {
	// Preflight checks - must be first, before any initialization
	if preflight.Run(preflight.Config{
		ScenarioName: "scenario-to-cloud",
	}) {
		return // Process was re-exec'd after rebuild
	}

	srv, err := NewServer()
	if err != nil {
		log.Fatalf("failed to initialize server: %v", err)
	}

	// P0: some VPS operations (setup/deploy) can take several minutes; keep the server-side
	// response path alive rather than forcing async during early iterations.
	authConfig := srv.authz.Config().Authn
	if err := server.Run(server.Config{
		Handler:        srv.Router(),
		Authentication: &authConfig,
		WriteTimeout:   35 * time.Minute,
		Cleanup: func(ctx context.Context) error {
			if srv.db != nil {
				return srv.db.Close()
			}
			return nil
		},
	}); err != nil {
		log.Fatalf("server stopped with error: %v", err)
	}
}

// targetResolver adapts the deployment repository to the authorization
// boundary. It hands back identity only (id, environment, target key); the
// locator never reaches a refused caller.
type targetResolver struct {
	repo DeploymentRepository
}

func (t targetResolver) ResolveTarget(ctx context.Context, id string) (authz.Target, bool, error) {
	if t.repo == nil {
		return authz.Target{}, false, fmt.Errorf("deployment repository not configured")
	}
	dep, err := t.repo.GetDeployment(ctx, id)
	if err != nil {
		return authz.Target{}, false, err
	}
	if dep == nil {
		return authz.Target{}, false, nil
	}
	return authz.Target{DeploymentID: dep.ID, Environment: dep.Environment, Key: dep.Target.Key()}, true, nil
}

// sshConfigForTarget resolves the SSH connection for an ssh-bound target:
// the locator from the binding, and the key file from the credential binding
// vrooli/scenario-to-cloud:ssh-key of the deployment bound to that host. No
// key bytes are read here; a target without a binding is reached with the
// operator's ambient identity (agent, default identity files).
func (s *Server) sshConfigForTarget(ctx context.Context, target identity.TargetRef) (sshadapter.ConnectionConfig, error) {
	if strings.TrimSpace(target.Locator.Host) == "" {
		return sshadapter.ConnectionConfig{}, fmt.Errorf("target binding has no host locator")
	}
	keyPath := ""
	if s.repo != nil {
		refs, err := s.repo.ResolveDeployments(ctx, identity.Selector{Host: target.Locator.Host})
		if err != nil {
			return sshadapter.ConnectionConfig{}, err
		}
		for _, ref := range refs {
			bound, err := credentials.ResolveSSHKeyPath(ctx, s.repo, ref.ID)
			if err != nil {
				return sshadapter.ConnectionConfig{}, err
			}
			if bound != "" {
				keyPath = bound
				break
			}
		}
	}
	return sshadapter.NewConfig(target.Locator.Host, target.Locator.Port, target.Locator.User, keyPath), nil
}
