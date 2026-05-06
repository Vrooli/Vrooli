package execute

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"test-genie/cli/execute/report"
	"test-genie/cli/internal/phases"

	"github.com/vrooli/cli-core/cliutil"

	execTypes "test-genie/cli/internal/execute"
)

const UsageLine = "test-genie execute <scenario> [phases...] [--preset quick] [--skip performance] [--scenario-path PATH] [--logical-repo-root PATH] [--logical-scenario-relpath PATH] [--ui-url URL] [--api-url URL] [--browserless-url URL] [--fail-fast] [--json]"

// HelpText returns the framework-rendered help body for the execute command.
func HelpText() string {
	return `Execute a scenario suite and stream or summarize the phase results.

Examples:
  test-genie execute swarm-manager
  test-genie execute swarm-manager standards lint integration
  test-genie execute swarm-manager --preset quick --fail-fast
  test-genie execute swarm-manager --skip performance --json
  test-genie execute demo --scenario-path /tmp/vrooli/scenarios/demo --preset comprehensive
  test-genie execute demo --scenario-path /tmp/vrooli/scenarios/demo --logical-repo-root /workspace/Vrooli --logical-scenario-relpath scenarios/demo --preset comprehensive`
}

// Run executes the execute command.
func Run(client *Client, httpClient *cliutil.HTTPClient, args []string) error {
	parsed, err := ParseArgs(args)
	if err != nil {
		return err
	}

	scenarioPath := parsed.ScenarioPath
	if scenarioPath == "" {
		// Resolve the physical scenario directory from the scenario name.
		// cliutil owns local environment details; the execute request only
		// carries the resulting path as workspace identity.
		scenarioPath = cliutil.ResolveScenarioPath(parsed.Scenario)
	}

	req := Request{
		ScenarioName:           parsed.Scenario,
		Preset:                 parsed.Preset,
		Phases:                 parsed.Phases,
		Skip:                   parsed.Skip,
		FailFast:               parsed.FailFast,
		SuiteRequestID:         parsed.RequestID,
		UIURL:                  parsed.UIURL,
		APIURL:                 parsed.APIURL,
		BrowserlessURL:         parsed.BrowserlessURL,
		ScenarioPath:           scenarioPath,
		LogicalRepoRoot:        parsed.LogicalRepoRoot,
		LogicalScenarioRelPath: parsed.LogicalScenarioRelPath,
	}

	var (
		preview         execTypes.PlanPreview
		previewReady    bool
		progressPhases  []string
		estimateTargets map[string]time.Duration
		timeoutTargets  map[string]time.Duration
	)
	if planned, err := client.PreviewPlan(req); err == nil {
		preview = planned
		previewReady = true
		progressPhases = plannedPhaseNames(planned)
		estimateTargets, timeoutTargets = phaseTimingTargets(planned)
	} else if !parsed.JSON {
		fmt.Fprintf(os.Stderr, "Warning: unable to preview execution plan (%v)\n", err)
		if len(parsed.Phases) > 0 {
			progressPhases = append([]string(nil), parsed.Phases...)
		}
	}

	// Create printer early for pre-execution output
	pr := report.New(
		os.Stdout,
		parsed.Scenario,
		req.Preset,
		parsed.Phases,
		parsed.Skip,
		req.FailFast,
		nil,
		nil,
	)
	if previewReady {
		pr.SetPlanPreview(preview)
	}

	// Print header and test plan IMMEDIATELY (before API call)
	// This gives users instant feedback about what will run
	if !parsed.JSON {
		pr.PrintPreExecution(progressPhases)
	}

	// Determine streaming mode:
	// - Default to streaming for interactive TTY (better UX with live output)
	// - Use spinner for non-TTY (CI/piped output)
	// - Respect explicit --stream or --no-stream flags
	useStreaming := parsed.Stream
	if !parsed.NoStream && !parsed.JSON && isInteractiveTTY() {
		useStreaming = true
	}

	// Choose execution mode: SSE streaming vs regular
	var resp Response
	var raw []byte

	if useStreaming && !parsed.JSON {
		// SSE streaming mode: real-time output as phases complete
		// This is the default for interactive terminals
		var err error
		resp, err = client.RunWithSSE(req, pr, progressPhases)
		if err != nil {
			PrintError(os.Stdout, err, req, httpClient)
			return err
		}

		// Mark that observations were already streamed, skip re-rendering them
		pr.SetStreamedObservations(true)
		// Print final summary (SSE already showed phase progress)
		pr.PrintResults(resp)
	} else {
		// Standard execution mode with progress indicator
		// Used for CI, piped output, or when --no-stream is specified
		var stopProgress func()
		var tailer *LogTailer
		if !parsed.JSON && previewReady && len(progressPhases) > 0 {
			stopProgress = StartProgress(os.Stderr, progressPhases, estimateTargets, timeoutTargets)
		}

		var err error
		resp, raw, err = client.Run(req)
		if stopProgress != nil {
			stopProgress()
		}
		if tailer != nil {
			tailer.Stop()
		}
		if err != nil {
			PrintError(os.Stdout, err, req, httpClient)
			return err
		}
		if parsed.JSON {
			cliutil.PrintJSON(raw)
			return executionResultError(resp)
		}

		// Print results (header/plan already printed pre-execution)
		pr.PrintResults(resp)
	}

	if resp.Error != "" {
		fmt.Printf("\nError: %s\n", resp.Error)
	}

	return executionResultError(resp)
}

func executionResultError(resp Response) error {
	if resp.Success {
		return nil
	}
	return fmt.Errorf("suite execution completed with failures")
}

// isInteractiveTTY checks if stdout is connected to an interactive terminal.
// Returns true for interactive shells, false for piped output or CI environments.
func isInteractiveTTY() bool {
	fi, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	// Check if stdout is a character device (terminal)
	return (fi.Mode() & os.ModeCharDevice) != 0
}

// ParseArgs parses command line arguments for the execute command.
func ParseArgs(args []string) (Args, error) {
	if len(args) == 0 {
		return Args{}, usageError("usage: " + strings.TrimPrefix(UsageLine, "test-genie "))
	}
	out := Args{Scenario: args[0]}
	fs := flag.NewFlagSet("execute", flag.ContinueOnError)
	fs.StringVar(&out.Preset, "preset", "", "Preset name")
	fs.StringVar(&out.PhasesCSV, "phases", "", "Comma-separated phases to run")
	fs.StringVar(&out.SkipCSV, "skip", "", "Comma-separated phases to skip")
	fs.StringVar(&out.RequestID, "request-id", "", "Link to suite request")
	fs.BoolVar(&out.FailFast, "fail-fast", false, "Stop on first failure")
	fs.BoolVar(&out.Stream, "stream", false, "Force streaming mode (default for TTY)")
	fs.BoolVar(&out.NoStream, "no-stream", false, "Disable streaming, use progress spinner instead")
	fs.StringVar(&out.ScenarioPath, "scenario-path", "", "Absolute path to the scenario directory")
	fs.StringVar(&out.LogicalRepoRoot, "logical-repo-root", "", "Absolute repo root for repo-relative validation")
	fs.StringVar(&out.LogicalScenarioRelPath, "logical-scenario-relpath", "", "Logical scenario directory relative to --logical-repo-root")
	fs.StringVar(&out.UIURL, "ui-url", "", "UI URL for Lighthouse audits (e.g., http://localhost:3000)")
	fs.StringVar(&out.APIURL, "api-url", "", "API URL for integration checks (e.g., http://localhost:8080)")
	fs.StringVar(&out.BrowserlessURL, "browserless-url", "", "Browserless URL (default: BROWSERLESS_URL env or http://localhost:4110)")
	jsonOutput := cliutil.JSONFlag(fs)
	fs.SetOutput(flag.CommandLine.Output())
	if err := cliutil.ParseInterspersed(fs, args[1:]); err != nil {
		return Args{}, err
	}
	out.JSON = *jsonOutput
	out.ScenarioPath = strings.TrimSpace(out.ScenarioPath)
	out.LogicalRepoRoot = strings.TrimSpace(out.LogicalRepoRoot)
	out.LogicalScenarioRelPath = strings.TrimSpace(out.LogicalScenarioRelPath)
	if out.ScenarioPath != "" && !filepath.IsAbs(out.ScenarioPath) {
		return Args{}, fmt.Errorf("--scenario-path must be absolute")
	}
	if err := validateLogicalPlacementArgs(out.Scenario, out.LogicalRepoRoot, out.LogicalScenarioRelPath); err != nil {
		return Args{}, err
	}
	out.ExtraPhases = fs.Args()

	phaseList := cliutil.MergeArgs(cliutil.ParseCSV(out.PhasesCSV), out.ExtraPhases)
	skip := cliutil.ParseCSV(out.SkipCSV)

	normalizedPhases, err := phases.NormalizeSelection(phaseList)
	if err != nil {
		return Args{}, err
	}
	normalizedSkip, err := phases.NormalizeSelection(skip)
	if err != nil {
		return Args{}, err
	}
	out.Phases = normalizedPhases
	out.Skip = normalizedSkip
	return out, nil
}

func validateLogicalPlacementArgs(scenario, logicalRepoRoot, logicalScenarioRelPath string) error {
	if logicalRepoRoot == "" && logicalScenarioRelPath == "" {
		return nil
	}
	if logicalRepoRoot == "" || logicalScenarioRelPath == "" {
		return fmt.Errorf("--logical-repo-root and --logical-scenario-relpath must be provided together")
	}
	if !filepath.IsAbs(logicalRepoRoot) {
		return fmt.Errorf("--logical-repo-root must be absolute")
	}
	if filepath.IsAbs(logicalScenarioRelPath) {
		return fmt.Errorf("--logical-scenario-relpath must be relative")
	}
	cleanRel := filepath.Clean(logicalScenarioRelPath)
	if cleanRel == "." || cleanRel == "" {
		return fmt.Errorf("--logical-scenario-relpath must not be empty")
	}
	if cleanRel == ".." || strings.HasPrefix(cleanRel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("--logical-scenario-relpath must not escape the logical repo root")
	}
	if filepath.Base(cleanRel) != scenario {
		return fmt.Errorf("--logical-scenario-relpath must end with the scenario name")
	}
	return nil
}

// PrintError displays a formatted error box with debugging hints.
func PrintError(w io.Writer, err error, req Request, httpClient *cliutil.HTTPClient) {
	fmt.Fprintln(w, "╔═══════════════════════════════════════════════════════════════╗")
	fmt.Fprintf(w, "║  %-61s║\n", "TEST EXECUTION REQUEST FAILED")
	fmt.Fprintln(w, "╠═══════════════════════════════════════════════════════════════╣")
	fmt.Fprintf(w, "║  %-61s║\n", fmt.Sprintf("Scenario: %s", req.ScenarioName))
	if req.Preset != "" {
		fmt.Fprintf(w, "║  %-61s║\n", fmt.Sprintf("Preset: %s", req.Preset))
	}
	if len(req.Phases) > 0 {
		fmt.Fprintf(w, "║  %-61s║\n", fmt.Sprintf("Requested phases: %s", strings.Join(req.Phases, ", ")))
	}
	if len(req.Skip) > 0 {
		fmt.Fprintf(w, "║  %-61s║\n", fmt.Sprintf("Skip: %s", strings.Join(req.Skip, ", ")))
	}
	if req.FailFast {
		fmt.Fprintf(w, "║  %-61s║\n", "Fail-fast: enabled")
	}
	if httpClient != nil {
		if base := httpClient.BaseURL(); strings.TrimSpace(base) != "" {
			fmt.Fprintf(w, "║  %-61s║\n", fmt.Sprintf("API base: %s", base))
		}
		if timeout := httpClient.Timeout(); timeout > 0 {
			fmt.Fprintf(w, "║  %-61s║\n", fmt.Sprintf("HTTP timeout: %s", timeout))
		}
	}
	fmt.Fprintf(w, "║  %-61s║\n", fmt.Sprintf("Error: %v", err))
	fmt.Fprintln(w, "╚═══════════════════════════════════════════════════════════════╝")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Next steps:")
	fmt.Fprintf(w, "  • Check scenario logs: vrooli scenario logs %s\n", req.ScenarioName)
	fmt.Fprintf(w, "  • Verify scenario health: vrooli scenario status %s\n", req.ScenarioName)
	fmt.Fprintf(w, "  • Retry with streaming to inspect live output: test-genie execute %s --stream\n", req.ScenarioName)
}

func usageError(msg string) error {
	return errors.New(msg)
}

func plannedPhaseNames(preview execTypes.PlanPreview) []string {
	names := make([]string, 0, len(preview.Phases))
	for _, phase := range preview.Phases {
		if phase.Name == "" {
			continue
		}
		names = append(names, phase.Name)
	}
	return names
}

func phaseTimingTargets(preview execTypes.PlanPreview) (map[string]time.Duration, map[string]time.Duration) {
	estimates := make(map[string]time.Duration, len(preview.Phases))
	timeouts := make(map[string]time.Duration, len(preview.Phases))
	for _, phase := range preview.Phases {
		key := phases.NormalizeAlias(phases.NormalizeName(phase.Name))
		if key == "" {
			continue
		}
		if phase.EstimatedDurationSeconds > 0 {
			estimates[key] = time.Duration(phase.EstimatedDurationSeconds) * time.Second
		}
		if phase.TimeoutSeconds > 0 {
			timeouts[key] = time.Duration(phase.TimeoutSeconds) * time.Second
		}
	}
	return estimates, timeouts
}
