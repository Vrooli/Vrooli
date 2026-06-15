package runtime

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"test-genie/agentmanager"
	appelig "test-genie/internal/app/eligibility"
	apprun "test-genie/internal/app/runs"
	"test-genie/internal/eligibility"
	"test-genie/internal/execution"
	"test-genie/internal/fix"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/playbooksclaims"
	"test-genie/internal/queue"
	"test-genie/internal/requirements"
	"test-genie/internal/requirementsimprove"
	"test-genie/internal/runmanager"
	"test-genie/internal/scenarios"
	"test-genie/internal/toolexecution"
	"test-genie/internal/toolregistry"

	"github.com/vrooli/api-core/database"
	// Register modernc.org/sqlite as the pure-Go "sqlite" driver.
	_ "modernc.org/sqlite"
)

// Bootstrapped holds the concrete dependencies needed by the HTTP server.
type Bootstrapped struct {
	DB                         *database.RoutedDB
	SuiteRequests              *queue.SuiteRequestService
	ExecutionRepo              *execution.SuiteExecutionRepository
	ExecutionHistory           execution.ExecutionHistory
	ExecutionService           *execution.SuiteExecutionService
	ExecutionPlanner           execution.ExecutionPlanner
	RunManager                 *runmanager.Manager
	ScenarioService            *scenarios.ScenarioDirectoryService
	PhaseCatalog               phaseCatalogProvider
	AgentService               *agentmanager.AgentService
	FixService                 *fix.Service
	RequirementsImproveService *requirementsimprove.Service
	RequirementsSyncer         *RequirementsSyncerAdapter
	PlaybooksClaims            *playbooksclaims.Service
	EligibilityService         *appelig.Service
	RunsService                *apprun.Service
	// Tool Discovery Protocol support
	ToolRegistry *toolregistry.Registry
	ToolHandler  *toolexecution.Handler
}

// RequirementsSyncerAdapter adapts the requirements.Service to a simple Sync interface.
type RequirementsSyncerAdapter struct {
	svc *requirements.Service
}

// Sync performs requirements synchronization for a scenario directory. The
// structured sync report is discarded here; callers of this adapter only need
// success/failure.
func (a *RequirementsSyncerAdapter) Sync(ctx context.Context, scenarioDir string) error {
	_, err := a.svc.Sync(ctx, requirements.SyncInput{
		ScenarioDir: scenarioDir,
	})
	return err
}

type phaseCatalogProvider interface {
	DescribePhases() []phases.Descriptor
	GlobalPhaseToggles() (orchestrator.PhaseToggleConfig, error)
	SaveGlobalPhaseToggles(orchestrator.PhaseToggleConfig) (orchestrator.PhaseToggleConfig, error)
}

// BuildDependencies wires the runtime config into the persistence + orchestrator services.
func BuildDependencies(cfg *Config) (*Bootstrapped, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	// database.Open returns a *RoutedDB: at startup (no test mode) every query
	// goes to the primary pool, but the handle can route per-request to a test
	// pool installed via RoutingService — the in-place routed e2e path test-genie
	// uses to test scenarios (and itself) without a restart.
	db, err := database.Open(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          cfg.DatabaseDSN,
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := ensureDatabaseSchema(db); err != nil {
		return nil, fmt.Errorf("failed to apply database schema: %w", err)
	}

	runner, err := orchestrator.NewSuiteOrchestrator(cfg.ScenariosRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize orchestrator: %w", err)
	}

	suiteRequestRepo := queue.NewSQLiteSuiteRequestRepository(db)
	suiteRequestService := queue.NewSuiteRequestService(suiteRequestRepo)
	executionRepo := execution.NewSuiteExecutionRepository(db)
	executionHistory := execution.NewExecutionHistoryService(executionRepo)
	executionPlanner := execution.NewExecutionPlanService(runner, executionRepo)
	scenarioRepo := scenarios.NewScenarioDirectoryRepository(db)
	scenarioLister := scenarios.NewVrooliScenarioLister()
	scenarioService := scenarios.NewScenarioDirectoryService(scenarioRepo, scenarioLister, cfg.ScenariosRoot)

	executionSvc := execution.NewSuiteExecutionService(runner, executionRepo, suiteRequestService)

	// The run manager owns durable run execution decoupled from any client
	// request: it is the single engine every door (blocking REST, SSE gateway,
	// Connect run surface) funnels through, so a run survives client cancellation.
	runManager := runmanager.New(executionSvc, cfg.ScenariosRoot)
	if swept, err := runManager.Sweep(); err != nil {
		log.Printf("[test-genie] run-index startup sweep failed: %v", err)
	} else if swept > 0 {
		log.Printf("[test-genie] startup sweep: marked %d orphaned in-progress run(s) aborted", swept)
	}

	claimsRepo := playbooksclaims.NewSqliteRepository(db)
	claimsService := playbooksclaims.NewService(playbooksclaims.Config{Repo: claimsRepo})
	runner.SetClaims(claimsService)

	// Construct the routed-test-db eligibility checker once at process
	// startup and share it between the playbooks phase and the Connect
	// EligibilityService handler. Sharing the instance lets a CLI/GCT
	// eligibility lookup reuse the scan cache primed by the playbooks
	// phase (and vice versa).
	routingEligibility := eligibility.NewChecker(0)
	phases.SetRoutingChecker(routingEligibility)
	eligibilityService := appelig.NewService(routingEligibility, cfg.ScenariosRoot)

	// RunsService exposes the append-only run index AND the durable run
	// lifecycle (start/follow/wait/abort/status) over Connect-RPC, delegating
	// execution to the run manager.
	runsService := apprun.NewService(cfg.ScenariosRoot, runManager, executionPlanner)

	// Create agent-manager service
	agentEnabled := os.Getenv("AGENT_MANAGER_ENABLED") != "false"
	profileKey := os.Getenv("AGENT_MANAGER_PROFILE_KEY")
	if profileKey == "" {
		profileKey = "test-genie"
	}

	agentService := agentmanager.NewAgentService(agentmanager.Config{
		ProfileName: "Test Genie Agent",
		ProfileKey:  profileKey,
		Timeout:     30 * time.Second,
		Enabled:     agentEnabled,
	})

	// Initialize profile at startup (non-blocking)
	if agentEnabled {
		go func() {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			if err := agentService.Initialize(ctx, agentmanager.DefaultProfileConfig()); err != nil {
				log.Printf("[agent-manager] Warning: failed to initialize profile: %v", err)
			}
		}()
	}

	// Create fix service (for agent-based test fixing)
	fixService := fix.NewService(agentService)

	// Create requirements improve service (for agent-based requirements improvement)
	reqImproveService := requirementsimprove.NewService(agentService)

	// Create requirements syncer
	reqSyncer := &RequirementsSyncerAdapter{
		svc: requirements.NewService(),
	}

	// Create tool registry for Tool Discovery Protocol
	toolReg := toolregistry.NewRegistry(toolregistry.RegistryConfig{
		ScenarioName:        "test-genie",
		ScenarioVersion:     "1.0.0",
		ScenarioDescription: "Automated testing and quality assurance for Vrooli scenarios",
	})

	// Register all tool providers
	toolReg.RegisterProvider(toolregistry.NewTestingToolProvider())
	toolReg.RegisterProvider(toolregistry.NewFixToolProvider())
	toolReg.RegisterProvider(toolregistry.NewRequirementsToolProvider())

	// Create tool executor with all required dependencies
	toolExec := toolexecution.NewServerExecutor(toolexecution.ServerExecutorConfig{
		ExecutionHistory:    executionHistory,
		SuiteExecutor:       executionSvc,
		ScenarioDirectory:   scenarioService,
		PhaseCatalog:        runner,
		FixService:          fixService,
		RequirementsImprove: reqImproveService,
		RequirementsSyncer:  reqSyncer,
	})
	toolHandler := toolexecution.NewHandler(toolExec)

	log.Printf("[test-genie] Tool Discovery Protocol enabled with %d tools", len(toolReg.ListToolNames(context.Background())))

	return &Bootstrapped{
		DB:                         db,
		SuiteRequests:              suiteRequestService,
		ExecutionRepo:              executionRepo,
		ExecutionHistory:           executionHistory,
		ExecutionService:           executionSvc,
		ExecutionPlanner:           executionPlanner,
		RunManager:                 runManager,
		ScenarioService:            scenarioService,
		PhaseCatalog:               runner,
		AgentService:               agentService,
		FixService:                 fixService,
		RequirementsImproveService: reqImproveService,
		RequirementsSyncer:         reqSyncer,
		PlaybooksClaims:            claimsService,
		EligibilityService:         eligibilityService,
		RunsService:                runsService,
		ToolRegistry:               toolReg,
		ToolHandler:                toolHandler,
	}, nil
}
