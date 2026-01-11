package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/health"
	corepreflight "github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"scenario-to-cloud/agentmanager"
	"scenario-to-cloud/bundle"
	"scenario-to-cloud/deployment"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/investigation"
	"scenario-to-cloud/manifest"
	"scenario-to-cloud/persistence"
	"scenario-to-cloud/secrets"
	"scenario-to-cloud/ssh"
	"scenario-to-cloud/tlsinfo"
	"scenario-to-cloud/vps"
	"scenario-to-cloud/vps/preflight"
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
	historyRecorder  deployment.HistoryRecorder
	orchestrator     *deployment.Orchestrator

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
	tlsService := tlsinfo.NewService(tlsinfo.WithTimeout(10 * time.Second))
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
		sshRunner:        sshRunner,
		scpRunner:        scpRunner,
		secretsFetcher:   secretsFetcher,
		secretsGenerator: secretsGenerator,
		dnsService:       dnsService,
		tlsService:       tlsService,
		tlsALPNRunner:    tlsALPNRunner,
	}

	// Initialize the deployment orchestrator with all dependencies
	srv.orchestrator = deployment.NewOrchestrator(deployment.OrchestratorConfig{
		Repo:             repo,
		ProgressHub:      progressHub,
		SSHRunner:        sshRunner,
		SCPRunner:        scpRunner,
		SecretsFetcher:   secretsFetcher,
		SecretsGenerator: secretsGenerator,
		DNSService:       dnsService,
		HistoryRecorder:  repo,
		Logger:           srv.log,
	})

	srv.setupRoutes()
	return srv, nil
}

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
	api.HandleFunc("/manifest/validate", s.handleManifestValidate).Methods("POST")
	api.HandleFunc("/bundle/build", s.handleBundleBuild).Methods("POST")
	api.HandleFunc("/bundles", bundle.HandleListBundles()).Methods("GET")
	api.HandleFunc("/bundles/stats", bundle.HandleBundleStats()).Methods("GET")
	api.HandleFunc("/bundles/cleanup", bundle.HandleBundleCleanup(s.sshRunner)).Methods("POST")
	api.HandleFunc("/bundles/vps/list", bundle.HandleListVPSBundles(s.sshRunner)).Methods("POST")
	api.HandleFunc("/bundles/vps/delete", bundle.HandleDeleteVPSBundle(s.sshRunner)).Methods("POST")
	api.HandleFunc("/bundles/{sha256}", bundle.HandleDeleteBundle()).Methods("DELETE")
	api.HandleFunc("/preflight", preflight.HandlePreflight(preflight.HandlerDeps{
		SSHRunner:         s.sshRunner,
		DNSService:        s.dnsService,
		ValidateManifest:  manifest.ValidateAndNormalize,
		HasBlockingIssues: manifest.HasBlockingIssues,
	})).Methods("POST")
	api.HandleFunc("/secrets/{scenario}", secrets.HandleGetSecrets(s.secretsFetcher)).Methods("GET")
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

	// Live state inspection (Ground Truth Redesign - Phase 1)
	api.HandleFunc("/deployments/{id}/live-state", s.handleGetLiveState).Methods("GET")
	api.HandleFunc("/deployments/{id}/files", s.handleGetFiles).Methods("GET")
	api.HandleFunc("/deployments/{id}/files/content", s.handleGetFileContent).Methods("GET")
	api.HandleFunc("/deployments/{id}/drift", s.handleGetDrift).Methods("GET")
	api.HandleFunc("/deployments/{id}/actions/kill", s.handleKillProcess).Methods("POST")
	api.HandleFunc("/deployments/{id}/actions/restart", s.handleRestartProcess).Methods("POST")
	api.HandleFunc("/deployments/{id}/actions/process", s.handleProcessControl).Methods("POST")
	api.HandleFunc("/deployments/{id}/actions/vps", s.handleVPSAction).Methods("POST")

	// History & Logs (Ground Truth Redesign - Phase 7)
	api.HandleFunc("/deployments/{id}/history", s.handleGetHistory).Methods("GET")
	api.HandleFunc("/deployments/{id}/history", s.handleAddHistoryEvent).Methods("POST")
	api.HandleFunc("/deployments/{id}/logs", s.handleGetLogs).Methods("GET")

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
	api.HandleFunc("/ssh/keys", ssh.HandleListKeys).Methods("GET")
	api.HandleFunc("/ssh/keys", ssh.HandleDeleteKey).Methods("DELETE")
	api.HandleFunc("/ssh/keys/generate", ssh.HandleGenerateKey).Methods("POST")
	api.HandleFunc("/ssh/keys/public", ssh.HandleGetPublicKey).Methods("POST")
	api.HandleFunc("/ssh/test", ssh.HandleTestConnection).Methods("POST")
	api.HandleFunc("/ssh/copy-key", ssh.HandleCopyKey).Methods("POST")

	// Preflight fix actions
	api.HandleFunc("/preflight/fix/ports", preflight.HandleStopPortServices(s.sshRunner)).Methods("POST")
	api.HandleFunc("/preflight/fix/firewall", preflight.HandleOpenFirewallPorts(s.sshRunner)).Methods("POST")
	api.HandleFunc("/preflight/fix/stop-processes", preflight.HandleStopScenarioProcesses(s.sshRunner, adaptStopScenarioFunc)).Methods("POST")
	api.HandleFunc("/preflight/disk/usage", preflight.HandleDiskUsage(s.sshRunner)).Methods("POST")
	api.HandleFunc("/preflight/disk/cleanup", preflight.HandleDiskCleanup(s.sshRunner)).Methods("POST")

	// Investigation endpoints (agent-manager integration)
	api.HandleFunc("/deployments/{id}/investigate", s.handleInvestigateDeployment).Methods("POST")
	api.HandleFunc("/deployments/{id}/investigations", s.handleListInvestigations).Methods("GET")
	api.HandleFunc("/deployments/{id}/investigations/{invId}", s.handleGetInvestigation).Methods("GET")
	api.HandleFunc("/deployments/{id}/investigations/{invId}/stop", s.handleStopInvestigation).Methods("POST")
	api.HandleFunc("/deployments/{id}/investigations/{invId}/apply-fixes", s.handleApplyFixes).Methods("POST")
	api.HandleFunc("/agent-manager/status", s.handleCheckAgentManagerStatus).Methods("GET")
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
func adaptStopScenarioFunc(ctx context.Context, sshRunner ssh.Runner, cfg ssh.Config, workdir, scenarioID string, targetPorts []int) preflight.StopScenarioResult {
	result := vps.StopExistingScenario(ctx, sshRunner, cfg, workdir, scenarioID, targetPorts)
	return preflight.StopScenarioResult{
		OK:      result.OK,
		Message: result.Message,
	}
}

func main() {
	// Preflight checks - must be first, before any initialization
	if corepreflight.Run(corepreflight.Config{
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
