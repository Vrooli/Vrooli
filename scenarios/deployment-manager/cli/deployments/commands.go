package deployments

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

type Commands struct {
	api *cliutil.APIClient
}

func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

type DeploymentLog struct {
	Timestamp string `json:"timestamp"`
	Level     string `json:"level"`
	Message   string `json:"message"`
}

// Deployments

func (c *Commands) Deploy(args []string) error {
	fs := flag.NewFlagSet("deploy", flag.ContinueOnError)
	dryRun := fs.Bool("dry-run", false, "dry run")
	async := fs.Bool("async", false, "async deploy")
	validateOnly := fs.Bool("validate-only", false, "validate only")
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	if *validateOnly {
		return c.Validate([]string{id})
	}
	payload := map[string]interface{}{
		"dry_run": *dryRun,
		"async":   *async,
	}
	body, err := c.api.Request("POST", "/api/v1/deploy/"+id, nil, payload)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) Deployment(args []string) error {
	if len(args) == 0 {
		return errors.New("deployment subcommand is required")
	}
	sub := args[0]
	rest := args[1:]
	switch sub {
	case "status":
		return c.deploymentStatus(rest)
	default:
		return errors.New("unknown deployment subcommand: " + sub)
	}
}

func (c *Commands) deploymentStatus(args []string) error {
	fs := flag.NewFlagSet("deployment status", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("deployment id is required")
	}
	id := remaining[0]
	body, err := c.api.Get("/api/v1/deployments/"+id, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

// Packagers

func (c *Commands) Packagers(args []string) error {
	fs := flag.NewFlagSet("packagers", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	_, _ = cmdutil.ParseArgs(fs, args)

	message := "Packager discovery is deprecated here; use deploy-desktop or scenario-to-* CLIs directly."
	packagers := []string{
		"scenario-to-desktop (desktop bundler)",
		"scenario-to-ios (stub)",
		"scenario-to-cloud (stub)",
	}

	resolvedFormat := cmdutil.ResolveFormat(*format)
	if strings.ToLower(resolvedFormat) == "json" {
		payload := map[string]interface{}{
			"status":    "stubbed",
			"message":   message,
			"packagers": packagers,
		}
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}
		cmdutil.PrintByFormat(resolvedFormat, data)
		return nil
	}

	fmt.Println(message)
	fmt.Println("Available packagers (stubbed):")
	for _, p := range packagers {
		fmt.Printf(" - %s\n", p)
	}
	return nil
}

func (c *Commands) PackageProfile(args []string) error {
	fs := flag.NewFlagSet("package", flag.ContinueOnError)
	packager := fs.String("packager", "", "packager name")
	dryRun := fs.Bool("dry-run", false, "dry run")
	format := fs.String("format", "", "output format (json)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	if *packager == "" {
		return errors.New("--packager is required")
	}
	id := remaining[0]

	resolvedFormat := cmdutil.ResolveFormat(*format)
	response := map[string]interface{}{
		"status":     "stubbed",
		"profile_id": id,
		"packager":   *packager,
		"dry_run":    *dryRun,
		"message":    "Package command is legacy-only; use deploy-desktop for end-to-end bundling.",
	}
	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal package response: %w", err)
	}

	if strings.ToLower(resolvedFormat) == "json" {
		cmdutil.PrintByFormat(resolvedFormat, data)
		return nil
	}

	fmt.Fprintf(os.Stdout, "Packager hand-off is deprecated; use deploy-desktop instead.\n")
	fmt.Fprintf(os.Stdout, "Stubbed package request for profile %s via %s (dry-run=%t)\n", id, *packager, *dryRun)
	return nil
}

// Validation/Estimates

func (c *Commands) Validate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	verbose := fs.Bool("verbose", false, "verbose output")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	path := "/api/v1/profiles/" + id + "/validate"
	if *verbose {
		path += "?verbose=true"
	}
	body, err := c.api.Get(path, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

func (c *Commands) EstimateCost(args []string) error {
	fs := flag.NewFlagSet("estimate-cost", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	verbose := fs.Bool("verbose", false, "verbose output")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	path := "/api/v1/profiles/" + id + "/cost-estimate"
	if *verbose {
		path += "?verbose=true"
	}
	body, err := c.api.Get(path, nil)
	if err != nil {
		return err
	}
	cmdutil.PrintByFormat(*format, body)
	return nil
}

// Logs (deployment scoped)

func (c *Commands) Logs(args []string) error {
	fs := flag.NewFlagSet("logs", flag.ContinueOnError)
	level := fs.String("level", "", "log level filter")
	search := fs.String("search", "", "search term")
	format := fs.String("format", "", "output format (json|table)")
	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}
	if len(remaining) == 0 {
		return errors.New("profile id is required")
	}
	id := remaining[0]
	q := url.Values{}
	if *level != "" {
		q.Set("level", *level)
	}
	if *search != "" {
		q.Set("search", *search)
	}
	body, err := c.api.Get("/api/v1/logs/"+id, q)
	if err != nil {
		return err
	}
	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) == "table" {
		if err := printLogsTable(body); err == nil {
			return nil
		}
	}
	cmdutil.PrintByFormat(formatVal, body)
	return nil
}

func printLogsTable(body []byte) error {
	var logs []DeploymentLog
	if err := json.Unmarshal(body, &logs); err != nil {
		return err
	}
	rows := make([][]string, 0, len(logs))
	for _, entry := range logs {
		rows = append(rows, []string{
			entry.Timestamp,
			entry.Level,
			entry.Message,
		})
	}
	cmdutil.PrintTable([]string{"Timestamp", "Level", "Message"}, rows)
	return nil
}

// BuildResult mirrors the API response structure.
type BuildResult struct {
	Platform   string `json:"platform"`
	OutputPath string `json:"output_path"`
	Success    bool   `json:"success"`
	Error      string `json:"error,omitempty"`
}

// BuildAllResult mirrors the API response for all platforms.
type BuildAllResult struct {
	ServiceID    string        `json:"service_id"`
	Results      []BuildResult `json:"results"`
	AllSucceeded bool          `json:"all_succeeded"`
}

// BuildResponse mirrors the API build response.
type BuildResponse struct {
	Status   string           `json:"status"`
	Scenario string           `json:"scenario"`
	Results  []BuildAllResult `json:"results"`
	Duration string           `json:"duration,omitempty"`
	Message  string           `json:"message,omitempty"`
}

// Build cross-compiles service binaries for a profile.
func (c *Commands) Build(args []string) error {
	fs := flag.NewFlagSet("build", flag.ContinueOnError)
	profileID := fs.String("profile", "", "profile ID (required)")
	scenario := fs.String("scenario", "", "scenario name (optional if profile specified)")
	platforms := fs.String("platforms", "", "comma-separated platforms (linux-x64,darwin-arm64,win-x64)")
	services := fs.String("services", "", "comma-separated service IDs to build")
	dryRun := fs.Bool("dry-run", false, "show what would be built without building")
	format := fs.String("format", "", "output format (json)")
	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, `Cross-compile service binaries for desktop bundling.

This command builds service binaries for all target platforms using the
build configuration specified in each service's manifest.

Usage:
  deployment-manager build [flags]

Flags:
`)
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, `
Supported platforms:
  linux-x64     Linux x86_64
  linux-arm64   Linux ARM64
  darwin-x64    macOS Intel
  darwin-arm64  macOS Apple Silicon
  win-x64       Windows x86_64

Examples:
  # Build all services for all platforms using a profile
  deployment-manager build --profile my-desktop-profile

  # Build for specific platforms only
  deployment-manager build --profile my-profile --platforms linux-x64,darwin-arm64

  # Build specific services
  deployment-manager build --profile my-profile --services api,worker

  # Dry run to see what would be built
  deployment-manager build --profile my-profile --dry-run

Build configuration is read from each service's "build" field in the manifest:
  {
    "build": {
      "type": "go",                           // go, rust, npm, custom
      "source_dir": "api",                    // relative to scenario
      "entry_point": "./cmd/api",             // build target
      "output_pattern": "bin/{{platform}}/api{{ext}}",
      "args": ["-ldflags", "-s -w"],          // extra build args
      "env": {"CGO_ENABLED": "0"}             // build environment
    }
  }
`)
	}

	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}

	// Allow profile ID as positional argument
	if *profileID == "" && len(remaining) > 0 {
		*profileID = remaining[0]
	}

	if *profileID == "" && *scenario == "" {
		fs.Usage()
		return errors.New("--profile or --scenario is required")
	}

	payload := map[string]interface{}{
		"dry_run": *dryRun,
	}

	if *profileID != "" {
		payload["profile_id"] = *profileID
	}
	if *scenario != "" {
		payload["scenario"] = *scenario
	}

	if *platforms != "" {
		platformList := strings.Split(*platforms, ",")
		for i, p := range platformList {
			platformList[i] = strings.TrimSpace(p)
		}
		payload["platforms"] = platformList
	}

	if *services != "" {
		serviceList := strings.Split(*services, ",")
		for i, s := range serviceList {
			serviceList[i] = strings.TrimSpace(s)
		}
		payload["service_ids"] = serviceList
	}

	body, err := c.api.Request("POST", "/api/v1/build", nil, payload)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	// Pretty print results if not JSON format
	formatVal := cmdutil.ResolveFormat(*format)
	if strings.ToLower(formatVal) != "json" {
		var resp BuildResponse
		if err := json.Unmarshal(body, &resp); err == nil {
			printBuildResults(resp)
			return nil
		}
	}

	cmdutil.PrintByFormat(formatVal, body)
	return nil
}

func printBuildResults(resp BuildResponse) {
	fmt.Printf("Build %s for %s\n", resp.Status, resp.Scenario)
	if resp.Duration != "" {
		fmt.Printf("Duration: %s\n", resp.Duration)
	}
	if resp.Message != "" {
		fmt.Printf("Message: %s\n", resp.Message)
	}
	fmt.Println()

	for _, svcResult := range resp.Results {
		allOk := "✓"
		if !svcResult.AllSucceeded {
			allOk = "✗"
		}
		fmt.Printf("%s Service: %s\n", allOk, svcResult.ServiceID)

		for _, r := range svcResult.Results {
			status := "✓"
			if !r.Success {
				status = "✗"
			}
			fmt.Printf("  %s %-14s %s\n", status, r.Platform, r.OutputPath)
			if r.Error != "" {
				fmt.Printf("    Error: %s\n", r.Error)
			}
		}
		fmt.Println()
	}
}

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
	fs.Usage = func() {
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

	remaining, err := cmdutil.ParseArgs(fs, args)
	if err != nil {
		return err
	}

	// Allow profile ID as positional argument
	if *profileID == "" && len(remaining) > 0 {
		*profileID = remaining[0]
	}

	if *profileID == "" {
		fs.Usage()
		return errors.New("--profile is required")
	}

	payload := map[string]interface{}{
		"profile_id":      *profileID,
		"skip_build":      *skipBuild,
		"skip_validation": *skipValidation,
		"skip_packaging":  *skipPackaging,
		"skip_installers": *skipInstallers,
		"deployment_mode": *deploymentMode,
		"dry_run":         *dryRun,
		"timeout_seconds": int(timeout.Seconds()),
	}

	if *outputDir != "" {
		payload["output_dir"] = *outputDir
	}

	if *platforms != "" {
		platformList := strings.Split(*platforms, ",")
		for i, p := range platformList {
			platformList[i] = strings.TrimSpace(p)
		}
		payload["platforms"] = platformList
	}

	// Load and include signing config if provided
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

	// Pretty print results if not JSON format
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

// printFallbackArtifacts gives the user actionable next steps when the API call fails
// (e.g., EOF/timeout) but artifacts may still exist on disk.
func (c *Commands) printFallbackArtifacts(profileID string) {
	// Try to resolve scenario name from profile
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
	// Header
	statusIcon := "✓"
	if resp.Status != "success" {
		statusIcon = "✗"
	}
	fmt.Printf("%s Desktop Deployment: %s\n", statusIcon, resp.Status)
	fmt.Printf("  Profile:  %s\n", resp.ProfileID)
	fmt.Printf("  Scenario: %s\n", resp.Scenario)
	if resp.Duration != "" {
		fmt.Printf("  Duration: %s\n", resp.Duration)
	}
	fmt.Println()

	// Steps
	fmt.Println("Steps:")
	for _, step := range resp.Steps {
		icon := "○"
		switch step.Status {
		case "success":
			icon = "✓"
		case "failed":
			icon = "✗"
		case "skipped":
			icon = "⊘"
		case "running":
			icon = "◐"
		}
		fmt.Printf("  %s %s", icon, step.Name)
		if step.Message != "" {
			fmt.Printf(" - %s", step.Message)
		}
		if step.Error != "" {
			fmt.Printf(" [ERROR: %s]", step.Error)
		}
		fmt.Println()
	}
	fmt.Println()

	// Manifest path
	if resp.ManifestPath != "" {
		fmt.Printf("Manifest: %s\n", resp.ManifestPath)
	}

	// Build results summary
	if resp.BuildResults != nil && len(resp.BuildResults.Results) > 0 {
		successCount := 0
		for _, r := range resp.BuildResults.Results {
			if r.Success {
				successCount++
			}
		}
		fmt.Printf("Binaries: %d/%d succeeded\n", successCount, len(resp.BuildResults.Results))
	}

	// Desktop wrapper info
	if resp.DesktopPath != "" {
		fmt.Printf("Desktop Wrapper: %s\n", resp.DesktopPath)
	}

	// Installers
	if len(resp.Installers) > 0 {
		fmt.Println("\nInstallers:")
		for platform, path := range resp.Installers {
			fmt.Printf("  %-8s %s\n", platform+":", path)
		}
	}
	fmt.Println()

	// Next steps
	if len(resp.NextSteps) > 0 {
		fmt.Println("Next steps:")
		for _, step := range resp.NextSteps {
			fmt.Printf("  $ %s\n", step)
		}
	}
}
