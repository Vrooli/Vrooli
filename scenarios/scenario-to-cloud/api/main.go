package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
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
	"github.com/vrooli/api-core/preflight"
	"github.com/vrooli/api-core/server"

	"scenario-to-cloud/agentmanager"
	"scenario-to-cloud/dns"
	"scenario-to-cloud/persistence"
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
	progressHub      *ProgressHub
	agentSvc         *agentmanager.AgentService
	investigationSvc *InvestigationService
	historyRecorder  HistoryRecorder

	// Seam: SSH command execution (defaults to ExecSSHRunner)
	sshRunner SSHRunner
	// Seam: SCP file transfer (defaults to ExecSCPRunner)
	scpRunner SCPRunner
	// Seam: Secrets fetching (defaults to NewSecretsClient())
	secretsFetcher SecretsFetcher
	// Seam: Secrets generation (defaults to NewSecretsGenerator())
	secretsGenerator SecretsGeneratorFunc
	// Seam: DNS services (defaults to dns.NewService(dns.NetResolver{}, dns.WithTimeout(...)))
	dnsService dns.Service
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

	progressHub := NewProgressHub()

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

	srv := &Server{
		config:           cfg,
		router:           mux.NewRouter(),
		db:               db,
		repo:             repo,
		progressHub:      progressHub,
		historyRecorder:  repo,
		agentSvc:         agentSvc,
		investigationSvc: NewInvestigationService(repo, agentSvc, progressHub),
		// Initialize seams with production implementations
		sshRunner:        ExecSSHRunner{},
		scpRunner:        ExecSCPRunner{},
		secretsFetcher:   NewSecretsClient(),
		secretsGenerator: NewSecretsGenerator(),
		dnsService:       dns.NewService(dns.NetResolver{}, dns.WithTimeout(10*time.Second)),
	}

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
	api.HandleFunc("/bundles", s.handleListBundles).Methods("GET")
	api.HandleFunc("/bundles/stats", s.handleBundleStats).Methods("GET")
	api.HandleFunc("/bundles/cleanup", s.handleBundleCleanup).Methods("POST")
	api.HandleFunc("/bundles/vps/list", s.handleListVPSBundles).Methods("POST")
	api.HandleFunc("/bundles/vps/delete", s.handleDeleteVPSBundle).Methods("POST")
	api.HandleFunc("/bundles/{sha256}", s.handleDeleteBundle).Methods("DELETE")
	api.HandleFunc("/preflight", s.handlePreflight).Methods("POST")
	api.HandleFunc("/secrets/{scenario}", s.handleGetSecrets).Methods("GET")
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
	api.HandleFunc("/ssh/keys", s.handleListSSHKeys).Methods("GET")
	api.HandleFunc("/ssh/keys", s.handleDeleteSSHKey).Methods("DELETE")
	api.HandleFunc("/ssh/keys/generate", s.handleGenerateSSHKey).Methods("POST")
	api.HandleFunc("/ssh/keys/public", s.handleGetPublicKey).Methods("POST")
	api.HandleFunc("/ssh/test", s.handleTestSSHConnection).Methods("POST")
	api.HandleFunc("/ssh/copy-key", s.handleCopySSHKey).Methods("POST")

	// Preflight fix actions
	api.HandleFunc("/preflight/fix/ports", s.handleStopPortServices).Methods("POST")
	api.HandleFunc("/preflight/fix/stop-processes", s.handleStopScenarioProcesses).Methods("POST")
	api.HandleFunc("/preflight/disk/usage", s.handleDiskUsage).Methods("POST")
	api.HandleFunc("/preflight/disk/cleanup", s.handleDiskCleanup).Methods("POST")

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

// loggingMiddleware prints simple request logs
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
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

func decodeJSON[T any](r io.Reader, maxBytes int64) (T, error) {
	var zero T
	if r == nil {
		return zero, errors.New("missing request body")
	}
	body := io.LimitReader(r, maxBytes)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(&zero); err != nil {
		return zero, err
	}
	if err := dec.Decode(&struct{}{}); err == nil {
		return zero, errors.New("unexpected extra JSON values")
	} else if !errors.Is(err, io.EOF) {
		return zero, err
	}
	return zero, nil
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
