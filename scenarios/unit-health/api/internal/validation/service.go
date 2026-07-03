// Package validation is Unit Health's core engine: it discovers test surfaces,
// plans and (in later phases) runs the canonical test commands, analyzes
// coverage/architecture/quality, and produces provider-local test maturity.
//
// Phase 2 ships the skeleton: a schema-valid Response that honestly reports a
// not-yet-implemented engine. Phase 3 wires Code Facts intake and the test plan
// builder, Phase 4 the bounded executor, and Phase 5 the analyzers and the
// maturity assessor. The Response shape mirrors the validation.proto contract
// one-to-one so the handler conversion stays a flat field copy.
package validation

import (
	"context"
	"fmt"
	"runtime"
	"strings"
	"time"

	"unit-health/internal/discovery"
	"unit-health/internal/executor"
	"unit-health/internal/runhistory"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/maturity-go/assessment"
)

// Service runs Unit Health validation. The Discoverer (Code Facts intake) and
// Spec (local maturity ladder) are injected; the bounded executor and the
// coverage/architecture/quality analyzers land as fields in later phases.
type Service struct {
	// Discoverer is the Code Facts intake seam. Defaults to a live
	// CodeFactsClient when nil so production wiring stays a no-op.
	Discoverer discovery.Discoverer
	// Locator resolves scenario/path to a root dir for the default discoverer.
	Locator discovery.Locator
	// Spec is the parsed `.vrooli/maturity.json`. When set, the engine computes
	// the local maturity summary from emitted findings.
	Spec *assessment.Spec
	// Executor runs planned commands when execution is requested. Defaults to a
	// bounded os/exec runner when nil; tests inject a fake.
	Executor executor.Runner
	// MaxConcurrency bounds parallel command execution. Defaults to NumCPU/2.
	MaxConcurrency int
	// History persists executed runs and supplies cross-run timing/status
	// history for the diagnostics analyzer. Nil disables persistence; the
	// diagnostics then fall back to single-run signals only.
	History runhistory.Store
	Now     func() time.Time
}

// Request identifies the validation target and execution options.
type Request struct {
	Scenario         string
	Path             string
	Workspaces       []string
	IncludeExecution bool
	UseCache         bool
}

// Response is the engine's normalized result. It maps one-to-one onto
// validationv1.ValidateScenarioResponse.
type Response struct {
	RunID            string
	Status           string
	Summary          string
	Scenario         string
	TargetKind       string
	TargetPath       string
	DegradedReason   string
	Surfaces         []Surface
	Workspaces       []Workspace
	Plan             ExecutionPlan
	CommandResults   []CommandResult
	Coverage         []CoverageTarget
	ProjectionChecks []ProjectionCheck
	Findings         []Finding
	Diagnostics      []Diagnostic
	Maturity         Maturity
	NextSteps        []string
	Artifacts        []Artifact
}

// Artifact is a labeled, typed reference into a run's outputs (the run id, a
// command's working directory, a coverage artifact location). It is derived
// from the rest of the Response so callers can navigate "where is this run /
// where are its outputs" without reconstructing paths from other fields.
type Artifact struct {
	Label     string
	Kind      string
	Reference string
}

// Surface is a discovered scenario surface from Code Facts.
type Surface struct {
	ID             string
	Kind           string
	Language       string
	Framework      string
	RootPath       string
	PackageManager string
	Status         string
	Confidence     float64
}

// Workspace is a testable unit with its canonical framework and commands.
type Workspace struct {
	ID                 string
	Language           string
	RootPath           string
	Framework          string
	CanonicalFramework string
	InstallCommand     string
	TestCommand        string
	CoverageCommand    string
	PackageManager     string
	Status             string
	DegradedReason     string
}

// ExecutionPlan is the bounded set of commands Unit Health would run.
type ExecutionPlan struct {
	Commands []PlannedCommand
	Notes    string
}

// PlannedCommand is a single command in the execution plan.
type PlannedCommand struct {
	WorkspaceID      string
	Name             string
	Command          string
	WorkingDirectory string
	TimeoutSeconds   int
	// Kind is "install" for a pre-test dependency install step or "test" for
	// the test/coverage command itself. Install steps run (and are gated)
	// before their workspace's test step so a missing dependency classifies as
	// TEST_DEPENDENCY_MISSING instead of a generic test misconfiguration.
	Kind string
}

// Command kinds for PlannedCommand.Kind.
const (
	kindInstall = "install"
	kindTest    = "test"
)

// CommandResult is the outcome of one executed command.
type CommandResult struct {
	Name             string
	Command          string
	WorkingDirectory string
	Status           string
	ExitCode         int
	StdoutExcerpt    string
	StderrExcerpt    string
	TimeoutSeconds   int
	FailureReason    string
	FailureClass     string
	DurationMS       int64
}

// CoverageTarget is per-file/per-surface coverage.
type CoverageTarget struct {
	ID              string
	Language        string
	SurfaceID       string
	FilePath        string
	CoveredLines    int64
	TotalLines      int64
	CoveragePercent float64
	Threshold       float64
	Status          string
}

// ProjectionCheck compares one resolved unit policy expectation with the
// native file/config evidence that should project it.
type ProjectionCheck struct {
	ID          string
	WorkspaceID string
	SurfaceID   string
	Key         string
	Owner       string
	FilePath    string
	PolicyValue string
	NativeValue string
	Status      string
	Remediation string
	FindingCode string
}

// Finding is a normalized Unit Health finding. Code maps to a
// `.vrooli/maturity.json` entry.
type Finding struct {
	ID            string
	Scenario      string
	SurfaceID     string
	WorkspaceID   string
	Language      string
	Framework     string
	Code          string
	Category      string
	Severity      string
	FilePath      string
	Symbol        string
	Message       string
	Evidence      string
	Expected      string
	Observed      string
	WhyItMatters  string
	Remediation   string
	SourceCommand string
	CreatedAt     string
}

// Diagnostic is a flake/runtime/hang diagnostic.
type Diagnostic struct {
	Kind        string
	WorkspaceID string
	Message     string
	Evidence    string
	Severity    string
}

// Maturity is the provider-local maturity summary.
type Maturity struct {
	Rung      int
	Label     string
	Rationale string
}

// New returns a Service with default (real-clock) wiring.
func New() *Service { return &Service{} }

func (s *Service) now() time.Time {
	if s.Now != nil {
		return s.Now()
	}
	return time.Now()
}

// Validate runs the Unit Health validation for the requested target.
//
// Phase 3 implements Code Facts intake and the test-plan builder: it discovers
// surfaces, resolves canonical test workspaces, emits discovery/config-gap
// findings, and produces a dry-run execution plan. Bounded execution (Phase 4)
// and the coverage/architecture/quality analyzers (Phase 5) extend the same
// Response without changing this signature.
func (s *Service) Validate(ctx context.Context, req Request) (Response, error) {
	now := s.now()
	nowStr := now.UTC().Format(time.RFC3339)
	runID := "uh-" + now.UTC().Format("20060102-150405")

	collector := metricsFrom(ctx)

	// discover and static-analysis are ALWAYS-ON: they run regardless of the
	// include_execution flag (cheap source-fact reads), so they belong to every
	// validation's profile.
	disc := s.Discoverer
	if disc == nil {
		disc = discovery.CodeFactsClient{Locator: s.Locator}
	}
	discover := collector.Stage("discover")
	inv, err := disc.Discover(ctx, req.Scenario, req.Path, req.UseCache)
	if err != nil {
		discover.End()
		return Response{}, fmt.Errorf("discover surfaces: %w", err)
	}
	discover.Gauge("surfaces", float64(len(inv.Surfaces)))
	discover.End()

	scenario := inv.Scenario
	if scenario == "" {
		scenario = req.Scenario
	}

	static := collector.Stage("static-analysis")
	surfaces, workspaces, plan, findings := buildPlan(scenario, inv, nowStr)
	projectionChecks := buildProjectionChecks(inv.RootPath, workspaces)

	// Static analyzers run regardless of execution: they read source facts only.
	findings = append(findings, analyzeArchitecture(scenario, workspaces, nowStr)...)
	findings = append(findings, analyzeQuality(scenario, inv.RootPath, workspaces, nowStr)...)
	static.Gauge("workspaces", float64(len(workspaces)))
	static.End()

	// The execute stage is GATED: it is opened ONLY when execution is requested
	// and there is something to run. Default (no-execution) validations leave
	// the profile free of execute-path timing entirely.
	var commandResults []CommandResult
	var coverage []CoverageTarget
	if req.IncludeExecution && len(plan.Commands) > 0 {
		execStage := collector.Stage("execute")
		commandResults, findings = s.execute(ctx, scenario, plan, findings, nowStr, execStage)
		var covFindings []Finding
		coverage, covFindings = analyzeCoverage(scenario, inv.RootPath, workspaces, nowStr)
		findings = append(findings, covFindings...)
		execStage.End()
	}

	// Load cross-run history before computing diagnostics so runtime-growth and
	// flake reflect prior runs (the current run is persisted afterward, so it is
	// not double-counted). Best-effort: history failures must not fail a run.
	var history []runhistory.CommandSample
	if s.History != nil && req.IncludeExecution {
		if hist, herr := s.History.CommandHistory(ctx, scenario, historyRunWindow); herr == nil {
			history = hist
		}
	}

	// Diagnostics fold in runtime-growth/flake from history plus runtime/hang
	// evidence from any executed commands (commandResults is empty on a dry run).
	diagnostics, diagFindings := analyzeDiagnostics(scenario, workspaces, plan, commandResults, history, nowStr)
	findings = append(findings, diagFindings...)

	resp := Response{
		RunID:            runID,
		Scenario:         scenario,
		TargetKind:       orDefault(inv.TargetKind, "scenario"),
		TargetPath:       inv.RootPath,
		DegradedReason:   inv.DegradedReason,
		Surfaces:         surfaces,
		Workspaces:       workspaces,
		Plan:             plan,
		CommandResults:   commandResults,
		Coverage:         coverage,
		ProjectionChecks: projectionChecks,
		Diagnostics:      diagnostics,
		Findings:         findings,
	}
	resp.Status = deriveStatus(inv, findings)
	resp.Maturity = s.assessMaturity(findings)
	resp.Summary = summarize(scenario, surfaces, workspaces, findings)
	resp.NextSteps = nextSteps(resp.Status, inv)
	resp.Artifacts = buildArtifacts(resp)

	// Persist the run for cross-run diagnostics. Only executed runs carry timing
	// worth recording. Best-effort: a persistence failure must not fail a run.
	if s.History != nil && req.IncludeExecution {
		_ = s.History.Record(ctx, buildRunRecord(resp, plan, now))
	}
	return resp, nil
}

// buildArtifacts derives the labeled output references for a completed run from
// the Response. The run id anchors the list; the target root, each executed
// command's working directory, and each coverage-producing workspace's root
// (where coverage.out / coverage/ is written) follow. References are
// deduplicated so a workspace with many per-file coverage targets yields one
// coverage artifact, not one per file.
func buildArtifacts(resp Response) []Artifact {
	arts := make([]Artifact, 0, 2+len(resp.CommandResults)+len(resp.Workspaces))
	if resp.RunID != "" {
		arts = append(arts, Artifact{Label: "Validation run", Kind: "run", Reference: resp.RunID})
	}
	if resp.TargetPath != "" {
		arts = append(arts, Artifact{Label: "Target", Kind: "target", Reference: resp.TargetPath})
	}
	for _, r := range resp.CommandResults {
		if r.WorkingDirectory == "" {
			continue
		}
		label := r.Name
		if label == "" {
			label = r.Command
		}
		arts = append(arts, Artifact{Label: label, Kind: "command", Reference: r.WorkingDirectory})
	}
	// Coverage artifacts live under each workspace that ran a coverage command;
	// the per-file CoverageTarget rows carry file paths but the artifact (Go
	// coverage.out / Vitest coverage/) is workspace-scoped.
	covered := make(map[string]bool, len(resp.Coverage))
	for _, c := range resp.Coverage {
		covered[c.SurfaceID] = true
	}
	for _, ws := range resp.Workspaces {
		if ws.CoverageCommand == "" || ws.RootPath == "" || !covered[ws.ID] {
			continue
		}
		arts = append(arts, Artifact{Label: "Coverage (" + ws.ID + ")", Kind: "coverage", Reference: ws.RootPath})
	}
	return arts
}

// buildRunRecord projects a completed Response into a persisted run record.
func buildRunRecord(resp Response, plan ExecutionPlan, started time.Time) runhistory.RunRecord {
	rec := runhistory.RunRecord{
		RunID:        resp.RunID,
		Scenario:     resp.Scenario,
		StartedAt:    started,
		Status:       resp.Status,
		MaturityRung: resp.Maturity.Rung,
	}
	for _, r := range resp.CommandResults {
		rec.Commands = append(rec.Commands, runhistory.CommandSample{
			RunID:        resp.RunID,
			StartedAt:    started,
			WorkspaceID:  workspaceForCommand(plan, r),
			Command:      r.Command,
			DurationMS:   r.DurationMS,
			Status:       r.Status,
			FailureClass: r.FailureClass,
		})
	}
	for _, c := range resp.Coverage {
		rec.Coverage = append(rec.Coverage, runhistory.CoverageSample{
			WorkspaceID: c.SurfaceID,
			File:        c.FilePath,
			Percent:     c.CoveragePercent,
		})
	}
	return rec
}

// execute runs the planned commands under the bounded executor and appends
// execution findings (failures, missing dependencies, hangs, misconfig) to the
// supplied findings. It returns the command results and the augmented findings.
func (s *Service) execute(ctx context.Context, scenario string, plan ExecutionPlan, findings []Finding, now string, execStage *metrics.Stage) ([]CommandResult, []Finding) {
	runner := s.Executor
	if runner == nil {
		runner = executor.Bounded{}
	}
	concurrency := s.MaxConcurrency
	if concurrency < 1 {
		if concurrency = runtime.NumCPU() / 2; concurrency < 1 {
			concurrency = 1
		}
	}

	// Per-workspace child stages narrow execute-path timing to each workspace,
	// with a `tests` gauge counting the test commands planned for it. A nil
	// execStage (no collector) makes every child call a no-op.
	wsStages := map[string]*metrics.Stage{}
	testCounts := map[string]int{}
	for _, pc := range plan.Commands {
		if pc.Kind == kindTest {
			testCounts[pc.WorkspaceID]++
		}
	}
	for _, pc := range plan.Commands {
		if _, seen := wsStages[pc.WorkspaceID]; seen {
			continue
		}
		st := execStage.Stage(pc.WorkspaceID)
		st.Gauge("tests", float64(testCounts[pc.WorkspaceID]))
		wsStages[pc.WorkspaceID] = st
	}
	defer func() {
		for _, st := range wsStages {
			st.End()
		}
	}()

	// Pass 1: dependency installs. These are independent across workspaces, so
	// they run concurrently, but each gates its own workspace's test command —
	// a failed install means the test is never run (it would just fail with a
	// missing-module error that misclassifies the real cause).
	var installPlanned []PlannedCommand
	for _, pc := range plan.Commands {
		if pc.Kind == kindInstall {
			installPlanned = append(installPlanned, pc)
		}
	}
	out := make([]CommandResult, 0, len(plan.Commands))
	failedInstall := map[string]bool{}
	installResults := executor.RunAll(ctx, runner, buildExecCommands(installPlanned), concurrency)
	for i, r := range installResults {
		pc := installPlanned[i]
		out = append(out, toCommandResult(r, pc))
		if r.Status != executor.StatusPassed {
			failedInstall[pc.WorkspaceID] = true
			findings = append(findings, installFinding(scenario, pc, r, now))
		}
	}

	// Pass 2: test/coverage commands, skipping workspaces whose install failed.
	var testPlanned []PlannedCommand
	for _, pc := range plan.Commands {
		if pc.Kind == kindInstall {
			continue
		}
		if failedInstall[pc.WorkspaceID] {
			out = append(out, CommandResult{
				Name:             pc.Name,
				Command:          pc.Command,
				WorkingDirectory: pc.WorkingDirectory,
				Status:           statusSkipped,
				FailureClass:     executor.ClassMissingDependency,
				FailureReason:    "dependency install failed; test command not run",
				TimeoutSeconds:   pc.TimeoutSeconds,
			})
			continue
		}
		testPlanned = append(testPlanned, pc)
	}
	testResults := executor.RunAll(ctx, runner, buildExecCommands(testPlanned), concurrency)
	for i, r := range testResults {
		out = append(out, toCommandResult(r, testPlanned[i]))
		if f, ok := executionFinding(scenario, r, now); ok {
			findings = append(findings, f)
		}
	}
	return out, findings
}

// statusSkipped marks a test command that was not run because its dependency
// install failed. It is a validation-layer outcome, not an executor result.
const statusSkipped = "skipped"

func buildExecCommands(planned []PlannedCommand) []executor.Command {
	cmds := make([]executor.Command, 0, len(planned))
	for _, pc := range planned {
		cmds = append(cmds, executor.Command{
			WorkspaceID:    pc.WorkspaceID,
			Name:           pc.Name,
			Argv:           strings.Fields(pc.Command),
			Dir:            pc.WorkingDirectory,
			TimeoutSeconds: pc.TimeoutSeconds,
		})
	}
	return cmds
}

func toCommandResult(r executor.Result, pc PlannedCommand) CommandResult {
	return CommandResult{
		Name:             r.Name,
		Command:          r.Command,
		WorkingDirectory: pc.WorkingDirectory,
		Status:           r.Status,
		ExitCode:         r.ExitCode,
		StdoutExcerpt:    r.Stdout,
		StderrExcerpt:    r.Stderr,
		TimeoutSeconds:   pc.TimeoutSeconds,
		FailureReason:    r.FailureReason,
		FailureClass:     r.FailureClass,
		DurationMS:       r.DurationMS,
	}
}

// installFinding maps a failed dependency install onto a TEST_DEPENDENCY_MISSING
// finding so the real cause (a broken/absent install) is not misreported as a
// test misconfiguration.
func installFinding(scenario string, pc PlannedCommand, r executor.Result, now string) Finding {
	evidence := r.FailureReason
	if tail := strings.TrimSpace(r.Stderr); tail != "" {
		evidence = r.FailureReason + "\n--- stderr tail ---\n" + tail
	} else if tail := strings.TrimSpace(r.Stdout); tail != "" {
		evidence = r.FailureReason + "\n--- stdout tail ---\n" + tail
	}
	return Finding{
		ID:            codeTestDependencyMissing + "-" + pc.WorkspaceID,
		Scenario:      scenario,
		WorkspaceID:   pc.WorkspaceID,
		Code:          codeTestDependencyMissing,
		Category:      "execution",
		Severity:      codeSeverity[codeTestDependencyMissing],
		Message:       fmt.Sprintf("Dependency install %q failed for workspace %q; tests could not run.", pc.Command, pc.WorkspaceID),
		Evidence:      evidence,
		Expected:      "Dependencies install cleanly (lockfile-frozen) before the test command runs.",
		Observed:      fmt.Sprintf("status=%s, class=%s, exit=%d", r.Status, r.FailureClass, r.ExitCode),
		WhyItMatters:  "Without installed dependencies the test command cannot run, so the workspace is left unvalidated.",
		Remediation:   "Commit a valid lockfile and ensure dependencies install; inspect the install output for the root cause.",
		SourceCommand: pc.Command,
		CreatedAt:     now,
	}
}

// executionFinding maps a non-passing command result onto a maturity finding.
func executionFinding(scenario string, r executor.Result, now string) (Finding, bool) {
	if r.Status == executor.StatusPassed {
		return Finding{}, false
	}
	var code, category string
	switch r.FailureClass {
	case executor.ClassMissingDependency:
		code, category = codeTestDependencyMissing, "execution"
	case executor.ClassTimeoutHang, executor.ClassNoOutputStall:
		code, category = codeTestTimeoutHang, "diagnostics"
	case executor.ClassMisconfiguration:
		code, category = codeTestMisconfiguration, "execution"
	default:
		code, category = codeTestExecutionFailure, "execution"
	}
	evidence := r.FailureReason
	if tail := strings.TrimSpace(r.Stderr); tail != "" {
		evidence = r.FailureReason + "\n--- stderr tail ---\n" + tail
	} else if tail := strings.TrimSpace(r.Stdout); tail != "" {
		evidence = r.FailureReason + "\n--- stdout tail ---\n" + tail
	}
	return Finding{
		ID:            code + "-" + r.WorkspaceID,
		Scenario:      scenario,
		WorkspaceID:   r.WorkspaceID,
		Code:          code,
		Category:      category,
		Severity:      codeSeverity[code],
		Message:       fmt.Sprintf("Command %q in workspace %q %s.", r.Command, r.WorkspaceID, r.Status),
		Evidence:      evidence,
		Expected:      "The workspace's tests run to completion and pass within the timeout.",
		Observed:      fmt.Sprintf("status=%s, class=%s, exit=%d, %dms", r.Status, r.FailureClass, r.ExitCode, r.DurationMS),
		WhyItMatters:  "A failing, hanging, or unrunnable test command blocks the scenario from being validated or hardened.",
		Remediation:   "Inspect the command output, fix the failure, and re-run with --include-execution.",
		SourceCommand: r.Command,
		CreatedAt:     now,
	}, true
}

// assessMaturity computes the provider-local maturity summary from findings.
// When no spec is wired (e.g. unit tests of the bare service) it degrades to an
// L0/unknown summary rather than guessing.
func (s *Service) assessMaturity(findings []Finding) Maturity {
	if s.Spec == nil {
		return Maturity{Rung: 0, Label: "L0", Rationale: "Maturity spec not loaded."}
	}
	assessed := make([]assessment.Finding, 0, len(findings))
	for _, f := range findings {
		assessed = append(assessed, assessment.Finding{
			Code:     f.Code,
			Severity: f.Severity,
			Message:  f.Message,
			Phase:    s.Spec.Phase,
		})
	}
	local := assessment.LocalMaturity(*s.Spec, assessed)
	rung := levelIndex(s.Spec, local.CurrentLevel)
	rationale := "All assessed Unit Health contracts are clean."
	if len(local.BlockingFindingCodes) > 0 {
		rationale = fmt.Sprintf("Reaching %s is blocked by: %v", local.NextLevel, local.BlockingFindingCodes)
	}
	return Maturity{Rung: rung, Label: orDefault(local.CurrentLevel, "L0"), Rationale: rationale}
}

func levelIndex(spec *assessment.Spec, id string) int {
	for i, lvl := range spec.Levels {
		if lvl.ID == id {
			return i
		}
	}
	return 0
}

func deriveStatus(inv discovery.Inventory, findings []Finding) string {
	if len(inv.Surfaces) == 0 {
		return "degraded"
	}
	for _, f := range findings {
		if f.Severity == "error" {
			return "failed"
		}
	}
	if inv.DegradedReason != "" {
		return "degraded"
	}
	return "passed"
}

func summarize(scenario string, surfaces []Surface, workspaces []Workspace, findings []Finding) string {
	if len(surfaces) == 0 {
		return fmt.Sprintf("%s: no testable surfaces discovered.", scenario)
	}
	errs := 0
	for _, f := range findings {
		if f.Severity == "error" {
			errs++
		}
	}
	return fmt.Sprintf("%s: %d surface(s), %d workspace(s), %d finding(s) (%d blocking).",
		scenario, len(surfaces), len(workspaces), len(findings), errs)
}

func nextSteps(status string, inv discovery.Inventory) []string {
	switch {
	case len(inv.Surfaces) == 0:
		return []string{"Add a discoverable test workspace; ensure Code Facts can describe the scenario."}
	case status == "failed":
		return []string{"Resolve the blocking configuration findings, then run with --include-execution to run the planned tests."}
	default:
		return []string{"Run with --include-execution to execute the planned tests and analyze coverage."}
	}
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
