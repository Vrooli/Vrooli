package execute

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/cli-core/cliutil"

	"test-genie/cli/execute/report"
	"test-genie/cli/internal/phases"
)

// Run executes the execute command.
func Run(client *Client, httpClient *cliutil.HTTPClient, args []string) error {
	parsed, err := ParseArgs(args)
	if err != nil {
		return err
	}

	req := Request{
		ScenarioName:   parsed.Scenario,
		Preset:         parsed.Preset,
		Phases:         parsed.Phases,
		Skip:           parsed.Skip,
		FailFast:       parsed.FailFast,
		SuiteRequestID: parsed.RequestID,
	}

	var phaseDescriptors []phases.Descriptor
	if desc, err := client.ListPhases(); err == nil {
		phaseDescriptors = desc
	} else {
		fmt.Fprintf(os.Stderr, "Warning: unable to load phase catalog (%v)\n", err)
	}
	durationTargets := phases.TargetDurations(phaseDescriptors)

	// Determine which phases will run (for pre-execution display)
	progressPhases := parsed.Phases
	if len(progressPhases) == 0 {
		progressPhases = []string{"structure", "dependencies", "unit", "integration", "business", "performance"}
	}

	// Create printer early for pre-execution output
	pr := report.New(
		os.Stdout,
		parsed.Scenario,
		req.Preset,
		parsed.Phases,
		parsed.Skip,
		req.FailFast,
		phaseDescriptors,
	)

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

		// Print final summary (SSE already showed phase progress)
		pr.PrintResults(resp)
	} else {
		// Standard execution mode with progress indicator
		// Used for CI, piped output, or when --no-stream is specified
		var stopProgress func()
		var tailer *LogTailer
		if !parsed.JSON {
			stopProgress = StartProgress(os.Stderr, progressPhases, durationTargets)
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
			return nil
		}

		// Print results (header/plan already printed pre-execution)
		pr.PrintResults(resp)
	}

	if resp.Error != "" {
		fmt.Printf("\nError: %s\n", resp.Error)
	}

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
		return Args{}, usageError("usage: execute <scenario> [phases...] [--preset quick] [--skip performance] [--request-id id] [--fail-fast] [--no-stream] [--json]")
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
	jsonOutput := cliutil.JSONFlag(fs)
	fs.SetOutput(flag.CommandLine.Output())
	if err := fs.Parse(args[1:]); err != nil {
		return Args{}, err
	}
	out.JSON = *jsonOutput
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
