package phases

import (
	"context"
	"errors"
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
	"test-genie/internal/playbooksclaims"
	"test-genie/internal/selfidentity"
	"test-genie/internal/shared"

	"github.com/google/uuid"

	playbookregistry "test-genie/internal/playbooks/registry"

	"github.com/vrooli/api-core/apihttp"
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

// routedLeaseTTL bounds how long the scenario will hold the test pool
// without a HeartbeatTestPool. test-genie heartbeats at routedLeaseTTL/3.
const routedLeaseTTL = 90 * time.Second

// eligibilityChecker is the seam over eligibility.Checker so tests can swap
// in a stub without touching the auditor HTTP path.
//
// seam: routingChecker — production wires *eligibility.Checker; tests
// override the package var to inject canned PathDecisions / errors.
type eligibilityChecker interface {
	Check(ctx context.Context, scenario string, mapping workspace.Mapping) (eligibility.Eligibility, error)
	Invalidate(scenario string)
}

var routingChecker eligibilityChecker = eligibility.NewChecker(0)

// SetRoutingChecker overrides the package-level routing-eligibility checker.
// Called from app bootstrap so the HTTP Connect handler and the playbooks
// phase share a single Checker (and its per-process scan cache). The
// declaration-time default is kept so unit tests that don't go through
// bootstrap continue to work without setup.
func SetRoutingChecker(c eligibilityChecker) {
	if c == nil {
		return
	}
	routingChecker = c
}

// resolveScenarioRoutingClient returns a Connect-RPC client for the running
// scenario's RoutingService. Tests override it to return a stub.
var resolveScenarioRoutingClient = func(ctx context.Context, scenarioName string) (routing_v1connect.RoutingServiceClient, error) {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("resolve %s API URL: %w", scenarioName, err)
	}
	return routing_v1connect.NewRoutingServiceClient(http.DefaultClient, baseURL), nil
}

// probeRoutingServiceEnabled issues a cheap HTTP GET against the scenario's
// RoutingService route. A 404 means devrouting.Register did not mount the
// handler — almost always because the scenario is in production mode
// (projectmeta.IsDevelopment() returned false). Any other response (200,
// 405, 415, …) means the handler is mounted and the routed path is viable.
//
// seam: probeRoutingServiceEnabled — production hits the live scenario;
// tests override to short-circuit.
var probeRoutingServiceEnabled = func(ctx context.Context, scenarioName string) error {
	baseURL, err := discovery.ResolveScenarioURLDefault(ctx, scenarioName)
	if err != nil {
		return fmt.Errorf("resolve %s API URL: %w", scenarioName, err)
	}
	probeURL := strings.TrimRight(baseURL, "/") + routing_v1connect.RoutingServiceInstallTestPoolProcedure
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, probeURL, nil)
	if err != nil {
		return fmt.Errorf("build probe request: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("probe %s: %w", probeURL, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return errRoutingServiceDisabled
	}
	return nil
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

	if env.Claims == nil {
		return RunReport{
			Err:                   fmt.Errorf("playbooks claims service is not wired into the workspace environment"),
			FailureClassification: FailureClassSystem,
			Remediation:           "Bootstrap must call ScenarioWorkspace.SetClaims before running the playbooks phase.",
		}
	}

	leaseID := uuid.NewString()
	startedBy := resolveClaimActor()
	claim, claimErr := env.Claims.TryAcquire(ctx, playbooksclaims.AcquireInput{
		ScenarioName: env.ScenarioName,
		RunID:        leaseID,
		Mode:         playbooksclaims.ModeRouted, // refined below once eligibility decides
		StartedBy:    startedBy,
	})
	if claimErr != nil {
		var busy *playbooksclaims.ErrBusy
		if errors.As(claimErr, &busy) {
			shared.LogWarn(logWriter, "playbooks phase busy: scenario %q already held by run %s (started_by=%s, expires=%s)",
				env.ScenarioName, busy.Holder.RunID, busy.Holder.StartedBy, busy.Holder.ExpiresAt.Format(time.RFC3339))
			return RunReport{
				Err:                   claimErr,
				FailureClassification: FailureClassMisconfiguration,
				Remediation:           "Wait for the active run to finish, retry with --wait, or force-release the claim via test-genie claims release.",
			}
		}
		return RunReport{
			Err:                   fmt.Errorf("acquire playbooks claim: %w", claimErr),
			FailureClassification: FailureClassSystem,
		}
	}
	stopHeartbeat := env.Claims.StartHeartbeat(ctx, env.ScenarioName, claim.RunID)
	defer stopHeartbeat()
	defer func() {
		if err := env.Claims.Release(context.Background(), env.ScenarioName, claim.RunID); err != nil {
			shared.LogWarn(logWriter, "failed to release playbooks claim run=%s: %v", claim.RunID, err)
		}
	}()
	shared.LogStep(logWriter, "acquired playbooks claim run=%s (TTL %s, heartbeat %s)", claim.RunID, env.Claims.TTL(), playbooksclaims.HeartbeatInterval)

	playbooksCfg, err := config.Load(env.ScenarioDir)
	if err != nil {
		logPhaseStep(logWriter, "failed to load playbooks config: %v", err)
		playbooksCfg = config.Default()
	}

	// A per-run diagnostics preset (from `test-genie execute --diagnostics-preset`)
	// overrides whatever testing.json declared.
	if env.DiagnosticsPreset != "" {
		if diag, ok := config.DiagnosticsPreset(env.DiagnosticsPreset); ok {
			playbooksCfg.Diagnostics = diag
			logPhaseStep(logWriter, "diagnostics preset %q applied", env.DiagnosticsPreset)
		} else {
			shared.LogWarn(logWriter, "unknown diagnostics preset %q; using config defaults", env.DiagnosticsPreset)
		}
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

	if selfidentity.Is(env.ScenarioName) {
		return RunReport{
			Observations: []Observation{
				NewSkipObservation(fmt.Sprintf("playbooks skipped: restart-based isolation for %s would terminate the active self-test API process; the routed test-DB path is required for a playbooks self-test", env.ScenarioName)),
			},
			Remediation: "Migrate test-genie to the routed test-DB path (see docs/agent-system/routed-test-db.md), or execute playbooks against a different target scenario.",
		}
	}

	mapping := env.Mapping
	if strings.TrimSpace(mapping.PhysicalScenarioDir) == "" {
		mapping = workspace.Mapping{
			PhysicalScenarioDir: strings.TrimSpace(env.ScenarioDir),
			PhysicalAppRoot:     strings.TrimSpace(env.PhysicalAppRoot),
		}
	}

	// Routing eligibility — the routed path requires the scenario to have
	// been migrated to *database.RoutedDB and to expose RoutingService in
	// dev mode. The decision is consolidated into a single PathDecision so
	// the structured log block is the only operator-facing surface.
	elig, eligErr := routingChecker.Check(ctx, env.ScenarioName, mapping)
	defer routingChecker.Invalidate(env.ScenarioName)

	forcedFallback := os.Getenv("TEST_GENIE_FORCE_FALLBACK") == "1"
	decision := decidePlaybooksPath(elig, eligErr, forcedFallback)

	needs := resolveDBNeeds(ctx, env, logWriter)

	// Early routed preflight — client resolution + RoutingService probe. Both
	// only need the scenario name (not the isolation env), so running them
	// before isoManager.Prepare avoids burning a testcontainer when the
	// scenario can't accept the routed path (e.g. it's in production mode and
	// returns 404 on the RoutingService route). DSN extraction still has to
	// happen after Prepare since it reads the isolation env.
	var routingClient routing_v1connect.RoutingServiceClient
	if decision.IsRouted() {
		client, prefErr := runPlaybooksRoutedClientPreflight(ctx, env.ScenarioName)
		if prefErr != nil {
			decision = preflightFailureDecision(prefErr)
		} else {
			routingClient = client
		}
	}

	// If we're committed to the fallback path and the runtime manager isn't
	// wired, fail before spending time on isolation. The routed path doesn't
	// touch TargetRuntime, so this gate only applies to the fallback branch.
	if !decision.IsRouted() && env.TargetRuntime == nil {
		writePathDecisionBlock(logWriter, decision)
		return RunReport{
			Err:                   fmt.Errorf("target runtime manager is not configured"),
			FailureClassification: FailureClassSystem,
			Remediation:           "Run playbooks through test-genie execute so the target scenario lifecycle can be managed.",
		}
	}

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

	if decision.IsRouted() {
		dsn, dsnErr := extractTestDSN(isoResult.Env, needs.PrimaryDriver)
		if dsnErr != nil {
			decision = preflightFailureDecision(fmt.Errorf("%w: %v", errNoTestDSN, dsnErr))
			writePathDecisionBlock(logWriter, decision)
			return runPlaybooksFallback(ctx, env, logWriter, playbooksCfg, registry, needs, isoResult, retainIsolation)
		}
		decision.LeaseID = leaseID
		decision.DSNDriver = needs.PrimaryDriver
		writePathDecisionBlock(logWriter, decision)
		return runPlaybooksRouted(ctx, env, logWriter, playbooksCfg, registry, needs, isoResult, retainIsolation, routingClient, dsn, leaseID)
	}

	writePathDecisionBlock(logWriter, decision)
	return runPlaybooksFallback(ctx, env, logWriter, playbooksCfg, registry, needs, isoResult, retainIsolation)
}

// decidePlaybooksPath consolidates the routed-vs-fallback choice into a
// single PathDecision so all branches emit the same structured log block.
func decidePlaybooksPath(elig eligibility.Eligibility, eligErr error, forcedFallback bool) eligibility.PathDecision {
	if forcedFallback {
		return eligibility.PathDecision{
			Path:   eligibility.PathFallbackForcedEnv,
			Reason: "TEST_GENIE_FORCE_FALLBACK=1 forces the fallback path",
		}
	}
	if eligErr != nil {
		return eligibility.PathDecision{
			Path:   eligibility.PathFallbackAuditorUnreachable,
			Reason: fmt.Sprintf("scenario-auditor scan did not complete: %v", eligErr),
		}
	}
	if elig.RuleAssertion != nil {
		return eligibility.PathDecision{
			Path:          eligibility.PathFallbackRules,
			RuleAssertion: elig.RuleAssertion,
			Reason:        "scenario-auditor scan did not include one or more required routing rules",
		}
	}
	if !elig.Routed {
		return eligibility.PathDecision{
			Path:       eligibility.PathFallbackRules,
			Violations: elig.Violations,
			Reason:     "scenario-auditor flagged routing-rule violations",
		}
	}
	return eligibility.PathDecision{
		Path:   eligibility.PathRouted,
		Reason: "routed e2e path — no scenario restart",
	}
}

// preflightFailureDecision wraps a pre-flight error into a PathDecision with
// the correct PreflightFailure tag (or PathFallbackProductionMode when the
// scenario's RoutingService route is not mounted).
func preflightFailureDecision(err error) eligibility.PathDecision {
	switch {
	case errors.Is(err, errRoutingServiceDisabled):
		return eligibility.PathDecision{
			Path:   eligibility.PathFallbackProductionMode,
			Reason: err.Error(),
		}
	case errors.Is(err, errNoTestDSN):
		return eligibility.PathDecision{
			Path:             eligibility.PathFallbackPreflight,
			PreflightFailure: eligibility.PreflightFailureNoDSN,
			Reason:           err.Error(),
		}
	case errors.Is(err, errRoutingClientUnreachable):
		return eligibility.PathDecision{
			Path:             eligibility.PathFallbackPreflight,
			PreflightFailure: eligibility.PreflightFailureRoutingUnreachable,
			Reason:           err.Error(),
		}
	}
	return eligibility.PathDecision{
		Path:             eligibility.PathFallbackPreflight,
		PreflightFailure: eligibility.PreflightFailureNone,
		Reason:           err.Error(),
	}
}

// errNoTestDSN and errRoutingClientUnreachable are the two typed pre-flight
// errors. Pre-flight wraps the underlying reason with these sentinels so the
// PathDecision can classify the failure.
var (
	errNoTestDSN                = errors.New("routed pre-flight: no usable test DSN in isolation env")
	errRoutingClientUnreachable = errors.New("routed pre-flight: scenario routing client unreachable")
	errRoutingServiceDisabled   = errors.New("routed pre-flight: scenario RoutingService is not mounted (production mode?)")
)

// runPlaybooksRoutedClientPreflight verifies the env-independent routed
// pre-flight checks: the scenario's RoutingService client is resolvable and
// the RoutingService route is mounted (i.e. the scenario is not in
// production mode). Runs before isolation Prepare so a 404 doesn't waste a
// testcontainer boot.
//
// DSN extraction is intentionally not part of this preflight — it needs the
// isolation env and so runs after Prepare in runPlaybooksPhase.
func runPlaybooksRoutedClientPreflight(ctx context.Context, scenarioName string) (routing_v1connect.RoutingServiceClient, error) {
	client, err := resolveScenarioRoutingClient(ctx, scenarioName)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", errRoutingClientUnreachable, err)
	}
	if err := probeRoutingServiceEnabled(ctx, scenarioName); err != nil {
		if errors.Is(err, errRoutingServiceDisabled) {
			return nil, err
		}
		return nil, fmt.Errorf("%w: %v", errRoutingClientUnreachable, err)
	}
	return client, nil
}

// resolveClaimActor reports the actor that started a playbooks run. Best
// effort — falls back to "test-genie" if no env hint is set.
func resolveClaimActor() string {
	for _, key := range []string{"TEST_GENIE_STARTED_BY", "USER", "LOGNAME"} {
		if v := strings.TrimSpace(os.Getenv(key)); v != "" {
			return v
		}
	}
	return "test-genie"
}

// writePathDecisionBlock emits the structured routed-vs-fallback decision
// summary at the top of the phase log. Every routed-or-fallback choice goes
// through this — no silent fall-throughs.
//
// See docs/agent-system/routed-test-db.md ("Decision log block") for the
// canonical format.
func writePathDecisionBlock(logWriter io.Writer, decision eligibility.PathDecision) {
	switch decision.Path {
	case eligibility.PathRouted:
		shared.LogStep(logWriter, "✓ Routed e2e path — lease=%s driver=%s reason=%s", decision.LeaseID, decision.DSNDriver, decision.Reason)
		return
	default:
		shared.LogWarn(logWriter, "⚠ Playbooks path=%s — fallback used. Reason: %s", decision.Path, decision.Reason)
	}

	switch decision.Path {
	case eligibility.PathFallbackRules:
		if decision.RuleAssertion != nil && len(decision.RuleAssertion.MissingRules) > 0 {
			shared.LogWarn(logWriter, "  Missing routing rules in scan (rule unregistered or disabled): %s", strings.Join(decision.RuleAssertion.MissingRules, ", "))
			shared.LogWarn(logWriter, "  Remediation: enable the rules in scenario-auditor; see docs/agent-system/routed-test-db.md")
			return
		}
		if len(decision.Violations) == 0 {
			shared.LogWarn(logWriter, "  (no specific excerpts; run scenario-auditor standards scan for details)")
			return
		}
		for _, v := range decision.Violations {
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
	case eligibility.PathFallbackPreflight:
		shared.LogWarn(logWriter, "  Pre-flight check: %s", decision.PreflightFailure)
	case eligibility.PathFallbackAuditorUnreachable:
		shared.LogWarn(logWriter, "  Remediation: run `scenario-auditor system status` to confirm the auditor is reachable.")
	case eligibility.PathFallbackForcedEnv:
		shared.LogWarn(logWriter, "  Unset TEST_GENIE_FORCE_FALLBACK to allow the routed path.")
	case eligibility.PathFallbackProductionMode:
		shared.LogWarn(logWriter, "  The target scenario is in production mode; set VROOLI_TEST_MODE_FORCE_ENABLE=1 or switch .vrooli/service.json mode to \"development\".")
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
	leaseID string,
) (report RunReport) {
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

	installReq := &routingv1.InstallTestPoolRequest{
		Dsn:        dsn,
		LeaseId:    leaseID,
		LeaseTtlMs: int64(routedLeaseTTL / time.Millisecond),
	}
	if _, err := client.InstallTestPool(ctx, connect.NewRequest(installReq)); err != nil {
		_ = isoResult.Cleanup(context.Background())
		return RunReport{
			Err:                   fmt.Errorf("install test pool on %s: %w", env.ScenarioName, err),
			FailureClassification: FailureClassSystem,
			Remediation:           "Check the scenario's logs for routing/install errors; confirm projectmeta.IsDevelopment() returns true and that devrouting.Register is called.",
		}
	}
	shared.LogStep(logWriter, "installed routed test pool on %s (lease TTL %s)", env.ScenarioName, routedLeaseTTL)

	// Heartbeat the lease at 1/3 of the TTL so a stuck orchestrator never
	// silently passes the expiry threshold.
	hbCtx, hbCancel := context.WithCancel(ctx)
	hbDone := make(chan struct{})
	go func() {
		defer close(hbDone)
		ticker := time.NewTicker(routedLeaseTTL / 3)
		defer ticker.Stop()
		for {
			select {
			case <-hbCtx.Done():
				return
			case <-ticker.C:
				if _, err := client.HeartbeatTestPool(hbCtx, connect.NewRequest(&routingv1.HeartbeatTestPoolRequest{LeaseId: leaseID})); err != nil {
					shared.LogWarn(logWriter, "routed lease heartbeat failed: %v", err)
				}
			}
		}
	}()

	defer func() {
		hbCancel()
		<-hbDone

		clearResp, clearErr := client.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID}))
		if clearErr != nil {
			shared.LogWarn(logWriter, "failed to clear routed test pool: %v", clearErr)
		} else if stats := clearResp.Msg.GetStats(); stats != nil {
			shared.LogInfo(logWriter, "routed lease stats: test_pool_requests=%d primary_during_test_mode_requests=%d", stats.GetTestPoolRequests(), stats.GetPrimaryDuringTestModeRequests())
			applyLeaseStatsResult(&report, stats.GetTestPoolRequests(), stats.GetPrimaryDuringTestModeRequests(), playbooksCfg)
		}
		if err := isoResult.Cleanup(context.Background()); err != nil {
			shared.LogWarn(logWriter, "failed to clean up isolation resources: %v", err)
		}
	}()

	// Attach the test-mode header to every browser request BAS makes during
	// playbook execution. BAS forwards this map as Playwright's
	// extraHTTPHeaders on the browser context, so each UI→API call carries
	// it and the scenario's TestModeMiddleware routes the request to the
	// installed test pool.
	extraHeaders := map[string]string{
		apihttp.TestModeHeader: apihttp.TestModeValue,
	}

	report = runLoadedPlaybooksPhase(ctx, env, logWriter, playbooksCfg, registry, isoResult.Env, extraHeaders)

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

// applyLeaseStatsResult promotes the routed-lease post-run stats into hard
// failures unless the scenario has opted out via
// playbooks.allow_empty_test_pool in .vrooli/testing.json.
//
//   - primary_during_test_mode_requests > 0 ("bypass"): a request carried
//     X-Vrooli-Test-Mode:1 but was served from the primary pool. This is
//     always a real defect — some code path holds a raw *sql.DB instead of
//     going through RoutedDB. Hard failure unless opted out.
//   - test_pool_requests == 0 ("empty pool"): no request ever exercised the
//     test pool during the run. Either the header didn't round-trip, the
//     playbooks never touched the API, or the playbooks are pure UI smoke.
//     Hard failure unless opted out.
//
// If the report already carries an error from the playbooks execution, the
// stats issues are appended as warning observations instead — the playbook
// failure is the more useful signal and shouldn't be overwritten.
func applyLeaseStatsResult(report *RunReport, testPoolRequests, primaryDuringTestMode int64, playbooksCfg *config.Config) {
	allowEmpty := playbooksCfg != nil && playbooksCfg.AllowEmptyTestPool

	if primaryDuringTestMode > 0 {
		msg := fmt.Sprintf("routed bypass detected: %d request(s) carried X-Vrooli-Test-Mode:1 but were served from the primary pool — some code path holds a raw *sql.DB", primaryDuringTestMode)
		switch {
		case allowEmpty:
			report.Observations = append(report.Observations, NewWarningObservation(msg))
		case report.Err != nil:
			report.Observations = append(report.Observations, NewWarningObservation(msg))
		default:
			report.Err = errors.New(msg)
			report.FailureClassification = FailureClassMisconfiguration
			report.Remediation = "Migrate the bypassing call site to *database.RoutedDB (run scenario-auditor to find raw *sql.DB captures), or set playbooks.allow_empty_test_pool=true in .vrooli/testing.json to acknowledge this advisory."
		}
	}

	if testPoolRequests == 0 {
		msg := "routed e2e ran without exercising the test pool: 0 test-mode requests reached RoutedDB — verify the X-Vrooli-Test-Mode header reaches the API or that the playbooks touch the API at all"
		switch {
		case allowEmpty:
			report.Observations = append(report.Observations, NewWarningObservation(msg))
		case report.Err != nil:
			report.Observations = append(report.Observations, NewWarningObservation(msg))
		default:
			report.Err = errors.New(msg)
			report.FailureClassification = FailureClassMisconfiguration
			report.Remediation = "Add at least one UI→API interaction to the playbooks so the test pool is exercised, or set playbooks.allow_empty_test_pool=true in .vrooli/testing.json if the playbooks are read-only by design."
		}
	}
}

// extractTestDSN pulls the test database DSN out of the isolation env map.
// primaryDriver must name the scenario's primary driver ("postgres" or
// "sqlite") so the chosen DSN aligns with what the scenario's RoutedDB was
// opened with — installing a postgres URL into a sqlite-driven pool hangs
// PingContext indefinitely. An empty primaryDriver is treated as
// disqualifying (db-detect couldn't pick a winner — better to fall back
// than to guess).
func extractTestDSN(env map[string]string, primaryDriver string) (string, error) {
	switch primaryDriver {
	case "postgres":
		if v := strings.TrimSpace(env["POSTGRES_URL"]); v != "" {
			return v, nil
		}
		if v := strings.TrimSpace(env["DATABASE_URL"]); v != "" {
			return v, nil
		}
		return "", fmt.Errorf("primary driver is postgres but isolation env has no POSTGRES_URL or DATABASE_URL")
	case "sqlite":
		if v := strings.TrimSpace(env["SQLITE_PATH"]); v != "" {
			return v, nil
		}
		if v := strings.TrimSpace(env["SQLITE_DB"]); v != "" {
			return v, nil
		}
		return "", fmt.Errorf("primary driver is sqlite but isolation env has no SQLITE_PATH or SQLITE_DB")
	case "":
		return "", fmt.Errorf("db-detect did not pick a primary driver — skipping routed path")
	default:
		return "", fmt.Errorf("unsupported primary driver %q", primaryDriver)
	}
}

func runLoadedPlaybooksPhase(
	ctx context.Context,
	env workspace.Environment,
	logWriter io.Writer,
	playbooksCfg *config.Config,
	registry playbooks.Registry,
	seedEnv map[string]string,
	extraHeaders map[string]string,
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
			if len(extraHeaders) > 0 {
				opts = append(opts, playbooks.WithExtraHeaders(extraHeaders))
			}
			runner := playbooks.New(playbooks.Config{
				ScenarioDir:  env.ScenarioDir,
				ScenarioName: env.ScenarioName,
				RunID:        env.RunID,
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
