// Package pipeline provides CLI commands for pipeline management.
package pipeline

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"net/url"
	"regexp"
	"strconv"
	"strings"

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

// Run starts a new pipeline.
func (c *Commands) Run(args []string) error {
	fs := flag.NewFlagSet("pipeline-run", flag.ContinueOnError)
	stages := fs.String("stages", "", "Comma-separated stages to run")
	platforms := fs.String("platforms", "", "Comma-separated target platforms (default: current platform)")
	wait := fs.Bool("wait", false, "Block until pipeline completes (recommended for scripts/agents)")
	timeout := fs.Int("timeout", 600, "Max wait time in seconds (when --wait is used)")
	debug := fs.Bool("debug", false, "Show full JSON response on error")
	jsonOutput := cliutil.JSONFlag(fs)

	// Reorder args so flags come before positional arguments (Go's flag package stops at first non-flag)
	reordered := reorderArgsForFlags(args)
	if err := fs.Parse(reordered); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-run <scenario> [--stages bundle,preflight,generate,build,smoketest,distribution] [--platforms win,mac,linux] [--wait] [--timeout N]")
	}

	scenario := fs.Args()[0]
	req := map[string]interface{}{
		"scenario_name": scenario,
	}
	if *platforms != "" {
		req["platforms"] = strings.Split(*platforms, ",")
	}
	if *stages != "" {
		req["stages"] = strings.Split(*stages, ",")
	}

	// Build query params for blocking mode
	query := url.Values{}
	if *wait {
		query.Set("block", "true")
		query.Set("timeout", strconv.Itoa(*timeout))
	}

	body, err := c.api.Request("POST", c.apiPath("/pipeline/run"), query, req)
	if err != nil {
		// In blocking mode, the API returns an error but the response body may contain
		// a full pipeline status with rich error info. Try to extract it.
		if *wait {
			if printed := printPipelineAPIError(err, *debug); printed {
				return &ErrAlreadyPrinted{Err: fmt.Errorf("pipeline failed")}
			}
		}
		printAPIError(err, *debug)
		return &ErrAlreadyPrinted{Err: err}
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// If blocking mode, the response is a Status object
	if *wait {
		var status pipelineStatus
		if err := json.Unmarshal(body, &status); err != nil {
			cliutil.PrintJSON(body)
			return nil
		}

		if status.Status == "completed" {
			fmt.Printf("Pipeline completed: %s\n", status.PipelineID)
			return nil
		} else if status.Status == "failed" {
			fmt.Printf("Pipeline failed: %s\n", status.PipelineID)
			printPipelineError(&status)
			return &ErrAlreadyPrinted{Err: fmt.Errorf("pipeline failed")}
		} else {
			fmt.Printf("Pipeline %s: %s\n", status.Status, status.PipelineID)
			return nil
		}
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
	jsonOutput := cliutil.JSONFlag(fs)

	// Reorder args so flags come before positional arguments (Go's flag package stops at first non-flag)
	reordered := reorderArgsForFlags(args)
	if err := fs.Parse(reordered); err != nil {
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
	if len(resp.Stages) > 0 {
		fmt.Println("Stages:")
		for name, stage := range resp.Stages {
			fmt.Printf("  %-12s %s\n", name+":", stage.Status)
		}
	}

	// Show error details for failed pipelines
	if resp.Status == "failed" {
		fmt.Println()
		printPipelineError(&resp)
	}

	return nil
}

// Resume resumes a stopped pipeline.
func (c *Commands) Resume(args []string) error {
	fs := flag.NewFlagSet("pipeline-resume", flag.ContinueOnError)
	jsonOutput := cliutil.JSONFlag(fs)

	reordered := reorderArgsForFlags(args)
	if err := fs.Parse(reordered); err != nil {
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

	reordered := reorderArgsForFlags(args)
	if err := fs.Parse(reordered); err != nil {
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
	if err := fs.Parse(args); err != nil {
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

	reordered := reorderArgsForFlags(args)
	if err := fs.Parse(reordered); err != nil {
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

	reordered := reorderArgsForFlags(args)
	if err := fs.Parse(reordered); err != nil {
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

	reordered := reorderArgsForFlags(args)
	if err := fs.Parse(reordered); err != nil {
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

	reordered := reorderArgsForFlags(args)
	if err := fs.Parse(reordered); err != nil {
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
	jsonOutput := cliutil.JSONFlag(fs)

	// Reorder args so flags come before positional arguments (Go's flag package stops at first non-flag)
	reordered := reorderArgsForFlags(args)
	if err := fs.Parse(reordered); err != nil {
		return err
	}

	if len(fs.Args()) == 0 {
		return fmt.Errorf("usage: pipeline-start <scenario> [--stages ...] [--platforms ...] [--wait] [--timeout N]")
	}

	scenario := fs.Args()[0]
	req := map[string]interface{}{}
	if *stages != "" {
		req["stages"] = strings.Split(*stages, ",")
	}
	if *platforms != "" {
		req["platforms"] = strings.Split(*platforms, ",")
	}

	// Build query params for blocking mode
	query := url.Values{}
	if *wait {
		query.Set("block", "true")
		query.Set("timeout", strconv.Itoa(*timeout))
	}

	var body []byte
	var err error
	if len(req) > 0 {
		body, err = c.api.Request("POST", c.apiPath("/scenarios/"+scenario+"/pipeline/start"), query, req)
	} else {
		body, err = c.api.Request("POST", c.apiPath("/scenarios/"+scenario+"/pipeline/start"), query, nil)
	}
	if err != nil {
		// In blocking mode, the API returns an error but the response body may contain
		// a full pipeline status with rich error info. Try to extract it.
		if *wait {
			if printed := printPipelineAPIError(err, *debug); printed {
				return &ErrAlreadyPrinted{Err: fmt.Errorf("pipeline failed")}
			}
		}
		printAPIError(err, *debug)
		return &ErrAlreadyPrinted{Err: err}
	}

	if *jsonOutput {
		cliutil.PrintJSON(body)
		return nil
	}

	// If blocking mode, the response is a Status object
	if *wait {
		var status pipelineStatus
		if err := json.Unmarshal(body, &status); err != nil {
			cliutil.PrintJSON(body)
			return nil
		}

		if status.Status == "completed" {
			fmt.Printf("Pipeline completed: %s\n", status.PipelineID)
			return nil
		} else if status.Status == "failed" {
			fmt.Printf("Pipeline failed: %s\n", status.PipelineID)
			printPipelineError(&status)
			return &ErrAlreadyPrinted{Err: fmt.Errorf("pipeline failed")}
		} else {
			fmt.Printf("Pipeline %s: %s\n", status.Status, status.PipelineID)
			return nil
		}
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
	PipelineID string                  `json:"pipeline_id"`
	Status     string                  `json:"status"`
	Error      string                  `json:"error,omitempty"`
	Stages     map[string]*stageResult `json:"stages,omitempty"`
}

// stageResult represents a stage result with optional error info.
type stageResult struct {
	Status    string          `json:"status"`
	Error     string          `json:"error,omitempty"`
	ErrorInfo *stageErrorInfo `json:"error_info,omitempty"`
	Logs      []string        `json:"logs,omitempty"`
	Details   json.RawMessage `json:"details,omitempty"`
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
	LastStdout   string            `json:"last_stdout,omitempty"`
	LastStderr   string            `json:"last_stderr,omitempty"`
	Error        string            `json:"error,omitempty"`
	ErrorKind    int               `json:"error_kind,omitempty"`
	ErrorContext map[string]string `json:"error_context,omitempty"`
	CurrentState string            `json:"current_state,omitempty"`
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

// printPipelineError prints detailed error information from a failed pipeline.
func printPipelineError(status *pipelineStatus) {
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

		// Print lifecycle state context for smoke test failures
		if smokeDetails != nil {
			lifecycleState := smokeDetails.getLifecycleState()
			if lifecycleState != "" || smokeDetails.CurrentState != "" {
				state := lifecycleState
				if state == "" {
					state = smokeDetails.CurrentState
				}
				fmt.Printf("\nLifecycle state: %s\n", state)
				fmt.Printf("  %s\n", lifecycleStateDescription(state))
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

		// Only show first failed stage
		break
	}
}

// printPipelineAPIError attempts to extract a pipeline status from an API error response
// and print rich error information. Returns true if it successfully printed pipeline error info.
func printPipelineAPIError(err error, debug bool) bool {
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
	printPipelineError(&status)

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

// reorderArgsForFlags moves flag arguments before positional arguments.
// Go's flag package stops parsing at the first non-flag argument, so we need
// to ensure flags come first for them to be parsed correctly.
// Example: ["scenario", "--platforms", "linux"] -> ["--platforms", "linux", "scenario"]
func reorderArgsForFlags(args []string) []string {
	var flags []string
	var positional []string

	i := 0
	for i < len(args) {
		arg := args[i]
		if strings.HasPrefix(arg, "-") {
			// This is a flag
			flags = append(flags, arg)
			// Check if the next arg is a value for this flag (not another flag)
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				// Check if this flag takes a value (not a boolean flag)
				// Boolean flags: --wait, --verbose, --json, --no-create
				// Flags with values: --stages, --platforms, --timeout, --limit, --target, --path, etc.
				if !isBooleanFlag(arg) {
					i++
					flags = append(flags, args[i])
				}
			}
		} else {
			positional = append(positional, arg)
		}
		i++
	}

	return append(flags, positional...)
}

// isBooleanFlag returns true if the flag is a known boolean flag that doesn't take a value.
func isBooleanFlag(flag string) bool {
	// Strip leading dashes
	name := strings.TrimLeft(flag, "-")

	booleanFlags := map[string]bool{
		"wait":      true,
		"verbose":   true,
		"json":      true,
		"no-create": true,
		"force":     true,
		"debug":     true,
	}
	return booleanFlags[name]
}
