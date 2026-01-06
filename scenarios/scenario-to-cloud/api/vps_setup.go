package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"scenario-to-cloud/domain"
)

// Type aliases for backward compatibility and shorter references within main package.
type (
	VPSPlanStep    = domain.VPSPlanStep
	VPSSetupResult = domain.VPSSetupResult
)

// bootstrapCommand installs system prerequisites on a fresh VPS.
// Uses noninteractive mode to prevent apt/debconf from hanging on prompts.
const bootstrapCommand = `export DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a && ` +
	`apt-get update -qq && ` +
	`apt-get install -y -qq curl git unzip tar jq ca-certificates gnupg lsb-release`
const firewallInboundCommand = "command -v ufw >/dev/null 2>/dev/null && { ufw allow 80/tcp; ufw allow 443/tcp; ufw reload; } || true"

func buildBootstrapCommand(manifest CloudManifest) string {
	command := bootstrapCommand
	if manifest.Edge.Caddy.Enabled {
		command = fmt.Sprintf("%s && %s", command, firewallInboundCommand)
	}
	return command
}

// VPSSetupRequest is the request body for VPS setup.
type VPSSetupRequest struct {
	Manifest   CloudManifest `json:"manifest"`
	BundlePath string        `json:"bundle_path"`
}

func BuildVPSSetupPlan(manifest CloudManifest, bundlePath string) ([]VPSPlanStep, error) {
	bundlePath = strings.TrimSpace(bundlePath)
	if bundlePath == "" {
		return nil, fmt.Errorf("bundle_path is required")
	}
	cfg := sshConfigFromManifest(manifest)

	remoteBundleDir := safeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud", "bundles")
	remoteBundlePath := safeRemoteJoin(remoteBundleDir, filepath.Base(bundlePath))

	autohealConfigPath := safeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud", "autoheal-scope.json")

	return []VPSPlanStep{
		{
			ID:          "mkdir",
			Title:       "Create remote directories",
			Description: "Ensure the deployment workdir and bundle directory exist.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("mkdir -p %s %s", shellQuoteSingle(manifest.Target.VPS.Workdir), shellQuoteSingle(remoteBundleDir))),
		},
		{
			ID:          "bootstrap",
			Title:       "Install system prerequisites",
			Description: "Update apt and install required packages (curl, git, unzip, etc.)",
			Command:     localSSHCommand(cfg, buildBootstrapCommand(manifest)),
		},
		{
			ID:          "upload",
			Title:       "Upload bundle",
			Description: "Copy the mini-Vrooli tarball to the VPS via scp.",
			Command:     localSCPCommand(cfg, bundlePath, remoteBundlePath),
		},
		{
			ID:          "extract",
			Title:       "Extract bundle",
			Description: "Extract the tarball into the deployment workdir.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("tar -xzf %s -C %s", shellQuoteSingle(remoteBundlePath), shellQuoteSingle(manifest.Target.VPS.Workdir))),
		},
		{
			ID:          "setup",
			Title:       "Run Vrooli setup",
			Description: "Runs production setup with only required resources.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("cd %s && ./scripts/manage.sh setup --yes yes --environment production", shellQuoteSingle(manifest.Target.VPS.Workdir))),
		},
		{
			ID:          "autoheal",
			Title:       "Write autoheal scope config",
			Description: "Writes a minimal config so vrooli-autoheal can scope checks to this deployment.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("mkdir -p %s && printf '%%s' %s > %s", shellQuoteSingle(safeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud")), shellQuoteSingle(minimalAutohealScopeJSON(manifest)), shellQuoteSingle(autohealConfigPath))),
		},
		{
			ID:          "verify",
			Title:       "Verify vrooli CLI",
			Description: "Sanity check that vrooli runs within the deployment directory.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("cd %s && vrooli --version", shellQuoteSingle(manifest.Target.VPS.Workdir))),
		},
	}, nil
}

// noopSetupProgressHub is a no-op implementation of progress broadcasting for RunVPSSetup.
type noopSetupProgressHub struct{}

func (noopSetupProgressHub) Broadcast(string, ProgressEvent) {}

// noopSetupProgressRepo is a no-op implementation of progress persistence for RunVPSSetup.
type noopSetupProgressRepo struct{}

func (noopSetupProgressRepo) UpdateDeploymentProgress(context.Context, string, string, float64) error {
	return nil
}

// RunVPSSetup executes VPS setup without progress tracking.
// This is a convenience wrapper around RunVPSSetupWithProgress that uses no-op progress callbacks.
func RunVPSSetup(ctx context.Context, manifest CloudManifest, bundlePath string, sshRunner SSHRunner, scpRunner SCPRunner) VPSSetupResult {
	progress := 0.0
	return RunVPSSetupWithProgress(ctx, manifest, bundlePath, sshRunner, scpRunner, noopSetupProgressHub{}, noopSetupProgressRepo{}, "", &progress)
}

// ProgressRepo is an interface for persisting progress.
type ProgressRepo interface {
	UpdateDeploymentProgress(ctx context.Context, id, step string, percent float64) error
}

// RunVPSSetupWithProgress runs VPS setup with progress tracking.
func RunVPSSetupWithProgress(
	ctx context.Context,
	manifest CloudManifest,
	bundlePath string,
	sshRunner SSHRunner,
	scpRunner SCPRunner,
	hub ProgressBroadcaster,
	repo ProgressRepo,
	deploymentID string,
	progress *float64,
) VPSSetupResult {
	if _, err := os.Stat(bundlePath); err != nil {
		return VPSSetupResult{OK: false, Error: fmt.Sprintf("bundle_path not accessible: %v", err), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	steps, err := BuildVPSSetupPlan(manifest, bundlePath)
	if err != nil {
		return VPSSetupResult{OK: false, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	cfg := sshConfigFromManifest(manifest)

	remoteBundleDir := safeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud", "bundles")
	remoteBundlePath := safeRemoteJoin(remoteBundleDir, filepath.Base(bundlePath))
	autohealConfigPath := safeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud", "autoheal-scope.json")

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
			log.Printf("setup progress update failed (step=%s): %v", stepID, err)
		}
	}

	// Helper to emit error and return failed result
	failStep := func(stepID, stepTitle, errMsg string) VPSSetupResult {
		event := ProgressEvent{
			Type:      "deployment_error",
			Step:      stepID,
			StepTitle: stepTitle,
			Progress:  *progress,
			Error:     errMsg,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
		}
		hub.Broadcast(deploymentID, event)
		return VPSSetupResult{OK: false, Steps: steps, Error: errMsg, FailedStep: stepID, Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	// run executes an SSH command and returns an error with output context if it fails.
	run := func(cmd string) error {
		return RunSSHWithOutput(ctx, sshRunner, cfg, cmd)
	}

	// Step: mkdir
	emit("step_started", "mkdir", "Creating directories")
	if err := run(fmt.Sprintf("mkdir -p %s %s", shellQuoteSingle(manifest.Target.VPS.Workdir), shellQuoteSingle(remoteBundleDir))); err != nil {
		return failStep("mkdir", "Creating directories", err.Error())
	}
	*progress += StepWeights["mkdir"]
	emit("step_completed", "mkdir", "Creating directories")

	// Step: bootstrap
	emit("step_started", "bootstrap", "Installing prerequisites")
	if err := run(buildBootstrapCommand(manifest)); err != nil {
		return failStep("bootstrap", "Installing prerequisites", err.Error())
	}
	*progress += StepWeights["bootstrap"]
	emit("step_completed", "bootstrap", "Installing prerequisites")

	// Step: upload
	emit("step_started", "upload", "Uploading bundle")
	if err := scpRunner.Copy(ctx, cfg, bundlePath, remoteBundlePath); err != nil {
		return failStep("upload", "Uploading bundle", err.Error())
	}
	*progress += StepWeights["upload"]
	emit("step_completed", "upload", "Uploading bundle")

	// Step: extract
	emit("step_started", "extract", "Extracting bundle")
	if err := run(fmt.Sprintf("tar -xzf %s -C %s", shellQuoteSingle(remoteBundlePath), shellQuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return failStep("extract", "Extracting bundle", err.Error())
	}
	*progress += StepWeights["extract"]
	emit("step_completed", "extract", "Extracting bundle")

	// Step: setup (production mode - skips dev tools, installs only required resources)
	emit("step_started", "setup", "Running setup")
	if err := run(fmt.Sprintf("cd %s && ./scripts/manage.sh setup --yes yes --environment production", shellQuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return failStep("setup", "Running setup", err.Error())
	}
	*progress += StepWeights["setup"]
	emit("step_completed", "setup", "Running setup")

	// Step: autoheal
	emit("step_started", "autoheal", "Configuring autoheal")
	if err := run(fmt.Sprintf("mkdir -p %s && printf '%%s' %s > %s", shellQuoteSingle(safeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud")), shellQuoteSingle(minimalAutohealScopeJSON(manifest)), shellQuoteSingle(autohealConfigPath))); err != nil {
		return failStep("autoheal", "Configuring autoheal", err.Error())
	}
	*progress += StepWeights["autoheal"]
	emit("step_completed", "autoheal", "Configuring autoheal")

	// Step: verify
	emit("step_started", "verify_setup", "Verifying installation")
	if err := run(fmt.Sprintf("cd %s && vrooli --version", shellQuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return failStep("verify_setup", "Verifying installation", err.Error())
	}
	*progress += StepWeights["verify_setup"]
	emit("step_completed", "verify_setup", "Verifying installation")

	return VPSSetupResult{OK: true, Steps: steps, Timestamp: time.Now().UTC().Format(time.RFC3339)}
}

func minimalAutohealScopeJSON(manifest CloudManifest) string {
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
