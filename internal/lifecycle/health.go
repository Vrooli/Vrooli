package lifecycle

import (
	"fmt"
	"time"

	"github.com/vrooli/vrooli/internal/process"
	"github.com/vrooli/vrooli/internal/scenario"
)

func (r *Runner) WaitForHealth(item scenario.Scenario, env map[string]string) (string, error) {
	health := item.Manifest.HealthConfig()
	if health == nil || len(health.Checks) == 0 {
		return "running", nil
	}
	ports := healthPortsFromEnv(item.Manifest, env)

	if health.StartupGracePeriod > 0 {
		time.Sleep(time.Duration(health.StartupGracePeriod) * time.Millisecond)
	}

	deadline := time.Now().Add(30 * time.Second)
	if health.Timeout > 0 {
		deadline = time.Now().Add(time.Duration(health.Timeout) * time.Millisecond)
	}

	interval := 500 * time.Millisecond
	if health.Interval > 0 {
		interval = time.Duration(health.Interval) * time.Millisecond
		if interval > 2*time.Second {
			interval = 2 * time.Second
		}
	}

	lastStatus := "unhealthy"
	for {
		lastStatus = scenario.EvaluateHealth(health, ports)
		if lastStatus == "healthy" {
			return lastStatus, nil
		}
		if time.Now().After(deadline) {
			if lastStatus == "degraded" {
				return lastStatus, nil
			}
			return lastStatus, fmt.Errorf("scenario %q failed health checks", item.Slug)
		}
		time.Sleep(interval)
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
		if err := scenario.PerformHealthCheck(check, r.runtimePorts(item.Manifest, records)); err != nil {
			return false
		}
	}
	return true
}

func (r *Runner) runtimePorts(manifest scenario.ServiceManifest, records []process.Record) map[string]int {
	portsByEnv := make(map[string]int)
	// Prefer the explicit step->port metadata captured in process records, then
	// fall back to reading *_PORT values from the live process environment.
	for _, record := range records {
		if record.Port <= 0 {
			continue
		}
		key := inferPortEnvVar(manifest, record.Step)
		if key == "" {
			continue
		}
		if _, exists := portsByEnv[key]; !exists {
			portsByEnv[key] = record.Port
		}
	}

	envPorts := process.ReadEnvironmentPorts(records, manifest.PortEnvVars())
	for key, port := range envPorts {
		if _, exists := portsByEnv[key]; !exists {
			portsByEnv[key] = port
		}
	}
	return portsByEnv
}
