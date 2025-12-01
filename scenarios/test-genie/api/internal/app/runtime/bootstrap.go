package runtime

import (
	"database/sql"
	"fmt"

	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/scenarios"
	"test-genie/internal/suite"
)

// Bootstrapped holds the concrete dependencies needed by the HTTP server.
type Bootstrapped struct {
	DB               *sql.DB
	SuiteRequests    *suite.SuiteRequestService
	ExecutionRepo    *suite.SuiteExecutionRepository
	ExecutionService *suite.SuiteExecutionService
	ScenarioService  *scenarios.ScenarioDirectoryService
	PhaseCatalog     phaseCatalogProvider
}

type phaseCatalogProvider interface {
	DescribePhases() []phases.Descriptor
}

// BuildDependencies wires the runtime config into the persistence + orchestrator services.
func BuildDependencies(cfg *Config) (*Bootstrapped, error) {
	if cfg == nil {
		return nil, fmt.Errorf("config is required")
	}
	db, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}
	if err := ensureDatabaseSchema(db); err != nil {
		return nil, fmt.Errorf("failed to apply database schema: %w", err)
	}

	runner, err := orchestrator.NewSuiteOrchestrator(cfg.ScenariosRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize orchestrator: %w", err)
	}

	suiteRequestRepo := suite.NewPostgresSuiteRequestRepository(db)
	suiteRequestService := suite.NewSuiteRequestService(suiteRequestRepo)
	executionRepo := suite.NewSuiteExecutionRepository(db)
	scenarioRepo := scenarios.NewScenarioDirectoryRepository(db)
	scenarioLister := scenarios.NewVrooliScenarioLister()
	scenarioService := scenarios.NewScenarioDirectoryService(scenarioRepo, scenarioLister)

	executionSvc := suite.NewSuiteExecutionService(runner, executionRepo, suiteRequestService)

	return &Bootstrapped{
		DB:               db,
		SuiteRequests:    suiteRequestService,
		ExecutionRepo:    executionRepo,
		ExecutionService: executionSvc,
		ScenarioService:  scenarioService,
		PhaseCatalog:     runner,
	}, nil
}
