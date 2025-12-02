package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
)

// allowedPhases enumerates the standard phase set the planner understands.
var allowedPhases = []string{
	"structure",
	"dependencies",
	"unit",
	"integration",
	"e2e",
	"business",
	"performance",
}

// generateArgs holds parsed CLI inputs for "generate".
type generateArgs struct {
	Scenario  string
	Types     string
	Coverage  int
	Priority  string
	Notes     string
	NotesFile string
	JSON      bool
}

// executeArgs holds parsed CLI inputs for "execute".
type executeArgs struct {
	Scenario    string
	Preset      string
	PhasesCSV   string
	SkipCSV     string
	Phases      []string
	Skip        []string
	RequestID   string
	FailFast    bool
	Stream      bool
	JSON        bool
	ExtraPhases []string
}

// runTestsArgs holds parsed CLI inputs for "run-tests".
type runTestsArgs struct {
	Scenario string
	Type     string
	JSON     bool
}

func (a *App) cmdStatus() error {
	body, resp, err := a.services.Health.Status()
	if err != nil {
		return err
	}

	fmt.Printf("Status: %s\n", defaultValue(resp.Status, "unknown"))
	if resp.Service != "" {
		fmt.Printf("Service: %s\n", resp.Service)
	}
	if resp.Version != "" {
		fmt.Printf("Version: %s\n", resp.Version)
	}

	if resp.Operations.LastExecution != nil {
		icon := "✓"
		if !resp.Operations.LastExecution.Success {
			icon = "✗"
		}
		fmt.Printf("Last execution: %s %s (phases=%d failed=%d) completed %s\n",
			icon,
			resp.Operations.LastExecution.Scenario,
			resp.Operations.LastExecution.PhaseSummary.Total,
			resp.Operations.LastExecution.PhaseSummary.Failed,
			defaultValue(resp.Operations.LastExecution.CompletedAt, "n/a"),
		)
	}

	q := resp.Operations.Queue
	fmt.Printf("Queue: pending=%d queued=%d delegated=%d running=%d failed=%d\n", q.Pending, q.Queued, q.Delegated, q.Running, q.Failed)
	if q.OldestQueuedAgeSecs > 0 {
		fmt.Printf("       oldest queued: %ds\n", q.OldestQueuedAgeSecs)
	}

	if len(resp.Dependencies) > 0 {
		fmt.Println("Dependencies:")
		cliutil.PrintJSONMap(resp.Dependencies, 2)
	}

	if resp.Status == "" && len(body) > 0 {
		cliutil.PrintJSON(body)
	}
	return nil
}

func (a *App) cmdGenerate(args []string) error {
	parsed, err := parseGenerateArgs(args)
	if err != nil {
		return err
	}

	payload := GenerateRequest{
		ScenarioName:   parsed.Scenario,
		RequestedTypes: cliutil.ParseCSV(parsed.Types),
		Priority:       strings.ToLower(parsed.Priority),
		Notes:          parsed.Notes,
	}
	if parsed.Coverage > 0 {
		val := parsed.Coverage
		payload.CoverageTarget = &val
	}
	if parsed.NotesFile != "" {
		content, err := cliutil.ReadFileString(parsed.NotesFile)
		if err != nil {
			return fmt.Errorf("read notes file: %w", err)
		}
		payload.Notes = content
	}

	resp, raw, err := a.services.Suite.Generate(payload)
	if err != nil {
		return err
	}
	if parsed.JSON {
		cliutil.PrintJSON(raw)
		return nil
	}

	fmt.Printf("Suite request queued for %s\n", resp.ScenarioName)
	if resp.ID != "" {
		fmt.Printf("  Request ID : %s\n", resp.ID)
	}
	if resp.Status != "" {
		fmt.Printf("  Status     : %s\n", resp.Status)
	}
	if len(resp.RequestedTypes) > 0 {
		fmt.Printf("  Types      : %s\n", strings.Join(resp.RequestedTypes, ", "))
	}
	if resp.CoverageTarget != nil {
		fmt.Printf("  Coverage   : %d%%\n", *resp.CoverageTarget)
	}
	if resp.Priority != "" {
		fmt.Printf("  Priority   : %s\n", resp.Priority)
	}
	if resp.EstimatedQueueSec > 0 {
		fmt.Printf("  ETA        : ~%ds\n", resp.EstimatedQueueSec)
	}
	return nil
}

func (a *App) cmdExecute(args []string) error {
	parsed, err := parseExecuteArgs(args)
	if err != nil {
		return err
	}

	req := ExecuteRequest{
		ScenarioName:   parsed.Scenario,
		Preset:         parsed.Preset,
		Phases:         parsed.Phases,
		Skip:           parsed.Skip,
		FailFast:       parsed.FailFast,
		SuiteRequestID: parsed.RequestID,
	}

	var phaseDescriptors []PhaseDescriptor
	if desc, err := a.services.Phases.List(); err == nil {
		phaseDescriptors = desc
	} else {
		fmt.Fprintf(os.Stderr, "Warning: unable to load phase catalog (%v)\n", err)
	}
	_, durationTargets := makeDescriptorMaps(phaseDescriptors)

	var stopProgress func()
	var tailer *logTailer
	if !parsed.JSON {
		progressPhases := parsed.Phases
		if len(progressPhases) == 0 {
			progressPhases = []string{"structure", "dependencies", "unit", "integration", "business", "performance"}
		}
		stopProgress = startExecuteProgress(os.Stderr, progressPhases, durationTargets)
		if parsed.Stream {
			if paths := discoverScenarioPaths(parsed.Scenario); paths.TestDir != "" {
				t, err := startLogTailer(os.Stderr, filepath.Join(paths.TestDir, "artifacts"))
				if err == nil {
					tailer = t
				} else {
					fmt.Fprintf(os.Stderr, "Warning: unable to start log streaming: %v\n", err)
				}
			} else {
				fmt.Fprintf(os.Stderr, "Warning: unable to locate scenario test dir for streaming\n")
			}
		}
	}

	resp, raw, err := a.services.Suite.Execute(req)
	if stopProgress != nil {
		stopProgress()
	}
	if tailer != nil {
		tailer.Stop()
	}
	if err != nil {
		printExecuteError(os.Stdout, err, req, a.core.HTTPClient)
		return err
	}
	if parsed.JSON {
		cliutil.PrintJSON(raw)
		return nil
	}

	printer := newExecutionPrinter(
		os.Stdout,
		parsed.Scenario,
		req.Preset,
		parsed.Phases,
		parsed.Skip,
		req.FailFast,
		phaseDescriptors,
	)
	printer.Print(resp)

	if resp.Error != "" {
		fmt.Printf("\nError: %s\n", resp.Error)
	}

	if resp.Success {
		return nil
	}
	return fmt.Errorf("suite execution completed with failures")
}

func (a *App) cmdRunTests(args []string) error {
	parsed, err := parseRunTestsArgs(args)
	if err != nil {
		return err
	}

	req := RunTestsRequest{}
	if parsed.Type != "" {
		req.Type = parsed.Type
	}

	resp, raw, err := a.services.RunTests.Run(parsed.Scenario, req)
	if err != nil {
		return err
	}
	if parsed.JSON {
		cliutil.PrintJSON(raw)
		return nil
	}

	fmt.Printf("Scenario tests completed via %s\n", defaultValue(resp.Type, "unknown"))
	if len(resp.Command.Command) > 0 {
		fmt.Printf("  Command : %s\n", strings.Join(resp.Command.Command, " "))
	}
	if resp.Command.WorkingDir != "" {
		fmt.Printf("  Workdir : %s\n", resp.Command.WorkingDir)
	}
	if resp.LogPath != "" {
		fmt.Printf("  Logs    : %s\n", resp.LogPath)
	}
	if resp.Status != "" {
		fmt.Printf("  Status  : %s\n", resp.Status)
	}
	return nil
}

func parseGenerateArgs(args []string) (generateArgs, error) {
	if len(args) == 0 {
		return generateArgs{}, usageError("usage: generate <scenario> [--types unit,integration] [--coverage 95] [--priority normal] [--notes text] [--notes-file path] [--json]")
	}
	out := generateArgs{Scenario: args[0]}
	fs := flag.NewFlagSet("generate", flag.ContinueOnError)
	fs.StringVar(&out.Types, "types", "", "Comma-separated types to request")
	fs.IntVar(&out.Coverage, "coverage", 0, "Coverage target (1-100)")
	fs.StringVar(&out.Priority, "priority", "", "Priority (low|normal|high|urgent)")
	fs.StringVar(&out.Notes, "notes", "", "Notes for this request")
	fs.StringVar(&out.NotesFile, "notes-file", "", "Path to notes file")
	jsonOutput := cliutil.JSONFlag(fs)
	fs.SetOutput(flag.CommandLine.Output())
	if err := fs.Parse(args[1:]); err != nil {
		return generateArgs{}, err
	}
	out.JSON = *jsonOutput

	if out.Coverage < 0 || out.Coverage > 100 {
		return generateArgs{}, usageError("coverage must be between 0 and 100")
	}
	if out.Priority != "" && !isAllowedPriority(out.Priority) {
		return generateArgs{}, usageError("priority must be one of: low, normal, high, urgent")
	}
	return out, nil
}

func parseExecuteArgs(args []string) (executeArgs, error) {
	if len(args) == 0 {
		return executeArgs{}, usageError("usage: execute <scenario> [phases...] [--preset quick] [--skip performance] [--request-id id] [--fail-fast] [--stream] [--json]")
	}
	out := executeArgs{Scenario: args[0]}
	fs := flag.NewFlagSet("execute", flag.ContinueOnError)
	fs.StringVar(&out.Preset, "preset", "", "Preset name")
	fs.StringVar(&out.PhasesCSV, "phases", "", "Comma-separated phases to run")
	fs.StringVar(&out.SkipCSV, "skip", "", "Comma-separated phases to skip")
	fs.StringVar(&out.RequestID, "request-id", "", "Link to suite request")
	fs.BoolVar(&out.FailFast, "fail-fast", false, "Stop on first failure")
	fs.BoolVar(&out.Stream, "stream", false, "Stream live phase logs while the run executes")
	jsonOutput := cliutil.JSONFlag(fs)
	fs.SetOutput(flag.CommandLine.Output())
	if err := fs.Parse(args[1:]); err != nil {
		return executeArgs{}, err
	}
	out.JSON = *jsonOutput
	out.ExtraPhases = fs.Args()

	phases := cliutil.MergeArgs(cliutil.ParseCSV(out.PhasesCSV), out.ExtraPhases)
	skip := cliutil.ParseCSV(out.SkipCSV)

	normalizedPhases, err := normalizePhaseSelection(phases)
	if err != nil {
		return executeArgs{}, err
	}
	normalizedSkip, err := normalizePhaseSelection(skip)
	if err != nil {
		return executeArgs{}, err
	}
	out.Phases = normalizedPhases
	out.Skip = normalizedSkip
	return out, nil
}

func parseRunTestsArgs(args []string) (runTestsArgs, error) {
	if len(args) == 0 {
		return runTestsArgs{}, usageError("usage: run-tests <scenario> [--type phased] [--json]")
	}
	out := runTestsArgs{Scenario: args[0]}
	fs := flag.NewFlagSet("run-tests", flag.ContinueOnError)
	fs.StringVar(&out.Type, "type", "", "Test runner type to request")
	jsonOutput := cliutil.JSONFlag(fs)
	fs.SetOutput(flag.CommandLine.Output())
	if err := fs.Parse(args[1:]); err != nil {
		return runTestsArgs{}, err
	}
	out.JSON = *jsonOutput
	return out, nil
}

func defaultValue(val, fallback string) string {
	if strings.TrimSpace(val) == "" {
		return fallback
	}
	return val
}

func usageError(msg string) error {
	return errors.New(msg)
}

func isAllowedPriority(priority string) bool {
	switch strings.ToLower(priority) {
	case "low", "normal", "high", "urgent":
		return true
	default:
		return false
	}
}

func normalizePhaseSelection(phases []string) ([]string, error) {
	if len(phases) == 0 {
		return nil, nil
	}

	// Allow "all" (and synonyms) to request the default planner behavior.
	for _, phase := range phases {
		if p := normalizePhaseName(phase); p == "all" || p == "default" {
			return nil, nil
		}
	}

	allowed := make(map[string]struct{}, len(allowedPhases))
	for _, phase := range allowedPhases {
		allowed[phase] = struct{}{}
	}

	var normalized []string
	seen := make(map[string]struct{}, len(phases))
	for _, phase := range phases {
		normalizedName := normalizePhaseName(phase)
		if normalizedName == "" {
			continue
		}
		normalizedName = normalizePhaseAlias(normalizedName)
		if _, exists := allowed[normalizedName]; !exists {
			return nil, usageError(fmt.Sprintf("unknown phase '%s' (allowed: %s)", phase, strings.Join(allowedPhases, ",")))
		}
		if _, dup := seen[normalizedName]; dup {
			continue
		}
		seen[normalizedName] = struct{}{}
		normalized = append(normalized, normalizedName)
	}
	return normalized, nil
}

func normalizePhaseName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

func normalizePhaseAlias(name string) string {
	switch name {
	case "e2e":
		return "integration"
	default:
		return name
	}
}

func startExecuteProgress(w io.Writer, phases []string, targets map[string]time.Duration) func() {
	if len(phases) == 0 {
		phases = []string{"structure", "unit", "integration"}
	}
	start := time.Now()
	ticker := time.NewTicker(2 * time.Second)
	done := make(chan struct{})
	go func() {
		idx := 0
		for {
			select {
			case <-done:
				ticker.Stop()
				return
			case <-ticker.C:
				phase := phases[idx%len(phases)]
				idx++
				elapsed := time.Since(start).Truncate(time.Second)
				target := ""
				if d, ok := targets[normalizePhaseName(phase)]; ok && d > 0 {
					target = fmt.Sprintf(" target=%s", d.Truncate(time.Second))
				}
				fmt.Fprintf(w, "\rExecuting %-12s (t+%s%s)", phase, elapsed, target)
			}
		}
	}()
	return func() {
		close(done)
		fmt.Fprint(w, "\r")
	}
}

func printExecuteError(w io.Writer, err error, req ExecuteRequest, httpClient *cliutil.HTTPClient) {
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
