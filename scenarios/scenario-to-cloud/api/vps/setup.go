package vps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/internal/shellutil"
	"scenario-to-cloud/ssh"
)

// bootstrapCommand installs system prerequisites on a fresh VPS.
// Uses noninteractive mode to prevent apt/debconf from hanging on prompts.
const bootstrapCommand = `export DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a && ` +
	`apt-get update -qq && ` +
	`apt-get install -y -qq curl git unzip tar jq ca-certificates gnupg lsb-release`

// firewallInboundCommand opens HTTP/HTTPS ports for Caddy ACME validation.
const firewallInboundCommand = "command -v ufw >/dev/null 2>/dev/null && { ufw allow 80/tcp; ufw allow 443/tcp; ufw reload; } || true"

// buildBootstrapCommand returns the bootstrap command, optionally including firewall rules.
func buildBootstrapCommand(manifest domain.CloudManifest) string {
	command := bootstrapCommand
	if manifest.Edge.Caddy.Enabled {
		command = fmt.Sprintf("%s && %s", command, firewallInboundCommand)
	}
	return command
}

func buildVrooliSetupCommand(workdir string) string {
	return shellutil.VrooliCommand(workdir, "vrooli setup --yes yes --environment production")
}

// SetupRequest is the request body for VPS setup.
type SetupRequest struct {
	Manifest   domain.CloudManifest `json:"manifest"`
	BundlePath string               `json:"bundle_path"`
}

// BuildSetupPlan creates a plan of steps to execute during VPS setup.
func BuildSetupPlan(manifest domain.CloudManifest, bundlePath string) ([]domain.VPSPlanStep, error) {
	bundlePath = strings.TrimSpace(bundlePath)
	if bundlePath == "" {
		return nil, fmt.Errorf("bundle_path is required")
	}
	cleanupCommand, err := buildScenarioCleanupCommand(manifest)
	if err != nil {
		return nil, err
	}
	cfg := ssh.ConfigFromManifest(manifest)

	remoteBundleDir := shellutil.SafeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud", "bundles")
	remoteBundlePath := shellutil.SafeRemoteJoin(remoteBundleDir, filepath.Base(bundlePath))

	autohealConfigPath := shellutil.SafeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud", "autoheal-scope.json")

	return []domain.VPSPlanStep{
		{
			ID:          "mkdir",
			Title:       "Create remote directories",
			Description: "Ensure the deployment workdir and bundle directory exist.",
			Command:     ssh.LocalSSHCommand(cfg, fmt.Sprintf("mkdir -p %s %s", shellutil.QuoteSingle(manifest.Target.VPS.Workdir), shellutil.QuoteSingle(remoteBundleDir))),
		},
		{
			ID:          "bootstrap",
			Title:       "Install system prerequisites",
			Description: "Update apt and install required packages (curl, git, unzip, etc.)",
			Command:     ssh.LocalSSHCommand(cfg, buildBootstrapCommand(manifest)),
		},
		{
			ID:          "upload",
			Title:       "Upload bundle",
			Description: "Copy the mini-Vrooli tarball to the VPS via scp.",
			Command:     ssh.LocalSCPCommand(cfg, bundlePath, remoteBundlePath),
		},
		{
			ID:          "cleanup_scenarios",
			Title:       "Clean scenario code targets",
			Description: "Removes old scenario code before extract while preserving explicitly configured mutable paths.",
			Command:     ssh.LocalSSHCommand(cfg, cleanupCommand),
		},
		{
			ID:          "extract",
			Title:       "Extract bundle",
			Description: "Extract the tarball into the deployment workdir.",
			Command:     ssh.LocalSSHCommand(cfg, fmt.Sprintf("tar -xzf %s -C %s", shellutil.QuoteSingle(remoteBundlePath), shellutil.QuoteSingle(manifest.Target.VPS.Workdir))),
		},
		{
			ID:          "install_vrooli",
			Title:       "Install native vrooli CLI",
			Description: "Build the native Linux vrooli binary locally for the detected target architecture, upload it to the deployment workdir, and mark it executable.",
			Command:     buildInstallVrooliPlanCommand(cfg, manifest.Target.VPS.Workdir),
		},
		{
			ID:          "setup",
			Title:       "Run Vrooli setup",
			Description: "Runs production setup with only required resources.",
			Command:     ssh.LocalSSHCommand(cfg, buildVrooliSetupCommand(manifest.Target.VPS.Workdir)),
		},
		{
			ID:          "autoheal",
			Title:       "Write autoheal scope config",
			Description: "Writes a minimal config so vrooli-autoheal can scope checks to this deployment.",
			Command:     ssh.LocalSSHCommand(cfg, fmt.Sprintf("mkdir -p %s && printf '%%s' %s > %s", shellutil.QuoteSingle(shellutil.SafeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud")), shellutil.QuoteSingle(minimalAutohealScopeJSON(manifest)), shellutil.QuoteSingle(autohealConfigPath))),
		},
		{
			ID:          "verify",
			Title:       "Verify vrooli CLI",
			Description: "Sanity check that vrooli runs within the deployment directory.",
			Command:     ssh.LocalSSHCommand(cfg, shellutil.VrooliCommand(manifest.Target.VPS.Workdir, "vrooli --version")),
		},
	}, nil
}

// RunSetup executes VPS setup without progress tracking.
// This is a convenience wrapper around RunSetupWithProgress that uses no-op progress callbacks.
func RunSetup(ctx context.Context, manifest domain.CloudManifest, bundlePath string, sshRunner ssh.Runner, scpRunner ssh.SCPRunner) domain.VPSSetupResult {
	progress := 0.0
	return RunSetupWithProgress(ctx, manifest, bundlePath, sshRunner, scpRunner, NoopProgressHub{}, NoopProgressRepo{}, "", &progress)
}

// RunSetupWithProgress runs VPS setup with progress tracking.
func RunSetupWithProgress(
	ctx context.Context,
	manifest domain.CloudManifest,
	bundlePath string,
	sshRunner ssh.Runner,
	scpRunner ssh.SCPRunner,
	hub ProgressBroadcaster,
	repo ProgressRepo,
	deploymentID string,
	progress *float64,
) domain.VPSSetupResult {
	start := time.Now()

	if _, err := os.Stat(bundlePath); err != nil {
		return domain.VPSSetupResult{OK: false, Error: fmt.Sprintf("bundle_path not accessible: %v", err), DurationMs: time.Since(start).Milliseconds(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	steps, err := BuildSetupPlan(manifest, bundlePath)
	if err != nil {
		return domain.VPSSetupResult{OK: false, Error: err.Error(), DurationMs: time.Since(start).Milliseconds(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	cfg := ssh.ConfigFromManifest(manifest)

	remoteBundleDir := shellutil.SafeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud", "bundles")
	remoteBundlePath := shellutil.SafeRemoteJoin(remoteBundleDir, filepath.Base(bundlePath))
	autohealConfigPath := shellutil.SafeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud", "autoheal-scope.json")
	cleanupCommand, err := buildScenarioCleanupCommand(manifest)
	if err != nil {
		return domain.VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), FailedStep: "cleanup_scenarios", DurationMs: time.Since(start).Milliseconds(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	// Helper to emit progress
	emit := func(eventType, stepID, stepTitle string) {
		event := NewProgressEvent(eventType, stepID, stepTitle, *progress)
		hub.Broadcast(deploymentID, event)
		if err := repo.UpdateDeploymentProgress(ctx, deploymentID, stepID, *progress); err != nil {
			log.Printf("setup progress update failed (step=%s): %v", stepID, err)
		}
	}

	// Helper to emit error and return failed result
	failStep := func(stepID, stepTitle string, err error) domain.VPSSetupResult {
		errMsg := err.Error()
		var info *domain.ErrorInfo
		var sshErr *ssh.SSHError
		if errors.As(err, &sshErr) {
			info = ssh.ErrorInfoFromSSHError(sshErr)
		}
		event := NewStructuredErrorEvent(stepID, stepTitle, *progress, errMsg, info)
		hub.Broadcast(deploymentID, event)
		return domain.VPSSetupResult{OK: false, Steps: steps, Error: errMsg, ErrorInfo: info, FailedStep: stepID, DurationMs: time.Since(start).Milliseconds(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	// runStep executes an SSH command with per-step timeout configuration.
	runStep := func(stepID, cmd string) error {
		if err := shellutil.ValidateTildeExpansion(cmd); err != nil {
			return err
		}
		return RunStepWithRetry(ctx, sshRunner, cfg, stepID, cmd)
	}

	// Step: mkdir
	emit("step_started", "mkdir", "Creating directories")
	if err := runStep("mkdir", fmt.Sprintf("mkdir -p %s %s", shellutil.QuoteSingle(manifest.Target.VPS.Workdir), shellutil.QuoteSingle(remoteBundleDir))); err != nil {
		return failStep("mkdir", "Creating directories", err)
	}
	*progress += StepWeights["mkdir"]
	emit("step_completed", "mkdir", "Creating directories")

	// Step: bootstrap
	emit("step_started", "bootstrap", "Installing prerequisites")
	if err := runStep("bootstrap", buildBootstrapCommand(manifest)); err != nil {
		return failStep("bootstrap", "Installing prerequisites", err)
	}
	*progress += StepWeights["bootstrap"]
	emit("step_completed", "bootstrap", "Installing prerequisites")

	// Step: upload
	emit("step_started", "upload", "Uploading bundle")
	if err := scpRunner.Copy(ctx, cfg, bundlePath, remoteBundlePath, ssh.DefaultSCPOptions()); err != nil {
		return failStep("upload", "Uploading bundle", err)
	}
	*progress += StepWeights["upload"]
	emit("step_completed", "upload", "Uploading bundle")

	// Step: cleanup_scenarios
	emit("step_started", "cleanup_scenarios", "Cleaning scenario code targets")
	if err := runStep("cleanup_scenarios", cleanupCommand); err != nil {
		return failStep("cleanup_scenarios", "Cleaning scenario code targets", err)
	}
	*progress += StepWeights["cleanup_scenarios"]
	emit("step_completed", "cleanup_scenarios", "Cleaning scenario code targets")

	// Step: extract
	emit("step_started", "extract", "Extracting bundle")
	if err := runStep("extract", fmt.Sprintf("tar -xzf %s -C %s", shellutil.QuoteSingle(remoteBundlePath), shellutil.QuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return failStep("extract", "Extracting bundle", err)
	}
	*progress += StepWeights["extract"]
	emit("step_completed", "extract", "Extracting bundle")

	// Step: install_vrooli
	emit("step_started", "install_vrooli", "Installing native vrooli CLI")
	if err := installRemoteVrooliCLI(ctx, cfg, manifest.Target.VPS.Workdir, sshRunner, scpRunner); err != nil {
		return failStep("install_vrooli", "Installing native vrooli CLI", err)
	}
	*progress += StepWeights["install_vrooli"]
	emit("step_completed", "install_vrooli", "Installing native vrooli CLI")

	// Step: setup (production mode - skips dev tools, installs only required resources)
	emit("step_started", "setup", "Running setup")
	if err := runStep("setup", buildVrooliSetupCommand(manifest.Target.VPS.Workdir)); err != nil {
		return failStep("setup", "Running setup", err)
	}
	*progress += StepWeights["setup"]
	emit("step_completed", "setup", "Running setup")

	// Step: autoheal
	emit("step_started", "autoheal", "Configuring autoheal")
	if err := runStep("autoheal", fmt.Sprintf("mkdir -p %s && printf '%%s' %s > %s", shellutil.QuoteSingle(shellutil.SafeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud")), shellutil.QuoteSingle(minimalAutohealScopeJSON(manifest)), shellutil.QuoteSingle(autohealConfigPath))); err != nil {
		return failStep("autoheal", "Configuring autoheal", err)
	}
	*progress += StepWeights["autoheal"]
	emit("step_completed", "autoheal", "Configuring autoheal")

	// Step: verify
	emit("step_started", "verify_setup", "Verifying installation")
	if err := runStep("verify_setup", shellutil.VrooliCommand(manifest.Target.VPS.Workdir, "vrooli --version")); err != nil {
		return failStep("verify_setup", "Verifying installation", err)
	}
	*progress += StepWeights["verify_setup"]
	emit("step_completed", "verify_setup", "Verifying installation")

	return domain.VPSSetupResult{OK: true, Steps: steps, DurationMs: time.Since(start).Milliseconds(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

func minimalAutohealScopeJSON(manifest domain.CloudManifest) string {
	payload := map[string]interface{}{
		"schema_version": "1.0.0",
		"environment":    manifest.Environment,
		"scenario_id":    manifest.Scenario.ID,
		"resources":      manifest.Dependencies.Resources,
		"scenarios":      manifest.Dependencies.Scenarios,
		"generated_at":   time.Now().UTC().Format(time.RFC3339),
	}
	b, _ := json.MarshalIndent(payload, "", "  ")
	return string(b)
}

func buildScenarioCleanupCommand(manifest domain.CloudManifest) (string, error) {
	workdir := strings.TrimSpace(manifest.Target.VPS.Workdir)
	if workdir == "" {
		return "", fmt.Errorf("target.vps.workdir is required for cleanup")
	}

	targetScenarios := stableUniqueValues(manifest.Bundle.Scenarios)
	if len(targetScenarios) == 0 {
		targetScenarios = stableUniqueValues([]string{manifest.Scenario.ID})
	}
	if len(targetScenarios) == 0 {
		return "", fmt.Errorf("cleanup requires at least one scenario id")
	}

	preserveByScenario := map[string][]string{}
	for _, fullPath := range stableUniqueValues(manifest.Target.VPS.PreservePaths) {
		clean := path.Clean(strings.TrimSpace(fullPath))
		parts := strings.Split(clean, "/")
		if len(parts) < 3 || parts[0] != "scenarios" {
			return "", fmt.Errorf("target.vps.preserve_paths entry %q must be scenarios/<scenario-id>/<path>", fullPath)
		}
		scenarioID := parts[1]
		rel := strings.Join(parts[2:], "/")
		if scenarioID == "" || rel == "" {
			return "", fmt.Errorf("target.vps.preserve_paths entry %q must include a relative subpath", fullPath)
		}
		preserveByScenario[scenarioID] = append(preserveByScenario[scenarioID], rel)
	}
	for scenarioID, paths := range preserveByScenario {
		preserveByScenario[scenarioID] = stableUniqueValues(paths)
	}

	mutableNames := []string{"data", "uploads", "storage", "state", "cache", "logs", "runtime", "tmp", "files"}
	var script strings.Builder
	script.WriteString("set -euo pipefail; ")
	script.WriteString("PRESERVE_ROOT=" + shellutil.QuoteSingle(shellutil.SafeRemoteJoin(workdir, ".vrooli", "cloud", "preserve")) + "; ")
	script.WriteString("mkdir -p \"$PRESERVE_ROOT\"; ")

	for _, scenarioID := range targetScenarios {
		scenarioDir := shellutil.SafeRemoteJoin(workdir, "scenarios", scenarioID)
		script.WriteString("SCENARIO_DIR=" + shellutil.QuoteSingle(scenarioDir) + "; ")
		script.WriteString("if [ -d \"$SCENARIO_DIR\" ]; then ")
		relPreserve := preserveByScenario[scenarioID]
		if len(relPreserve) == 0 {
			script.WriteString("PRESERVE_REL_LIST=''; ")
		} else {
			script.WriteString("PRESERVE_REL_LIST=" + shellutil.QuoteSingle(strings.Join(relPreserve, "\n")) + "; ")
		}
		// Block cleanup when known mutable paths exist under scenario directories but are not explicitly preserved.
		script.WriteString("MUTABLE_REL_LIST=$(find \"$SCENARIO_DIR\" -mindepth 1 -maxdepth 5 -type d \\( ")
		for idx, name := range mutableNames {
			if idx > 0 {
				script.WriteString(" -o ")
			}
			script.WriteString("-name " + shellutil.QuoteSingle(name))
		}
		script.WriteString(" \\) -printf '%P\\n' | sort -u); ")
		if len(relPreserve) == 0 {
			// Backward-compatible safety for legacy deployments: preserve detected mutable directories.
			script.WriteString("PRESERVE_REL_LIST=\"$MUTABLE_REL_LIST\"; ")
		} else {
			script.WriteString("printf '%s\\n' \"$MUTABLE_REL_LIST\" | while IFS= read -r rel; do ")
			script.WriteString("[ -z \"$rel\" ] && continue; covered=0; ")
			script.WriteString("for keep in $PRESERVE_REL_LIST; do ")
			script.WriteString("case \"$rel\" in \"$keep\"|\"$keep\"/*) covered=1; break;; esac; ")
			script.WriteString("case \"$keep\" in \"$rel\"/*) covered=1; break;; esac; ")
			script.WriteString("done; ")
			script.WriteString("if [ \"$covered\" -eq 0 ]; then echo \"cleanup blocked: mutable path '$SCENARIO_DIR/$rel' must be added to target.vps.preserve_paths\" >&2; exit 1; fi; ")
			script.WriteString("done; ")
		}

		tarPath := shellutil.SafeRemoteJoin(workdir, ".vrooli", "cloud", "preserve", scenarioID+".tar")
		script.WriteString("TAR_PATH=" + shellutil.QuoteSingle(tarPath) + "; rm -f \"$TAR_PATH\"; ")
		script.WriteString("if [ -n \"$PRESERVE_REL_LIST\" ]; then ")
		script.WriteString("(cd \"$SCENARIO_DIR\" && printf '%s\\n' \"$PRESERVE_REL_LIST\" | sed '/^$/d' | tr '\\n' '\\0' | xargs -0 -r tar -cf \"$TAR_PATH\" --ignore-failed-read --); ")
		script.WriteString("fi; ")
		script.WriteString("rm -rf \"$SCENARIO_DIR\"; mkdir -p \"$SCENARIO_DIR\"; ")
		script.WriteString("if [ -f \"$TAR_PATH\" ]; then tar -xf \"$TAR_PATH\" -C \"$SCENARIO_DIR\"; fi; ")
		script.WriteString("fi; ")
	}

	return script.String(), nil
}

func stableUniqueValues(in []string) []string {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(in))
	for _, value := range in {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}
