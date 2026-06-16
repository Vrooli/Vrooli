package execute

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"test-genie/cli/execute/report"
	"test-genie/cli/internal/phases"

	"github.com/vrooli/cli-core/cliutil"

	execTypes "test-genie/cli/internal/execute"
)

const UsageLine = "test-genie execute <scenario> [phases...] [--preset quick] [--skip performance] [--scenario-path PATH] [--logical-repo-root PATH] [--logical-scenario-relpath PATH] [--ui-url URL] [--api-url URL] [--fail-fast] [--wait] [--json] [--jsonl]"

// HelpText returns the framework-rendered help body for the execute command.
func HelpText() string {
	return `Execute a scenario suite. The run is owned by the test-genie server, so it
survives this command being interrupted: the run id and a re-attach command are
printed up front, and a known-long run is launched in the background (use --wait
to always block inline). Re-attach with 'test-genie runs wait <scenario> <id>'.

Examples:
  test-genie execute swarm-manager
  test-genie execute swarm-manager standards quality integration
  test-genie execute swarm-manager --preset quick --fail-fast
  test-genie execute swarm-manager --wait            # block to completion inline (CI)
  test-genie execute swarm-manager --skip performance --json
  test-genie execute swarm-manager --jsonl           # newline-delimited phase events for TUIs
  test-genie execute demo --scenario-path /tmp/vrooli/scenarios/demo --preset comprehensive
  test-genie execute demo --scenario-path /tmp/vrooli/scenarios/demo --logical-repo-root /workspace/Vrooli --logical-scenario-relpath scenarios/demo --preset comprehensive`
}

// Run executes the execute command.
func Run(client *Client, args []string) error {
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
		DiagnosticsPreset:      parsed.DiagnosticsPreset,
		UIURL:                  parsed.UIURL,
		APIURL:                 parsed.APIURL,
		ScenarioPath:           scenarioPath,
		LogicalRepoRoot:        parsed.LogicalRepoRoot,
		LogicalScenarioRelPath: parsed.LogicalScenarioRelPath,
	}

	baseURL := client.BaseURL()

	// JSONL machine stream: skip all human pre-execution rendering and emit the
	// canonical newline-delimited event stream (run id on the first line). The
	// process exits with the suite's real result code.
	if parsed.JSONL {
		return RunDurable(baseURL, req, DurableOptions{JSONL: true})
	}

	var (
		preview        execTypes.PlanPreview
		previewReady   bool
		progressPhases []string
	)
	if planned, err := client.PreviewPlan(req); err == nil {
		preview = planned
		previewReady = true
		progressPhases = plannedPhaseNames(planned)
	} else if !parsed.JSON {
		fmt.Fprintf(os.Stderr, "Warning: unable to preview execution plan (%v)\n", err)
		if len(parsed.Phases) > 0 {
			progressPhases = append([]string(nil), parsed.Phases...)
		}
	}

	// Create printer early for pre-execution output.
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
	// Best-effort COMPLETENESS supplement from the fast cached status layer
	// (2s budget; silently absent when the scoring CLI is missing or slow).
	pr.SetScoreRunner(report.RunScoreCLI)
	if previewReady {
		pr.SetPlanPreview(preview)
	}

	// Print header and test plan IMMEDIATELY (before the run starts) so the
	// user sees what will run before any work begins.
	if !parsed.JSON {
		pr.PrintPreExecution(progressPhases)
	}

	// --json: programmatic block-to-verdict over the blocking REST adapter (the
	// run is still server-owned and cancel-survivable). Returns the full result
	// schema unchanged for existing consumers.
	if parsed.JSON {
		resp, raw, err := client.Run(req)
		if err != nil {
			printJSONExecutionError(raw, err)
			return err
		}
		cliutil.PrintJSON(raw)
		return executionResultError(resp)
	}

	// Default human path: a server-owned, cancel-survivable run. Prints the run
	// id + re-attach command up front, auto-backgrounds known-long runs (unless
	// --wait), and follows inline otherwise.
	return RunDurable(baseURL, req, DurableOptions{Wait: parsed.Wait, Printer: pr})
}

func printJSONExecutionError(raw []byte, err error) {
	if json.Valid(bytes.TrimSpace(raw)) {
		cliutil.PrintJSON(raw)
		return
	}
	payload, marshalErr := json.Marshal(map[string]interface{}{
		"success": false,
		"error":   err.Error(),
	})
	if marshalErr != nil {
		fmt.Printf("{\"success\":false,\"error\":%q}\n", err.Error())
		return
	}
	cliutil.PrintJSON(payload)
}

func executionResultError(resp Response) error {
	if resp.Success {
		return nil
	}
	return fmt.Errorf("suite execution completed with failures")
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
	fs.StringVar(&out.DiagnosticsPreset, "diagnostics-preset", "", "Playbooks diagnostics capture: none|light|full (overrides testing.json)")
	fs.BoolVar(&out.FailFast, "fail-fast", false, "Stop on first failure")
	fs.BoolVar(&out.Wait, "wait", false, "Block to completion inline; never auto-background (CI / scripted use)")
	fs.BoolVar(&out.JSONL, "jsonl", false, "Stream canonical newline-delimited phase events (run_started…run_completed) for TUIs")
	fs.StringVar(&out.ScenarioPath, "scenario-path", "", "Absolute path to the scenario directory")
	fs.StringVar(&out.LogicalRepoRoot, "logical-repo-root", "", "Absolute repo root for repo-relative validation")
	fs.StringVar(&out.LogicalScenarioRelPath, "logical-scenario-relpath", "", "Logical scenario directory relative to --logical-repo-root")
	fs.StringVar(&out.UIURL, "ui-url", "", "UI URL for Lighthouse audits (e.g., http://localhost:3000)")
	fs.StringVar(&out.APIURL, "api-url", "", "API URL for integration checks (e.g., http://localhost:8080)")
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
