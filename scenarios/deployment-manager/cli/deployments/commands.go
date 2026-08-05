package deployments

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"os"
	"strings"

	"deployment-manager/cli/cmdutil"

	"github.com/vrooli/cli-core/cliapp"
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
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
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
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
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

// Validation/Estimates

func (c *Commands) Validate(args []string) error {
	fs := flag.NewFlagSet("validate", flag.ContinueOnError)
	format := fs.String("format", "", "output format (json)")
	verbose := fs.Bool("verbose", false, "verbose output")
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
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
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
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
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()
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

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}
	remaining := fs.Args()

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
	report := cliapp.MutationReport{
		Result: []string{
			fmt.Sprintf("Build %s for %s", resp.Status, resp.Scenario),
		},
	}
	if resp.Duration != "" {
		report.Result = append(report.Result, fmt.Sprintf("Duration: %s", resp.Duration))
	}
	if resp.Message != "" {
		report.Result = append(report.Result, fmt.Sprintf("Message: %s", resp.Message))
	}

	for _, svcResult := range resp.Results {
		allOK := "success"
		if !svcResult.AllSucceeded {
			allOK = "partial-failure"
		}
		report.Changes = append(report.Changes, fmt.Sprintf("Service %s: %s", svcResult.ServiceID, allOK))

		for _, r := range svcResult.Results {
			status := "success"
			if !r.Success {
				status = "failed"
			}
			line := fmt.Sprintf("%s %s output=%s", r.Platform, status, r.OutputPath)
			if r.Error != "" {
				line += fmt.Sprintf(" error=%s", r.Error)
			}
			report.Changes = append(report.Changes, line)
		}
	}
	report.NextCommand = []string{
		"deployment-manager bundle assemble --profile <profile-id>",
		"deployment-manager deploy-desktop --profile <profile-id>",
	}
	_ = cliapp.RenderMutationReport(os.Stdout, report)
}
