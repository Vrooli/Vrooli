package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"scenario-to-cloud/agentmanager"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/deployment"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/investigation"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/persistence"
	"scenario-to-cloud/secrets"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/tasks"
	"scenario-to-cloud/tlsinfo"
	"scenario-to-cloud/toolexecution"
	"scenario-to-cloud/toolhandlers"
	"scenario-to-cloud/toolregistry"
	"scenario-to-cloud/vps"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/discovery"
	"github.com/vrooli/api-core/health"
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	vpspreflight "scenario-to-cloud/vps/preflight"
)

// Config holds minimal runtime configuration
type Config struct {
	Port string
}

// Server wires the HTTP router.
// Fields marked with "// Seam:" are integration points that can be substituted
// for testing. If nil, defaults to the production implementation.
type Server struct {
	config           *Config
	router           *mux.Router
	db               *sql.DB
	repo             *persistence.Repository
	progressHub      *deployment.Hub
	agentSvc         *agentmanager.AgentService
	investigationSvc *investigation.Service
	taskSvc          *tasks.Service
	historyRecorder  deployment.HistoryRecorder
	orchestrator     *deployment.Orchestrator
	// Tool Discovery Protocol
	toolRegistry *toolregistry.Registry
	// Tool Execution Protocol
	toolExecutor *toolexecution.ServerExecutor

	// Seam: SSH command execution (defaults to ssh.ExecRunner)
	sshRunner ssh.Runner
	// Seam: SCP file transfer (defaults to ssh.ExecSCPRunner)
	scpRunner ssh.SCPRunner
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
}

// NewServer initializes configuration, database, and routes
func NewServer() (*Server, error) {
	cfg := &Config{
		Port: requireEnv("API_PORT"),
	}

	// Connect to database
	db, err := database.Connect(context.Background(), database.Config{
		Driver: database.DriverPostgres,
	})
	if err != nil {
		return nil, err
	}

	// Initialize repository and schema
	repo := persistence.NewRepository(db)
	if err := repo.InitSchema(context.Background()); err != nil {
		db.Close()
		return nil, err
	}

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
	sshRunner := ssh.ExecRunner{}
	scpRunner := ssh.ExecSCPRunner{}
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
	}

	// Initialize manifest refresher for rebuild operations
	manifestRefresher := deployment.NewManifestRefresher(deployment.ManifestRefresherConfig{
		SecretsFetcher: secretsFetcher,
		DepsFetcher: &deployment.DefaultDependenciesFetcher{
			AnalyzerFetcher:    createAnalyzerFetcher(),
			ServiceJSONFetcher: deployment.ServiceJSONDependenciesFetcher,
		},
		PortsFetcher: &deployment.DefaultPortsFetcher{},
		Logger:       srv.log,
	})

	// Initialize the deployment orchestrator with all dependencies
	srv.orchestrator = deployment.NewOrchestrator(deployment.OrchestratorConfig{
		Repo:              repo,
		ProgressHub:       progressHub,
		SSHRunner:         sshRunner,
		SCPRunner:         scpRunner,
		SecretsFetcher:    secretsFetcher,
		SecretsGenerator:  secretsGenerator,
		DNSService:        dnsService,
		HistoryRecorder:   repo,
		ManifestRefresher: manifestRefresher,
		Logger:            srv.log,
	})

	// Initialize Tool Discovery Protocol registry
	srv.toolRegistry = toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "scenario-to-cloud",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Deploys scenarios to VPS targets with full lifecycle management, preflight checks, and live state inspection.",
	})
	// Register all tool providers
	srv.toolRegistry.RegisterProvider(toolregistry.NewDeploymentToolProvider())
	srv.toolRegistry.RegisterProvider(toolregistry.NewInspectionToolProvider())
	srv.toolRegistry.RegisterProvider(toolregistry.NewValidationToolProvider())

	// Initialize Tool Execution Protocol executor
	srv.toolExecutor = toolexecution.NewServerExecutor(toolexecution.ServerExecutorConfig{
		Repo:         repo,
		Resolver:     toolregistry.NewResolver(repo),
		Orchestrator: srv.orchestrator,
		SSHRunner:    sshRunner,
		DNSService:   dnsService,
		Logger:       srv.log,
	})

	srv.setupRoutes()
	return srv, nil
}

// DOC: docs/reference/api-endpoints.md — complete endpoint reference
func (s *Server) setupRoutes() {
	s.router.Use(loggingMiddleware)
	// Health endpoint at both root (for infrastructure) and /api/v1 (for clients)
	healthHandler := health.Handler(health.DB(s.db))
	s.router.HandleFunc("/health", healthHandler).Methods("GET")

	api := s.router.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/health", healthHandler).Methods("GET")
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
	api.HandleFunc("/bundles/cleanup", bundle.HandleBundleCleanup(s.sshRunner)).Methods("POST")
	api.HandleFunc("/bundles/vps/list", bundle.HandleListVPSBundles(s.sshRunner)).Methods("POST")
	api.HandleFunc("/bundles/vps/delete", bundle.HandleDeleteVPSBundle(s.sshRunner)).Methods("POST")
	api.HandleFunc("/bundles/{sha256}", bundle.HandleDeleteBundle()).Methods("DELETE")

	// Deployment-scoped VPS bundle cache (recommended, selector-first via deployment resolve).
	api.HandleFunc("/deployments/{id}/bundles/vps", s.handleListDeploymentVPSBundles).Methods("GET")
	api.HandleFunc("/deployments/{id}/bundles/vps/gc", s.handleGCDeploymentVPSBundles).Methods("POST")
	api.HandleFunc("/preflight", vpspreflight.HandlePreflight(vpspreflight.HandlerDeps{
		SSHRunner:         s.sshRunner,
		DNSService:        s.dnsService,
		ValidateManifest:  manifest.ValidateAndNormalize,
		HasBlockingIssues: manifest.HasBlockingIssues,
	})).Methods("POST")
	api.HandleFunc("/preflight/requirements", vpspreflight.HandleRequirements()).Methods("GET")
	api.HandleFunc("/secrets/{scenario}", secrets.HandleGetSecrets(s.secretsFetcher)).Methods("GET")
	api.HandleFunc("/local-secrets/{scope}/{key}", secrets.HandleGetLocalSecret()).Methods("GET")
	api.HandleFunc("/local-secrets/{scope}/{key}", secrets.HandleSetLocalSecret()).Methods("PUT")
	api.HandleFunc("/local-secrets/{scope}/{key}", secrets.HandleDeleteLocalSecret()).Methods("DELETE")
	api.HandleFunc("/vps/setup/plan", s.handleVPSSetupPlan).Methods("POST")
	api.HandleFunc("/vps/setup/apply", s.handleVPSSetupApply).Methods("POST")
	api.HandleFunc("/vps/deploy/plan", s.handleVPSDeployPlan).Methods("POST")
	api.HandleFunc("/vps/deploy/apply", s.handleVPSDeployApply).Methods("POST")
	api.HandleFunc("/vps/inspect/plan", s.handleVPSInspectPlan).Methods("POST")
	api.HandleFunc("/vps/inspect/apply", s.handleVPSInspectApply).Methods("POST")

	// Deployment management
	api.HandleFunc("/deployments", s.handleListDeployments).Methods("GET")
	api.HandleFunc("/deployments", s.handleCreateDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}", s.handleGetDeployment).Methods("GET")
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
	api.HandleFunc("/deployments/{id}/actions/kill", s.handleKillProcess).Methods("POST")
	api.HandleFunc("/deployments/{id}/actions/restart", s.handleRestartProcess).Methods("POST")
	api.HandleFunc("/deployments/{id}/actions/process", s.handleProcessControl).Methods("POST")
	api.HandleFunc("/deployments/{id}/actions/vps", s.handleVPSAction).Methods("POST")

	// History & Logs (Ground Truth Redesign - Phase 7)
	api.HandleFunc("/deployments/{id}/history", s.handleGetHistory).Methods("GET")
	api.HandleFunc("/deployments/{id}/history", s.handleAddHistoryEvent).Methods("POST")
	api.HandleFunc("/deployments/{id}/logs", s.handleGetLogs).Methods("GET")

	// VPS Secrets Management (Post-deployment secret CRUD)
	secretsMgmtDeps := secrets.ManagementDeps{Repo: s.repo, SSHRunner: s.sshRunner}
	api.HandleFunc("/deployments/{id}/secrets", secrets.HandleListVPSSecrets(secretsMgmtDeps)).Methods("GET")
	api.HandleFunc("/deployments/{id}/secrets", secrets.HandleCreateVPSSecret(secretsMgmtDeps)).Methods("POST")
	api.HandleFunc("/deployments/{id}/secrets/{key}", secrets.HandleGetVPSSecret(secretsMgmtDeps)).Methods("GET")
	api.HandleFunc("/deployments/{id}/secrets/{key}", secrets.HandleUpdateVPSSecret(secretsMgmtDeps)).Methods("PUT")
	api.HandleFunc("/deployments/{id}/secrets/{key}", secrets.HandleDeleteVPSSecret(secretsMgmtDeps)).Methods("DELETE")

	// Expected Secrets (secrets defined in scenario's service.json)
	api.HandleFunc("/deployments/{id}/expected-secrets", s.handleGetExpectedSecrets).Methods("GET")

	// Terminal (Ground Truth Redesign - Phase 8)
	api.HandleFunc("/deployments/{id}/terminal", s.handleTerminalWebSocket).Methods("GET")

	// Edge/TLS Management (Ground Truth Redesign - Enhancement)
	api.HandleFunc("/deployments/{id}/edge/dns-check", s.handleDNSCheck).Methods("GET")
	api.HandleFunc("/deployments/{id}/edge/dns-records", s.handleDNSRecords).Methods("GET")
	api.HandleFunc("/deployments/{id}/edge/caddy", s.handleCaddyControl).Methods("POST")
	api.HandleFunc("/deployments/{id}/edge/tls", s.handleTLSInfo).Methods("GET")
	api.HandleFunc("/deployments/{id}/edge/tls/renew", s.handleTLSRenew).Methods("POST")

	// Documentation
	api.HandleFunc("/docs/manifest", s.handleGetDocsManifest).Methods("GET")
	api.HandleFunc("/docs/content", s.handleGetDocContent).Methods("GET")

	// SSH Key Management
	keySvc := ssh.NewKeyService(nil, "")
	keyCopier := ssh.ExecKeyCopier{}
	api.HandleFunc("/ssh/keys", ssh.HandleListKeys(keySvc)).Methods("GET")
	api.HandleFunc("/ssh/keys", ssh.HandleDeleteKey(keySvc)).Methods("DELETE")
	api.HandleFunc("/ssh/keys/generate", ssh.HandleGenerateKey(keySvc)).Methods("POST")
	api.HandleFunc("/ssh/keys/public", ssh.HandleGetPublicKey(keySvc)).Methods("POST")
	api.HandleFunc("/ssh/test", ssh.HandleTestConnection(s.sshRunner, ssh.DefaultHandlerOptions())).Methods("POST")
	api.HandleFunc("/ssh/copy-key", ssh.HandleCopyKey(keyCopier, ssh.DefaultHandlerOptions())).Methods("POST")

	// Preflight fix actions
	api.HandleFunc("/preflight/fix/ports", vpspreflight.HandleStopPortServices(s.sshRunner)).Methods("POST")
	api.HandleFunc("/preflight/fix/firewall", vpspreflight.HandleOpenFirewallPorts(s.sshRunner)).Methods("POST")
	api.HandleFunc("/preflight/fix/stop-processes", vpspreflight.HandleStopScenarioProcesses(s.sshRunner, adaptStopScenarioFunc)).Methods("POST")
	api.HandleFunc("/preflight/disk/usage", vpspreflight.HandleDiskUsage(s.sshRunner)).Methods("POST")
	api.HandleFunc("/preflight/disk/cleanup", vpspreflight.HandleDiskCleanup(s.sshRunner)).Methods("POST")

	// Investigation endpoints (agent-manager integration) - legacy, kept for backward compatibility
	api.HandleFunc("/deployments/{id}/investigate", s.handleInvestigateDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/investigations", s.handleListInvestigations).Methods("GET")
	api.HandleFunc("/deployments/{id}/investigations/{invId}", s.handleGetInvestigation).Methods("GET")
	api.HandleFunc("/deployments/{id}/investigations/{invId}/stop", s.handleStopInvestigation).Methods("POST")
	api.HandleFunc("/deployments/{id}/investigations/{invId}/apply-fixes", s.handleApplyFixes).Methods("POST")
	api.HandleFunc("/agent-manager/status", s.handleCheckAgentManagerStatus).Methods("GET")

	// New unified task endpoints
	s.registerTaskRoutes(api)

	// Tool Discovery Protocol endpoints
	toolsHandler := toolhandlers.NewToolsHandler(s.toolRegistry)
	api.HandleFunc("/tools", toolsHandler.GetTools).Methods("GET", "OPTIONS")
	api.HandleFunc("/tools/{name}", toolsHandler.GetTool).Methods("GET", "OPTIONS")

	// Tool Execution Protocol endpoint
	toolExecHandler := toolexecution.NewHandler(s.toolExecutor)
	api.HandleFunc("/tools/execute", toolExecHandler.Execute).Methods("POST", "OPTIONS")
}

// Router returns the HTTP handler for use with server.Run
func (s *Server) Router() http.Handler {
	return handlers.RecoveryHandler()(s.router)
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

// adaptStopScenarioFunc adapts vps.StopExistingScenario to the preflight package interface.
func adaptStopScenarioFunc(ctx context.Context, sshRunner ssh.Runner, cfg ssh.Config, workdir, scenarioID string, targetPorts []int) vpspreflight.StopScenarioResult {
	result := vps.StopExistingScenario(ctx, sshRunner, cfg, workdir, scenarioID, targetPorts)
	return vpspreflight.StopScenarioResult{
		OK:      result.OK,
		Message: result.Message,
	}
}

// createAnalyzerFetcher returns a function that fetches dependencies from the scenario-dependency-analyzer.
// This is used by ManifestRefresher to get current dependencies when rebuilding.
func createAnalyzerFetcher() func(ctx context.Context, scenarioID string) (resources, scenarios []string, err error) {
	return func(ctx context.Context, scenarioID string) (resources, scenarios []string, err error) {
		baseURL, err := discovery.ResolveScenarioURLDefault(ctx, "scenario-dependency-analyzer")
		if err != nil {
			return nil, nil, fmt.Errorf("analyzer not available: %w", err)
		}

		// Call the analyzer API
		url := fmt.Sprintf("%s/api/v1/analyze/%s", strings.TrimSuffix(baseURL, "/"), scenarioID)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, nil, fmt.Errorf("create analyzer request: %w", err)
		}

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			return nil, nil, fmt.Errorf("analyzer request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return nil, nil, fmt.Errorf("analyzer returned status %d", resp.StatusCode)
		}

		// Parse analyzer response
		var analyzerResp struct {
			Resources []struct {
				DependencyName string `json:"dependency_name"`
				Required       bool   `json:"required"`
				Configuration  struct {
					Enabled bool `json:"enabled"`
				} `json:"configuration"`
			} `json:"resources"`
			Scenarios []struct {
				DependencyName string `json:"dependency_name"`
				Required       bool   `json:"required"`
				Configuration  struct {
					Enabled bool `json:"enabled"`
				} `json:"configuration"`
			} `json:"scenarios"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&analyzerResp); err != nil {
			return nil, nil, fmt.Errorf("parse analyzer response: %w", err)
		}

		// Collect enabled/required resources
		for _, r := range analyzerResp.Resources {
			if r.Required || r.Configuration.Enabled {
				resources = append(resources, r.DependencyName)
			}
		}

		// Collect enabled/required scenarios
		for _, s := range analyzerResp.Scenarios {
			if s.Required || s.Configuration.Enabled {
				scenarios = append(scenarios, s.DependencyName)
			}
		}

		return resources, scenarios, nil
	}
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
	if err := server.Run(server.Config{
		Handler:      srv.Router(),
		WriteTimeout: 35 * time.Minute,
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
