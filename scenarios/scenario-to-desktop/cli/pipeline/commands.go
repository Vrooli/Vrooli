// Package pipeline provides CLI commands for pipeline management.
package pipeline

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"scenario-to-desktop/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

const appName = "scenario-to-desktop"

// ErrAlreadyPrinted is a sentinel error indicating the error was already printed to the user.
// This prevents duplicate error output when the error is returned up the call stack.
type ErrAlreadyPrinted struct {
	Err error
}

func (e *ErrAlreadyPrinted) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}
	return "error already printed"
}

func (e *ErrAlreadyPrinted) Unwrap() error {
	return e.Err
}

// IsAlreadyPrinted checks if an error is an ErrAlreadyPrinted.
func IsAlreadyPrinted(err error) bool {
	var ap *ErrAlreadyPrinted
	return errors.As(err, &ap)
}

// Commands provides pipeline CLI commands.
type Commands struct {
	api *cliutil.APIClient
}

// New creates a new pipeline Commands instance.
func New(api *cliutil.APIClient) *Commands {
	return &Commands{api: api}
}

func (c *Commands) apiPath(path string) string {
	return cmdutil.APIPath(path)
}

func (c *Commands) apiGet(path string, query map[string]string) ([]byte, error) {
	return c.api.Get(c.apiPath(path), cmdutil.MapToValues(query))
}

func (c *Commands) apiPost(path string, body interface{}) ([]byte, error) {
	return c.api.Request("POST", c.apiPath(path), nil, body)
}

func (c *Commands) waitForPipeline(pipelineID string, timeoutSeconds int, deployRequested bool, notice *versionUpdateNotice, showOutput bool) error {
	// Human-first progress: print only when the status/progress meaningfully changes.
	fmt.Printf("Pipeline: %s\n", pipelineID)

	deadline := time.Now().Add(time.Duration(timeoutSeconds) * time.Second)
	var lastKey string

	for {
		status, err := c.fetchPipelineStatus(pipelineID, false)
		if err != nil {
			return err
		}

		key := fmt.Sprintf("%s|%s|%d|%s", status.Status, status.CurrentStage, status.ProgressPercent, status.ProgressMessage)
		if key != lastKey {
			lastKey = key

			msg := strings.TrimSpace(status.ProgressMessage)
			stage := strings.TrimSpace(status.CurrentStage)
			if msg != "" && stage != "" {
				fmt.Printf("Status: %s (%d%%) stage=%s (%s)\n", status.Status, status.ProgressPercent, stage, msg)
			} else if stage != "" {
				fmt.Printf("Status: %s (%d%%) stage=%s\n", status.Status, status.ProgressPercent, stage)
			} else if msg != "" {
				fmt.Printf("Status: %s (%d%%) %s\n", status.Status, status.ProgressPercent, msg)
			} else {
				fmt.Printf("Status: %s (%d%%)\n", status.Status, status.ProgressPercent)
			}
		}

		switch status.Status {
		case "completed":
			// If a deploy was requested, fetch the verbose status so we can print derived update URLs
			// and other deploy details as part of the default success contract.
			if deployRequested {
				if verboseStatus, err := c.fetchPipelineStatus(status.PipelineID, true); err == nil {
					status = verboseStatus
				}
			}
			printPipelineSuccess(status, notice)
			return nil
		case "failed":
			fmt.Printf("Pipeline failed: %s\n", status.PipelineID)
			printPipelineError(status, showOutput)
			return &ErrAlreadyPrinted{Err: fmt.Errorf("pipeline failed")}
		}

		if time.Now().After(deadline) {
			fmt.Printf("Pipeline still running after %ds: %s\n", timeoutSeconds, pipelineID)
			fmt.Printf("Check status: %s pipeline-status %s --verbose\n", appName, pipelineID)
			return fmt.Errorf("pipeline timed out")
		}

		time.Sleep(2 * time.Second)
	}
}

// Run starts a new pipeline.
func (c *Commands) Run(args []string) error {
	fs := flag.NewFlagSet("pipeline-run", flag.ContinueOnError)
	stages := fs.String("stages", "", "Comma-separated stages to run")
	platforms := fs.String("platforms", "", "Comma-separated target platforms (default: current platform)")
	deploymentMode := fs.String("deployment-mode", "", "Deployment mode: bundled (default), external-server, cloud-api, proxy")
	locationMode := fs.String("location-mode", "", "Output location: proper (default), staging, temp")
	clean := fs.Bool("clean", false, "Remove existing desktop output before running the pipeline")
	version := fs.String("version", "", "Override version for this run (no file updates)")
	setVersion := fs.String("set-version", "", "Persist scenario version before running the pipeline")
	bumpVersion := fs.String("bump-version", "", "Bump scenario version (patch, minor, medium, major, auto) and persist")
	versionSource := fs.String("version-source", "both", "Version source to update when persisting: both, service, ui")
	allowDowngrade := fs.Bool("allow-downgrade", false, "Allow setting a version lower than the current scenario version")
	wait := fs.Bool("wait", false, "Block until pipeline completes (recommended for scripts/agents)")
	timeout := fs.Int("timeout", 600, "Max wait time in seconds (when --wait is used)")
	debug := fs.Bool("debug", false, "Show full JSON response on error")
	showOutput := fs.Bool("show-output", false, "Show app stdout/stderr on failure (useful for debugging)")
	deployTarget := fs.String("deploy-target", "", "Saved deploy target name from deploy-targets.json")
	deployTo := fs.String("deploy-to", "", "LPBS scenario name to deploy through (inline)")
	remoteProfile := fs.String("remote-profile", "", "Remote profile tag on the LPBS instance (inline)")
	appKey := fs.String("app-key", "", "App key for the download app in LPBS (required for deploy)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-run <scenario> [--deployment-mode bundled|external-server] [--location-mode proper|staging] [--clean] [--stages bundle,...] [--platforms win,mac,linux] [--version X.Y.Z | --set-version X.Y.Z | --bump-version patch|minor|medium|major|auto] [--deploy-target <name> | --deploy-to <scenario> --remote-profile <tag>] [--app-key <key>] [--wait] [--timeout N]")
	}

	scenario := fs.Args()[0]
	deployRequested := false
	req := map[string]interface{}{
		"scenario_name": scenario,
	}
	if *platforms != "" {
		req["platforms"] = strings.Split(*platforms, ",")
	}
	if *stages != "" {
		req["stages"] = strings.Split(*stages, ",")
	}
	if *deploymentMode != "" {
		req["deployment_mode"] = *deploymentMode
	}
	if *locationMode != "" {
		req["location_mode"] = *locationMode
	}
	if *clean {
		req["clean"] = true
	}

	versionFlags := 0
	notice := &versionUpdateNotice{}
	if *version != "" {
		versionFlags++
		notice.requested = true
		notice.expectedVersion = *version
	}
	if *setVersion != "" {
		versionFlags++
		notice.requested = true
		notice.expectedVersion = *setVersion
	}
	if *bumpVersion != "" {
		versionFlags++
		notice.requested = true
		notice.bumpRequested = true
	}
	if versionFlags > 1 {
		return fmt.Errorf("only one of --version, --set-version, or --bump-version may be specified")
	}
	if *allowDowngrade && versionFlags == 0 {
		return fmt.Errorf("--allow-downgrade requires --version, --set-version, or --bump-version")
	}

	if *version != "" {
		req["version_update"] = map[string]interface{}{
			"mode":            "set",
			"version":         *version,
			"persist":         false,
			"allow_downgrade": *allowDowngrade,
		}
	}
	if *setVersion != "" {
		req["version_update"] = map[string]interface{}{
			"mode":            "set",
			"version":         *setVersion,
			"persist":         true,
			"source":          *versionSource,
			"allow_downgrade": *allowDowngrade,
		}
	}
	if *bumpVersion != "" {
		normalizedBump, err := normalizeBumpValue(*bumpVersion)
		if err != nil {
			return err
		}
		notice.bumpValue = normalizedBump
		req["version_update"] = map[string]interface{}{
			"mode":            "bump",
			"bump":            normalizedBump,
			"persist":         true,
			"source":          *versionSource,
			"allow_downgrade": *allowDowngrade,
		}
	}

	// Build deploy config if any deploy flags are set
	if *deployTarget != "" || *deployTo != "" || *appKey != "" {
		deployRequested = true
		deploy := map[string]interface{}{}
		if *deployTarget != "" {
			deploy["target_name"] = *deployTarget
		}
		if *deployTo != "" {
			deploy["scenario_name"] = *deployTo
		}
		if *remoteProfile != "" {
			deploy["remote_profile"] = *remoteProfile
		}
		if *appKey != "" {
			deploy["app_key"] = *appKey
		}
		req["deploy"] = deploy
	}

	// Build query params for blocking mode
	if *wait {
		// IMPORTANT: /pipeline/run with blocking semantics has historically been able to create a pipeline
		// without actually starting it (leaving it in "created"). To provide a reliable CLI contract for
		// --wait, we do:
		// 1) run (create/configure) asynchronously
		// 2) start asynchronously
		// 3) poll status client-side until completion/failure/timeout

		createBody, err := c.api.Request("POST", c.apiPath("/pipeline/run"), nil, req)
		if err != nil {
			printAPIError(err, *debug)
			return &ErrAlreadyPrinted{Err: err}
		}

		if *jsonOutput {
			cliutil.PrintJSON(createBody)
			return nil
		}

		var createResp struct {
			PipelineID string `json:"pipeline_id"`
			StatusURL  string `json:"status_url"`
			Message    string `json:"message"`
		}
		if err := json.Unmarshal(createBody, &createResp); err != nil || createResp.PipelineID == "" {
			cliutil.PrintJSON(createBody)
			return nil
		}

		startReq := map[string]interface{}{}
		if *platforms != "" {
			startReq["platforms"] = strings.Split(*platforms, ",")
		}
		if *stages != "" {
			startReq["stages"] = strings.Split(*stages, ",")
		}
		if *deployTarget != "" || *deployTo != "" || *appKey != "" {
			deploy := map[string]interface{}{}
			if *deployTarget != "" {
				deploy["target_name"] = *deployTarget
			}
			if *deployTo != "" {
				deploy["scenario_name"] = *deployTo
			}
			if *remoteProfile != "" {
				deploy["remote_profile"] = *remoteProfile
			}
			if *appKey != "" {
				deploy["app_key"] = *appKey
			}
			startReq["deploy"] = deploy
		}

		var startBody []byte
		if len(startReq) > 0 {
			startBody, err = c.api.Request("POST", c.apiPath("/scenarios/"+scenario+"/pipeline/start"), nil, startReq)
		} else {
			startBody, err = c.api.Request("POST", c.apiPath("/scenarios/"+scenario+"/pipeline/start"), nil, nil)
		}
		if err != nil {
			printAPIError(err, *debug)
			return &ErrAlreadyPrinted{Err: err}
		}

		// Even if the start response isn't parseable (API shape may change), we can still poll the pipeline ID
		// returned by the run/create step.
		_ = startBody

		return c.waitForPipeline(createResp.PipelineID, *timeout, deployRequested, notice, *showOutput)
	}

	body, err := c.api.Request("POST", c.apiPath("/pipeline/run"), nil, req)
	if err != nil {
		printAPIError(err, *debug)
		return &ErrAlreadyPrinted{Err: err}
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// Async mode response
	var resp struct {
		PipelineID string `json:"pipeline_id"`
		StatusURL  string `json:"status_url"`
		Message    string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Pipeline started: %s\n", resp.PipelineID)
	fmt.Printf("Check status: %s pipeline-status %s\n", appName, resp.PipelineID)
	return nil
}

// Status gets pipeline status.
func (c *Commands) Status(args []string) error {
	fs := flag.NewFlagSet("pipeline-status", flag.ContinueOnError)
	verbose := fs.Bool("verbose", false, "Include detailed stage logs")
	showOutput := fs.Bool("show-output", false, "Show app stdout/stderr on failure (useful for debugging)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-status <id> [--verbose]")
	}

	pipelineID := fs.Args()[0]
	query := make(map[string]string)
	if *verbose {
		query["verbose"] = "true"
	}

	body, err := c.apiGet("/pipeline/"+pipelineID, query)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp pipelineStatus
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	// Also parse progress for display
	var progressResp struct {
		Progress int `json:"progress_percent"`
	}
	_ = json.Unmarshal(body, &progressResp)

	fmt.Printf("Pipeline: %s\n", resp.PipelineID)
	fmt.Printf("Status: %s (%d%% complete)\n", resp.Status, progressResp.Progress)
	if resp.ScenarioName != "" {
		fmt.Printf("Scenario: %s\n", resp.ScenarioName)
	}
	if resp.Config != nil && resp.Config.Version != "" {
		fmt.Printf("Version: %s\n", resp.Config.Version)
	}
	if len(resp.Stages) > 0 {
		fmt.Println("Stages:")
		for name, stage := range resp.Stages {
			fmt.Printf("  %-12s %s\n", name+":", stage.Status)
		}
	}

	// Show error details for failed pipelines
	if resp.Status == "failed" {
		fmt.Println()
		printPipelineError(&resp, *showOutput)
	}

	return nil
}

// Resume resumes a stopped pipeline.
func (c *Commands) Resume(args []string) error {
	fs := flag.NewFlagSet("pipeline-resume", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-resume <id>")
	}

	pipelineID := fs.Args()[0]
	body, err := c.apiPost("/pipeline/"+pipelineID+"/resume", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Pipeline resumed")
	cliutil.PrintJSON(body)
	return nil
}

// Cancel cancels a running pipeline.
func (c *Commands) Cancel(args []string) error {
	fs := flag.NewFlagSet("pipeline-cancel", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-cancel <id>")
	}

	pipelineID := fs.Args()[0]
	body, err := c.apiPost("/pipeline/"+pipelineID+"/cancel", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Pipeline cancellation requested")
	return nil
}

// List lists all pipelines.
func (c *Commands) List(args []string) error {
	fs := flag.NewFlagSet("pipeline-list", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)
	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	body, err := c.apiGet("/pipelines", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	var resp struct {
		Pipelines []struct {
			PipelineID   string `json:"pipeline_id"`
			ScenarioName string `json:"scenario_name"`
			Status       string `json:"status"`
		} `json:"pipelines"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	if len(resp.Pipelines) == 0 {
		fmt.Println("No pipelines found")
		return nil
	}

	fmt.Println("Pipelines:")
	for _, p := range resp.Pipelines {
		fmt.Printf("  %-36s %-20s %s\n", p.PipelineID, p.ScenarioName, p.Status)
	}
	return nil
}

// Active gets active pipeline for scenario.
func (c *Commands) Active(args []string) error {
	fs := flag.NewFlagSet("pipeline-active", flag.ContinueOnError)
	noCreate := fs.Bool("no-create", false, "Don't create if none exists")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-active <scenario> [--no-create]")
	}

	scenario := fs.Args()[0]
	query := make(map[string]string)
	if *noCreate {
		query["auto_create"] = "false"
	}

	body, err := c.apiGet("/scenarios/"+scenario+"/pipeline/active", query)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// Create creates a new pipeline for scenario.
func (c *Commands) Create(args []string) error {
	fs := flag.NewFlagSet("pipeline-create", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-create <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiPost("/scenarios/"+scenario+"/pipeline", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Pipeline created")
	cliutil.PrintJSON(body)
	return nil
}

// Reset resets active pipeline for scenario.
func (c *Commands) Reset(args []string) error {
	fs := flag.NewFlagSet("pipeline-reset", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-reset <scenario>")
	}

	scenario := fs.Args()[0]
	body, err := c.apiPost("/scenarios/"+scenario+"/pipeline/reset", nil)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Println("Pipeline reset")
	cliutil.PrintJSON(body)
	return nil
}

// History gets pipeline history.
func (c *Commands) History(args []string) error {
	fs := flag.NewFlagSet("pipeline-history", flag.ContinueOnError)
	limit := fs.Int("limit", 10, "Number of pipelines to return")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-history <scenario> [--limit N]")
	}

	scenario := fs.Args()[0]
	query := map[string]string{"limit": fmt.Sprintf("%d", *limit)}

	body, err := c.apiGet("/scenarios/"+scenario+"/pipeline/history", query)
	if err != nil {
		return err
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	cliutil.PrintJSON(body)
	return nil
}

// Start starts active pipeline.
func (c *Commands) Start(args []string) error {
	fs := flag.NewFlagSet("pipeline-start", flag.ContinueOnError)
	stages := fs.String("stages", "", "Comma-separated stages to run")
	platforms := fs.String("platforms", "", "Comma-separated target platforms")
	wait := fs.Bool("wait", false, "Block until pipeline completes (recommended for scripts/agents)")
	timeout := fs.Int("timeout", 600, "Max wait time in seconds (when --wait is used)")
	debug := fs.Bool("debug", false, "Show full JSON response on error")
	showOutput := fs.Bool("show-output", false, "Show app stdout/stderr on failure (useful for debugging)")
	deployTarget := fs.String("deploy-target", "", "Saved deploy target name from deploy-targets.json")
	deployTo := fs.String("deploy-to", "", "LPBS scenario name to deploy through (inline)")
	remoteProfile := fs.String("remote-profile", "", "Remote profile tag on the LPBS instance (inline)")
	appKey := fs.String("app-key", "", "App key for the download app in LPBS (required for deploy)")
	jsonOutput := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-start <scenario> [--stages ...] [--platforms ...] [--deploy-target <name> | --deploy-to <scenario> --remote-profile <tag>] [--app-key <key>] [--wait] [--timeout N]")
	}

	scenario := fs.Args()[0]
	req := map[string]interface{}{}
	if *stages != "" {
		req["stages"] = strings.Split(*stages, ",")
	}
	if *platforms != "" {
		req["platforms"] = strings.Split(*platforms, ",")
	}

	// Build deploy config if any deploy flags are set
	if *deployTarget != "" || *deployTo != "" || *appKey != "" {
		deployConfig := map[string]interface{}{}
		if *deployTarget != "" {
			deployConfig["target_name"] = *deployTarget
		}
		if *deployTo != "" {
			deployConfig["scenario_name"] = *deployTo
		}
		if *remoteProfile != "" {
			deployConfig["remote_profile"] = *remoteProfile
		}
		if *appKey != "" {
			deployConfig["app_key"] = *appKey
		}
		req["deploy"] = deployConfig
	}

	var body []byte
	var err error
	if len(req) > 0 {
		body, err = c.api.Request("POST", c.apiPath("/scenarios/"+scenario+"/pipeline/start"), nil, req)
	} else {
		body, err = c.api.Request("POST", c.apiPath("/scenarios/"+scenario+"/pipeline/start"), nil, nil)
	}
	if err != nil {
		printAPIError(err, *debug)
		return &ErrAlreadyPrinted{Err: err}
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	if *wait {
		// Pipeline start returns a small response. Polling is the reliable progress + exit-code contract.
		var resp struct {
			Pipeline struct {
				PipelineID string `json:"pipeline_id"`
			} `json:"pipeline"`
			Message string `json:"message"`
		}
		if err := json.Unmarshal(body, &resp); err != nil || resp.Pipeline.PipelineID == "" {
			cliutil.PrintJSON(body)
			return nil
		}

		return c.waitForPipeline(resp.Pipeline.PipelineID, *timeout, false, nil, *showOutput)
	}

	// Async mode response
	var resp struct {
		Pipeline struct {
			PipelineID string `json:"pipeline_id"`
		} `json:"pipeline"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		cliutil.PrintJSON(body)
		return nil
	}

	fmt.Printf("Pipeline started: %s\n", resp.Pipeline.PipelineID)
	fmt.Printf("Check status: %s pipeline-status %s\n", appName, resp.Pipeline.PipelineID)
	return nil
}

// pipelineStatus represents a full pipeline status response with error info.
type pipelineStatus struct {
	PipelineID      string                  `json:"pipeline_id"`
	Status          string                  `json:"status"`
	CurrentStage    string                  `json:"current_stage,omitempty"`
	CurrentState    string                  `json:"current_state,omitempty"`
	ProgressPercent int                     `json:"progress_percent,omitempty"`
	ProgressMessage string                  `json:"progress_message,omitempty"`
	ScenarioName    string                  `json:"scenario_name,omitempty"`
	FinalArtifacts  map[string]string       `json:"final_artifacts,omitempty"`
	StartedAt       int64                   `json:"started_at,omitempty"`
	CompletedAt     int64                   `json:"completed_at,omitempty"`
	Error           string                  `json:"error,omitempty"`
	Stages          map[string]*stageResult `json:"stages,omitempty"`
	Config          *pipelineConfig         `json:"config,omitempty"`
}

type pipelineConfig struct {
	Version   string   `json:"version,omitempty"`
	Platforms []string `json:"platforms,omitempty"`
	Deploy    *struct {
		ScenarioName  string `json:"scenario_name,omitempty"`
		RemoteProfile string `json:"remote_profile,omitempty"`
		AppKey        string `json:"app_key,omitempty"`
		Channel       string `json:"channel,omitempty"`
	} `json:"deploy,omitempty"`
}

// stageResult represents a stage result with optional error info.
type stageResult struct {
	Status    string          `json:"status"`
	Error     string          `json:"error,omitempty"`
	ErrorInfo *stageErrorInfo `json:"error_info,omitempty"`
	Logs      []string        `json:"logs,omitempty"`
	Details   json.RawMessage `json:"details,omitempty"`
}

type deployStageDetails struct {
	UpdateURL string `json:"update_url"`
}

func pipelineManifestFilename(platform string) string {
	switch strings.ToLower(strings.TrimSpace(platform)) {
	case "win", "windows":
		return "latest.yml"
	case "mac", "macos", "darwin":
		return "latest-mac.yml"
	case "linux":
		return "latest-linux.yml"
	default:
		return ""
	}
}

func (c *Commands) fetchPipelineStatus(pipelineID string, verbose bool) (*pipelineStatus, error) {
	query := map[string]string{}
	if verbose {
		query["verbose"] = "true"
	}
	body, err := c.apiGet("/pipeline/"+pipelineID, query)
	if err != nil {
		return nil, err
	}
	var resp pipelineStatus
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// getSmokeTestDetails extracts smoke test details from the raw JSON details.
func (s *stageResult) getSmokeTestDetails() *smokeTestDetails {
	if len(s.Details) == 0 {
		return nil
	}
	var details smokeTestDetails
	if err := json.Unmarshal(s.Details, &details); err != nil {
		return nil
	}
	// Check if we actually got meaningful data - now also check error_context
	if details.LastStdout == "" && details.LastStderr == "" && len(details.ErrorContext) == 0 {
		return nil
	}
	return &details
}

// getStderr returns stderr from either LastStderr or ErrorContext.
func (d *smokeTestDetails) getStderr() string {
	if d.LastStderr != "" {
		return d.LastStderr
	}
	if d.ErrorContext != nil {
		return d.ErrorContext["stderr"]
	}
	return ""
}

// getLifecycleState returns the last lifecycle state from ErrorContext.
func (d *smokeTestDetails) getLifecycleState() string {
	if d.ErrorContext != nil {
		return d.ErrorContext["last_lifecycle_state"]
	}
	return ""
}

// getAppReportedError returns formatted app-reported error info if available.
func (d *smokeTestDetails) getAppReportedError() string {
	if d.AppReportedError == nil || d.AppReportedError.Message == "" {
		return ""
	}
	return d.AppReportedError.Message
}

// getAppReportedErrorContext returns additional context about the app-reported error.
func (d *smokeTestDetails) getAppReportedErrorContext() string {
	if d.AppReportedError == nil {
		return ""
	}
	var parts []string
	if d.AppReportedError.DeploymentMode != "" {
		parts = append(parts, fmt.Sprintf("deployment_mode=%s", d.AppReportedError.DeploymentMode))
	}
	if d.AppReportedError.Event != "" {
		parts = append(parts, fmt.Sprintf("event=%s", d.AppReportedError.Event))
	}
	if len(parts) > 0 {
		return strings.Join(parts, ", ")
	}
	return ""
}

// getProgressStages extracts SMOKE_TEST_STAGE markers from stdout.
// Returns the stages completed before failure.
func (d *smokeTestDetails) getProgressStages() []string {
	if d == nil {
		return nil
	}
	return extractProgressStages(d.LastStdout)
}

// stageErrorInfo contains structured error information for stage failures.
type stageErrorInfo struct {
	Code         string      `json:"code,omitempty"`
	Message      string      `json:"message,omitempty"`
	Recovery     string      `json:"recovery,omitempty"`
	RecoveryHint string      `json:"recovery_hint,omitempty"`
	AutoFix      *autoFix    `json:"auto_fix,omitempty"`
	ManualSteps  []string    `json:"manual_steps,omitempty"`
	Diagnostic   *diagnostic `json:"diagnostic,omitempty"`
}

// autoFix contains an auto-fix command suggestion.
type autoFix struct {
	Command     string `json:"command,omitempty"`
	Description string `json:"description,omitempty"`
	Safe        bool   `json:"safe,omitempty"`
}

// diagnostic contains process diagnostic information.
type diagnostic struct {
	Process *processDiagnostic `json:"process,omitempty"`
}

// processDiagnostic contains process execution details.
type processDiagnostic struct {
	LastOutput string `json:"last_output,omitempty"`
	ExitCode   int    `json:"exit_code,omitempty"`
}

// smokeTestDetails contains smoke test execution details from the API response.
type smokeTestDetails struct {
	LastStdout            string               `json:"last_stdout,omitempty"`
	LastStderr            string               `json:"last_stderr,omitempty"`
	Error                 string               `json:"error,omitempty"`
	ErrorKind             int                  `json:"error_kind,omitempty"`
	ErrorContext          map[string]string    `json:"error_context,omitempty"`
	CurrentState          string               `json:"current_state,omitempty"`
	AppReportedError      *appReportedErrorDTO `json:"app_reported_error,omitempty"`
	AppSessionID          string               `json:"app_session_id,omitempty"`
	AppReportedErrorStale bool                 `json:"app_reported_error_stale,omitempty"`
	ErrorSessionMismatch  bool                 `json:"error_session_mismatch,omitempty"`
	Logs                  []string             `json:"logs,omitempty"`
}

// getPrereqWarnings extracts prerequisite check warnings from logs.
// Returns warnings that may be relevant to the failure.
func (d *smokeTestDetails) getPrereqWarnings() []string {
	if d == nil || len(d.Logs) == 0 {
		return nil
	}

	var warnings []string
	prereqWarningPattern := regexp.MustCompile(`\[prereq:(\w+)\].*\(suggestion: ([^)]+)\)`)

	for _, log := range d.Logs {
		matches := prereqWarningPattern.FindStringSubmatch(log)
		if len(matches) >= 3 {
			// Extract the warning type and suggestion
			warningType := matches[1]
			suggestion := matches[2]
			warnings = append(warnings, fmt.Sprintf("%s: %s", warningType, suggestion))
		}
	}
	return warnings
}

// appReportedErrorDTO represents an error extracted from app telemetry.
type appReportedErrorDTO struct {
	Event          string `json:"event"`
	Message        string `json:"message"`
	DeploymentMode string `json:"deployment_mode,omitempty"`
	SessionID      string `json:"session_id,omitempty"`
	Timestamp      string `json:"timestamp,omitempty"`
}

// stderrPattern represents a pattern to match in stderr with associated recovery hint.
type stderrPattern struct {
	pattern  *regexp.Regexp
	hint     string
	category string // For grouping related errors
}

// stderrPatterns contains patterns to match common errors in stderr.
var stderrPatterns = []stderrPattern{
	{
		pattern:  regexp.MustCompile(`no go\.mod|unable to resolve paths for staleness`),
		hint:     "The bundled binary is trying to find Go source files. Ensure VROOLI_API_SKIP_STALE_CHECK=true is set in the bundle environment.",
		category: "go_module",
	},
	{
		pattern:  regexp.MustCompile(`(?i)permission denied`),
		hint:     "Check file permissions. Ensure the artifact is executable: chmod +x <artifact>",
		category: "permissions",
	},
	{
		pattern:  regexp.MustCompile(`(?i)GLIBC.*not found|version.*GLIBC`),
		hint:     "System GLIBC version mismatch. The binary was built for a different Linux version. Rebuild on a compatible system or use a container.",
		category: "glibc",
	},
	{
		pattern:  regexp.MustCompile(`(?i)ENOENT|no such file or directory`),
		hint:     "Required file or dependency not found. Check the bundle contains all required files.",
		category: "missing_file",
	},
	{
		pattern:  regexp.MustCompile(`(?i)EACCES|access denied`),
		hint:     "Access denied. Check permissions and ensure the app isn't blocked by security software.",
		category: "access",
	},
	{
		pattern:  regexp.MustCompile(`(?i)libgtk|libX11|cannot open shared object|libGL`),
		hint:     "Missing system library. Install Electron dependencies: sudo apt-get install libgtk-3-0 libnotify4 libnss3 libxss1",
		category: "shared_lib",
	},
	{
		pattern:  regexp.MustCompile(`(?i)ECONNREFUSED|connection refused`),
		hint:     "Server connection refused. Ensure the target server is running and accessible.",
		category: "connection",
	},
	{
		pattern:  regexp.MustCompile(`(?i)ETIMEDOUT|timeout|timed out`),
		hint:     "Connection or operation timed out. Check network connectivity and increase timeout if needed.",
		category: "timeout",
	},
	{
		pattern:  regexp.MustCompile(`(?i)out of memory|OOM|heap`),
		hint:     "Out of memory. The system may not have enough RAM. Try closing other applications.",
		category: "memory",
	},
	{
		pattern:  regexp.MustCompile(`(?i)segmentation fault|SIGSEGV`),
		hint:     "App crashed with segmentation fault. This may indicate a binary incompatibility or corrupted artifact.",
		category: "crash",
	},
}

// analyzeStderr matches stderr against known patterns and returns a targeted hint.
func analyzeStderr(stderr string) string {
	for _, p := range stderrPatterns {
		if p.pattern.MatchString(stderr) {
			return p.hint
		}
	}
	return ""
}

// lifecycleStateDescription returns a human-readable description of where the failure occurred.
func lifecycleStateDescription(state string) string {
	switch state {
	case "":
		return "App crashed before smoke test initialization code ran. This usually indicates an Electron startup failure or missing dependencies."
	case "init":
		return "App started smoke test but crashed during initialization. A bundled service likely failed to start."
	// Granular bundled-mode states (occur between init and ready)
	case "bundle_resolving":
		return "App is locating the bundle directory. Check if bundle is packaged correctly in extraResources."
	case "runtime_starting":
		return "App is spawning the bundled runtime process. Check runtime binary permissions and dependencies."
	case "waiting_for_token":
		return "App is waiting for runtime auth token file. The runtime process started but may not be creating its token. Check if the bundled API supports --token-path flag."
	case "runtime_healthz":
		return "App is waiting for runtime /healthz endpoint. The runtime may still be starting or crashed."
	case "runtime_readyz":
		return "App is waiting for runtime /readyz endpoint. Runtime started but services not ready."
	case "runtime_ports":
		return "App is querying runtime /ports endpoint. Services are ready but port configuration may be wrong."
	case "ui_server_check":
		return "App is verifying the UI server responds with HTTP 2xx. The server is returning an error status code (e.g., 404 Not Found)."
	case "ready":
		return "App initialized but failed during server connectivity check. The target server may not be running or accessible."
	case "result":
		return "App reported a result but crashed during cleanup. This is usually non-fatal."
	case "exit":
		return "App completed the smoke test lifecycle. If still failing, there may be a race condition in result reporting."
	default:
		return fmt.Sprintf("App reached state '%s' before failing.", state)
	}
}

// smokeTestErrorPattern matches SMOKE_TEST_ERROR markers in app output.
// Format: SMOKE_TEST_ERROR kind=<kind> msg="<message>"
var smokeTestErrorPattern = regexp.MustCompile(`SMOKE_TEST_ERROR kind=(\w+) msg="([^"]+)"`)

// smokeTestStagePattern matches SMOKE_TEST_STAGE markers in app output.
var smokeTestStagePattern = regexp.MustCompile(`SMOKE_TEST_STAGE=(\w+)`)

// extractProgressStages extracts SMOKE_TEST_STAGE markers from stdout.
// Returns the stages in order, representing how far the app progressed.
func extractProgressStages(stdout string) []string {
	if stdout == "" {
		return nil
	}
	matches := smokeTestStagePattern.FindAllStringSubmatch(stdout, -1)
	if len(matches) == 0 {
		return nil
	}

	var stages []string
	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) >= 2 {
			stage := match[1]
			// Avoid duplicates while preserving order
			if !seen[stage] {
				seen[stage] = true
				stages = append(stages, stage)
			}
		}
	}
	return stages
}

// stageDisplayName returns a human-friendly name for a smoke test stage.
func stageDisplayName(stage string) string {
	names := map[string]string{
		"bundle_resolving":  "Bundle resolved",
		"runtime_starting":  "Runtime starting",
		"waiting_for_token": "Waiting for auth token",
		"runtime_healthz":   "Runtime health check",
		"runtime_readyz":    "Waiting for services ready",
		"runtime_ports":     "Getting port configuration",
		"ui_server_check":   "Verifying UI server",
		"ready":             "App ready",
		"result":            "Smoke test completed",
	}
	if name, ok := names[stage]; ok {
		return name
	}
	return stage
}

// extractSmokeTestErrorHint extracts the first config/validation error from smoke test output.
// Returns the error message if found, empty string otherwise.
func extractSmokeTestErrorHint(stdout, stderr string) string {
	// Check stdout first (where structured markers are written)
	matches := smokeTestErrorPattern.FindAllStringSubmatch(stdout, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			kind := match[1]
			msg := match[2]
			// Prioritize config errors as they're most actionable
			if kind == "config" || kind == "validation" {
				return msg
			}
		}
	}

	// Check stderr as fallback
	matches = smokeTestErrorPattern.FindAllStringSubmatch(stderr, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			return match[2] // Return first error found
		}
	}

	return ""
}

// printPipelineSuccess prints helpful information after a successful pipeline.
// Includes scenario name, duration, artifact locations, and next steps.
type versionUpdateNotice struct {
	requested       bool
	expectedVersion string
	bumpRequested   bool
	bumpValue       string
}

func printPipelineSuccess(status *pipelineStatus, notice *versionUpdateNotice) {
	fmt.Printf("Pipeline completed: %s\n", status.PipelineID)
	fmt.Println()

	// Scenario name
	if status.ScenarioName != "" {
		fmt.Printf("Scenario: %s\n", status.ScenarioName)
	}
	if status.Config != nil && status.Config.Version != "" {
		fmt.Printf("Version: %s\n", status.Config.Version)
	}
	if notice != nil && notice.requested {
		printVersionUpdateWarning(status, notice)
	}

	// Duration
	if status.StartedAt > 0 && status.CompletedAt > 0 {
		durationSec := status.CompletedAt - status.StartedAt
		if durationSec >= 60 {
			minutes := durationSec / 60
			seconds := durationSec % 60
			fmt.Printf("Duration: %dm %ds\n", minutes, seconds)
		} else {
			fmt.Printf("Duration: %ds\n", durationSec)
		}
	}

	// Artifacts
	if len(status.FinalArtifacts) > 0 {
		fmt.Println()
		fmt.Println("Artifacts:")
		for platform, path := range status.FinalArtifacts {
			fmt.Printf("  %s: %s\n", platform, path)
		}
	}

	// Deploy/update URLs (when deploy stage ran)
	if deployStage, ok := status.Stages["deploy"]; ok && deployStage != nil && len(deployStage.Details) > 0 {
		var details deployStageDetails
		if err := json.Unmarshal(deployStage.Details, &details); err == nil && strings.TrimSpace(details.UpdateURL) != "" {
			updateURL := strings.TrimSpace(details.UpdateURL)
			channel := "stable"
			if status.Config != nil && status.Config.Deploy != nil && strings.TrimSpace(status.Config.Deploy.Channel) != "" {
				channel = strings.TrimSpace(status.Config.Deploy.Channel)
			}

			fmt.Println()
			fmt.Println("Updates:")
			fmt.Printf("  Base:    %s\n", updateURL)
			fmt.Printf("  Channel: %s\n", channel)
			if len(status.FinalArtifacts) > 0 {
				fmt.Println("  Manifests:")
				for platform := range status.FinalArtifacts {
					manifest := pipelineManifestFilename(platform)
					if manifest == "" {
						continue
					}
					fmt.Printf("    %s: %s/%s/%s\n", platform, updateURL, channel, manifest)
				}
			}
		}
	}

	// Next steps
	fmt.Println()
	fmt.Println("Next steps:")
	if len(status.FinalArtifacts) == 1 {
		// Single artifact - suggest running it directly
		for _, path := range status.FinalArtifacts {
			fmt.Printf("  Run the app:     %s\n", path)
			break
		}
	} else if len(status.FinalArtifacts) > 1 {
		fmt.Printf("  Run an artifact from the paths listed above\n")
	}
	fmt.Printf("  View full logs:  %s pipeline-status %s --verbose\n", appName, status.PipelineID)
	fmt.Printf("  View as JSON:    %s pipeline-status %s --json\n", appName, status.PipelineID)
}

func printVersionUpdateWarning(status *pipelineStatus, notice *versionUpdateNotice) {
	reported := ""
	if status.Config != nil {
		reported = status.Config.Version
	}

	if notice.expectedVersion != "" {
		if reported == "" {
			fmt.Fprintf(os.Stderr, "Warning: version update requested (%s), but pipeline reported no version. Check pipeline logs or update the scenario-to-desktop API.\n", notice.expectedVersion)
			return
		}
		if reported != notice.expectedVersion {
			fmt.Fprintf(os.Stderr, "Warning: version update requested (%s), but pipeline reported version %s. Check pipeline logs or update the scenario-to-desktop API.\n", notice.expectedVersion, reported)
		}
		return
	}

	if notice.bumpRequested && reported == "" {
		bump := notice.bumpValue
		if bump == "" {
			bump = "patch"
		}
		fmt.Fprintf(os.Stderr, "Warning: version bump (%s) requested, but pipeline reported no version. Check pipeline logs or update the scenario-to-desktop API.\n", bump)
	}
}

// printPipelineError prints detailed error information from a failed pipeline.
// When showOutput is true, it displays the full stdout/stderr from the smoke test.
func printPipelineError(status *pipelineStatus, showOutput bool) {
	// Print the top-level error if present
	if status.Error != "" {
		fmt.Printf("Error: %s\n", status.Error)
	}

	// Find the failed stage and print its error info
	for stageName, stage := range status.Stages {
		if stage.Status != "failed" {
			continue
		}

		// Print stage error if different from top-level error
		if stage.Error != "" && stage.Error != status.Error {
			fmt.Printf("Stage '%s' failed: %s\n", stageName, stage.Error)
		}

		// For smoketest failures, extract and display rich diagnostic information
		var smokeDetails *smokeTestDetails
		if stageName == "smoketest" {
			smokeDetails = stage.getSmokeTestDetails()
		}

		// Determine if app-reported error is stale/mismatched
		appErrorIsStale := smokeDetails != nil && (smokeDetails.ErrorSessionMismatch || smokeDetails.AppReportedErrorStale)

		// When app error is stale, show lifecycle state FIRST as the primary issue
		// Otherwise, show app error first (it's the most actionable info)
		if smokeDetails != nil {
			appError := smokeDetails.getAppReportedError()
			lifecycleState := smokeDetails.getLifecycleState()
			if lifecycleState == "" {
				lifecycleState = smokeDetails.CurrentState
			}

			if appErrorIsStale {
				// Stale error: Show lifecycle state as primary, app error as historical context
				if lifecycleState != "" {
					fmt.Printf("\nLifecycle state: %s\n", lifecycleState)
					fmt.Printf("  %s\n", lifecycleStateDescription(lifecycleState))
				}

				// Show stale app error as secondary info
				if appError != "" {
					fmt.Printf("\nHistorical context (from previous session):\n")
					fmt.Printf("  Previous error: %s\n", appError)
					if ctx := smokeDetails.getAppReportedErrorContext(); ctx != "" {
						fmt.Printf("  (%s)\n", ctx)
					}
					fmt.Printf("  ⚠️  Note: This error is from a different session and may not reflect the current issue.\n")
					fmt.Printf("  The current run reached '%s' state before timing out.\n", lifecycleState)
				}
			} else {
				// Fresh error: Show app error prominently first
				if appError != "" {
					fmt.Printf("\nApp reported error: %s\n", appError)
					if ctx := smokeDetails.getAppReportedErrorContext(); ctx != "" {
						fmt.Printf("  (%s)\n", ctx)
					}
				}

				// Then show lifecycle state context
				if lifecycleState != "" {
					fmt.Printf("\nLifecycle state: %s\n", lifecycleState)
					fmt.Printf("  %s\n", lifecycleStateDescription(lifecycleState))
				}
			}

			// Show progress summary from stdout SMOKE_TEST_STAGE markers
			if stages := smokeDetails.getProgressStages(); len(stages) > 0 {
				fmt.Printf("\nApp progress (from stdout markers):\n")
				for i, stage := range stages {
					if i == len(stages)-1 {
						// Last stage - this is where it got stuck
						fmt.Printf("  ⏳ %s (timed out here)\n", stageDisplayName(stage))
					} else {
						fmt.Printf("  ✓ %s\n", stageDisplayName(stage))
					}
				}
			}

			// Show prereq warnings that may be relevant to the failure
			if prereqWarnings := smokeDetails.getPrereqWarnings(); len(prereqWarnings) > 0 {
				fmt.Printf("\nPotential issues detected during prerequisites:\n")
				for _, warning := range prereqWarnings {
					fmt.Printf("  ⚠️  %s\n", warning)
				}
			}
		}

		// Print rich error info if available
		if stage.ErrorInfo != nil {
			info := stage.ErrorInfo

			// Error code for programmatic use
			if info.Code != "" {
				fmt.Printf("Error code: %s\n", info.Code)
			}

			// For smoketest failures, build a prioritized recovery hint:
			// 1. Pattern-matched stderr analysis (most specific)
			// 2. SMOKE_TEST_ERROR markers from app
			// 3. Generic recovery hint
			recoveryHint := ""
			if smokeDetails != nil {
				stderr := smokeDetails.getStderr()

				// First, try to analyze stderr for common patterns
				if stderr != "" {
					recoveryHint = analyzeStderr(stderr)
				}

				// If no pattern match, try SMOKE_TEST_ERROR markers
				if recoveryHint == "" {
					recoveryHint = extractSmokeTestErrorHint(smokeDetails.LastStdout, smokeDetails.LastStderr)
				}

				// Show stderr prominently if it contains useful info
				if stderr != "" && !strings.Contains(stderr, "ExperimentalWarning") {
					stderrDisplay := strings.TrimSpace(stderr)
					if len(stderrDisplay) > 500 {
						stderrDisplay = stderrDisplay[:500] + "...(truncated)"
					}
					if stderrDisplay != "" {
						fmt.Printf("\nRoot cause (stderr):\n  %s\n", strings.ReplaceAll(stderrDisplay, "\n", "\n  "))
					}
				}
			}

			// Recovery guidance - prefer specific hint over generic
			if recoveryHint != "" {
				fmt.Printf("\nRecovery: %s\n", recoveryHint)
			} else if info.RecoveryHint != "" {
				fmt.Printf("\nRecovery: %s\n", info.RecoveryHint)
			} else if info.Recovery != "" {
				fmt.Printf("\nRecovery action: %s\n", info.Recovery)
			}

			// Auto-fix suggestion
			if info.AutoFix != nil && info.AutoFix.Command != "" {
				safeLabel := ""
				if info.AutoFix.Safe {
					safeLabel = " (safe to run)"
				}
				fmt.Printf("\nAuto-fix%s:\n  %s\n", safeLabel, info.AutoFix.Command)
				if info.AutoFix.Description != "" {
					fmt.Printf("  → %s\n", info.AutoFix.Description)
				}
			}

			// Manual steps
			if len(info.ManualSteps) > 0 {
				fmt.Printf("\nManual steps:\n")
				for i, step := range info.ManualSteps {
					fmt.Printf("  %d. %s\n", i+1, step)
				}
			}

			// Last output from process (truncated) - only show if no stderr already displayed
			if info.Diagnostic != nil && info.Diagnostic.Process != nil {
				if info.Diagnostic.Process.LastOutput != "" && smokeDetails == nil {
					output := info.Diagnostic.Process.LastOutput
					// Truncate if too long
					if len(output) > 500 {
						output = output[:500] + "...(truncated)"
					}
					fmt.Printf("\nLast output:\n%s\n", output)
				}
			}
		}

		// Build failures: surface decisive excerpt from build output logs.
		// The API often returns a generic BUILD_FAILED code; the useful detail lives in stage.details.
		if stageName == "build" && len(stage.Details) > 0 {
			type buildPlatformResult struct {
				Status   string   `json:"status,omitempty"`
				ErrorLog []string `json:"error_log,omitempty"`
			}
			type buildDetails struct {
				PlatformResults map[string]buildPlatformResult `json:"platform_results,omitempty"`
				BuildLog        []string                       `json:"build_log,omitempty"`
				ErrorLog        []string                       `json:"error_log,omitempty"`
			}

			var details buildDetails
			if err := json.Unmarshal(stage.Details, &details); err == nil {
				var excerpt string
				for _, pr := range details.PlatformResults {
					if pr.Status != "failed" || len(pr.ErrorLog) == 0 {
						continue
					}
					excerpt = pr.ErrorLog[len(pr.ErrorLog)-1]
					break
				}
				if excerpt == "" && len(details.ErrorLog) > 0 {
					excerpt = details.ErrorLog[len(details.ErrorLog)-1]
				}
				if excerpt != "" {
					excerpt = strings.TrimSpace(excerpt)
					if len(excerpt) > 1200 {
						excerpt = excerpt[:1200] + "...(truncated)"
					}
					fmt.Printf("\nRoot cause (build output):\n  %s\n", strings.ReplaceAll(excerpt, "\n", "\n  "))
				}

				// When explicitly requested, also show the last few build log entries for context.
				if showOutput && len(details.BuildLog) > 0 {
					fmt.Printf("\n--- Build log (tail) ---\n")
					start := len(details.BuildLog) - 3
					if start < 0 {
						start = 0
					}
					for i := start; i < len(details.BuildLog); i++ {
						entry := strings.TrimSpace(details.BuildLog[i])
						if len(entry) > 800 {
							entry = entry[:800] + "...(truncated)"
						}
						fmt.Printf("%s\n\n", entry)
					}
				}
			}
		}

		// Show full stdout/stderr when --show-output is enabled
		if showOutput && smokeDetails != nil {
			if smokeDetails.LastStdout != "" {
				fmt.Printf("\n--- App stdout ---\n%s\n", smokeDetails.LastStdout)
			}
			if smokeDetails.LastStderr != "" {
				fmt.Printf("\n--- App stderr ---\n%s\n", smokeDetails.LastStderr)
			}
			if smokeDetails.LastStdout == "" && smokeDetails.LastStderr == "" {
				fmt.Printf("\n--- No app output captured ---\n")
				fmt.Printf("Tip: App may have crashed before producing output. Check system logs.\n")
			}
		}

		// Only show first failed stage
		break
	}
}

// printPipelineAPIError attempts to extract a pipeline status from an API error response
// and print rich error information. Returns true if it successfully printed pipeline error info.
func printPipelineAPIError(err error, debug bool, showOutput bool) bool {
	var apiErr *cliutil.APIError
	if !errors.As(err, &apiErr) || len(apiErr.RawResponse) == 0 {
		return false
	}

	// Try to parse the response as a pipeline status
	var status pipelineStatus
	if parseErr := json.Unmarshal(apiErr.RawResponse, &status); parseErr != nil {
		return false
	}

	// Check if this is a valid pipeline status with error info
	if status.PipelineID == "" {
		return false
	}

	// Print pipeline failure info
	fmt.Printf("Pipeline failed: %s\n", status.PipelineID)
	printPipelineError(&status, showOutput)

	if debug {
		fmt.Println("\n--- Debug: Raw Response ---")
		cliutil.PrintJSON(apiErr.RawResponse)
	}

	return true
}

// printAPIError displays a structured API error with recovery information.
func printAPIError(err error, debug bool) {
	var apiErr *cliutil.APIError
	if errors.As(err, &apiErr) && apiErr.IsStructured() {
		fmt.Print(apiErr.FormatConcise())
		if debug && len(apiErr.RawResponse) > 0 {
			fmt.Println("\n--- Debug: Raw Response ---")
			cliutil.PrintJSON(apiErr.RawResponse)
		}
	} else {
		fmt.Printf("Error: %s\n", err)
	}
}

func normalizeBumpValue(input string) (string, error) {
	value := strings.ToLower(strings.TrimSpace(input))
	switch value {
	case "patch", "minor", "medium", "major":
		return value, nil
	case "auto":
		return "patch", nil
	default:
		return "", fmt.Errorf("invalid --bump-version %q (expected patch, minor, medium, major, auto)", input)
	}
}
