package lifecycle

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/scenario"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

func (r *Runner) WaitForHealth(item scenario.Scenario, env map[string]string) (string, error) {
	return r.waitForHealth(context.Background(), item, env)
}

func (r *Runner) waitForHealth(ctx context.Context, item scenario.Scenario, env map[string]string) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	health := item.Manifest.HealthConfig()
	if err := r.awaitScenarioReadiness(ctx, item, env); err != nil {
		return "unhealthy", err
	}
	if health == nil || len(health.Checks) == 0 {
		r.logInfo("Scenario has no health checks; treating as running", logx.AttrScenario, item.Slug, logx.AttrStatus, "running")
		return "running", nil
	}
	ports := healthPortsFromEnv(item.Manifest, env)
	r.logDebug("Waiting for scenario health", logx.AttrScenario, item.Slug, logx.AttrChecks, len(health.Checks), logx.AttrPorts, ports)

	var lastStatus string
	err := AwaitContext(ctx, r.awaitClock(), healthAwaitPolicy(health.Timeout, health.Interval), func() (bool, error) {
		lastStatus = scenario.EvaluateHealth(health, ports)
		return lastStatus == "healthy", nil
	})
	if err == nil {
		r.logInfo("Scenario reported healthy", logx.AttrScenario, item.Slug, logx.AttrStatus, lastStatus)
		return lastStatus, nil
	}
	// Deadline expiry. Degraded-after-timeout is a success path: non-critical
	// checks failing must not fail the start.
	if lastStatus == "degraded" {
		r.logWarn("Scenario health checks degraded after timeout", logx.AttrScenario, item.Slug, logx.AttrStatus, lastStatus)
		return lastStatus, nil
	}
	r.logWarn("Scenario health checks failed before timeout", logx.AttrScenario, item.Slug, logx.AttrStatus, lastStatus)
	return lastStatus, fmt.Errorf("scenario %q failed health checks", item.Slug)
}

// awaitScenarioReadiness evaluates every launched component before the
// scenario-level health contract. StartupGracePeriod is a failure ceiling: a
// component that becomes ready immediately proceeds immediately, while a
// failing probe is allowed to retry until the ceiling.
func (r *Runner) awaitScenarioReadiness(ctx context.Context, item scenario.Scenario, env map[string]string) error {
	if len(item.Manifest.Components) == 0 {
		return nil
	}
	policy := scenarioReadinessPolicy
	if health := item.Manifest.HealthConfig(); health != nil && health.StartupGracePeriod > 0 {
		policy.Timeout = time.Duration(health.StartupGracePeriod) * time.Millisecond
	}
	names := make([]string, 0, len(item.Manifest.Components))
	for name := range item.Manifest.Components {
		names = append(names, name)
	}
	sort.Strings(names)
	lastErrors := make(map[string]error, len(names))
	err := AwaitContext(ctx, r.awaitClock(), policy, func() (bool, error) {
		ready := true
		for _, name := range names {
			component := item.Manifest.Components[name]
			if ok, _, conditionErr := stepConditionsMet(item, component.Run.Condition, env); conditionErr != nil {
				return false, conditionErr
			} else if !ok {
				continue
			}
			probeErr := r.checkComponentReadinessNamed(recordSlug(item), item.Manifest, name, component, env)
			if probeErr != nil {
				lastErrors[name] = probeErr
				ready = false
			}
		}
		return ready, nil
	})
	if err != nil {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		for _, name := range names {
			if probeErr := lastErrors[name]; probeErr != nil {
				return fmt.Errorf("scenario %q component %q was not ready before %s: %w", item.Slug, name, policy.Timeout, probeErr)
			}
		}
		return fmt.Errorf("scenario %q components were not ready before %s", item.Slug, policy.Timeout)
	}
	return nil
}

// isRegistryRuntimeHealthy decides whether an authoritative registry runtime
// is data-plane ready, using the registry-bound ports as the probe target.
// Lease freshness and reconciliation already proved the scenario is *running*
// (that authority lives in the registry); this only evaluates whether the
// manifest's health checks pass against the bound ports.
func (r *Runner) isRegistryRuntimeHealthy(item scenario.Scenario, view registryRuntimeView) bool {
	if !view.Authoritative {
		return false
	}
	deps := r.runtimeDeps()
	// Orphan-squat guard — align lifecycle health with test-genie's
	// targetruntime.resolveURLs, which requires the recorded owner PID to be alive
	// before trusting a port probe. Reconciliation keeps an instance authoritative
	// for the whole heartbeat TTL even if its owner died mid-window, so a foreign
	// process squatting a bound port could answer the manifest probe and read as
	// "healthy" while the real owner is gone. When the recorded owner PID is known
	// and not running, the data plane belongs to an orphan, not this instance —
	// report unhealthy no matter who answers the port. Unknown owner PID (nil) or
	// an unavailable liveness probe is NOT condemned (positive bad evidence only).
	if pid := view.Instance.OwnerPID; pid != nil && view.Instance.OwnerKind != scenarioruntime.OwnerKindSupervisor &&
		deps.isPIDRunning != nil && !deps.isPIDRunning(*pid) {
		r.logWarn("Registry runtime owner pid is not alive; a bound port answered by another process would be an orphan squat",
			logx.AttrScenario, item.Slug, "owner_pid", *pid)
		return false
	}
	health := item.Manifest.HealthConfig()
	if health == nil || len(health.Checks) == 0 {
		return true
	}
	// Retry transient probe failures before condemning a running dependency:
	// a single dropped/slow probe must not trigger a stop+rebuild+restart of a
	// process that is actually fine. Bounded to a few quick attempts
	// (registryHealthRetryPolicy: 3 over ~3s) since the registry already
	// proved the instance is *running* — we are only confirming data-plane
	// readiness. We fail toward reuse only after the probe is persistently bad.
	healthy := false
	_ = Await(r.awaitClock(), registryHealthRetryPolicy, func() (bool, error) {
		healthy = allHealthChecksPass(health.Checks, view.Ports)
		return healthy, nil
	})
	return healthy
}

func allHealthChecksPass(checks []scenario.HealthCheck, ports map[string]int) bool {
	for _, check := range checks {
		if err := scenario.PerformHealthCheck(check, ports); err != nil {
			return false
		}
	}
	return true
}
