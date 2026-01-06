package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/domain"
)

// Type aliases for backward compatibility and shorter references within main package.
type (
	VPSDeployResult   = domain.VPSDeployResult
	MissingSecretInfo = domain.MissingSecretInfo
)

// VPSDeployRequest is the request body for VPS deployment (unused but kept for API docs).
type VPSDeployRequest struct {
	Manifest CloudManifest `json:"manifest"`
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

// buildPortEnvVars builds exported environment variable assignments for all ports in the manifest.
// Uses "export VAR=value &&" format to ensure environment variables are inherited by all
// child processes (API, UI servers, etc.) spawned by vrooli scenario start.
func buildPortEnvVars(ports ManifestPorts) string {
	if len(ports) == 0 {
		return ""
	}
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
	// Use "export VAR1=val1 VAR2=val2 &&" format so variables are exported and available
	// to all child processes started by the subsequent command
	return fmt.Sprintf("export %s &&", strings.Join(parts, " "))
}

func containsTildeInSingleQuotes(command string) bool {
	inSingleQuote := false
	for _, r := range command {
		if r == '\'' {
			inSingleQuote = !inSingleQuote
			continue
		}
		if inSingleQuote && r == '~' {
			return true
		}
	}
	return false
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
			ID:          "scenario_stop",
			Title:       "Stop existing scenario",
			Description: "Gracefully stop any running instance of the scenario to free ports.",
			Command:     "(vrooli scenario stop + pkill fallback + port cleanup)",
		},
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
	if manifest.Edge.Caddy.Enabled {
		steps = append(steps, VPSPlanStep{
			ID:          "firewall_inbound",
			Title:       "Open inbound HTTP/HTTPS",
			Description: "Allow inbound 80/443 so Caddy can complete ACME validation.",
			Command:     localSSHCommand(cfg, firewallInboundCommand),
		})
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
			Description: "Checks the UI health endpoint locally on the VPS. On failure, provides detailed diagnostics including API connectivity status and investigation commands.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("curl http://127.0.0.1:%d/health (with detailed error reporting)", uiPort)),
		},
		VPSPlanStep{
			ID:          "verify_https",
			Title:       "Verify HTTPS health",
			Description: "Checks https://<domain>/health via Caddy + Let's Encrypt. On failure, provides detailed diagnostics.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("curl https://%s/health (with detailed error reporting)", manifest.Edge.Domain)),
		},
		VPSPlanStep{
			ID:          "verify_origin",
			Title:       "Verify origin reachability",
			Description: "Checks https://<domain>/health against the VPS origin directly (bypasses proxy).",
			Command:     fmt.Sprintf("curl --resolve %s:443:%s https://%s/health (from deployment runner)", manifest.Edge.Domain, manifest.Target.VPS.Host, manifest.Edge.Domain),
		},
		VPSPlanStep{
			ID:          "verify_public",
			Title:       "Verify public reachability",
			Description: "Checks https://<domain>/health from the deployment runner (outside the VPS).",
			Command:     fmt.Sprintf("curl https://%s/health (from deployment runner)", manifest.Edge.Domain),
		},
	)

	return steps, nil
}

// RunVPSDeploy executes VPS deployment without progress tracking.
// This is a convenience wrapper around RunVPSDeployWithProgress that uses no-op progress callbacks.
func RunVPSDeploy(ctx context.Context, manifest CloudManifest, sshRunner SSHRunner, secretsGen SecretsGeneratorFunc) VPSDeployResult {
	progress := 0.0
	return RunVPSDeployWithProgress(ctx, manifest, sshRunner, secretsGen, noopProgressHub{}, noopProgressRepo{}, "", &progress)
}

// noopProgressHub is a no-op implementation of progress broadcasting for RunVPSDeploy.
type noopProgressHub struct{}

func (noopProgressHub) Broadcast(string, ProgressEvent) {}

// noopProgressRepo is a no-op implementation of progress persistence for RunVPSDeploy.
type noopProgressRepo struct{}

func (noopProgressRepo) UpdateDeploymentProgress(context.Context, string, string, float64) error {
	return nil
}

// RunVPSDeployWithProgress runs VPS deployment with progress tracking.
// The secretsGen parameter enables testing with deterministic secret values.
// Pass nil to use the default NewSecretsGenerator().
// The hub parameter accepts any ProgressBroadcaster, allowing use with the real ProgressHub
// or a no-op implementation for callers that don't need progress tracking.
func RunVPSDeployWithProgress(
	ctx context.Context,
	manifest CloudManifest,
	sshRunner SSHRunner,
	secretsGen SecretsGeneratorFunc,
	hub ProgressBroadcaster,
	repo ProgressRepo,
	deploymentID string,
	progress *float64,
) VPSDeployResult {
	// Default to production implementation if nil
	if secretsGen == nil {
		secretsGen = NewSecretsGenerator()
	}
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
			log.Printf("deployment progress update failed (step=%s): %v", stepID, err)
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

	// run executes an SSH command and returns an error with output context if it fails.
	run := func(cmd string) error {
		return RunSSHWithOutput(ctx, sshRunner, cfg, cmd)
	}

	// Step: scenario_stop - Stop existing scenario before deployment
	emit("step_started", "scenario_stop", "Stopping existing scenario")
	var targetPorts []int
	for _, port := range manifest.Ports {
		targetPorts = append(targetPorts, port)
	}
	stopResult := StopExistingScenario(ctx, sshRunner, cfg, workdir, manifest.Scenario.ID, targetPorts)
	if !stopResult.OK {
		return failStep("scenario_stop", "Stopping existing scenario", stopResult.Error)
	}
	*progress += StepWeights["scenario_stop"]
	emit("step_completed", "scenario_stop", "Stopping existing scenario")

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
	// Idempotency: Only write Caddyfile and reload if content differs from current
	emit("step_started", "caddy_config", "Configuring Caddy")
	caddyfilePath := "/etc/caddy/Caddyfile"
	caddyfile := buildCaddyfile(manifest.Edge.Domain, uiPort)

	// Check if current Caddyfile matches desired content (idempotent write)
	checkCmd := fmt.Sprintf("cat %s 2>/dev/null || echo ''", shellQuoteSingle(caddyfilePath))
	currentCaddyfile, _ := sshRunner.Run(ctx, cfg, checkCmd)
	currentContent := strings.TrimSpace(currentCaddyfile.Stdout)
	desiredContent := strings.TrimSpace(caddyfile)

	if currentContent != desiredContent {
		// Content differs, write new config
		if err := run(fmt.Sprintf("printf '%%s' %s > %s", shellQuoteSingle(caddyfile), shellQuoteSingle(caddyfilePath))); err != nil {
			return failStep("caddy_config", "Configuring Caddy", err.Error())
		}
		// Only reload if we actually changed the config
		if err := run("systemctl reload caddy"); err != nil {
			return failStep("caddy_config", "Configuring Caddy", err.Error())
		}
	}
	// If content matches, skip write and reload (already configured correctly)
	*progress += StepWeights["caddy_config"]
	emit("step_completed", "caddy_config", "Configuring Caddy")

	if manifest.Edge.Caddy.Enabled {
		emit("step_started", "firewall_inbound", "Opening inbound HTTP/HTTPS")
		firewallCmd := firewallInboundCommand
		if err := run(firewallCmd); err != nil {
			return failStep("firewall_inbound", "Opening inbound HTTP/HTTPS", err.Error())
		}
		*progress += StepWeights["firewall_inbound"]
		emit("step_completed", "firewall_inbound", "Opening inbound HTTP/HTTPS")
	} else {
		*progress += StepWeights["firewall_inbound"]
	}

	// Step: secrets_provision - Generate and write secrets BEFORE resource startup
	if manifest.Secrets != nil && len(manifest.Secrets.BundleSecrets) > 0 {
		emit("step_started", "secrets_provision", "Provisioning secrets")

		// Generate per_install_generated secrets using the injected generator (seam)
		generated, err := secretsGen.GenerateSecrets(manifest.Secrets.BundleSecrets)
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

	// Step: wait_for_ui - Wait for UI port to be listening before health check
	emit("step_started", "wait_for_ui", "Waiting for UI to listen")
	waitForUIScript := buildWaitForPortScript("127.0.0.1", uiPort, 30, "UI")
	waitCmd := fmt.Sprintf("bash -c %s", shellQuoteSingle(waitForUIScript))
	if err := run(waitCmd); err != nil {
		healthCheckCmd := fmt.Sprintf("curl -fsS --max-time 3 http://127.0.0.1:%d/health", uiPort)
		healthResult, healthErr := sshRunner.Run(ctx, cfg, healthCheckCmd)
		if healthErr == nil && healthResult.ExitCode == 0 {
			return failStep("wait_for_ui", "Waiting for UI to listen", fmt.Sprintf("wait_for_ui failed but /health responded successfully; likely wait script or port check issue. wait error: %s", err.Error()))
		}
		return failStep("wait_for_ui", "Waiting for UI to listen", err.Error())
	}
	*progress += StepWeights["wait_for_ui"]
	emit("step_completed", "wait_for_ui", "Waiting for UI to listen")

	// Step: verify_local - Use detailed health check script for actionable error messages
	emit("step_started", "verify_local", "Verifying local health")
	localHealthURL := fmt.Sprintf("http://127.0.0.1:%d/health", uiPort)
	localHealthScript := buildHealthCheckScript(localHealthURL, 5, "local")
	logFileName := fmt.Sprintf("verify_local_%s.log", manifest.Scenario.ID)
	preflightLogsCmd := "log_dir=\"$HOME/.vrooli/logs\"; mkdir -p \"$log_dir\" && test -w \"$log_dir\""
	if err := run(preflightLogsCmd); err != nil {
		return failStep("verify_local", "Verifying local health", fmt.Sprintf("verify log directory not writable: %s", err.Error()))
	}
	verifyLocalCmd := fmt.Sprintf("log_dir=\"$HOME/.vrooli/logs\"; log_file=\"$log_dir\"/%s; tmp_log=\"$(mktemp)\"; if bash -c %s &> \"$tmp_log\"; then cat \"$tmp_log\" > \"$log_file\" 2>/dev/null || true; else cat \"$tmp_log\"; cat \"$tmp_log\" > \"$log_file\" 2>/dev/null || true; exit 1; fi", shellQuoteSingle(logFileName), shellQuoteSingle(localHealthScript))
	if err := run(verifyLocalCmd); err != nil {
		return failStep("verify_local", "Verifying local health", err.Error())
	}
	*progress += StepWeights["verify_local"]
	emit("step_completed", "verify_local", "Verifying local health")

	// Step: verify_https - Use detailed health check script for actionable error messages
	emit("step_started", "verify_https", "Verifying HTTPS")
	httpsHealthURL := fmt.Sprintf("https://%s/health", manifest.Edge.Domain)
	httpsHealthScript := buildHealthCheckScript(httpsHealthURL, 10, "https")
	if err := run(fmt.Sprintf("bash -c %s", shellQuoteSingle(httpsHealthScript))); err != nil {
		return failStep("verify_https", "Verifying HTTPS", err.Error())
	}
	*progress += StepWeights["verify_https"]
	emit("step_completed", "verify_https", "Verifying HTTPS")

	// Step: verify_origin - Verify reachability to origin directly (bypasses proxy)
	emit("step_started", "verify_origin", "Verifying origin reachability")
	if err := checkOriginHealth(ctx, manifest.Edge.Domain, manifest.Target.VPS.Host, 10*time.Second); err != nil {
		if manifest.Edge.Caddy.Enabled {
			if hint := caddyACMEOriginUnreachableHint(fetchCaddyLogs(ctx, sshRunner, cfg, 200)); hint != "" {
				return failStep("verify_origin", "Verifying origin reachability", hint)
			}
		}
		return failStep("verify_origin", "Verifying origin reachability", err.Error())
	}
	*progress += StepWeights["verify_origin"]
	emit("step_completed", "verify_origin", "Verifying origin reachability")

	// Step: verify_public - Verify public reachability from deployment runner
	emit("step_started", "verify_public", "Verifying public reachability")
	if err := checkPublicHealth(ctx, httpsHealthURL, 10*time.Second); err != nil {
		return failStep("verify_public", "Verifying public reachability", err.Error())
	}
	*progress += StepWeights["verify_public"]
	emit("step_completed", "verify_public", "Verifying public reachability")

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

// buildHealthCheckScript returns a shell script that performs a health check with detailed error reporting.
// It captures the HTTP status code, response body, and provides actionable diagnostics.
func buildHealthCheckScript(url string, timeoutSecs int, checkType string) string {
	// This script:
	// 1. Captures HTTP status and response body separately
	// 2. Parses JSON response for error details
	// 3. Provides contextual error messages based on status codes
	// 4. Includes investigation commands for common failures
	return fmt.Sprintf(`
set -e
URL="%s"
TIMEOUT=%d
CHECK_TYPE="%s"

# Perform the request, capturing status code and body
RESPONSE=$(curl -sS --max-time "$TIMEOUT" -w '\n%%{http_code}' "$URL") || {
    EXIT_CODE=$?
    case $EXIT_CODE in
        7)  echo "❌ Connection refused: No service listening on $URL"
            echo ""
            echo "Investigation:"
            if [[ "$CHECK_TYPE" == "local" ]]; then
                echo "  • Check if UI server is running: ps aux | grep -E 'node|server'"
                echo "  • Check port binding: ss -tlnp | grep -E '35000|15000'"
                echo "  • Check UI logs: tail -50 scenarios/*/logs/ui.log"
            fi
            ;;
        28) echo "❌ Timeout: $URL did not respond within ${TIMEOUT}s"
            echo ""
            echo "Investigation:"
            echo "  • Service may be starting up - wait and retry"
            echo "  • Check system resources: top -bn1 | head -20"
            ;;
        *)  echo "❌ curl failed with exit code $EXIT_CODE"
            echo "Response: $RESPONSE"
            ;;
    esac
    exit 1
}

# Split response body and status code
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)
BODY=$(echo "$RESPONSE" | sed '$d')

if [ "$HTTP_CODE" -ge 200 ] && [ "$HTTP_CODE" -lt 300 ]; then
    echo "✓ Health check passed ($HTTP_CODE)"
    exit 0
fi

# Handle error responses with detailed diagnostics
echo "❌ Health check failed: HTTP $HTTP_CODE"
echo ""
echo "URL: $URL"
echo ""

case $HTTP_CODE in
    503)
        echo "Status: Service Unavailable"
        echo ""
        # Try to parse JSON response for details
        if echo "$BODY" | jq -e '.api_connectivity' &> /dev/null; then
            API_CONNECTED=$(echo "$BODY" | jq -r '.api_connectivity.connected // "unknown"')
            API_ERROR=$(echo "$BODY" | jq -r '.api_connectivity.error.message // "no details"')
            echo "API Connectivity: $API_CONNECTED"
            if [ "$API_CONNECTED" = "false" ]; then
                echo "API Error: $API_ERROR"
                echo ""
                echo "The UI server is running but cannot reach the API server."
                echo ""
                echo "Investigation:"
                echo "  • Check API process: ps aux | grep -E 'api|go'"
                echo "  • Check API port: ss -tlnp | grep 15000"
                echo "  • Check API logs: tail -50 scenarios/*/logs/api.log"
                echo "  • Verify API_PORT env var was set: echo \$API_PORT"
            fi
        elif echo "$BODY" | jq -e '.status' &> /dev/null; then
            STATUS=$(echo "$BODY" | jq -r '.status')
            echo "Service status: $STATUS"
            echo ""
            echo "Response body:"
            echo "$BODY" | jq . 2>/dev/null || echo "$BODY"
        else
            echo "Response body:"
            echo "$BODY" | head -20
        fi
        ;;
    502)
        echo "Status: Bad Gateway"
        echo "The reverse proxy (Caddy) cannot reach the upstream service."
        echo ""
        echo "Investigation:"
        echo "  • Check Caddyfile: cat /etc/caddy/Caddyfile"
        echo "  • Check Caddy status: systemctl status caddy"
        echo "  • Verify UI is running on configured port"
        ;;
    404)
        echo "Status: Not Found"
        echo "The /health endpoint does not exist at this URL."
        echo ""
        echo "Investigation:"
        echo "  • Verify the service exposes /health"
        echo "  • Check if correct port is being used"
        ;;
    *)
        echo "Response body:"
        echo "$BODY" | head -20
        ;;
esac

exit 1
`, url, timeoutSecs, checkType)
}

func checkPublicHealth(ctx context.Context, url string, timeout time.Duration) error {
	client := &http.Client{Timeout: timeout}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("build public health request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("public health check failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			return fmt.Errorf("public health check returned %s: %s", resp.Status, msg)
		}
		return fmt.Errorf("public health check returned %s", resp.Status)
	}
	return nil
}

func checkOriginHealth(ctx context.Context, domain, host string, timeout time.Duration) error {
	if strings.TrimSpace(domain) == "" {
		return fmt.Errorf("origin health check failed: domain is empty")
	}
	ip, err := resolveHostIP(ctx, host)
	if err != nil {
		return fmt.Errorf("origin health check failed: resolve VPS host %q: %w", host, err)
	}
	targetURL := fmt.Sprintf("https://%s/health", domain)
	dialer := &net.Dialer{Timeout: timeout}
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, network, net.JoinHostPort(ip, "443"))
		},
		TLSClientConfig: &tls.Config{
			ServerName: strings.TrimSpace(domain),
		},
	}
	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, targetURL, nil)
	if err != nil {
		return fmt.Errorf("origin health check failed: build request: %w", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("origin health check failed for %s via %s: %w", domain, ip, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		msg := strings.TrimSpace(string(body))
		if msg != "" {
			return fmt.Errorf("origin health check returned %s via %s: %s", resp.Status, ip, msg)
		}
		return fmt.Errorf("origin health check returned %s via %s", resp.Status, ip)
	}
	return nil
}

func fetchCaddyLogs(ctx context.Context, sshRunner SSHRunner, cfg SSHConfig, lines int) string {
	if lines <= 0 {
		lines = 200
	}
	cmd := fmt.Sprintf("journalctl -u caddy --no-pager -n %d 2>/dev/null || true", lines)
	res, err := sshRunner.Run(ctx, cfg, cmd)
	if err != nil {
		return ""
	}
	return res.Stdout
}

func caddyACMEOriginUnreachableHint(logs string) string {
	if strings.TrimSpace(logs) == "" {
		return ""
	}
	lower := strings.ToLower(logs)
	if !strings.Contains(lower, "acme") {
		return ""
	}
	if !strings.Contains(lower, "522") && !strings.Contains(lower, "403") {
		return ""
	}

	var matches []string
	for _, line := range strings.Split(logs, "\n") {
		lineLower := strings.ToLower(line)
		if strings.Contains(lineLower, "acme") && (strings.Contains(lineLower, "522") || strings.Contains(lineLower, "403")) {
			matches = append(matches, strings.TrimSpace(line))
		}
	}

	var sb strings.Builder
	sb.WriteString("Caddy ACME validation shows origin unreachable (522/403). ")
	sb.WriteString("Open inbound 80/443 or use DNS-01/DNS-only during issuance.")
	if len(matches) > 0 {
		sb.WriteString(" Recent Caddy logs: ")
		for i, line := range matches {
			if i >= 3 {
				break
			}
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(line)
		}
	}
	return sb.String()
}

func resolveHostIP(ctx context.Context, host string) (string, error) {
	host = strings.TrimSpace(host)
	if host == "" {
		return "", fmt.Errorf("host is empty")
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.String(), nil
	}
	ips, err := net.DefaultResolver.LookupIPAddr(ctx, host)
	if err != nil {
		return "", err
	}
	for _, ip := range ips {
		if ip.IP != nil {
			return ip.IP.String(), nil
		}
	}
	return "", fmt.Errorf("no IPs found for host")
}

// buildWaitForPortScript returns a shell script that waits for a TCP port to be listening.
func buildWaitForPortScript(host string, port int, timeoutSecs int, serviceName string) string {
	return fmt.Sprintf(`
set -e
HOST="%s"
PORT=%d
TIMEOUT=%d
SERVICE="%s"

start_time=$(date +%%s)
echo "Waiting for $SERVICE on $HOST:$PORT to accept connections..."
while true; do
    if ss -tln | awk '{print $4}' | grep -Eq "[:.]${PORT}$"; then
        echo "$SERVICE is listening on $HOST:$PORT"
        exit 0
    fi
    now=$(date +%%s)
    elapsed=$((now - start_time))
    if (( elapsed >= TIMEOUT )); then
        echo "❌ Timeout waiting for $SERVICE on $HOST:$PORT after ${TIMEOUT}s"
        echo "Investigation:"
        echo "  • Check UI logs: tail -50 $HOME/.vrooli/logs/scenarios/*/vrooli.develop.*.start-ui.log"
        echo "  • Check port bindings: ss -tlnp | grep -E '35000|15000'"
        exit 1
    fi
    sleep 1
done
`, host, port, timeoutSecs, serviceName)
}
