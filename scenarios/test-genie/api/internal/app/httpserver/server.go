package httpserver

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/api-core/apihttp"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/devrouting"
	apiserver "github.com/vrooli/api-core/server"

	"test-genie/agentmanager"
	appelig "test-genie/internal/app/eligibility"
	apprun "test-genie/internal/app/runs"
	appvalidation "test-genie/internal/app/validation"
	"test-genie/internal/dbexec"
	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/playbooksclaims"
	"test-genie/internal/remediation"
	"test-genie/internal/runmanager"
	"test-genie/internal/scenarios"
	"test-genie/internal/selfhealthsnapshots"

	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"

	scenariovalidationconnect "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-validation/v1/scenariovalidationv1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/eligibility/eligibility_v1connect"
	"github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs/runs_v1connect"
)

// Config controls the HTTP transport settings.
type Config struct {
	Port        string
	ServiceName string
}

// Logger captures the logging surface the HTTP layer relies on.
type Logger interface {
	Print(v ...interface{})
	Printf(format string, v ...interface{})
}

// Dependencies encapsulates the services the HTTP layer needs to operate.
type Dependencies struct {
	DB                  *database.RoutedDB
	HealthDB            dbexec.HealthProbe
	Executions          execution.ExecutionHistory
	ExecutionPlanner    executionPlanner
	RunManager          *runmanager.Manager
	Scenarios           scenarioDirectory
	PhaseCatalog        phaseCatalog
	AgentService        *agentmanager.AgentService
	RemediationService  remediationService
	RemediationLauncher remediation.Launcher
	RequirementsSyncer  requirementsSyncer
	PlaybooksClaims     *playbooksclaims.Service
	EligibilityService  *appelig.Service
	RunsService         *apprun.Service
	ValidationService   *appvalidation.Service
	// StartBackground receives a process-owned context after the HTTP listener
	// is live. It is for advisory work only; serving must not depend on it.
	StartBackground func(context.Context)
	SweepStatus     *selfhealthsnapshots.StatusStore
	Logger          Logger
}

type executionPlanner interface {
	Preview(ctx context.Context, req orchestrator.SuiteExecutionRequest) (*execution.ExecutionPlanPreview, error)
}

type scenarioDirectory interface {
	ListSummaries(ctx context.Context) ([]scenarios.ScenarioSummary, error)
	GetSummary(ctx context.Context, name string) (*scenarios.ScenarioSummary, error)
	RunScenarioTests(ctx context.Context, name string, preferred string, extraArgs []string, scenarioDirOverride string) (*scenarios.TestingCommand, *scenarios.TestingRunnerResult, error)
	ListFiles(ctx context.Context, name string, opts scenarios.FileListOptions) ([]scenarios.FileNode, error)
	ListFilesWithMeta(ctx context.Context, name string, opts scenarios.FileListOptions) (scenarios.FileListResult, error)
	ScenarioRoot() string
}

type phaseCatalog interface {
	DescribePhases() []phases.Descriptor
	GlobalPhaseToggles() (orchestrator.PhaseToggleConfig, error)
	SaveGlobalPhaseToggles(orchestrator.PhaseToggleConfig) (orchestrator.PhaseToggleConfig, error)
}

type requirementsSyncer interface {
	Sync(ctx context.Context, scenarioDir string) error
}

type remediationService interface {
	Create(context.Context, remediation.Plan, []string, []string, string) (remediation.Job, error)
	Get(context.Context, string) (remediation.Job, error)
	List(context.Context, string, int) ([]remediation.Job, error)
	Cancel(context.Context, string) (remediation.Job, error)
	PrepareLaunch(context.Context, string, string) (remediation.Job, error)
	RetryLaunch(context.Context, string) (remediation.Job, error)
	RecordLaunchFailure(context.Context, string, string) (remediation.Job, error)
	MarkRunning(context.Context, string, remediation.Attribution) (remediation.Job, error)
	MarkAgentCompleted(context.Context, string, string) (remediation.Job, error)
	StartVerification(context.Context, string, remediation.Verification) (remediation.Job, error)
	ReserveVerification(context.Context, string) (remediation.Job, error)
	SetVerificationRun(context.Context, string, remediation.Verification) (remediation.Job, error)
	ReleaseVerificationReservation(context.Context, string) (remediation.Job, error)
	CompleteVerification(context.Context, string, remediation.Verification, remediation.FindingDelta, remediation.RequirementDelta, string) (remediation.Job, error)
	Fail(context.Context, string, string) (remediation.Job, error)
}

// Server wires the HTTP router, configuration, and service dependencies behind intentional seams.
type Server struct {
	config                 Config
	db                     *database.RoutedDB
	healthDB               dbexec.HealthProbe
	router                 *mux.Router
	executionHistory       execution.ExecutionHistory
	executionPlanner       executionPlanner
	runManager             *runmanager.Manager
	scenarios              scenarioDirectory
	phaseCatalog           phaseCatalog
	logger                 Logger
	agentService           *agentmanager.AgentService
	remediationService     remediationService
	remediationLauncher    remediation.Launcher
	requirementsSyncer     requirementsSyncer
	playbooksClaims        *playbooksclaims.Service
	eligibilityService     *appelig.Service
	runsService            *apprun.Service
	validationService      *appvalidation.Service
	startBackground        func(context.Context)
	sweepStatus            *selfhealthsnapshots.StatusStore
	seedSessions           map[string]*seedSession
	seedSessionsByScenario map[string]string
	seedSessionsMu         sync.Mutex
}

// New creates a configured HTTP server instance.
func New(config Config, deps Dependencies) (*Server, error) {
	if strings.TrimSpace(config.Port) == "" {
		return nil, fmt.Errorf("api port is required")
	}
	if deps.DB == nil {
		return nil, fmt.Errorf("database dependency is required")
	}
	if deps.Executions == nil {
		return nil, fmt.Errorf("execution history service is required")
	}
	if deps.ExecutionPlanner == nil {
		return nil, fmt.Errorf("execution planner is required")
	}
	if deps.RunManager == nil {
		return nil, fmt.Errorf("run manager is required")
	}
	if deps.Scenarios == nil {
		return nil, fmt.Errorf("scenario directory service is required")
	}
	if deps.PhaseCatalog == nil {
		return nil, fmt.Errorf("phase catalog dependency is required")
	}
	if deps.AgentService == nil {
		return nil, fmt.Errorf("agent service dependency is required")
	}

	logger := deps.Logger
	if logger == nil {
		logger = log.Default()
	}

	srv := &Server{
		config:                 config,
		db:                     deps.DB,
		healthDB:               deps.HealthDB,
		router:                 mux.NewRouter(),
		executionHistory:       deps.Executions,
		executionPlanner:       deps.ExecutionPlanner,
		runManager:             deps.RunManager,
		scenarios:              deps.Scenarios,
		phaseCatalog:           deps.PhaseCatalog,
		logger:                 logger,
		agentService:           deps.AgentService,
		remediationService:     deps.RemediationService,
		remediationLauncher:    deps.RemediationLauncher,
		requirementsSyncer:     deps.RequirementsSyncer,
		playbooksClaims:        deps.PlaybooksClaims,
		eligibilityService:     deps.EligibilityService,
		runsService:            deps.RunsService,
		validationService:      deps.ValidationService,
		startBackground:        deps.StartBackground,
		sweepStatus:            deps.SweepStatus,
		seedSessions:           make(map[string]*seedSession),
		seedSessionsByScenario: make(map[string]string),
	}

	srv.setupRoutes()
	return srv, nil
}

func (s *Server) setupRoutes() {
	s.router.Use(s.securityHeadersMiddleware)
	s.router.Use(s.loggingMiddleware)
	// Health endpoint at root for infrastructure agents
	s.router.HandleFunc("/health", s.handleHealth).Methods("GET")

	apiRouter := s.router.PathPrefix("/api/v1").Subrouter()
	apiRouter.HandleFunc("/health", s.handleHealth).Methods("GET")
	apiRouter.HandleFunc("/admission", s.handleAdmissionStatus).Methods("GET")
	apiRouter.HandleFunc("/admission/profile", s.handleAdmissionProfile).Methods("POST")
	apiRouter.HandleFunc("/config", s.handleGetConfig).Methods("GET")
	apiRouter.HandleFunc("/phases", s.handleListPhases).Methods("GET")
	apiRouter.HandleFunc("/phases/applicability", s.handlePreviewPhaseApplicability).Methods("GET")
	apiRouter.HandleFunc("/phases/settings", s.handleGetPhaseSettings).Methods("GET")
	apiRouter.HandleFunc("/phases/settings", s.handleUpdatePhaseSettings).Methods("PUT")
	apiRouter.HandleFunc("/phases/{phase}", s.handleInspectPhase).Methods("GET")
	apiRouter.HandleFunc("/executions", s.handleExecuteSuite).Methods("POST")
	apiRouter.HandleFunc("/executions/plan", s.handlePreviewExecutionPlan).Methods("POST")
	apiRouter.HandleFunc("/executions/stream", s.handleExecuteSuiteStream).Methods("POST")
	apiRouter.HandleFunc("/executions", s.handleListExecutions).Methods("GET")
	apiRouter.HandleFunc("/executions/{id}", s.handleGetExecution).Methods("GET")
	apiRouter.HandleFunc("/scenarios", s.handleListScenarios).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}", s.handleGetScenario).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}/run-tests", s.handleRunScenarioTests).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/playbooks/seed/apply", s.handlePlaybooksSeedApply).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/playbooks/seed/cleanup", s.handlePlaybooksSeedCleanup).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/playbooks/seed/cleanup-force", s.handlePlaybooksSeedCleanupForce).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/files", s.handleListScenarioFiles).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}/remediation/plans/{executionID}", s.handleGetRemediationPlan).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}/remediation/jobs", s.handleCreateRemediationJob).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/remediation/jobs", s.handleListRemediationJobs).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}/remediation/jobs/{id}", s.handleGetRemediationJob).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}/remediation/jobs/{id}/cancel", s.handleCancelRemediationJob).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/remediation/jobs/{id}/recover", s.handleRecoverRemediationJob).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/remediation/jobs/{id}/retry", s.handleRetryRemediationJob).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/remediation/jobs/{id}/agent-status", s.handleRefreshRemediationAgent).Methods("POST")
	apiRouter.HandleFunc("/scenarios/{name}/remediation/jobs/{id}/verify", s.handleVerifyRemediationJob).Methods("POST")

	// Agent Manager exposes portable role choices. Remediation jobs are the only
	// supported Test Genie surface that creates or controls agent runs.
	apiRouter.HandleFunc("/agents/roles", s.handleListAgentRoles).Methods("GET")

	// Docs endpoints for in-app documentation browser
	apiRouter.HandleFunc("/docs/manifest", s.handleGetDocsManifest).Methods("GET")
	apiRouter.HandleFunc("/docs/content", s.handleGetDocContent).Methods("GET")

	// Requirements endpoints for requirements sync UI
	apiRouter.HandleFunc("/scenarios/{name}/requirements", s.handleGetScenarioRequirements).Methods("GET")
	apiRouter.HandleFunc("/scenarios/{name}/requirements/sync", s.handleSyncScenarioRequirements).Methods("POST")

	// Workflow seed claim routes (legacy playbooks URL retained for clients)
	apiRouter.HandleFunc("/playbooks/claims", s.handleListPlaybooksClaims).Methods("GET")
	apiRouter.HandleFunc("/playbooks/claims/{scenario}", s.handleGetPlaybooksClaim).Methods("GET")
	apiRouter.HandleFunc("/playbooks/claims/{scenario}/release", s.handleReleasePlaybooksClaim).Methods("POST")

	// Eligibility Connect-RPC service. The handler resolves a per-scenario
	// auditor scan and reports whether the scenario qualifies for the
	// routed-test-db path. gorilla/mux matches the full path prefix the
	// Connect handler emits; the {rest:.*} suffix forwards every method
	// under that prefix to the generated handler.
	if s.eligibilityService != nil {
		path, handler := eligibility_v1connect.NewEligibilityServiceHandler(s.eligibilityService)
		s.router.PathPrefix(path).Handler(handler)
	}

	// Provider-conformance ScenarioValidationService: Test Genie's own
	// descriptor-backed phase. Validates target scenarios that declare
	// .vrooli/test-genie.json phase descriptors (descriptor, embedded maturity,
	// policy, stale-file, and live provider-contract conformance).
	if s.validationService != nil {
		path, handler := scenariovalidationconnect.NewScenarioValidationServiceHandler(s.validationService.Served())
		s.router.PathPrefix(path).Handler(handler)
	}

	// Runs Connect-RPC service: enumerates/compares/pins the append-only run
	// index. Consumed by the test-genie CLI and GCT baseline adapters.
	if s.runsService != nil {
		path, handler := runs_v1connect.NewRunsServiceHandler(s.runsService)
		s.router.PathPrefix(path).Handler(handler)

		// Opaque binary artifact route: metadata is enumerated through the typed
		// RunsService catalog while bytes use REST so media range requests work
		// without buffering entire recordings through protobuf.
		apiRouter.HandleFunc("/scenarios/{name}/runs/{runId}/artifacts/{artifactId}", s.handleGetRunArtifactByID).Methods("GET")
	}
}

// Handler preserves Test Genie's router, development routing, test-mode
// isolation, and recovery middleware while delegating lifecycle and receipt
// observation to api-core/server.
func (s *Server) Handler() http.Handler {
	rootMux := http.NewServeMux()
	if s.db != nil {
		devrouting.Register(rootMux, s.db)
	}
	rootMux.Handle("/", s.router)
	return handlers.RecoveryHandler()(apihttp.TestModeMiddleware(rootMux))
}

// Start launches through the standard API Core server so receipt observation,
// verified provenance, streaming-safe wrappers, and graceful shutdown are
// shared with the rest of the fleet.
func (s *Server) Start() error {
	s.log("starting server", map[string]interface{}{
		"service": s.serviceName(),
		"port":    s.config.Port,
	})

	backgroundCtx, cancelBackground := context.WithCancel(context.Background())
	defer cancelBackground()
	if s.startBackground != nil {
		s.startBackground(backgroundCtx)
	}
	return apiserver.Run(apiserver.Config{
		Handler:      s.Handler(),
		Port:         s.config.Port,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 20 * time.Minute,
		IdleTimeout:  120 * time.Second,
		Cleanup: func(context.Context) error {
			cancelBackground()
			if s.runManager != nil {
				s.runManager.Shutdown()
			}
			if s.healthDB != nil {
				return s.healthDB.Close()
			}
			return nil
		},
	})
}

// loggingMiddleware prints simple request logs.
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Printf("[%s] %s %s", r.Method, r.RequestURI, time.Since(start))
	})
}

// securityHeadersMiddleware sets baseline OWASP security headers and the
// permissive CORS policy used across the API on every response. Origin is a
// wildcard without credentials, so no credentialed-wildcard exposure exists.
func (s *Server) securityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Strict-Transport-Security", "max-age=31536000")
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Vrooli-Actor, X-Update-Key")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) log(msg string, fields map[string]interface{}) {
	if s.logger == nil {
		return
	}
	if len(fields) == 0 {
		s.logger.Print(msg)
		return
	}
	s.logger.Printf("%s | %v", msg, fields)
}

func (s *Server) serviceName() string {
	name := strings.TrimSpace(s.config.ServiceName)
	if name == "" {
		return "Test Genie API"
	}
	return name
}
