package runtime

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"test-genie/agentmanager"
	appelig "test-genie/internal/app/eligibility"
	apprun "test-genie/internal/app/runs"
	appvalidation "test-genie/internal/app/validation"
	"test-genie/internal/dbexec"
	"test-genie/internal/eligibility"
	"test-genie/internal/execution"
	"test-genie/internal/orchestrator"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/playbooksclaims"
	"test-genie/internal/remediation"
	"test-genie/internal/requirements"
	"test-genie/internal/runmanager"
	"test-genie/internal/scenarios"
	"test-genie/internal/selfhealthsnapshots"
	sharedruns "test-genie/internal/shared/runs"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/maturity-go/assessment"
	// Register modernc.org/sqlite as the pure-Go "sqlite" driver.
	_ "modernc.org/sqlite"
)

// Bootstrapped holds the concrete dependencies needed by the HTTP server.
type Bootstrapped struct {
	DB *database.RoutedDB
	// HealthDB is a dedicated, read-only lifecycle probe connection. It is
	// intentionally separate from the single runtime SQLite pool so background
	// analytics cannot make lifecycle health queue behind ordinary work.
	HealthDB            dbexec.HealthProbe
	ExecutionRepo       *execution.SuiteExecutionRepository
	ExecutionHistory    execution.ExecutionHistory
	ExecutionService    *execution.SuiteExecutionService
	ExecutionPlanner    execution.ExecutionPlanner
	RunManager          *runmanager.Manager
	ScenarioService     *scenarios.ScenarioDirectoryService
	PhaseCatalog        phaseCatalogProvider
	AgentService        *agentmanager.AgentService
	RemediationService  *remediation.Service
	RemediationLauncher remediation.Launcher
	RequirementsSyncer  *RequirementsSyncerAdapter
	PlaybooksClaims     *playbooksclaims.Service
	EligibilityService  *appelig.Service
	RunsService         *apprun.Service
	ValidationService   *appvalidation.Service
	// StartBackground is invoked by the HTTP transport only after its listener is
	// accepting requests. Expensive advisory work must never begin while the
	// lifecycle health endpoint is still competing for the one SQLite connection.
	StartBackground func(context.Context)
	SweepStatus     *selfhealthsnapshots.StatusStore
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
	healthDB, err := openHealthDatabase(cfg.DatabaseDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open dedicated health database: %w", err)
	}

	runner, err := orchestrator.NewSuiteOrchestrator(cfg.ScenariosRoot)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize orchestrator: %w", err)
	}

	executionRepo := execution.NewSuiteExecutionRepository(db)
	executionHistory := execution.NewExecutionHistoryService(executionRepo)
	executionPlanner := execution.NewExecutionPlanService(runner, executionRepo)
	scenarioRepo := scenarios.NewScenarioDirectoryRepository(db)
	scenarioLister := scenarios.NewVrooliScenarioLister()
	scenarioService := scenarios.NewScenarioDirectoryService(scenarioRepo, scenarioLister, cfg.ScenariosRoot)

	executionSvc := execution.NewSuiteExecutionService(runner, executionRepo)
	executionSvc.SetRetentionCollector(func(ctx context.Context, scenario string) {
		scenarioDir := filepath.Join(cfg.ScenariosRoot, scenario)
		if _, err := sharedruns.NewRetentionService(scenarioDir, sharedruns.DefaultRetentionPolicy()).WithDetailStore(executionRepo).Collect(ctx); err != nil {
			log.Printf("run retention failed for %s: %v", scenario, err)
		}
	})

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
	if err := playbooksclaims.Migrate(context.Background(), db); err != nil {
		return nil, fmt.Errorf("migrate playbooks claims storage: %w", err)
	}
	claimsService := playbooksclaims.NewService(playbooksclaims.Config{Repo: claimsRepo})
	runner.SetClaims(claimsService)

	// Construct the routed-test-db eligibility checker once at process startup
	// for the Connect EligibilityService handler.
	routingEligibility := eligibility.NewChecker()
	eligibilityService := appelig.NewService(routingEligibility, cfg.ScenariosRoot)

	// RunsService exposes the append-only run index AND the durable run
	// lifecycle (start/follow/wait/abort/status) over Connect-RPC, delegating
	// execution to the run manager.
	runsService := apprun.NewService(cfg.ScenariosRoot, runManager, executionPlanner, executionRepo)
	runsService.SetRetentionStore(executionRepo)

	// Provider-conformance ScenarioValidationService: Test Genie's own
	// descriptor-backed phase. The maturity spec comes from Test Genie's own
	// .vrooli/test-genie.json; a load failure disables the handler with a log
	// line rather than blocking startup.
	repoRoot := repoRootFromScenariosRoot(cfg.ScenariosRoot)
	conformanceSpec, specErr := assessment.LoadSpecFromScenario(filepath.Join(cfg.ScenariosRoot, "test-genie"))
	if specErr != nil {
		log.Printf("[test-genie] provider-conformance maturity spec unavailable: %v", specErr)
	}
	validationService := appvalidation.NewService(log.Default(), repoRoot, conformanceSpec)

	// Persisted self-health trend store. Its advisory sweeper is deliberately
	// constructed here but started by the serving transport after it owns a
	// listening socket; dependency construction must remain foreground-only.
	// The read path (GetSelfHealth trend delta/series) composes the repo; the
	// sweeper is the sole writer, digest-deduped + env-disableable.
	selfHealthSnapshots := selfhealthsnapshots.NewSqliteRepository(db)
	sweepStatus := &selfhealthsnapshots.StatusStore{}
	runsService.SetSnapshotReader(selfHealthSnapshots)
	selfHealthJob := func(ctx context.Context) {
		runSelfHealthSweeper(ctx, selfHealthSnapshots, newSelfHealthRollupBuilder(executionRepo, repoRootFromScenariosRoot(cfg.ScenariosRoot)), sweepStatus, poolSweepObserver(db.Primary(), sweepStatus))
	}

	// GetFleetHealth aggregates stored runs across the fleet (Stage 3 fleet
	// backbone, read side). The roster lists on-disk scenario directories so the
	// ledger can surface never-tested-in-window coverage gaps honestly.
	runsService.SetFleetSource(executionRepo, fleetRosterFromScenariosRoot(cfg.ScenariosRoot))

	// Priority-weighted background fleet scheduler (Stage 3 fleet backbone).
	// DEFAULT-OFF: it cycles real full suites across the fleet only when
	// explicitly enabled via TEST_GENIE_FLEET_SCHEDULER_ENABLED, bounded by
	// concurrency + per-cycle + wall-clock budgets, and respects the run
	// manager's one-in-progress-per-scenario invariant.
	fleetSchedulerJob := func(ctx context.Context) { runFleetScheduler(ctx, runManager) }

	// Create agent-manager service
	agentEnabled := os.Getenv("AGENT_MANAGER_ENABLED") != "false"
	profileKey := os.Getenv("AGENT_MANAGER_PROFILE_KEY")
	if profileKey == "" {
		profileKey = "test-genie/generation"
	}

	agentService := agentmanager.NewAgentService(agentmanager.Config{
		ProfileKey: profileKey,
		Timeout:    30 * time.Second,
		Enabled:    agentEnabled,
	})

	// Agent initialization is advisory and must share the serving lifetime with
	// every other background job; it cannot compete with listener startup.
	var agentInitializationJob func(context.Context)
	if agentEnabled {
		agentInitializationJob = func(parent context.Context) {
			ctx, cancel := context.WithTimeout(parent, 30*time.Second)
			defer cancel()
			if err := agentService.Initialize(ctx); err != nil {
				log.Printf("[agent-manager] Warning: failed to initialize profile: %v", err)
			}
		}
	}
	background := NewBackgroundCoordinator(selfHealthJob, fleetSchedulerJob, agentInitializationJob)

	remediationService := remediation.NewService(remediation.NewSQLiteRepository(db), nil)
	if err := remediation.Migrate(context.Background(), db); err != nil {
		return nil, fmt.Errorf("migrate remediation storage: %w", err)
	}
	remediationLauncher := remediation.NewAgentManagerAdapter(agentService)

	// Create requirements syncer
	reqSyncer := &RequirementsSyncerAdapter{
		svc: requirements.NewService(),
	}

	return &Bootstrapped{
		DB:                  db,
		HealthDB:            healthDB,
		ExecutionRepo:       executionRepo,
		ExecutionHistory:    executionHistory,
		ExecutionService:    executionSvc,
		ExecutionPlanner:    executionPlanner,
		RunManager:          runManager,
		ScenarioService:     scenarioService,
		PhaseCatalog:        runner,
		AgentService:        agentService,
		RemediationService:  remediationService,
		RemediationLauncher: remediationLauncher,
		RequirementsSyncer:  reqSyncer,
		PlaybooksClaims:     claimsService,
		EligibilityService:  eligibilityService,
		RunsService:         runsService,
		ValidationService:   validationService,
		StartBackground:     background.Start,
		SweepStatus:         sweepStatus,
	}, nil
}

func openHealthDatabase(dsn string) (dbexec.HealthProbe, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	return db, nil
}
