package deployments

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

// OrchestrationStep mirrors the API response.
type OrchestrationStep struct {
	Name     string `json:"name"`
	Status   string `json:"status"`
	Duration string `json:"duration,omitempty"`
	Message  string `json:"message,omitempty"`
	Error    string `json:"error,omitempty"`
}

// DeployDesktopResponse mirrors the API response.
type DeployDesktopResponse struct {
	Status           string              `json:"status"`
	ProfileID        string              `json:"profile_id"`
	Scenario         string              `json:"scenario"`
	Steps            []OrchestrationStep `json:"steps"`
	ManifestPath     string              `json:"manifest_path,omitempty"`
	BuildResults     *BuildAllResult     `json:"build_results,omitempty"`
	DesktopBuildID   string              `json:"desktop_build_id,omitempty"`
	DesktopPath      string              `json:"desktop_path,omitempty"`
	InstallerBuildID string              `json:"installer_build_id,omitempty"`
	Installers       map[string]string   `json:"installers,omitempty"`
	Duration         string              `json:"duration,omitempty"`
	NextSteps        []string            `json:"next_steps,omitempty"`
}

// DeployDesktop orchestrates the complete bundled desktop deployment workflow.
func (c *Commands) DeployDesktop(args []string) error {
	fs := flag.NewFlagSet("deploy-desktop", flag.ContinueOnError)
	profileID := fs.String("profile", "", "profile ID (required)")
	outputDir := fs.String("output", "", "output directory for bundle (defaults to scenario/platforms/electron/bundle)")
	platforms := fs.String("platforms", "", "comma-separated platforms to build (win,mac,linux)")
	skipBuild := fs.Bool("skip-build", false, "skip binary compilation")
	skipValidation := fs.Bool("skip-validation", false, "skip profile validation")
	skipPackaging := fs.Bool("skip-packaging", false, "skip Electron wrapper generation (manifest + binaries only)")
	skipInstallers := fs.Bool("skip-installers", false, "skip building platform installers (MSI/PKG/AppImage)")
	deploymentMode := fs.String("mode", "bundled", "deployment mode: bundled (offline), external-server (thin client), cloud-api")
	dryRun := fs.Bool("dry-run", false, "show what would be done without doing it")
	format := fs.String("format", "", "output format (json)")
	signingConfig := fs.String("signing-config", "", "path to JSON file with signing configuration (applies to scenario-to-desktop)")
	timeout := fs.Duration("timeout", 10*time.Minute, "timeout for the deploy operation (e.g. 10m, 15m)")
	fs.Usage = deployDesktopUsage(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()

	if *profileID == "" && len(remaining) > 0 {
		*profileID = remaining[0]
	}
	if *profileID == "" {
		fs.Usage()
		return errors.New("--profile is required")
	}

	payload := buildDeployPayload(*profileID, *outputDir, *platforms, *skipBuild, *skipValidation, *skipPackaging, *skipInstallers, *deploymentMode, *dryRun, timeout)

	if *signingConfig != "" {
		signingData, err := os.ReadFile(*signingConfig)
		if err != nil {
			return fmt.Errorf("failed to read signing config file: %w", err)
		}
		var signingPayload map[string]interface{}
		if err := json.Unmarshal(signingData, &signingPayload); err != nil {
			return fmt.Errorf("failed to parse signing config JSON: %w", err)
		}
		payload["signing_config"] = signingPayload
	}

	body, err := c.api.Request("POST", "/api/v1/deploy-desktop", nil, payload)
	if err != nil {
		c.printFallbackArtifacts(*profileID)
		return fmt.Errorf("deploy-desktop failed: %w", err)
	}

	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) != "json" {
		var resp DeployDesktopResponse
		if err := json.Unmarshal(body, &resp); err == nil {
			printDeployDesktopResults(resp)
			return nil
		}
	}

	cmdutil.PrintByFormat(formatVal, body)
	return nil
}

func deployDesktopUsage(fs *flag.FlagSet) func() {
	return func() {
		fmt.Fprintf(os.Stderr, `Orchestrate complete bundled desktop deployment.

This command runs the full desktop deployment workflow:
  1. Load and validate the deployment profile
  2. Apply signing configuration (if provided)
  3. Assemble bundle manifest with profile swaps applied
  4. Export manifest to bundle directory
  5. Cross-compile service binaries for all platforms
  6. Generate Electron wrapper via scenario-to-desktop
  7. Build platform installers (MSI/PKG/AppImage)
  8. Return installer paths and next steps

Usage:
  deployment-manager deploy-desktop --profile <profile-id> [flags]

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Deployment Modes:
  bundled          Self-contained offline app (UI + API + resources)
  external-server  Thin client connecting to running Vrooli server
  cloud-api        Cloud-hosted API backend

Signing Configuration:
  Use --signing-config to provide a JSON file with signing settings.
  The config is applied to scenario-to-desktop before building.
  See 'scenario-to-desktop signing help' for config format.

Examples:
  # Full end-to-end deployment (assembles manifest, builds binaries, generates Electron, builds installers)
  deployment-manager deploy-desktop --profile my-desktop-profile

  # Bundled offline app for specific platforms
  deployment-manager deploy-desktop --profile my-profile --platforms win,mac

  # With signing configuration for signed installers
  deployment-manager deploy-desktop --profile my-profile --signing-config ./signing.json

  # Thin client mode (UI only, connects to server)
  deployment-manager deploy-desktop --profile my-profile --mode external-server

  # Manifest + binaries only (no Electron packaging)
  deployment-manager deploy-desktop --profile my-profile --skip-packaging

  # Generate Electron wrapper but skip installer builds
  deployment-manager deploy-desktop --profile my-profile --skip-installers

  # Dry run to preview all steps
  deployment-manager deploy-desktop --profile my-profile --dry-run
`)
	}
}

func buildDeployPayload(profileID, outputDir, platforms string, skipBuild, skipValidation, skipPackaging, skipInstallers bool, deploymentMode string, dryRun bool, timeout *time.Duration) map[string]interface{} {
	payload := map[string]interface{}{
		"profile_id":      profileID,
		"skip_build":      skipBuild,
		"skip_validation": skipValidation,
		"skip_packaging":  skipPackaging,
		"skip_installers": skipInstallers,
		"deployment_mode": deploymentMode,
		"dry_run":         dryRun,
		"timeout_seconds": int(timeout.Seconds()),
	}
	if outputDir != "" {
		payload["output_dir"] = outputDir
	}
	if platforms != "" {
		platformList := strings.Split(platforms, ",")
		for i, p := range platformList {
			platformList[i] = strings.TrimSpace(p)
		}
		payload["platforms"] = platformList
	}
	return payload
}

// printFallbackArtifacts gives the user actionable next steps when the API call fails.
func (c *Commands) printFallbackArtifacts(profileID string) {
	body, err := c.api.Get(fmt.Sprintf("/api/v1/profiles/%s", profileID), nil)
	if err != nil {
		return
	}
	var resp map[string]interface{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return
	}
	scenario, _ := resp["scenario"].(string)
	if scenario == "" {
		return
	}

	distDir := filepath.Join("scenarios", scenario, "platforms", "electron", "dist-electron")
	patterns := []string{"*.AppImage", "*.deb", "*.rpm"}
	var found []string
	for _, p := range patterns {
		matches, _ := filepath.Glob(filepath.Join(distDir, p))
		found = append(found, matches...)
	}
	if len(found) == 0 {
		fmt.Fprintf(os.Stderr, "⚠️  API call failed, and no local artifacts found under %s\n", distDir)
		return
	}
	fmt.Fprintf(os.Stderr, "⚠️  API call failed; check these locally-built artifacts (likely completed):\n")
	for _, f := range found {
		fmt.Fprintf(os.Stderr, "  - %s\n", f)
	}
	fmt.Fprintf(os.Stderr, "If present, import the public key from signing/gnupg or public-key.asc to verify signatures.\n")
}

func printDeployDesktopResults(resp DeployDesktopResponse) {
	report := cliapp.OperationalReport{
		Status: []string{
			fmt.Sprintf("Desktop deployment: %s", resp.Status),
			fmt.Sprintf("Profile: %s", resp.ProfileID),
			fmt.Sprintf("Scenario: %s", resp.Scenario),
		},
	}
	if resp.Status != "success" {
		report.Status[0] = fmt.Sprintf("Desktop deployment: %s", resp.Status)
	}
	if resp.Duration != "" {
		report.Status = append(report.Status, fmt.Sprintf("Duration: %s", resp.Duration))
	}
	stepsGroup := cliapp.TriageGroup{Heading: "Workflow Steps"}
	for _, step := range resp.Steps {
		line := fmt.Sprintf("%s status=%s", step.Name, step.Status)
		if step.Message != "" {
			line += fmt.Sprintf(" message=%s", step.Message)
		}
		if step.Error != "" {
			line += fmt.Sprintf(" error=%s", step.Error)
		}
		stepsGroup.Items = append(stepsGroup.Items, line)
	}
	report.Triage = append(report.Triage, stepsGroup)

	if resp.ManifestPath != "" {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Manifest",
			Items:   []string{resp.ManifestPath},
		})
	}

	if resp.BuildResults != nil && len(resp.BuildResults.Results) > 0 {
		successCount := 0
		group := cliapp.TriageGroup{Heading: "Build Results"}
		for _, r := range resp.BuildResults.Results {
			if r.Success {
				successCount++
			}
			line := fmt.Sprintf("%s success=%v output=%s", r.Platform, r.Success, r.OutputPath)
			if r.Error != "" {
				line += fmt.Sprintf(" error=%s", r.Error)
			}
			group.Items = append(group.Items, line)
		}
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Binary Summary",
			Items:   []string{fmt.Sprintf("%d/%d succeeded", successCount, len(resp.BuildResults.Results))},
		})
		report.Triage = append(report.Triage, group)
	}

	if resp.DesktopPath != "" {
		report.Triage = append(report.Triage, cliapp.TriageGroup{
			Heading: "Desktop Wrapper",
			Items:   []string{resp.DesktopPath},
		})
	}

	if len(resp.Installers) > 0 {
		group := cliapp.TriageGroup{Heading: "Installers"}
		for platform, path := range resp.Installers {
			group.Items = append(group.Items, fmt.Sprintf("%s: %s", platform, path))
		}
		report.Triage = append(report.Triage, group)
	}
	if len(resp.NextSteps) > 0 {
		report.NextSteps = append(report.NextSteps, resp.NextSteps...)
	}
	_ = cliapp.RenderOperationalReport(os.Stdout, report)
}
