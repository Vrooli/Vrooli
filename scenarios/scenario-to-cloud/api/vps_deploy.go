package main

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"
)

type VPSDeployRequest struct {
	Manifest CloudManifest `json:"manifest"`
}

type VPSDeployResult struct {
	OK         bool          `json:"ok"`
	Steps      []VPSPlanStep `json:"steps"`
	Error      string        `json:"error,omitempty"`
	FailedStep string        `json:"failed_step,omitempty"`
	Timestamp  string        `json:"timestamp"`
}

// buildPortEnvVars builds environment variable assignments for all ports in the manifest
func buildPortEnvVars(ports ManifestPorts) string {
	var parts []string
	// Sort keys for deterministic output
	keys := make([]string, 0, len(ports))
	for k := range ports {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, key := range keys {
		// Convert port key to env var name: ui -> UI_PORT, api -> API_PORT, playwright_driver -> PLAYWRIGHT_DRIVER_PORT
		envVar := strings.ToUpper(key) + "_PORT"
		parts = append(parts, fmt.Sprintf("%s=%d", envVar, ports[key]))
	}
	return strings.Join(parts, " ")
}

func BuildVPSDeployPlan(manifest CloudManifest) ([]VPSPlanStep, error) {
	cfg := sshConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	uiPort := manifest.Ports["ui"]

	caddyfile := buildCaddyfile(manifest.Edge.Domain, uiPort)
	caddyfilePath := "/etc/caddy/Caddyfile"

	steps := []VPSPlanStep{
		{
			ID:          "caddy_install",
			Title:       "Install Caddy",
			Description: "Ensure Caddy is installed (apt) and enabled (systemd).",
			Command: localSSHCommand(cfg, strings.Join([]string{
				"command -v caddy >/dev/null || (apt-get update -y && apt-get install -y caddy)",
				"systemctl enable --now caddy",
			}, " && ")),
		},
		{
			ID:          "caddy_config",
			Title:       "Configure Caddy",
			Description: "Write a minimal Caddyfile and reload.",
			Command: localSSHCommand(cfg, strings.Join([]string{
				fmt.Sprintf("printf '%%s' %s > %s", shellQuoteSingle(caddyfile), shellQuoteSingle(caddyfilePath)),
				"systemctl reload caddy",
			}, " && ")),
		},
	}

	for _, res := range stableUniqueStrings(manifest.Dependencies.Resources) {
		steps = append(steps, VPSPlanStep{
			ID:          "resource_start_" + res,
			Title:       "Start resource: " + res,
			Description: "Start required Vrooli resources via the mini install.",
			Command:     localSSHCommand(cfg, vrooliCommand(workdir, fmt.Sprintf("vrooli resource start %s", shellQuoteSingle(res)))),
		})
	}

	for _, scen := range stableUniqueStrings(manifest.Dependencies.Scenarios) {
		if scen == manifest.Scenario.ID {
			continue
		}
		steps = append(steps, VPSPlanStep{
			ID:          "scenario_start_" + scen,
			Title:       "Start scenario: " + scen,
			Description: "Start dependent scenarios (excluding the target).",
			Command:     localSSHCommand(cfg, vrooliCommand(workdir, fmt.Sprintf("vrooli scenario start %s", shellQuoteSingle(scen)))),
		})
	}

	// Build port environment variables from manifest
	portEnvVars := buildPortEnvVars(manifest.Ports)

	steps = append(steps,
		VPSPlanStep{
			ID:          "scenario_start_target",
			Title:       "Start target scenario with fixed ports",
			Description: "Starts the target scenario with port overrides from the manifest.",
			Command:     localSSHCommand(cfg, vrooliCommand(workdir, fmt.Sprintf("%s vrooli scenario start %s", portEnvVars, shellQuoteSingle(manifest.Scenario.ID)))),
		},
		VPSPlanStep{
			ID:          "verify_local",
			Title:       "Verify local health",
			Description: "Checks the UI health endpoint locally on the VPS.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("curl -fsS --max-time 5 http://127.0.0.1:%d/health", uiPort)),
		},
		VPSPlanStep{
			ID:          "verify_https",
			Title:       "Verify HTTPS health",
			Description: "Checks https://<domain>/health via Caddy + Let's Encrypt.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("curl -fsS --max-time 10 https://%s/health", manifest.Edge.Domain)),
		},
	)

	return steps, nil
}

func RunVPSDeploy(ctx context.Context, manifest CloudManifest, sshRunner SSHRunner) VPSDeployResult {
	steps, err := BuildVPSDeployPlan(manifest)
	if err != nil {
		return VPSDeployResult{OK: false, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	cfg := sshConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	uiPort := manifest.Ports["ui"]

	run := func(cmd string) error {
		res, err := sshRunner.Run(ctx, cfg, cmd)
		if err != nil {
			// Collect output from both stderr and stdout for better error context.
			// Many CLI tools (including vrooli) use `2>&1 | tee` which sends errors to stdout.
			var outputParts []string
			if res.Stderr != "" {
				outputParts = append(outputParts, "stderr: "+res.Stderr)
			}
			if res.Stdout != "" {
				// Limit stdout to last 50 lines to avoid overwhelming error messages
				lines := strings.Split(res.Stdout, "\n")
				if len(lines) > 50 {
					lines = lines[len(lines)-50:]
					outputParts = append(outputParts, "stdout (last 50 lines): "+strings.Join(lines, "\n"))
				} else {
					outputParts = append(outputParts, "stdout: "+res.Stdout)
				}
			}
			if len(outputParts) > 0 {
				return fmt.Errorf("%w\n%s", err, strings.Join(outputParts, "\n"))
			}
			return err
		}
		return nil
	}

	if err := run("command -v caddy >/dev/null || (apt-get update -y && apt-get install -y caddy)"); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run("systemctl enable --now caddy"); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	caddyfilePath := "/etc/caddy/Caddyfile"
	caddyfile := buildCaddyfile(manifest.Edge.Domain, uiPort)
	if err := run(fmt.Sprintf("printf '%%s' %s > %s", shellQuoteSingle(caddyfile), shellQuoteSingle(caddyfilePath))); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run("systemctl reload caddy"); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	for _, res := range stableUniqueStrings(manifest.Dependencies.Resources) {
		if err := run(vrooliCommand(workdir, fmt.Sprintf("vrooli resource start %s", shellQuoteSingle(res)))); err != nil {
			return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
		}
	}

	for _, scen := range stableUniqueStrings(manifest.Dependencies.Scenarios) {
		if scen == manifest.Scenario.ID {
			continue
		}
		if err := run(vrooliCommand(workdir, fmt.Sprintf("vrooli scenario start %s", shellQuoteSingle(scen)))); err != nil {
			return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
		}
	}

	// Build port environment variables from manifest
	portEnvVars := buildPortEnvVars(manifest.Ports)
	if err := run(vrooliCommand(workdir, fmt.Sprintf("%s vrooli scenario start %s", portEnvVars, shellQuoteSingle(manifest.Scenario.ID)))); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	if err := run(fmt.Sprintf("curl -fsS --max-time 5 http://127.0.0.1:%d/health", uiPort)); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(fmt.Sprintf("curl -fsS --max-time 10 https://%s/health", manifest.Edge.Domain)); err != nil {
		return VPSDeployResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	return VPSDeployResult{OK: true, Steps: steps, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

// RunVPSDeployWithProgress runs VPS deployment with progress tracking.
func RunVPSDeployWithProgress(
	ctx context.Context,
	manifest CloudManifest,
	sshRunner SSHRunner,
	hub *ProgressHub,
	repo ProgressRepo,
	deploymentID string,
	progress *float64,
) VPSDeployResult {
	steps, err := BuildVPSDeployPlan(manifest)
	if err != nil {
		return VPSDeployResult{OK: false, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	cfg := sshConfigFromManifest(manifest)
	workdir := manifest.Target.VPS.Workdir
	uiPort := manifest.Ports["ui"]

	// Helper to emit progress
	emit := func(eventType, stepID, stepTitle string) {
		event := ProgressEvent{
			Type:      eventType,
			Step:      stepID,
			StepTitle: stepTitle,
			Progress:  *progress,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		hub.Broadcast(deploymentID, event)
		if err := repo.UpdateDeploymentProgress(ctx, deploymentID, stepID, *progress); err != nil {
			// Log but don't fail
		}
	}

	// Helper to emit error and return failed result
	failStep := func(stepID, stepTitle, errMsg string) VPSDeployResult {
		event := ProgressEvent{
			Type:      "deployment_error",
			Step:      stepID,
			StepTitle: stepTitle,
			Progress:  *progress,
			Error:     errMsg,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		hub.Broadcast(deploymentID, event)
		return VPSDeployResult{OK: false, Steps: steps, Error: errMsg, FailedStep: stepID, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	run := func(cmd string) error {
		res, err := sshRunner.Run(ctx, cfg, cmd)
		if err != nil {
			// Collect output from both stderr and stdout for better error context.
			// Many CLI tools (including vrooli) use `2>&1 | tee` which sends errors to stdout.
			var outputParts []string
			if res.Stderr != "" {
				outputParts = append(outputParts, "stderr: "+res.Stderr)
			}
			if res.Stdout != "" {
				// Limit stdout to last 50 lines to avoid overwhelming error messages
				lines := strings.Split(res.Stdout, "\n")
				if len(lines) > 50 {
					lines = lines[len(lines)-50:]
					outputParts = append(outputParts, "stdout (last 50 lines): "+strings.Join(lines, "\n"))
				} else {
					outputParts = append(outputParts, "stdout: "+res.Stdout)
				}
			}
			if len(outputParts) > 0 {
				return fmt.Errorf("%w\n%s", err, strings.Join(outputParts, "\n"))
			}
			return err
		}
		return nil
	}

	// Step: caddy_install
	emit("step_started", "caddy_install", "Installing Caddy")
	if err := run("command -v caddy >/dev/null || (apt-get update -y && apt-get install -y caddy)"); err != nil {
		return failStep("caddy_install", "Installing Caddy", err.Error())
	}
	if err := run("systemctl enable --now caddy"); err != nil {
		return failStep("caddy_install", "Installing Caddy", err.Error())
	}
	*progress += StepWeights["caddy_install"]
	emit("step_completed", "caddy_install", "Installing Caddy")

	// Step: caddy_config
	emit("step_started", "caddy_config", "Configuring Caddy")
	caddyfilePath := "/etc/caddy/Caddyfile"
	caddyfile := buildCaddyfile(manifest.Edge.Domain, uiPort)
	if err := run(fmt.Sprintf("printf '%%s' %s > %s", shellQuoteSingle(caddyfile), shellQuoteSingle(caddyfilePath))); err != nil {
		return failStep("caddy_config", "Configuring Caddy", err.Error())
	}
	if err := run("systemctl reload caddy"); err != nil {
		return failStep("caddy_config", "Configuring Caddy", err.Error())
	}
	*progress += StepWeights["caddy_config"]
	emit("step_completed", "caddy_config", "Configuring Caddy")

	// Step: resource_start
	resources := stableUniqueStrings(manifest.Dependencies.Resources)
	if len(resources) > 0 {
		emit("step_started", "resource_start", "Starting resources")
		for _, res := range resources {
			if err := run(vrooliCommand(workdir, fmt.Sprintf("vrooli resource start %s", shellQuoteSingle(res)))); err != nil {
				return failStep("resource_start", "Starting resources", err.Error())
			}
		}
		*progress += StepWeights["resource_start"]
		emit("step_completed", "resource_start", "Starting resources")
	} else {
		*progress += StepWeights["resource_start"]
	}

	// Step: scenario_deps
	depScenarios := []string{}
	for _, scen := range stableUniqueStrings(manifest.Dependencies.Scenarios) {
		if scen != manifest.Scenario.ID {
			depScenarios = append(depScenarios, scen)
		}
	}
	if len(depScenarios) > 0 {
		emit("step_started", "scenario_deps", "Starting dependencies")
		for _, scen := range depScenarios {
			if err := run(vrooliCommand(workdir, fmt.Sprintf("vrooli scenario start %s", shellQuoteSingle(scen)))); err != nil {
				return failStep("scenario_deps", "Starting dependencies", err.Error())
			}
		}
		*progress += StepWeights["scenario_deps"]
		emit("step_completed", "scenario_deps", "Starting dependencies")
	} else {
		*progress += StepWeights["scenario_deps"]
	}

	// Step: scenario_target
	emit("step_started", "scenario_target", "Starting scenario")
	portEnvVars := buildPortEnvVars(manifest.Ports)
	if err := run(vrooliCommand(workdir, fmt.Sprintf("%s vrooli scenario start %s", portEnvVars, shellQuoteSingle(manifest.Scenario.ID)))); err != nil {
		return failStep("scenario_target", "Starting scenario", err.Error())
	}
	*progress += StepWeights["scenario_target"]
	emit("step_completed", "scenario_target", "Starting scenario")

	// Step: verify_local
	emit("step_started", "verify_local", "Verifying local health")
	if err := run(fmt.Sprintf("curl -fsS --max-time 5 http://127.0.0.1:%d/health", uiPort)); err != nil {
		return failStep("verify_local", "Verifying local health", err.Error())
	}
	*progress += StepWeights["verify_local"]
	emit("step_completed", "verify_local", "Verifying local health")

	// Step: verify_https
	emit("step_started", "verify_https", "Verifying HTTPS")
	if err := run(fmt.Sprintf("curl -fsS --max-time 10 https://%s/health", manifest.Edge.Domain)); err != nil {
		return failStep("verify_https", "Verifying HTTPS", err.Error())
	}
	*progress += StepWeights["verify_https"]
	emit("step_completed", "verify_https", "Verifying HTTPS")

	return VPSDeployResult{OK: true, Steps: steps, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

func buildCaddyfile(domain string, uiPort int) string {
	domain = strings.TrimSpace(domain)
	if domain == "" {
		domain = "example.com"
	}
	return fmt.Sprintf(`%s {
  reverse_proxy 127.0.0.1:%d
}`, domain, uiPort)
}
