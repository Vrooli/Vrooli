package vps

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
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
			ID:          "extract",
			Title:       "Extract bundle",
			Description: "Extract the tarball into the deployment workdir.",
			Command:     ssh.LocalSSHCommand(cfg, fmt.Sprintf("tar -xzf %s -C %s", shellutil.QuoteSingle(remoteBundlePath), shellutil.QuoteSingle(manifest.Target.VPS.Workdir))),
		},
		{
			ID:          "setup",
			Title:       "Run Vrooli setup",
			Description: "Runs production setup with only required resources.",
			Command:     ssh.LocalSSHCommand(cfg, fmt.Sprintf("cd %s && ./scripts/manage.sh setup --yes yes --environment production", shellutil.QuoteSingle(manifest.Target.VPS.Workdir))),
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
			Command:     ssh.LocalSSHCommand(cfg, fmt.Sprintf("cd %s && vrooli --version", shellutil.QuoteSingle(manifest.Target.VPS.Workdir))),
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

	// Step: extract
	emit("step_started", "extract", "Extracting bundle")
	if err := runStep("extract", fmt.Sprintf("tar -xzf %s -C %s", shellutil.QuoteSingle(remoteBundlePath), shellutil.QuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return failStep("extract", "Extracting bundle", err)
	}
	*progress += StepWeights["extract"]
	emit("step_completed", "extract", "Extracting bundle")

	// Step: setup (production mode - skips dev tools, installs only required resources)
	emit("step_started", "setup", "Running setup")
	if err := runStep("setup", fmt.Sprintf("cd %s && ./scripts/manage.sh setup --yes yes --environment production", shellutil.QuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
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
	if err := runStep("verify_setup", fmt.Sprintf("cd %s && vrooli --version", shellutil.QuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
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
