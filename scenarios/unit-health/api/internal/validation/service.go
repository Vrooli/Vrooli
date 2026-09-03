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
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"unit-health/internal/discovery"
	"unit-health/internal/evidence"
	"unit-health/internal/executor"
	"unit-health/internal/readiness"
	"unit-health/internal/runhistory"

	"github.com/vrooli/api-core/metrics"
	"github.com/vrooli/envkit-go"
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
	// MaxConcurrency bounds parallel command execution. Defaults to the
	// build-width lever so admission and width stop multiplying.
	MaxConcurrency int
	Admission      *executor.Admission
	// History persists executed runs and supplies cross-run timing/status
	// history for the diagnostics analyzer. Nil disables persistence; the
	// diagnostics then fall back to single-run signals only.
	History runhistory.Store
	// DependencyResolver reads the target closure from Scenario Dependency
	// Analyzer. Tests inject a deterministic closure; production defaults to
	// the live target-DAG export and remains conservative if it is unavailable.
	DependencyResolver DependencyResolver
	// ReadinessResolver consumes governed dependency readiness. It is distinct
	// from DependencyResolver, which only supplies graph closure for static
	// architecture analysis.
	ReadinessResolver interface {
		Check(context.Context, string, string, string) (readiness.Report, error)
	}
	// EvidenceStore is optional. When configured, exact complete execution
	// responses can be reused without starting runner children.
	EvidenceStore     *evidence.Store
	PolicyDigest      string
	RunnerProfile     string
	ToolchainIdentity string
	Now               func() time.Time
	cacheMu           sync.Mutex
	cacheFlights      map[string]*cacheFlight
}

type cacheFlight struct {
	done chan struct{}
}

func (s *Service) beginCacheFlight(key string) (bool, *cacheFlight) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.cacheFlights == nil {
		s.cacheFlights = make(map[string]*cacheFlight)
	}
	if flight, exists := s.cacheFlights[key]; exists {
		return false, flight
	}
	flight := &cacheFlight{done: make(chan struct{})}
	s.cacheFlights[key] = flight
	return true, flight
}

func (s *Service) finishCacheFlight(key string, flight *cacheFlight) {
	s.cacheMu.Lock()
	if current, exists := s.cacheFlights[key]; exists && current == flight {
		delete(s.cacheFlights, key)
		close(flight.done)
	}
	s.cacheMu.Unlock()
}

func cachedResponse(record evidence.Record, runID string) (Response, bool) {
	var response Response
	if err := json.Unmarshal(record.Payload, &response); err != nil {
		return Response{}, false
	}
	var savedWallTime int64
	response.CacheSavedCPUTimeMS = 0
	for _, command := range response.CommandResults {
		savedWallTime += command.DurationMS
		response.CacheSavedCPUTimeMS += command.CPUTimeMS
	}
	response.RunID = runID
	response.CacheHit = true
	response.CacheMissReason = ""
	response.CacheInvalidatedDimensions = nil
	response.CacheSavedWallTimeMS = savedWallTime
	response.CacheRetainedBytes = int64(len(record.Payload))
	response.Artifacts = buildArtifacts(response)
	return response, true
}

// Request identifies the validation target and execution options.
type Request struct {
	Scenario         string
	TargetKind       string
	Path             string
	Workspaces       []string
	IncludeExecution bool
	UseCache         bool
	FastTestOnly     bool
}

// Response is the engine's normalized result. It maps one-to-one onto
// validationv1.ValidateScenarioResponse.
type Response struct {
	RunID                      string
	Status                     string
	Summary                    string
	Scenario                   string
	TargetKind                 string
	TargetPath                 string
	DegradedReason             string
	Surfaces                   []Surface
	Workspaces                 []Workspace
	Plan                       ExecutionPlan
	CommandResults             []CommandResult
	Coverage                   []CoverageTarget
	ProjectionChecks           []ProjectionCheck
	Findings                   []Finding
	SuppressedFindings         []Finding
	Diagnostics                []Diagnostic
	Maturity                   Maturity
	NextSteps                  []string
	Artifacts                  []Artifact
	CacheHit                   bool
	CacheMissReason            string
	CacheInvalidatedDimensions []string
	CacheSavedWallTimeMS       int64
	CacheSavedCPUTimeMS        int64
	CacheRetainedBytes         int64
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
	ID                string
	Kind              string
	Language          string
	Framework         string
	RootPath          string
	PackageManager    string
	Status            string
	Confidence        float64
	ToolchainIdentity string
}

// Workspace is a testable unit with its canonical framework and commands.
type Workspace struct {
	ID                     string
	Language               string
	RootPath               string
	Framework              string
	CanonicalFramework     string
	TestCommand            string
	CoverageCommand        string
	TestExecutable         string
	TestArgs               []string
	TypecheckCommand       string
	TypecheckExecutable    string
	TypecheckArgs          []string
	CoverageExecutable     string
	CoverageArgs           []string
	TestArtifacts          []Artifact
	TestPath               string
	AdapterID              string
	AdapterVersion         string
	TestKind               string
	RunnerProfile          string
	Resource               ResourceLimits
	TimeoutSeconds         int
	NoOutputTimeoutSeconds int
	Hermetic               executor.HermeticPolicy
	PackageManager         string
	ToolchainIdentity      string
	Status                 string
	DegradedReason         string
}

// ExecutionPlan is the bounded set of commands Unit Health would run.
type ExecutionPlan struct {
	Commands []PlannedCommand
	Notes    string
}

// PlannedCommand is a single command in the execution plan.
type PlannedCommand struct {
	WorkspaceID            string
	Name                   string
	Command                string
	Executable             string
	Args                   []string
	Env                    map[string]string
	Artifacts              []Artifact
	Resource               ResourceLimits
	WorkingDirectory       string
	TimeoutSeconds         int
	NoOutputTimeoutSeconds int
	// Kind identifies the command graph stage. Unit Health currently submits
	// only test/coverage commands; dependency setup is owned externally by
	// Scenario Dependency Analyzer.
	Kind     string
	TestKind string
	Hermetic executor.HermeticPolicy
}

type ResourceLimits struct {
	CPUWeight   int
	MemoryBytes int64
	MaxWorkers  int
}

// Command kinds for PlannedCommand.Kind.
const kindTest = "test"

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
	CPUTimeMS        int64
	PeakRSSBytes     int64
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
	Suppressed    bool
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
func New() *Service {
	capacity := runtime.NumCPU() / 2
	if capacity < 1 {
		capacity = 1
	}
	// Keep aggregate memory admission bounded even when the host does not
	// expose a portable memory API. The floor accommodates one constrained UI
	// profile; CPU admission still scales with the host.
	memoryCapacity := int64(capacity) * 512 << 20
	if memoryCapacity < 2<<30 {
		memoryCapacity = 2 << 30
	}
	return &Service{Admission: executor.NewAdmission(capacity, memoryCapacity)}
}

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
	targetKind := orDefault(strings.TrimSpace(req.TargetKind), "scenario")
	inv, err := disc.Discover(ctx, req.Scenario, targetKind, req.Path, req.UseCache)
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
	if targetKind != "scenario" && targetKind != "package" && targetKind != "control-plane" {
		findings := []Finding{unsupportedTargetFinding(scenario, targetKind, nowStr)}
		resp := Response{
			RunID:      runID,
			Scenario:   scenario,
			TargetKind: targetKind,
			TargetPath: inv.RootPath,
			Status:     "failed",
			Summary:    fmt.Sprintf("%s target kind is not supported by unit-health.", targetKind),
			Findings:   findings,
			Maturity:   Maturity{Rung: 0, Label: "L0", Rationale: fmt.Sprintf("Target kind %q is unsupported; no maturity is claimed.", targetKind)},
		}
		resp.NextSteps = []string{"Route this target to a provider that declares and implements its target kind."}
		resp.Artifacts = buildArtifacts(resp)
		return resp, nil
	}
	inv = filterRequestedWorkspaces(inv, req.Workspaces)

	surfaces, workspaces, plan, findings := buildPlan(scenario, inv, nowStr)
	if req.FastTestOnly {
		plan = buildExecutionPlanForMode(workspaces, true)
	}
	readinessReport, readinessBlocks := s.checkDependencyReadiness(ctx, targetKind, scenario, inv.RootPath, req.IncludeExecution)
	if readinessBlocks {
		for _, requirement := range readinessReport.Requirements {
			findings = append(findings, Finding{
				ID:       codeTestDependencyMissing + "-" + requirement.ID,
				Scenario: scenario, Code: codeTestDependencyMissing, Category: "dependency", Severity: codeSeverity[codeTestDependencyMissing],
				Message: "Required test dependency is not ready.", Evidence: fmt.Sprintf("%s=%s source=%s", requirement.ID, requirement.Status, requirement.Source),
				Expected: "All declared unit-test dependencies are ready.", Observed: string(requirement.Status),
				WhyItMatters: "Unit Health must not run a test against an absent or stale toolchain.", Remediation: requirement.Remediation, CreatedAt: nowStr,
			})
		}
	}

	// Exact cache hits return after discovery, planning, and dependency
	// readiness, but before source analyzers. This keeps the provider's warm
	// path cheap without allowing a previously-valid result to mask a currently
	// unavailable governed dependency.
	var cacheKey evidence.Key
	cacheMissReason := ""
	cacheInvalidatedDimensions := []string(nil)
	var activeFlight *cacheFlight
	var activeFlightKey string
	defer func() {
		if activeFlight != nil {
			s.finishCacheFlight(activeFlightKey, activeFlight)
		}
	}()
	if req.IncludeExecution && req.UseCache && !readinessBlocks && s.EvidenceStore != nil && len(plan.Commands) > 0 {
		if key, keyErr := s.evidenceKeyForMode(inv.RootPath, targetKind, workspaces, req.FastTestOnly); keyErr == nil {
			cacheKey = key
			for {
				if cached, getErr := s.EvidenceStore.Get(key, now); getErr == nil {
					if cachedResponse, ok := cachedResponse(cached, runID); ok {
						return cachedResponse, nil
					}
					cacheMissReason = "corrupt_or_unreadable_evidence"
					cacheInvalidatedDimensions = []string{"evidence_integrity"}
				} else {
					if errors.Is(getErr, evidence.ErrCorrupt) {
						cacheMissReason = "corrupt_or_unreadable_evidence"
						cacheInvalidatedDimensions = []string{"evidence_integrity"}
					} else if errors.Is(getErr, evidence.ErrStale) {
						cacheMissReason = "stale_evidence"
						cacheInvalidatedDimensions = []string{"evidence_age"}
					} else if errors.Is(getErr, evidence.ErrMiss) {
						cacheMissReason = "exact_evidence_not_found_or_expired"
						cacheInvalidatedDimensions = []string{"exact_key_miss"}
					} else {
						cacheMissReason = "evidence_store_error"
						cacheInvalidatedDimensions = []string{"evidence_store"}
					}
				}
				owner, flight := s.beginCacheFlight(key.Digest)
				if owner {
					activeFlight, activeFlightKey = flight, key.Digest
					break
				}
				select {
				case <-flight.done:
					continue
				case <-ctx.Done():
					return Response{}, ctx.Err()
				}
			}
			// A miss is expected when source, policy, platform, or artifacts
			// changed. It must never prevent a correct fresh run.
		} else {
			cacheMissReason = "evidence_key_unavailable"
			cacheInvalidatedDimensions = []string{"source", "configuration", "dependency", "toolchain", "adapter", "policy", "runner_profile", "platform", "coverage"}
		}
	}

	static := collector.Stage("static-analysis")
	closure := DependencyClosure{}
	resolver := s.DependencyResolver
	if resolver == nil {
		resolver = sdaDependencyResolver{}
	}
	// A failed resolve is not fatal and must not silently weaken the companion
	// rules: each workspace widens this closure with its own go.mod before
	// matching, so reachability stays provable when the analyzer is unavailable.
	if resolved, resolveErr := resolver.Resolve(ctx, targetKind, scenario, inv.RootPath); resolveErr == nil {
		closure = resolved
	}
	var suppressedFindings []Finding
	var projectionChecks []ProjectionCheck
	if targetKind == "scenario" {
		projectionChecks = buildProjectionChecks(inv.RootPath, workspaces)
	}

	// Static analyzers run regardless of execution: they read source facts only.
	if targetKind == "scenario" {
		findings = append(findings, analyzeArchitectureWithClosure(scenario, workspaces, nowStr, closure)...)
		findings = append(findings, analyzeQuality(scenario, inv.RootPath, workspaces, nowStr)...)
	} else if targetKind == "package" || targetKind == "control-plane" {
		findings = append(findings, analyzePackageArchitectureWithClosure(scenario, workspaces, nowStr, closure)...)
	}
	static.Gauge("workspaces", float64(len(workspaces)))
	static.End()

	// The execute stage is GATED: it is opened ONLY when execution is requested
	// and there is something to run. Default (no-execution) validations leave
	// the profile free of execute-path timing entirely.
	var commandResults []CommandResult
	var coverage []CoverageTarget
	if req.IncludeExecution && len(plan.Commands) > 0 && !readinessBlocks {
		execStage := collector.Stage("execute")
		commandResults, findings = s.execute(ctx, scenario, plan, findings, nowStr, execStage)
		if targetKind == "scenario" {
			var covFindings []Finding
			coverage, covFindings = analyzeCoverage(scenario, inv.RootPath, workspaces, nowStr)
			findings = append(findings, covFindings...)
		}
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
	var diagnostics []Diagnostic
	var diagFindings []Finding
	if targetKind == "scenario" {
		diagnostics, diagFindings = analyzeDiagnostics(scenario, workspaces, plan, commandResults, history, nowStr)
	}
	findings = append(findings, diagFindings...)
	findings = applyConfiguredUnitPolicyWaivers(findings, scenario, inv.RootPath, nowStr)
	findings, suppressedFindings = splitSuppressedFindings(findings)

	resp := Response{
		RunID:                      runID,
		Scenario:                   scenario,
		TargetKind:                 targetKind,
		TargetPath:                 inv.RootPath,
		DegradedReason:             inv.DegradedReason,
		Surfaces:                   surfaces,
		Workspaces:                 workspaces,
		Plan:                       plan,
		CommandResults:             commandResults,
		Coverage:                   coverage,
		ProjectionChecks:           projectionChecks,
		Diagnostics:                diagnostics,
		Findings:                   findings,
		SuppressedFindings:         suppressedFindings,
		CacheMissReason:            cacheMissReason,
		CacheInvalidatedDimensions: cacheInvalidatedDimensions,
	}
	resp.Status = deriveStatus(inv, findings)
	if targetKind == "scenario" {
		resp.Maturity = s.assessMaturity(findings)
	} else {
		resp.Maturity = Maturity{Rung: 0, Label: "L0", Rationale: fmt.Sprintf("%s targets use package-aware validation; scenario maturity is not claimed.", targetKind)}
	}
	resp.Summary = summarize(scenario, surfaces, workspaces, findings)
	resp.NextSteps = nextSteps(resp.Status, inv)
	resp.Artifacts = buildArtifacts(resp)
	if req.IncludeExecution && req.UseCache && s.EvidenceStore != nil && cacheKey.Digest != "" && cacheableResponse(resp) {
		if payload, marshalErr := json.Marshal(resp); marshalErr == nil {
			resp.CacheRetainedBytes = int64(len(payload))
			if updated, updatedErr := json.Marshal(resp); updatedErr == nil {
				payload = updated
			}
			_ = s.EvidenceStore.Put(cacheKey, payload, now)
		}
	}

	// Persist the run for cross-run diagnostics. Only executed runs carry timing
	// worth recording. Best-effort: a persistence failure must not fail a run.
	if s.History != nil && req.IncludeExecution {
		_ = s.History.Record(ctx, buildRunRecord(resp, plan, now))
	}
	return resp, nil
}

// filterRequestedWorkspaces applies the CLI/API workspace selector before
// planning, analysis, caching, and execution. A selector is allowed to name a
// discovered surface id, its workspace root, or the root basename so callers
// can use stable ids without learning filesystem layout. Unknown selectors
// intentionally produce the normal explicit no-surface finding rather than
// silently validating every workspace.
func filterRequestedWorkspaces(inv discovery.Inventory, requested []string) discovery.Inventory {
	if len(requested) == 0 {
		return inv
	}
	wanted := make(map[string]struct{}, len(requested))
	for _, value := range requested {
		value = strings.TrimSpace(value)
		if value != "" {
			wanted[strings.ToLower(filepath.Clean(value))] = struct{}{}
		}
	}
	if len(wanted) == 0 {
		return inv
	}
	selected := make([]discovery.Surface, 0, len(wanted))
	for _, surface := range inv.Surfaces {
		candidates := []string{surface.ID, surface.RootPath, filepath.Base(filepath.Clean(surface.RootPath))}
		matched := false
		for _, candidate := range candidates {
			if _, ok := wanted[strings.ToLower(filepath.Clean(candidate))]; ok {
				matched = true
				break
			}
		}
		if matched {
			selected = append(selected, surface)
		}
	}
	if len(selected) == 0 {
		inv.DegradedReason = fmt.Sprintf("requested workspace(s) %q were not found among discovered surfaces", requested)
	}
	inv.Surfaces = selected
	return inv
}

func (s *Service) checkDependencyReadiness(ctx context.Context, targetKind, scenario, root string, executionRequested bool) (readiness.Report, bool) {
	if !executionRequested || s.ReadinessResolver == nil {
		return readiness.Report{Status: readiness.Ready, Source: "not-requested"}, false
	}
	report, err := s.ReadinessResolver.Check(ctx, targetKind, scenario, root)
	if err != nil {
		return readiness.Report{Status: readiness.Unavailable, Source: "scenario-dependency-analyzer", Requirements: []readiness.Requirement{{ID: "dependency-readiness", Kind: "readiness", Status: readiness.Unavailable, Source: "scenario-dependency-analyzer", Remediation: "Restore Scenario Dependency Analyzer readiness evidence before executing unit tests."}}}, true
	}
	if validateErr := report.Validate(); validateErr != nil {
		return readiness.Report{Status: readiness.Unavailable, Source: "scenario-dependency-analyzer", Requirements: []readiness.Requirement{{ID: "dependency-readiness", Kind: "readiness", Status: readiness.Unavailable, Source: "scenario-dependency-analyzer", Remediation: "Publish valid dependency-readiness evidence through Scenario Dependency Analyzer."}}}, true
	}
	return report, report.BlocksExecution()
}

func cacheableResponse(resp Response) bool {
	for _, result := range resp.CommandResults {
		if result.Status != executor.StatusPassed {
			return false
		}
	}
	return len(resp.CommandResults) > 0
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
	// adapter-declared coverage artifacts are workspace-scoped.
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
		concurrency = envkit.BuildWidthFrom(envkit.Env(os.Environ()))
	}
	for _, command := range plan.Commands {
		if command.Resource.MaxWorkers > 0 && command.Resource.MaxWorkers < concurrency {
			concurrency = command.Resource.MaxWorkers
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

	out := make([]CommandResult, 0, len(plan.Commands))
	// Validation is observational. Dependency setup belongs to Scenario
	// Dependency Analyzer or an explicit lifecycle workflow; Unit Health never
	// mutates a workspace or invokes a package manager.
	commands := buildExecCommands(plan.Commands)
	var testResults []executor.Result
	if s.Admission != nil {
		testResults = executor.RunAllWithAdmission(ctx, runner, commands, concurrency, s.Admission)
	} else {
		testResults = executor.RunAll(ctx, runner, commands, concurrency)
	}
	for i, r := range testResults {
		out = append(out, toCommandResult(r, plan.Commands[i]))
		if f, ok := executionFinding(scenario, r, now); ok {
			findings = append(findings, f)
		}
	}
	return out, findings
}

func buildExecCommands(planned []PlannedCommand) []executor.Command {
	cmds := make([]executor.Command, 0, len(planned))
	for _, pc := range planned {
		cmds = append(cmds, executor.Command{
			WorkspaceID:     pc.WorkspaceID,
			Name:            pc.Name,
			Executable:      pc.Executable,
			Args:            append([]string(nil), pc.Args...),
			Dir:             pc.WorkingDirectory,
			TimeoutSeconds:  pc.TimeoutSeconds,
			NoOutputTimeout: time.Duration(pc.NoOutputTimeoutSeconds) * time.Second,
			Hermetic:        pc.Hermetic,
			Env:             pc.Env,
			Artifacts:       toExecutorArtifacts(pc.Artifacts),
			Resources:       executor.ResourceLimits{CPUWeight: pc.Resource.CPUWeight, MemoryBytes: pc.Resource.MemoryBytes, MaxWorkers: pc.Resource.MaxWorkers},
		})
	}
	return cmds
}

func toExecutorArtifacts(in []Artifact) []executor.Artifact {
	out := make([]executor.Artifact, 0, len(in))
	for _, artifact := range in {
		out = append(out, executor.Artifact{Label: artifact.Label, Kind: artifact.Kind, Path: artifact.Reference})
	}
	return out
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
		CPUTimeMS:        r.CPUTimeMS,
		PeakRSSBytes:     r.PeakRSSBytes,
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
	case executor.ClassUnsupported:
		code, category = codeUnitPolicyInvalid, "policy"
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

func splitSuppressedFindings(findings []Finding) ([]Finding, []Finding) {
	active := make([]Finding, 0, len(findings))
	suppressed := make([]Finding, 0)
	for _, finding := range findings {
		if finding.Suppressed {
			finding.Suppressed = false
			suppressed = append(suppressed, finding)
			continue
		}
		active = append(active, finding)
	}
	return active, suppressed
}

func orDefault(v, def string) string {
	if v == "" {
		return def
	}
	return v
}
