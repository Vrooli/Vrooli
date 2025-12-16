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
	OK        bool         `json:"ok"`
	Steps     []VPSPlanStep `json:"steps"`
	Error     string       `json:"error,omitempty"`
	Timestamp string       `json:"timestamp"`
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
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := scpRunner.Copy(ctx, cfg, bundlePath, remoteBundlePath); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(fmt.Sprintf("tar -xzf %s -C %s", shellQuoteSingle(remoteBundlePath), shellQuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(fmt.Sprintf("cd %s && ./scripts/manage.sh setup --yes yes", shellQuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(fmt.Sprintf("mkdir -p %s && printf '%%s' %s > %s", shellQuoteSingle(safeRemoteJoin(manifest.Target.VPS.Workdir, ".vrooli", "cloud")), shellQuoteSingle(minimalAutohealScopeJSON(manifest)), shellQuoteSingle(autohealConfigPath))); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}
	if err := run(fmt.Sprintf("cd %s && vrooli --version", shellQuoteSingle(manifest.Target.VPS.Workdir))); err != nil {
		return VPSSetupResult{OK: false, Steps: steps, Error: err.Error(), Timestamp: time.Now().UTC().Format(time.RFC3339)}
	}

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
