package phases

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"

	"test-genie/internal/eligibility"
	"test-genie/internal/orchestrator/workspace"
	"test-genie/internal/playbooks"
	"test-genie/internal/playbooks/config"
	"test-genie/internal/playbooks/isolation"
	"test-genie/internal/shared"

	playbookregistry "test-genie/internal/playbooks/registry"

	"github.com/vrooli/api-core/discovery"

	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	"github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
)

// isolationProvider lets tests stub isolation without requiring Docker.
type isolationProvider interface {
	Prepare(ctx context.Context) (*isolation.Result, error)
}

// isolationManagerFactory creates the default isolation manager.
var isolationManagerFactory = func(cfg isolation.Config) isolationProvider {
	return isolation.NewManager(cfg)
}

// routingChecker is the eligibility decider; tests override it.
var routingChecker = eligibility.NewChecker(0)

// resolveScenarioRoutingClient returns a Connect-RPC client for the running
// scenario's RoutingService. Tests override it to return a stub.
var resolveScenarioRoutingClient = func(ctx context.Context, scenarioName string) (routing_v1connect.RoutingServiceClient, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("resolve %s API URL: %w", scenarioName, err)
	}
	return routing_v1connect.NewRoutingServiceClient(http.DefaultClient, baseURL), nil
}

type staticRegistryLoader struct {
	registry playbooks.Registry
}

func (s staticRegistryLoader) Load() (playbooks.Registry, error) {
	return s.registry, nil
}

// runPlaybooksPhase executes BAS playbook workflows using the playbooks
// package. It branches at the top on routing eligibility: scenarios that
// qualify take the in-place "routed" path (no scenario restart; runtime
// test-pool install via RoutingService); scenarios that don't qualify take
// the original "fallback" path with a leading violations block in the log.
func runPlaybooksPhase(ctx context.Context, env workspace.Environment, logWriter io.Writer) RunReport {
	if err := ctx.Err(); err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassSystem,
		}
	}

	playbooksCfg, err := config.Load(env.ScenarioDir)
	if err != nil {
		logPhaseStep(logWriter, "failed to load playbooks config: %v", err)
		playbooksCfg = config.Default()
	}

	if playbooksCfg != nil && !playbooksCfg.Enabled {
		shared.LogWarn(logWriter, "playbooks phase disabled via .vrooli/testing.json (playbooks.enabled=false)")
		return RunReport{
			Observations: []Observation{
				NewSkipObservation("playbooks phase disabled via .vrooli/testing.json"),
			},
		}
	}

	retainIsolation := isolation.ShouldRetainFromEnv()

	if os.Getenv("TEST_GENIE_SKIP_PLAYBOOKS") == "1" {
		shared.LogWarn(logWriter, "playbooks phase disabled via TEST_GENIE_SKIP_PLAYBOOKS (skipping isolation/restart)")
		return RunReport{
			Observations: []Observation{
				NewSkipObservation("playbooks phase disabled via TEST_GENIE_SKIP_PLAYBOOKS"),
			},
		}
	}

	registry, err := playbookregistry.NewLoader(env.ScenarioDir).Load()
	if err != nil {
		return RunReport{
			Err:                   err,
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Regenerate bas/registry.json via playbook builder.",
		}
	}

	if len(registry.Playbooks) == 0 {
		shared.LogInfo(logWriter, "playbooks registry contains no workflows; skipping isolation/restart")
		return runLoadedPlaybooksPhase(ctx, env, logWriter, playbooksCfg, registry, nil, nil)
	}

	if registry.UsesObserverMode() {
		shared.LogInfo(logWriter, "playbooks registry execution_mode=%s; skipping isolation/restart", registry.NormalizedExecutionMode())
		return runLoadedPlaybooksPhase(ctx, env, logWriter, playbooksCfg, registry, nil, nil)
	}

	if env.ScenarioName == "test-genie" {
		return RunReport{
			Err:                   fmt.Errorf("playbooks for %s require isolation/restart, which would terminate the active test-genie API process", env.ScenarioName),
			FailureClassification: FailureClassMisconfiguration,
			Remediation:           "Set bas/registry.json metadata.execution_mode to \"observer\" for self-tests, or execute playbooks against a different target scenario.",
		}
	}

	mapping := env.Mapping
	if strings.TrimSpace(mapping.PhysicalScenarioDir) == "" {
		mapping = workspace.Mapping{
			PhysicalScenarioDir: strings.TrimSpace(env.ScenarioDir),
			PhysicalAppRoot:     strings.TrimSpace(env.PhysicalAppRoot),
		}
	}

	// Routing eligibility check — the routed path requires the scenario to
	// have been migrated to *database.RoutedDB and to expose RoutingService
	// in dev mode. If anything is off, we drop straight into the fallback
	// path with a violations block.
	elig, eligErr := routingChecker.Check(ctx, env.ScenarioName, mapping)
	useRouted := eligErr == nil && elig.Routed && os.Getenv("TEST_GENIE_FORCE_FALLBACK") != "1"

	if eligErr != nil {
		shared.LogWarn(logWriter, "routed-path eligibility check failed: %v (falling back to restart-based path)", eligErr)
	} else if !elig.Routed {
		writeFallbackViolationsBlock(logWriter, elig.Violations)
	}

	needs := resolveDBNeeds(ctx, env, logWriter)
	isoManager := isolationManagerFactory(isolation.Config{
		ScenarioName:    env.ScenarioName,
		RequirePostgres: needs.RequirePostgres,
		RequireRedis:    needs.RequireRedis,
		RequireSQLite:   needs.RequireSQLite,
		SQLiteEnvVars:   needs.SQLiteEnvVars,
		Retain:          retainIsolation,
		LogWriter:       logWriter,
		Timeout:         2 * time.Minute,
	})

	isoResult, err := isoManager.Prepare(ctx)
	if err != nil {
		return RunReport{
			Err:                   fmt.Errorf("failed to prepare playbooks isolation: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure Docker is available for testcontainers or provide access to start the temporary database and cache resources required by the target scenario.",
		}
	}

	if useRouted {
		dsn, dsnErr := extractTestDSN(isoResult.Env)
		var client routing_v1connect.RoutingServiceClient
		var clientErr error
		if dsnErr == nil {
			client, clientErr = resolveScenarioRoutingClient(ctx, env.ScenarioName)
		}
		if dsnErr != nil || clientErr != nil {
			reason := dsnErr
			if reason == nil {
				reason = clientErr
			}
			shared.LogWarn(logWriter, "routed-path pre-flight failed (%v); using fallback path", reason)
			return runPlaybooksFallback(ctx, env, logWriter, playbooksCfg, registry, needs, isoResult, retainIsolation)
		}
		return runPlaybooksRouted(ctx, env, logWriter, playbooksCfg, registry, needs, isoResult, retainIsolation, client, dsn)
	}
	return runPlaybooksFallback(ctx, env, logWriter, playbooksCfg, registry, needs, isoResult, retainIsolation)
}

// writeFallbackViolationsBlock prepends a structured violations summary to
// the phase log so operators understand why the fallback path was taken.
func writeFallbackViolationsBlock(logWriter io.Writer, violations []eligibility.ViolationExcerpt) {
	shared.LogWarn(logWriter, "⚠ Routed e2e path unavailable — fallback used. Violations:")
	if len(violations) == 0 {
		shared.LogWarn(logWriter, "  (no specific excerpts; run scenario-auditor standards scan for details)")
		return
	}
	for _, v := range violations {
		loc := v.FilePath
		if v.LineNumber > 0 {
			loc = fmt.Sprintf("%s:%d", v.FilePath, v.LineNumber)
		}
		if loc == "" {
			loc = "(see scenario-auditor)"
		}
		title := v.Title
		if title == "" {
			title = v.RuleID
		}
		shared.LogWarn(logWriter, "  [%s] %s %s — %s", strings.ToUpper(v.Severity), v.RuleID, loc, title)
	}
}

// runPlaybooksRouted runs the in-place routed path: provision the test DB,
// install it as a runtime test pool on the live scenario via RoutingService,
// run seeds, run the playbooks with the X-Vrooli-Test-Mode header injected
// via initial_params, then clear the test pool in defer. No scenario
// restart.
func runPlaybooksRouted(
	ctx context.Context,
	env workspace.Environment,
	logWriter io.Writer,
	playbooksCfg *config.Config,
	registry playbooks.Registry,
	needs resourceNeeds,
	isoResult *isolation.Result,
	retainIsolation bool,
	client routing_v1connect.RoutingServiceClient,
	dsn string,
) RunReport {
	shared.LogStep(logWriter, "playbooks routed path (run=%s) — no scenario restart", isoResult.RunID)
	for _, res := range isoResult.Resources {
		shared.LogInfo(logWriter, "  %s -> %s", res.Name, res.Endpoint)
	}

	// Apply migrations to the test DB before installing it as the test pool.
	if err := applyPlaybooksMigrations(ctx, env, needs, logWriter); err != nil {
		_ = isoResult.Cleanup(context.Background())
		return RunReport{
			Err:                   fmt.Errorf("failed to apply playbooks migrations: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure psql is available and migrations under bas/seeds/migrations/ are valid.",
		}
	}

	if _, err := client.InstallTestPool(ctx, connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: dsn})); err != nil {
		_ = isoResult.Cleanup(context.Background())
		return RunReport{
			Err:                   fmt.Errorf("install test pool on %s: %w", env.ScenarioName, err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Check the scenario's logs for routing/install errors; confirm projectmeta.IsDevelopment() returns true and that devrouting.Register is called.",
		}
	}
	shared.LogStep(logWriter, "installed routed test pool on %s", env.ScenarioName)

	defer func() {
		_, clearErr := client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{}))
		if clearErr != nil {
			shared.LogWarn(logWriter, "failed to clear routed test pool: %v", clearErr)
		}
		if err := isoResult.Cleanup(context.Background()); err != nil {
			shared.LogWarn(logWriter, "failed to clean up isolation resources: %v", err)
		}
	}()

	extraInitial := map[string]any{
		"test_mode_header": "X-Vrooli-Test-Mode",
		"test_mode_value":  "1",
	}

	report := runLoadedPlaybooksPhase(ctx, env, logWriter, playbooksCfg, registry, isoResult.Env, extraInitial)

	if retainIsolation && len(isoResult.Resources) > 0 {
		for _, res := range isoResult.Resources {
			for _, cmd := range res.InspectCommands {
				report.Observations = append(report.Observations, NewInfoObservation(fmt.Sprintf("retain %s: %s", res.Name, cmd)))
			}
		}
	}

	return report
}

// runPlaybooksFallback runs the historical restart-based path. Preserved as
// a first-class path — not deprecated.
func runPlaybooksFallback(
	ctx context.Context,
	env workspace.Environment,
	logWriter io.Writer,
	playbooksCfg *config.Config,
	registry playbooks.Registry,
	needs resourceNeeds,
	isoResult *isolation.Result,
	retainIsolation bool,
) RunReport {
	restoreEnv := isolation.ApplyEnv(isoResult.Env)
	envApplied := true
	shared.LogStep(logWriter, "playbooks isolation ready (run=%s)", isoResult.RunID)
	for _, res := range isoResult.Resources {
		shared.LogInfo(logWriter, "  %s -> %s", res.Name, res.Endpoint)
		if retainIsolation && len(res.InspectCommands) > 0 {
			for _, cmd := range res.InspectCommands {
				shared.LogInfo(logWriter, "    inspect: %s", cmd)
			}
		}
	}

	if err := applyPlaybooksMigrations(ctx, env, needs, logWriter); err != nil {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		_ = isoResult.Cleanup(context.Background())
		return RunReport{
			Err:                   fmt.Errorf("failed to apply playbooks migrations: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Ensure psql is available and migrations under bas/seeds/migrations/ are valid.",
		}
	}

	if env.TargetRuntime == nil {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		_ = isoResult.Cleanup(context.Background())
		return RunReport{
			Err:                   fmt.Errorf("target runtime manager is not configured"),
			FailureClassification: FailureClassSystem,
			Remediation:           "Run playbooks through test-genie execute so the target scenario lifecycle can be managed.",
		}
	}

	if err := env.TargetRuntime.RestartWithEnv(ctx, isoResult.Env, logWriter); err != nil {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		_ = isoResult.Cleanup(context.Background())
		return RunReport{
			Err:                   fmt.Errorf("failed to restart scenario with playbooks isolation: %w", err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Check lifecycle logs for restart failures and ensure the scenario can connect to the temporary resources provisioned for the playbooks run.",
		}
	}

	if envApplied {
		restoreEnv()
		envApplied = false
	}

	defer func() {
		if envApplied {
			restoreEnv()
			envApplied = false
		}
		if err := env.TargetRuntime.Restore(context.Background(), logWriter); err != nil {
			shared.LogWarn(logWriter, "failed to restart scenario back to normal resources: %v", err)
		}
		if err := isoResult.Cleanup(context.Background()); err != nil {
			shared.LogWarn(logWriter, "failed to clean up playbooks isolation resources: %v", err)
		}
	}()

	report := runLoadedPlaybooksPhase(ctx, env, logWriter, playbooksCfg, registry, isoResult.Env, nil)

	if retainIsolation && len(isoResult.Resources) > 0 {
		for _, res := range isoResult.Resources {
			for _, cmd := range res.InspectCommands {
				report.Observations = append(report.Observations, NewInfoObservation(fmt.Sprintf("retain %s: %s", res.Name, cmd)))
			}
		}
	}

	return report
}

// extractTestDSN pulls the test database DSN out of the isolation env map.
// Postgres URL wins; SQLite path is the fallback. Returns an error if
// neither is present.
func extractTestDSN(env map[string]string) (string, error) {
	if v := strings.TrimSpace(env["POSTGRES_URL"]); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(env["DATABASE_URL"]); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(env["SQLITE_PATH"]); v != "" {
		return v, nil
	}
	if v := strings.TrimSpace(env["SQLITE_DB"]); v != "" {
		return v, nil
	}
	return "", fmt.Errorf("no test DSN found in isolation env (looked for POSTGRES_URL, DATABASE_URL, SQLITE_PATH, SQLITE_DB)")
}

func runLoadedPlaybooksPhase(
	ctx context.Context,
	env workspace.Environment,
	logWriter io.Writer,
	playbooksCfg *config.Config,
	registry playbooks.Registry,
	seedEnv map[string]string,
	extraInitialParams map[string]any,
) RunReport {
	return RunPhase(ctx, logWriter, "playbooks",
		func() (*playbooks.RunResult, error) {
			opts := []playbooks.Option{
				playbooks.WithLogger(logWriter),
				playbooks.WithPlaybooksConfig(playbooksCfg),
				playbooks.WithRegistryLoader(staticRegistryLoader{registry: registry}),
				playbooks.WithSeedEnv(seedEnv),
				playbooks.WithPortResolver(func(ctx context.Context, scenarioName, portName string) (string, error) {
					return ResolveScenarioPort(ctx, logWriter, scenarioName, portName)
				}),
				playbooks.WithScenarioStarter(func(ctx context.Context, scenario string) error {
					shared.LogStep(logWriter, "ensuring scenario %s is running", scenario)
					return phaseCommandExecutor(ctx, "", logWriter, "vrooli", "scenario", "start", scenario, "--clean-stale")
				}),
			}
			if len(extraInitialParams) > 0 {
				opts = append(opts, playbooks.WithExtraInitialParams(extraInitialParams))
			}
			runner := playbooks.New(playbooks.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				TestDir:      env.TestDir,
				AppRoot:      env.AppRoot,
			}, opts...)
			return runner.Run(ctx), nil
		},
		func(r *playbooks.RunResult) PhaseResult[playbooks.Observation] {
			return ExtractSimple(
				r.Success,
				r.Error,
				r.FailureClass,
				r.Remediation,
				r.Observations,
			)
		},
	)
}
