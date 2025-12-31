package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// bootstrapCommand installs system prerequisites on a fresh VPS.
// Uses noninteractive mode to prevent apt/debconf from hanging on prompts.
const bootstrapCommand = `export DEBIAN_FRONTEND=noninteractive NEEDRESTART_MODE=a && ` +
	`apt-get update -qq && ` +
	`apt-get install -y -qq curl git unzip tar jq ca-certificates gnupg lsb-release`

type VPSSetupRequest struct {
	Manifest   CloudManifest `json:"manifest"`
	BundlePath string        `json:"bundle_path"`
}

type VPSPlanStep struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description,omitempty"`
	Command     string `json:"command,omitempty"`
}

type VPSSetupResult struct {
	OK         bool          `json:"ok"`
	Steps      []VPSPlanStep `json:"steps"`
	Error      string        `json:"error,omitempty"`
	FailedStep string        `json:"failed_step,omitempty"`
	Timestamp  string        `json:"timestamp"`
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
			Command:     localSSHCommand(cfg, bootstrapCommand),
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
			Description: "Runs ./scripts/manage.sh setup --yes yes inside the mini bundle.",
			Command:     localSSHCommand(cfg, fmt.Sprintf("cd %s && ./scripts/manage.sh setup --yes yes", shellQuoteSingle(manifest.Target.VPS.Workdir))),
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

func RunVPSSetup(ctx context.Context, manifest CloudManifest, bundlePath string, sshRunner SSHRunner, scpRunner SCPRunner) VPSSetupResult {
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

	run := func(cmd string) error {
		res, err := sshRunner.Run(ctx, cfg, cmd)
		if err != nil {
			if res.Stderr != "" {
				return fmt.Errorf("%w: %s", err, res.Stderr)
			}
			return err
		}
		return nil
	}

	if err := run(fmt.Sprintf("mkdir -p %s %s", shellQuoteSingle(manifest.Target.VPS.Workdir), shellQuoteSingle(remoteBundleDir))); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), FailedStep: "mkdir", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(bootstrapCommand); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), FailedStep: "bootstrap", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := scpRunner.Copy(ctx, cfg, bundlePath, remoteBundlePath); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), FailedStep: "upload", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(fmt.Sprintf("tar -xzf %s -C %s", shellQuoteSingle(remoteBundlePath), shellQuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), FailedStep: "extract", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(fmt.Sprintf("cd %s && ./scripts/manage.sh setup --yes yes", shellQuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), FailedStep: "setup", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(fmt.Sprintf("mkdir -p %s && printf '%%s' %s > %s", shellQuoteSingle(safeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud")), shellQuoteSingle(minimalAutohealScopeJSON(manifest)), shellQuoteSingle(autohealConfigPath))); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), FailedStep: "autoheal", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(fmt.Sprintf("cd %s && vrooli --version", shellQuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), FailedStep: "verify", Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

	return VPSSetupResult{OK: true, Steps: steps, Timestamp: time.Now().UTC().Format(time.RFC3339)}
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
	hub *ProgressHub,
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
			// Log but don't fail
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

	run := func(cmd string) error {
		res, err := sshRunner.Run(ctx, cfg, cmd)
		if err != nil {
			if res.Stderr != "" {
				return fmt.Errorf("%w: %s", err, res.Stderr)
			}
			return err
		}
		return nil
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
	if err := run(bootstrapCommand); err != nil {
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

	// Step: setup
	emit("step_started", "setup", "Running setup")
	if err := run(fmt.Sprintf("cd %s && ./scripts/manage.sh setup --yes yes", shellQuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
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
