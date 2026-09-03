package execute

import (
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

const UsageLine = "test-genie execute <target> [phases...] [--preset quick] [--skip performance] [--capture-profile baseline] [--scenario-path PATH] [--logical-repo-root PATH] [--logical-scenario-relpath PATH] [--ui-url URL] [--api-url URL] [--fail-fast] [--wait] [--json] [--jsonl]"

// HelpText returns the framework-rendered help body for the execute command.
func HelpText() string {
	return `Execute a scenario suite. The run is owned by the test-genie server, so it
survives this command being interrupted: the run id and a re-attach command are
printed up front, and a known-long run is launched in the background (use --wait
to always block inline). Re-attach with 'test-genie runs wait <scenario> <id>'.

Run-handle timing by output mode (all three share one server-owned StartRun):
  human   run id + re-attach command printed to stderr immediately; a known-long
          run auto-backgrounds (unless --wait).
  --jsonl first stdout event is run_started (carries run_id); each phase event
          streams as one JSON line; last event is run_completed.
  --json  blocks to completion and prints one final SuiteExecutionResult object
          to stdout (with executionId + runHandle). Because stdout stays a single
          object, the early run handle is emitted as one run_started JSON line on
          stderr at start, so a long run is never opaque until it finishes. Start
          and follow failures print a parseable {"success":false,...} object with
          the scenario (and run id, once started).

Examples:
  test-genie execute swarm-manager
  test-genie execute swarm-manager quality docs unit
  test-genie execute swarm-manager --preset quick --fail-fast
  test-genie execute swarm-manager --wait            # block to completion inline (CI)
  test-genie execute swarm-manager --skip performance --json
  test-genie execute swarm-manager --jsonl           # newline-delimited phase events for TUIs
  test-genie execute demo --scenario-path /tmp/vrooli/scenarios/demo --preset comprehensive
  test-genie execute demo structure --capture-profile baseline --wait
  test-genie execute demo --scenario-path /tmp/vrooli/scenarios/demo --logical-repo-root /workspace/Vrooli --logical-scenario-relpath scenarios/demo --preset comprehensive`
}

// Run executes the execute command.
func Run(client *Client, args []string) error {
	parsed, err := ParseArgs(args)
	if err != nil {
		return err
	}
	if !parsed.JSON && !parsed.JSONL {
		for _, warning := range parsed.PhaseWarnings {
			fmt.Fprintf(os.Stderr, "Warning: %s\n", warning)
		}
	}

	scenarioPath, err := resolveWorkspacePaths(&parsed)
	if err != nil {
		return err
	}
	if scenarioPath == "" && !strings.Contains(parsed.Scenario, ":") {
		// Resolve the physical scenario directory from the scenario name.
		// cliutil owns local environment details; the execute request only
		// carries the resulting path as workspace identity.
		scenarioPath = cliutil.ResolveScenarioPath(parsed.Scenario)
	}

	req := Request{
		ScenarioName:           parsed.Scenario,
		Target:                 targetExpression(parsed.Scenario),
		Preset:                 parsed.Preset,
		Phases:                 parsed.Phases,
		Skip:                   parsed.Skip,
		FailFast:               parsed.FailFast,
		DiagnosticsPreset:      parsed.DiagnosticsPreset,
		CaptureProfile:         parsed.CaptureProfile,
		RetainForEvidence:      parsed.RetainForEvidence,
		RetentionReason:        parsed.RetentionReason,
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

	// --json: block to completion over the same durable run path and emit the
	// final Response as one JSON object. Shares StartRun with the human and
	// JSONL modes — no separate blocking execution path — so --json is purely an
	// output contract. Skips the human pre-execution preview/printer entirely.
	if parsed.JSON {
		return RunDurable(baseURL, req, DurableOptions{JSON: true})
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
	} else {
		// Human path only (--json/--jsonl returned above).
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
	// user sees what will run before any work begins. (--json/--jsonl already
	// returned above; only the human path reaches here.)
	pr.PrintPreExecution(progressPhases)

	// Default human path: a server-owned, cancel-survivable run. Prints the run
	// id + re-attach command up front, auto-backgrounds known-long runs (unless
	// --wait), and follows inline otherwise.
	return RunDurable(baseURL, req, DurableOptions{Wait: parsed.Wait, Printer: pr})
}

func resolveWorkspacePaths(parsed *Args) (string, error) {
	scenarioPath := parsed.ScenarioPath
	if scenarioPath == "" {
		if hostMerged := strings.TrimSpace(os.Getenv("VROOLI_SANDBOX_MERGED_HOST")); hostMerged != "" {
			scenarioPath = filepath.Join(hostMerged, "scenarios", parsed.Scenario)
		}
	}
	if scenarioPath != "" {
		if _, err := os.Stat(scenarioPath); err != nil {
			return "", fmt.Errorf("physical scenario root %q is unreachable: %w", scenarioPath, err)
		}
	}
	if parsed.LogicalRepoRoot == "" && parsed.LogicalScenarioRelPath == "" {
		if repoRoot := strings.TrimSpace(os.Getenv("VROOLI_SANDBOX_REPO_ROOT")); repoRoot != "" {
			parsed.LogicalRepoRoot = repoRoot
			parsed.LogicalScenarioRelPath = filepath.Join("scenarios", parsed.Scenario)
		}
	}
	return scenarioPath, nil
}

func targetExpression(value string) string {
	if strings.Contains(value, ":") {
		return value
	}
	return ""
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
	fs.StringVar(&out.DiagnosticsPreset, "diagnostics-preset", "", "Workflow diagnostics capture: none|light|full (overrides testing.json)")
	fs.StringVar(&out.CaptureProfile, "capture-profile", "", "Capture depth: baseline disables phase-cache reuse for measurement runs")
	fs.BoolVar(&out.RetainForEvidence, "retain-for-evidence", false, "Retain this run's measurements under an expiring server-owned lease")
	fs.StringVar(&out.RetentionReason, "retention-reason", "", "Reason recorded for an evidence-retention lease")
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
	out.PhaseWarnings = append(out.PhaseWarnings, phases.DeprecatedAliasWarnings(phaseList)...)
	out.PhaseWarnings = append(out.PhaseWarnings, phases.DeprecatedAliasWarnings(skip)...)
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
