package runtime

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"time"

	"test-genie/agentmanager"
	"test-genie/internal/execution"
	"test-genie/internal/fix"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/queue"
	"test-genie/internal/requirements"
	"test-genie/internal/requirementsimprove"
	"test-genie/internal/scenarios"
	"test-genie/internal/toolexecution"
	"test-genie/internal/toolregistry"

	"github.com/vrooli/api-core/database"
)

// Bootstrapped holds the concrete dependencies needed by the HTTP server.
type Bootstrapped struct {
	DB                         *sql.DB
	SuiteRequests              *queue.SuiteRequestService
	ExecutionRepo              *execution.SuiteExecutionRepository
	ExecutionHistory           execution.ExecutionHistory
	ExecutionService           *execution.SuiteExecutionService
	ScenarioService            *scenarios.ScenarioDirectoryService
	PhaseCatalog               phaseCatalogProvider
	AgentService               *agentmanager.AgentService
	FixService                 *fix.Service
	RequirementsImproveService *requirementsimprove.Service
	RequirementsSyncer         *RequirementsSyncerAdapter
	// Tool Discovery Protocol support
	ToolRegistry *toolregistry.Registry
	ToolHandler  *toolexecution.Handler
}

// RequirementsSyncerAdapter adapts the requirements.Service to a simple Sync interface.
type RequirementsSyncerAdapter struct {
	svc *requirements.Service
}

// Sync performs requirements synchronization for a scenario directory.
func (a *RequirementsSyncerAdapter) Sync(ctx context.Context, scenarioDir string) error {
	return a.svc.Sync(ctx, requirements.SyncInput{
		ScenarioDir: scenarioDir,
	})
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
	db, err := database.Connect(context.Background(), database.Config{
		Driver: "postgres",
		DSN:    cfg.DatabaseURL,
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

	suiteRequestRepo := queue.NewPostgresSuiteRequestRepository(db)
	suiteRequestService := queue.NewSuiteRequestService(suiteRequestRepo)
	executionRepo := execution.NewSuiteExecutionRepository(db)
	executionHistory := execution.NewExecutionHistoryService(executionRepo)
	scenarioRepo := scenarios.NewScenarioDirectoryRepository(db)
	scenarioLister := scenarios.NewVrooliScenarioLister()
	scenarioService := scenarios.NewScenarioDirectoryService(scenarioRepo, scenarioLister, cfg.ScenariosRoot)

	executionSvc := execution.NewSuiteExecutionService(runner, executionRepo, suiteRequestService)

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
		ScenarioService:            scenarioService,
		PhaseCatalog:               runner,
		AgentService:               agentService,
		FixService:                 fixService,
		RequirementsImproveService: reqImproveService,
		RequirementsSyncer:         reqSyncer,
		ToolRegistry:               toolReg,
		ToolHandler:                toolHandler,
	}, nil
}
