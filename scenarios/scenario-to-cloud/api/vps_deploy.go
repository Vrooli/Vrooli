package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

// MissingSecretInfo describes a missing user_prompt secret.
type MissingSecretInfo struct {
	ID          string `json:"id"`
	Key         string `json:"key"`         // env var name
	Label       string `json:"label"`       // human-readable label
	Description string `json:"description"` // help text
}

// ValidateUserPromptSecrets checks that all required user_prompt secrets are provided.
// Returns nil if all required secrets are present, or an error with details about missing secrets.
func ValidateUserPromptSecrets(manifest CloudManifest, providedSecrets map[string]string) ([]MissingSecretInfo, error) {
	if manifest.Secrets == nil || len(manifest.Secrets.BundleSecrets) == 0 {
		return nil, nil // No secrets required
	}

	var missing []MissingSecretInfo
	for _, secret := range manifest.Secrets.BundleSecrets {
		if secret.Class != "user_prompt" {
			continue // Not a user-provided secret
		}
		if !secret.Required {
			continue // Optional secret
		}

		key := secret.Target.Name
		if key == "" {
			key = secret.ID
		}

		// Check if secret was provided
		if _, ok := providedSecrets[key]; ok {
			continue // Secret provided
		}

		// Secret is missing - collect info for error message
		label := key
		description := secret.Description
		if secret.Prompt != nil {
			if secret.Prompt.Label != "" {
				label = secret.Prompt.Label
			}
			if secret.Prompt.Description != "" {
				description = secret.Prompt.Description
			}
		}

		missing = append(missing, MissingSecretInfo{
			ID:          secret.ID,
			Key:         key,
			Label:       label,
			Description: description,
		})
	}

	if len(missing) == 0 {
		return nil, nil
	}

	// Build actionable error message
	var sb strings.Builder
	sb.WriteString("missing required secrets:\n")
	for _, m := range missing {
		sb.WriteString(fmt.Sprintf("  - %s (%s)", m.Label, m.Key))
		if m.Description != "" {
			sb.WriteString(": " + m.Description)
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\nProvide secrets via --secret KEY=VALUE flags or environment variables")

	return missing, fmt.Errorf("%s", sb.String())
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

func requiredResourcesForScenario(scenarioID string) ([]string, error) {
	repoRoot, err := FindRepoRootFromCWD()
	if err != nil {
		return nil, fmt.Errorf("repo root not found for dependency validation: %w", err)
	}
	serviceJSONPath := filepath.Join(repoRoot, "scenarios", scenarioID, ".vrooli", "service.json")
	data, err := os.ReadFile(serviceJSONPath)
	if err != nil {
		return nil, fmt.Errorf("read service.json for dependency validation: %w", err)
	}
	var svc ServiceJSON
	if err := json.Unmarshal(data, &svc); err != nil {
		return nil, fmt.Errorf("parse service.json for dependency validation: %w", err)
	}
	var required []string
	for name, dep := range svc.Dependencies.Resources {
		if dep.Enabled || dep.Required {
			required = append(required, name)
		}
	}
	return stableUniqueStrings(required), nil
}

func validateManifestResourceDependencies(manifest CloudManifest) error {
	required, err := requiredResourcesForScenario(manifest.Scenario.ID)
	if err != nil {
		return err
	}
	if len(required) == 0 {
		return nil
	}
	declared := stableUniqueStrings(manifest.Dependencies.Resources)
	var missing []string
	for _, name := range required {
		if !contains(declared, name) {
			missing = append(missing, name)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("deployment manifest missing required resources: %s. Re-export the manifest from scenario-dependency-analyzer or ensure .vrooli/service.json resources are captured in dependencies.resources", strings.Join(missing, ", "))
	}
	return nil
}

func BuildVPSDeployPlan(manifest CloudManifest) ([]VPSPlanStep, error) {
	if err := validateManifestResourceDependencies(manifest); err != nil {
		return nil, err
	}
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

	// Add secrets provisioning step if secrets are present
	if manifest.Secrets != nil && len(manifest.Secrets.BundleSecrets) > 0 {
		steps = append(steps, VPSPlanStep{
			ID:          "secrets_provision",
			Title:       "Provision secrets",
			Description: "Generate per-install secrets and write to .vrooli/secrets.json before resource startup.",
			Command:     "(custom step - secrets generated and written via API)",
		})
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

	// Secrets provisioning: Generate per_install_generated secrets and write to VPS BEFORE resource startup
	if manifest.Secrets != nil && len(manifest.Secrets.BundleSecrets) > 0 {
		generator := NewSecretsGenerator()
		generated, err := generator.GenerateSecrets(manifest.Secrets.BundleSecrets)
		if err != nil {
			return VPSDeployResult{OK: false, Steps: steps, Error: fmt.Sprintf("generate secrets: %v", err), Timestamp: time.Now().UTC().Format(time.RFC3339)}
		}
		if len(generated) > 0 {
			if err := WriteSecretsToVPS(ctx, sshRunner, cfg, workdir, generated, manifest.Scenario.ID); err != nil {
				return VPSDeployResult{OK: false, Steps: steps, Error: fmt.Sprintf("write secrets: %v", err), Timestamp: time.Now().UTC().Format(time.RFC3339)}
			}
		}
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

	// Step: secrets_provision - Generate and write secrets BEFORE resource startup
	if manifest.Secrets != nil && len(manifest.Secrets.BundleSecrets) > 0 {
		emit("step_started", "secrets_provision", "Provisioning secrets")

		// Generate per_install_generated secrets
		generator := NewSecretsGenerator()
		generated, err := generator.GenerateSecrets(manifest.Secrets.BundleSecrets)
		if err != nil {
			return failStep("secrets_provision", "Provisioning secrets", fmt.Sprintf("generate secrets: %v", err))
		}

		// Write secrets.json to VPS
		if len(generated) > 0 {
			if err := WriteSecretsToVPS(ctx, sshRunner, cfg, workdir, generated, manifest.Scenario.ID); err != nil {
				return failStep("secrets_provision", "Provisioning secrets", fmt.Sprintf("write secrets: %v", err))
			}
		}

		*progress += StepWeights["secrets_provision"]
		emit("step_completed", "secrets_provision", "Provisioning secrets")
	}

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
