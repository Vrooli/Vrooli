// Package pipeline provides CLI commands for pipeline management.
package pipeline

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"scenario-to-desktop/cli/cmdutil"

	"github.com/vrooli/cli-core/cliutil"
)

const appName = "scenario-to-desktop"

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
		return err
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
			return fmt.Errorf("pipeline failed")
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
	if err := fs.Parse(args); err != nil {
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
		return err
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
			return fmt.Errorf("pipeline failed")
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

		// Print rich error info if available
		if stage.ErrorInfo != nil {
			info := stage.ErrorInfo

			// Error code for programmatic use
			if info.Code != "" {
				fmt.Printf("Error code: %s\n", info.Code)
			}

			// Recovery guidance
			if info.RecoveryHint != "" {
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

			// Last output from process (truncated)
			if info.Diagnostic != nil && info.Diagnostic.Process != nil {
				if info.Diagnostic.Process.LastOutput != "" {
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
	}
	return booleanFlags[name]
}
