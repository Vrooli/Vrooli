package pipeline

import (
	"encoding/json"
	"flag"
	"fmt"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

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
			printProgressLine(status)
		}

		switch status.Status {
		case "completed":
			// Always fetch verbose status on completion to surface recording info and full details
			if verboseStatus, err := c.fetchPipelineStatus(status.PipelineID, true); err == nil {
				status = verboseStatus
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

func printProgressLine(status *pipelineStatus) {
	msg := strings.TrimSpace(status.ProgressMessage)
	stage := strings.TrimSpace(status.CurrentStage)
	switch {
	case msg != "" && stage != "":
		fmt.Printf("Status: %s (%d%%) stage=%s (%s)\n", status.Status, status.ProgressPercent, stage, msg)
	case stage != "":
		fmt.Printf("Status: %s (%d%%) stage=%s\n", status.Status, status.ProgressPercent, stage)
	case msg != "":
		fmt.Printf("Status: %s (%d%%) %s\n", status.Status, status.ProgressPercent, msg)
	default:
		fmt.Printf("Status: %s (%d%%)\n", status.Status, status.ProgressPercent)
	}
}

// runFlags holds parsed flags for the Run command.
type runFlags struct {
	stages         string
	platforms      string
	deploymentMode string
	locationMode   string
	clean          bool
	version        string
	setVersion     string
	bumpVersion    string
	versionSource  string
	allowDowngrade bool
	wait           bool
	timeout        int
	debug          bool
	showOutput     bool
	deployTarget      string
	deployTo          string
	remoteProfile     string
	appKey            string
	deploymentProfile string
	gateTimeout       string
	gatePollInterval  string
	jsonOutput        bool
}

func parseRunFlags(args []string) (*runFlags, *flag.FlagSet, error) {
	fs := flag.NewFlagSet("pipeline-run", flag.ContinueOnError)
	f := &runFlags{}
	fs.StringVar(&f.stages, "stages", "", "Comma-separated stages to run")
	fs.StringVar(&f.platforms, "platforms", "", "Comma-separated target platforms (default: current platform)")
	fs.StringVar(&f.deploymentMode, "deployment-mode", "", "Deployment mode: bundled (default), external-server, cloud-api, proxy")
	fs.StringVar(&f.locationMode, "location-mode", "", "Output location: proper (default), staging, temp")
	fs.BoolVar(&f.clean, "clean", false, "Remove existing desktop output before running the pipeline")
	fs.StringVar(&f.version, "version", "", "Override version for this run (no file updates)")
	fs.StringVar(&f.setVersion, "set-version", "", "Persist scenario version before running the pipeline")
	fs.StringVar(&f.bumpVersion, "bump-version", "", "Bump scenario version (patch, minor, medium, major, auto) and persist")
	fs.StringVar(&f.versionSource, "version-source", "both", "Version source to update when persisting: both, service, ui")
	fs.BoolVar(&f.allowDowngrade, "allow-downgrade", false, "Allow setting a version lower than the current scenario version")
	fs.BoolVar(&f.wait, "wait", false, "Block until pipeline completes (recommended for scripts/agents)")
	fs.IntVar(&f.timeout, "timeout", 600, "Max wait time in seconds (when --wait is used)")
	fs.BoolVar(&f.debug, "debug", false, "Show full JSON response on error")
	fs.BoolVar(&f.showOutput, "show-output", false, "Show app stdout/stderr on failure (useful for debugging)")
	fs.StringVar(&f.deployTarget, "deploy-target", "", "Saved deploy target name from deploy-targets.json")
	fs.StringVar(&f.deployTo, "deploy-to", "", "LPBS scenario name to deploy through (inline)")
	fs.StringVar(&f.remoteProfile, "remote-profile", "", "Remote profile tag on the LPBS instance (inline)")
	fs.StringVar(&f.appKey, "app-key", "", "App key for the download app in LPBS (required for deploy)")
	fs.StringVar(&f.deploymentProfile, "deployment-profile", "", "Deployment-manager profile ID for approval gates")
	fs.StringVar(&f.gateTimeout, "gate-timeout", "", "Max time to wait for approval gates (e.g. 30m)")
	fs.StringVar(&f.gatePollInterval, "gate-poll-interval", "", "Initial gate poll interval (e.g. 15s)")
	jsonPtr := cliutil.JSONFlag(fs)

	if err := cliutil.ParseInterspersed(fs, args); err != nil {
		return nil, nil, err
	}
	f.jsonOutput = *jsonPtr
	return f, fs, nil
}

// buildRunRequest constructs the API request body from parsed flags.
func buildRunRequest(scenario string, f *runFlags) map[string]interface{} {
	req := map[string]interface{}{
		"scenario_name": scenario,
	}
	if f.platforms != "" {
		req["platforms"] = strings.Split(f.platforms, ",")
	}
	if f.stages != "" {
		req["stages"] = strings.Split(f.stages, ",")
	}
	if f.deploymentMode != "" {
		req["deployment_mode"] = f.deploymentMode
	}
	if f.locationMode != "" {
		req["location_mode"] = f.locationMode
	}
	if f.clean {
		req["clean"] = true
	}
	return req
}

// buildVersionUpdate parses version flags and populates the request and notice.
// Returns an error if flags are invalid.
func buildVersionUpdate(f *runFlags, req map[string]interface{}) (*versionUpdateNotice, error) {
	notice := &versionUpdateNotice{}
	versionFlags := 0

	if f.version != "" {
		versionFlags++
		notice.requested = true
		notice.expectedVersion = f.version
	}
	if f.setVersion != "" {
		versionFlags++
		notice.requested = true
		notice.expectedVersion = f.setVersion
	}
	if f.bumpVersion != "" {
		versionFlags++
		notice.requested = true
		notice.bumpRequested = true
	}
	if versionFlags > 1 {
		return nil, fmt.Errorf("only one of --version, --set-version, or --bump-version may be specified")
	}
	if f.allowDowngrade && versionFlags == 0 {
		return nil, fmt.Errorf("--allow-downgrade requires --version, --set-version, or --bump-version")
	}

	if f.version != "" {
		req["version_update"] = map[string]interface{}{
			"mode": "set", "version": f.version,
			"persist": false, "allow_downgrade": f.allowDowngrade,
		}
	}
	if f.setVersion != "" {
		req["version_update"] = map[string]interface{}{
			"mode": "set", "version": f.setVersion,
			"persist": true, "source": f.versionSource, "allow_downgrade": f.allowDowngrade,
		}
	}
	if f.bumpVersion != "" {
		normalizedBump, err := normalizeBumpValue(f.bumpVersion)
		if err != nil {
			return nil, err
		}
		notice.bumpValue = normalizedBump
		req["version_update"] = map[string]interface{}{
			"mode": "bump", "bump": normalizedBump,
			"persist": true, "source": f.versionSource, "allow_downgrade": f.allowDowngrade,
		}
	}

	return notice, nil
}

// buildDeployConfig builds a deploy config map from deploy-related flags.
// Returns nil if no deploy flags are set.
func buildDeployConfig(f *runFlags) map[string]interface{} {
	if f.deployTarget == "" && f.deployTo == "" && f.appKey == "" {
		return nil
	}
	deploy := map[string]interface{}{}
	if f.deployTarget != "" {
		deploy["target_name"] = f.deployTarget
	}
	if f.deployTo != "" {
		deploy["scenario_name"] = f.deployTo
	}
	if f.remoteProfile != "" {
		deploy["remote_profile"] = f.remoteProfile
	}
	if f.appKey != "" {
		deploy["app_key"] = f.appKey
	}
	if f.deploymentProfile != "" {
		deploy["deployment_manager_profile_id"] = f.deploymentProfile
	}
	if f.gateTimeout != "" {
		deploy["gate_timeout"] = f.gateTimeout
	}
	if f.gatePollInterval != "" {
		deploy["gate_poll_interval"] = f.gatePollInterval
	}
	return deploy
}

// Run starts a new pipeline.
func (c *Commands) Run(args []string) error {
	f, fs, err := parseRunFlags(args)
	if err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-run <scenario> [--deployment-mode bundled|external-server] [--location-mode proper|staging] [--clean] [--stages bundle,...] [--platforms win,mac,linux] [--version X.Y.Z | --set-version X.Y.Z | --bump-version patch|minor|medium|major|auto] [--deploy-target <name> | --deploy-to <scenario> --remote-profile <tag>] [--app-key <key>] [--wait] [--timeout N]")
	}

	scenario := fs.Args()[0]
	req := buildRunRequest(scenario, f)

	notice, err := buildVersionUpdate(f, req)
	if err != nil {
		return err
	}

	deployConfig := buildDeployConfig(f)
	deployRequested := deployConfig != nil
	if deployConfig != nil {
		req["deploy"] = deployConfig
	}

	if f.wait {
		return c.runWaitMode(scenario, req, f, notice, deployRequested, deployConfig)
	}

	return c.runAsyncMode(req, f)
}

func (c *Commands) runWaitMode(scenario string, req map[string]interface{}, f *runFlags, notice *versionUpdateNotice, deployRequested bool, deployConfig map[string]interface{}) error {
	createBody, err := c.api.Request("POST", c.apiPath("/pipeline/run"), nil, req)
	if err != nil {
		printAPIError(err, f.debug)
		return &ErrAlreadyPrinted{Err: err}
	}

	if f.jsonOutput {
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
	if f.platforms != "" {
		startReq["platforms"] = strings.Split(f.platforms, ",")
	}
	if f.stages != "" {
		startReq["stages"] = strings.Split(f.stages, ",")
	}
	if deployConfig != nil {
		startReq["deploy"] = deployConfig
	}

	var startBody []byte
	if len(startReq) > 0 {
		startBody, err = c.api.Request("POST", c.apiPath("/scenarios/"+scenario+"/pipeline/start"), nil, startReq)
	} else {
		startBody, err = c.api.Request("POST", c.apiPath("/scenarios/"+scenario+"/pipeline/start"), nil, nil)
	}
	if err != nil {
		printAPIError(err, f.debug)
		return &ErrAlreadyPrinted{Err: err}
	}
	_ = startBody

	return c.waitForPipeline(createResp.PipelineID, f.timeout, deployRequested, notice, f.showOutput)
}

func (c *Commands) runAsyncMode(req map[string]interface{}, f *runFlags) error {
	body, err := c.api.Request("POST", c.apiPath("/pipeline/run"), nil, req)
	if err != nil {
		printAPIError(err, f.debug)
		return &ErrAlreadyPrinted{Err: err}
	}

	if f.jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

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
	deploymentProfile := fs.String("deployment-profile", "", "Deployment-manager profile ID for approval gates")
	gateTimeoutFlag := fs.String("gate-timeout", "", "Max time to wait for approval gates (e.g. 30m)")
	gatePollIntervalFlag := fs.String("gate-poll-interval", "", "Initial gate poll interval (e.g. 15s)")
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

	startFlags := &runFlags{
		deployTarget:      *deployTarget,
		deployTo:          *deployTo,
		remoteProfile:     *remoteProfile,
		appKey:            *appKey,
		deploymentProfile: *deploymentProfile,
		gateTimeout:       *gateTimeoutFlag,
		gatePollInterval:  *gatePollIntervalFlag,
	}
	if deployConfig := buildDeployConfig(startFlags); deployConfig != nil {
		req["deploy"] = deployConfig
	}

	body, err := c.postStartRequest(scenario, req)
	if err != nil {
		printAPIError(err, *debug)
		return &ErrAlreadyPrinted{Err: err}
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	pipelineID := parseStartPipelineID(body)
	if pipelineID == "" {
		cliutil.PrintJSON(body)
		return nil
	}

	if *wait {
		return c.waitForPipeline(pipelineID, *timeout, false, nil, *showOutput)
	}

	fmt.Printf("Pipeline started: %s\n", pipelineID)
	fmt.Printf("Check status: %s pipeline-status %s\n", appName, pipelineID)
	return nil
}

func (c *Commands) postStartRequest(scenario string, req map[string]interface{}) ([]byte, error) {
	if len(req) > 0 {
		return c.api.Request("POST", c.apiPath("/scenarios/"+scenario+"/pipeline/start"), nil, req)
	}
	return c.api.Request("POST", c.apiPath("/scenarios/"+scenario+"/pipeline/start"), nil, nil)
}

func parseStartPipelineID(body []byte) string {
	var resp struct {
		Pipeline struct {
			PipelineID string `json:"pipeline_id"`
		} `json:"pipeline"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return ""
	}
	return resp.Pipeline.PipelineID
}
