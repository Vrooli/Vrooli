package lifecycle

import (
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/logx"
	"github.com/vrooli/vrooli/internal/process"
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

func (r *Runner) isScenarioHealthyStrict(item scenario.Scenario, records []process.Record) bool {
	if len(process.LiveRecords(records)) == 0 {
		return false
	}
	health := item.Manifest.HealthConfig()
	if health == nil || len(health.Checks) == 0 {
		return true
	}
	for _, check := range health.Checks {
		if err := scenario.PerformHealthCheck(check, scenario.RuntimePorts(item.Manifest, records)); err != nil {
			return false
		}
	}
	return true
}

func (r *Runner) runtimePorts(manifest scenario.ServiceManifest, records []process.Record) map[string]int {
	return scenario.RuntimePorts(manifest, records)
}
