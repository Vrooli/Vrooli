package lifecycle

import (
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/scenario"
)

func (r *Runner) WaitForHealth(item scenario.Scenario, env map[string]string) (string, error) {
	deps := r.runtimeDeps()
	health := item.Manifest.HealthConfig()
	if health == nil || len(health.Checks) == 0 {
		r.logInfo("Scenario has no health checks; treating as running", logx.AttrScenario, item.Slug, logx.AttrStatus, "running")
		return "running", nil
	}
	ports := healthPortsFromEnv(item.Manifest, env)
	r.logDebug("Waiting for scenario health", logx.AttrScenario, item.Slug, logx.AttrChecks, len(health.Checks), logx.AttrPorts, ports)

	if health.StartupGracePeriod > 0 {
		deps.sleep(time.Duration(health.StartupGracePeriod) * time.Millisecond)
	}

	deadline := deps.now().Add(30 * time.Second)
	if health.Timeout > 0 {
		deadline = deps.now().Add(time.Duration(health.Timeout) * time.Millisecond)
	}

	interval := 500 * time.Millisecond
	if health.Interval > 0 {
		interval = time.Duration(health.Interval) * time.Millisecond
		if interval > 2*time.Second {
			interval = 2 * time.Second
		}
	}

	var lastStatus string
	for {
		lastStatus = scenario.EvaluateHealth(health, ports)
		if lastStatus == "healthy" {
			r.logInfo("Scenario reported healthy", logx.AttrScenario, item.Slug, logx.AttrStatus, lastStatus)
			return lastStatus, nil
		}
		if deps.now().After(deadline) {
			if lastStatus == "degraded" {
				r.logWarn("Scenario health checks degraded after timeout", logx.AttrScenario, item.Slug, logx.AttrStatus, lastStatus)
				return lastStatus, nil
			}
			r.logWarn("Scenario health checks failed before timeout", logx.AttrScenario, item.Slug, logx.AttrStatus, lastStatus)
			return lastStatus, fmt.Errorf("scenario %q failed health checks", item.Slug)
		}
		deps.sleep(interval)
	}
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
	health := item.Manifest.HealthConfig()
	if health == nil || len(health.Checks) == 0 {
		return true
	}
	// Retry transient probe failures before condemning a running dependency:
	// a single dropped/slow probe must not trigger a stop+rebuild+restart of a
	// process that is actually fine. Mirrors WaitForHealth's poll loop but
	// bounded to a few quick attempts (~3 over ~3s) since the registry already
	// proved the instance is *running* — we are only confirming data-plane
	// readiness. We fail toward reuse only after the probe is persistently bad.
	deps := r.runtimeDeps()
	const attempts = 3
	const interval = 1 * time.Second
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			deps.sleep(interval)
		}
		if allHealthChecksPass(health.Checks, view.Ports) {
			return true
		}
	}
	return false
}

func allHealthChecksPass(checks []scenario.HealthCheck, ports map[string]int) bool {
	for _, check := range checks {
		if err := scenario.PerformHealthCheck(check, ports); err != nil {
			return false
		}
	}
	return true
}
