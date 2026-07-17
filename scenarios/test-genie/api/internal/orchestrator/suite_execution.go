package orchestrator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"test-genie/internal/basprobe"
	"test-genie/internal/captureprofile"
	"test-genie/internal/executionevidence"
	"test-genie/internal/orchestrator/phasepolicy"
	"test-genie/internal/orchestrator/phaseregistry"
	"test-genie/internal/orchestrator/phases"
	"test-genie/internal/orchestrator/providerdescriptor"
	"test-genie/internal/orchestrator/providerreadiness"
	"test-genie/internal/orchestrator/requirements"
	"test-genie/internal/orchestrator/runnability"
	"test-genie/internal/orchestrator/targetruntime"
	"test-genie/internal/playbooksclaims"
	"test-genie/internal/selfidentity"
	"test-genie/internal/shared"

	"github.com/google/uuid"

	workspacepkg "test-genie/internal/orchestrator/workspace"

	playbooksconfig "test-genie/internal/playbooks/config"
	sharedartifacts "test-genie/internal/shared/artifacts"
	sharedruns "test-genie/internal/shared/runs"

	"github.com/vrooli/freshness-go/treedigest"
	"github.com/vrooli/vrooli/packages/proto/architecture/findingid"
	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"
	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	runspb "github.com/vrooli/vrooli/packages/proto/gen/go/test-genie/v1/runs"
)

var (
	defaultPhaseTimeout = phases.DefaultTimeout
	// defaultExecutionPresets is sourced from the canonical phases catalog so
	// preset composition has a single source of truth; ValidatePresets guards
	// it against drift at orchestrator construction.
	defaultExecutionPresets  = phases.DefaultPresets()
	defaultPhaseSortFallback = 1000
)

type (
	PhaseEnvironment = workspacepkg.Environment
	PhaseRunReport   = phases.RunReport
	PhaseName        = phases.Name
	PhaseDescriptor  = phases.Descriptor
	PhaseCatalog     = phases.Catalog
	phaseDefinition  = phases.Definition
	phaseSpec        = phases.Spec
)

const (
	failureClassMisconfiguration  = phases.FailureClassMisconfiguration
	failureClassMissingDependency = phases.FailureClassMissingDependency
	failureClassTimeout           = phases.FailureClassTimeout
	failureClassSystem            = phases.FailureClassSystem
	PhaseStructure                = phases.Structure
	PhaseDependencies             = phases.Dependencies
	PhaseQuality                  = phases.Quality
	PhaseDocs                     = phases.Docs
	PhaseUnit                     = phases.Unit
	PhaseBusiness                 = phases.Business
	PhasePerformance              = phases.Performance
)

const MaxExecutionHistory = 50

// Phase status vocabulary. The shared skip-status helper lets a runnability
// skip flow into requirements sync as a non-executed phase rather than a
// failure.
const (
	phaseStatusPassed              = "passed"
	phaseStatusFailed              = "failed"
	phaseStatusSkipped             = "skipped"
	phaseStatusMissing             = "missing"
	phaseStatusNotExecutable       = "not_executable"
	phaseStatusNotRun              = "not_run"
	phaseStatusProviderUnavailable = "provider_unavailable"
)

func isSkippedPhaseStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case phaseStatusSkipped, phaseStatusMissing, phaseStatusNotExecutable, phaseStatusNotRun:
		return true
	default:
		return false
	}
}

// Suite verdict vocabulary (tri-state). Replaces the old binary success flag:
//   - PASS: every selected phase ran and passed (optional-phase skips are fine).
//   - PARTIAL: nothing failed, but a non-optional phase could not run (a
//     runnability SKIP) — exit 0, loudly labeled, machine-readable.
//   - FAIL: at least one phase failed — non-zero exit.
const (
	SuiteVerdictPass    = "PASS"
	SuiteVerdictPartial = "PARTIAL"
	SuiteVerdictFail    = "FAIL"
)

// SuiteOrchestrator runs scenario-local test phases without relying on external bash runners.
type SuiteOrchestrator struct {
	scenariosRoot string
	projectRoot   string
	phaseTimeout  time.Duration
	catalog       *phases.Catalog
	registry      *phaseregistry.Registry
	requirements  requirements.Syncer
	phaseToggles  *phaseToggleStore
	newRuntime    func(name, scenarioDir string) *targetruntime.Manager
	readiness     *providerreadiness.Manager
	claims        *playbooksclaims.Service
}

// SetClaims wires the playbooks-claims service used by the playbooks phase
// to guard against concurrent runs against the same scenario.
func (o *SuiteOrchestrator) SetClaims(svc *playbooksclaims.Service) {
	if o == nil {
		return
	}
	o.claims = svc
}

// SuiteExecutionRequest configures a single test execution run.
type SuiteExecutionRequest struct {
	ScenarioName string   `json:"scenarioName"`
	Preset       string   `json:"preset,omitempty"`
	Phases       []string `json:"phases,omitempty"`
	Skip         []string `json:"skip,omitempty"`
	FailFast     bool     `json:"failFast"`

	// RunID, when set, is the durable run identifier the run manager mints up
	// front so it can register and return the id synchronously (before the
	// suite actually starts executing). When empty, prepareExecution mints one.
	// Threading the id keeps a single run-id scheme and makes the start→finalize
	// index upsert idempotent under the pre-minted id.
	RunID string `json:"runId,omitempty"`

	// DiagnosticsPreset ("none"|"light"|"full"), when set, overrides the
	// playbooks diagnostics config for this run (richer BAS artifact capture).
	DiagnosticsPreset string `json:"diagnosticsPreset,omitempty"`

	// CaptureProfile is the capture-depth dial (orthogonal to the phase set).
	// "" = default depth with no extra all-pages capture; "baseline" =
	// all-pages visual capture + video for baseline diffs. See
	// internal/captureprofile.
	CaptureProfile string `json:"captureProfile,omitempty"`

	// Runtime URLs for phases that need to connect to running services.
	// UIURL/APIURL are optional overrides; when omitted, Test Genie manages the
	// target scenario lifecycle and discovers URLs from lifecycle process metadata.
	UIURL  string `json:"uiUrl,omitempty"`
	APIURL string `json:"apiUrl,omitempty"`

	// ScenarioPath is the absolute physical scenario directory to read and write.
	// When empty, the orchestrator resolves it from ScenarioName.
	ScenarioPath string `json:"scenarioPath,omitempty"`
	// LogicalRepoRoot and LogicalScenarioRelPath describe where the physical
	// scenario should be treated as living for repo-relative validation.
	LogicalRepoRoot        string `json:"logicalRepoRoot,omitempty"`
	LogicalScenarioRelPath string `json:"logicalScenarioRelPath,omitempty"`
}

// SuiteExecutionResult captures the outcome of a run.
type SuiteExecutionResult struct {
	ExecutionID uuid.UUID `json:"executionId,omitempty"`
	RunID       string    `json:"runId,omitempty"`
	// ArtifactDir is the stable, first-class run artifact root
	// (coverage/runs/<runID>/) holding per-phase logs, validator JSON, and the
	// findings document. Surfaced so a `--jsonl` consumer / TUI can locate a
	// run's outputs without re-deriving the layout.
	ArtifactDir  string    `json:"artifactDir,omitempty"`
	ScenarioName string    `json:"scenarioName"`
	StartedAt    time.Time `json:"startedAt"`
	CompletedAt  time.Time `json:"completedAt"`
	Success      bool      `json:"success"`
	// Verdict is the tri-state outcome (PASS/PARTIAL/FAIL). Success is kept for
	// backward compatibility and is true for both PASS and PARTIAL (only FAIL is
	// a non-zero exit), so a self-test that skips an unrunnable phase is honestly
	// reported without failing CI.
	Verdict                  string                      `json:"verdict,omitempty"`
	PresetUsed               string                      `json:"preset,omitempty"`
	RequestedPreset          string                      `json:"requestedPreset,omitempty"`
	RequestedPhases          []string                    `json:"requestedPhases,omitempty"`
	RequestedSkipPhases      []string                    `json:"requestedSkipPhases,omitempty"`
	PlannedPhases            []string                    `json:"plannedPhases,omitempty"`
	PhaseSetDigest           string                      `json:"phaseSetDigest,omitempty"`
	DescriptorSnapshotDigest string                      `json:"descriptorSnapshotDigest,omitempty"`
	ConfigurationFingerprint string                      `json:"configurationFingerprint,omitempty"`
	FailFast                 bool                        `json:"failFast"`
	Phases                   []PhaseExecutionResult      `json:"phases"`
	PhaseSummary             PhaseSummary                `json:"phaseSummary"`
	ProviderReadiness        []providerreadiness.Outcome `json:"providerReadiness,omitempty"`
	Warnings                 []string                    `json:"warnings,omitempty"`
	WarningSummary           WarningSummary              `json:"warningSummary"`
	// CampaignNudge is present only when the audit finding load exceeded the
	// single-pass threshold, steering the agent to open a tracked
	// improvement campaign. Nil otherwise.
	CampaignNudge *CampaignNudge `json:"campaignNudge,omitempty"`
	// Requirements summarizes PRD operational-target and requirement status for
	// the scenario. It is populated on every run that has a requirements/ tree:
	// freshly synced when the full suite ran, or cached (Synced=false) with a
	// skip reason when sync was gated. Nil when the scenario has no requirements.
	Requirements *requirements.SyncOutcome `json:"requirements,omitempty"`
}

type PhaseExecutionResult = phases.ExecutionResult

// WarningDetail captures a warning surfaced by a phase in structured output.
type WarningDetail struct {
	Message      string `json:"message"`
	Source       string `json:"source,omitempty"`
	LogPath      string `json:"logPath,omitempty"`
	ArtifactPath string `json:"artifactPath,omitempty"`
}

// PhaseWarningSummary groups warnings by phase.
type PhaseWarningSummary struct {
	Name     string          `json:"name"`
	Count    int             `json:"count"`
	Warnings []WarningDetail `json:"warnings,omitempty"`
}

// WarningSummary aggregates all non-fatal phase warnings.
type WarningSummary struct {
	Total  int                   `json:"total"`
	Phases []PhaseWarningSummary `json:"phases,omitempty"`
}

// PhaseSummary aggregates phase telemetry for quick status surfaces.
type PhaseSummary struct {
	Total            int `json:"total"`
	Passed           int `json:"passed"`
	Failed           int `json:"failed"`
	Skipped          int `json:"skipped"`
	DurationSeconds  int `json:"durationSeconds"`
	ObservationCount int `json:"observationCount"`
}

// ExecutionEventType identifies the kind of streaming event.
type ExecutionEventType string

const (
	EventPhaseStart  ExecutionEventType = "phase_start"
	EventPhaseEnd    ExecutionEventType = "phase_end"
	EventObservation ExecutionEventType = "observation"
	EventProgress    ExecutionEventType = "progress"
	EventComplete    ExecutionEventType = "complete"
)

// ExecutionEvent represents a streaming event during suite execution.
type ExecutionEvent struct {
	Type      ExecutionEventType `json:"type"`
	Timestamp time.Time          `json:"timestamp"`

	// For phase_start events
	Phase      string `json:"phase,omitempty"`
	PhaseIndex int    `json:"phaseIndex,omitempty"`
	PhaseTotal int    `json:"phaseTotal,omitempty"`

	// For phase_end events
	Status          string `json:"status,omitempty"`
	DurationSeconds int    `json:"durationSeconds,omitempty"`
	Error           string `json:"error,omitempty"`
	// PhasePresentation and FindingsSummary carry the provider-computed per-phase
	// standing (Phase Capability Contract) on phase_end events, so a follower
	// derives the standing from the same payload the terminal result carries. nil
	// for native phases / providers with no ladder.
	PhasePresentation *commonv1.PhasePresentation  `json:"-"`
	FindingsSummary   *runspb.PhaseFindingsSummary `json:"-"`

	// For observation events
	Message string `json:"message,omitempty"`

	// For complete events
	Result *SuiteExecutionResult `json:"result,omitempty"`
}

// ExecutionEventCallback is called for each event during streaming execution.
type ExecutionEventCallback func(event ExecutionEvent)

type preparedExecution struct {
	env       workspacepkg.Environment
	config    *workspacepkg.Config
	plan      *phasePlan
	runID     string
	runLogDir string
	result    *SuiteExecutionResult
}

type phaseRunContext struct {
	start       time.Time
	timeout     time.Duration
	phaseCtx    context.Context
	cancel      context.CancelFunc
	definition  phases.Definition
	logPath     string
	logFile     *os.File
	logWriter   io.Writer
	projectRoot string
}

func NewSuiteOrchestrator(scenariosRoot string) (*SuiteOrchestrator, error) {
	if strings.TrimSpace(scenariosRoot) == "" {
		return nil, fmt.Errorf("scenarios root path is required")
	}
	absRoot, err := filepath.Abs(scenariosRoot)
	if err != nil {
		return nil, fmt.Errorf("invalid scenarios root: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("scenarios root not accessible: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scenarios root must be a directory: %s", absRoot)
	}
	repoRoot := filepath.Dir(absRoot)
	repoBacked := looksLikeVrooliRepoRoot(repoRoot)
	descriptorLoad := providerdescriptor.Load(providerdescriptor.LoadOptions{RepoRoot: repoRoot})
	if err := descriptorLoad.Err(); err != nil {
		return nil, fmt.Errorf("load provider descriptors: %w", err)
	}
	var (
		catalog  *phases.Catalog
		registry *phaseregistry.Registry
	)
	if len(descriptorLoad.Descriptors) == 0 {
		if repoBacked {
			return nil, fmt.Errorf("load provider descriptors: no %s files found under %s", providerdescriptor.RelPath, repoRoot)
		}
		catalog = phases.NewDefaultCatalog(defaultPhaseTimeout)
		defaultDescriptorLoad := providerdescriptor.Load(providerdescriptor.LoadOptions{RepoRoot: sourceRepoRoot()})
		if err := defaultDescriptorLoad.Err(); err != nil {
			return nil, fmt.Errorf("load default provider descriptors: %w", err)
		}
		builtRegistry, err := phases.BuildDescriptorRegistry(defaultDescriptorLoad.Descriptors)
		if err != nil {
			return nil, fmt.Errorf("build default descriptor phase registry: %w", err)
		}
		registry = builtRegistry
		if err := phases.ValidatePresets(catalog); err != nil {
			return nil, fmt.Errorf("invalid default presets: %w", err)
		}
	} else {
		builtRegistry, err := phases.BuildDescriptorRegistry(descriptorLoad.Descriptors)
		if err != nil {
			return nil, err
		}
		registry = builtRegistry
		if repoBacked {
			catalog = phases.NewCatalogFromSpecs(defaultPhaseTimeout, phases.SpecsFromRegistry(registry)...)
		} else {
			catalog = phases.NewCatalogFromSpecs(defaultPhaseTimeout, phases.SpecsFromRegistry(registry)...)
		}
		if repoBacked {
			if err := phases.ValidatePresets(catalog); err != nil {
				return nil, fmt.Errorf("invalid descriptor-backed presets: %w", err)
			}
		}
	}
	return &SuiteOrchestrator{
		scenariosRoot: absRoot,
		projectRoot:   repoRoot,
		phaseTimeout:  defaultPhaseTimeout,
		catalog:       catalog,
		registry:      registry,
		// Use NewSyncer (not NewNodeSyncer): NewNodeSyncer returns a nil Syncer
		// when the legacy scripts/requirements/report.js is absent — which it is
		// in current installs — leaving requirements sync a silent no-op on every
		// execute. NewSyncer falls back to the native Go syncer so requirement/OT
		// status is actually derived and surfaced in the report.
		requirements: requirements.NewSyncer(filepath.Dir(absRoot)),
		phaseToggles: newPhaseToggleStore(),
		newRuntime:   targetruntime.New,
		readiness:    providerreadiness.NewManager(),
	}, nil
}

func sourceRepoRoot() string {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		return ""
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", "..", ".."))
}

// Execute performs a phased test run and returns the recorded result.
func (o *SuiteOrchestrator) Execute(ctx context.Context, req SuiteExecutionRequest) (*SuiteExecutionResult, error) {
	return o.execute(ctx, req, nil)
}

// ExecuteWithEvents performs a phased test run while streaming events via
// callback, enabling real-time progress reporting for SSE/Connect clients.
func (o *SuiteOrchestrator) ExecuteWithEvents(ctx context.Context, req SuiteExecutionRequest, emit ExecutionEventCallback) (*SuiteExecutionResult, error) {
	return o.execute(ctx, req, emit)
}

// execute is the single phased-run implementation. A nil emit runs silently
// (the blocking path); a non-nil emit streams phase events. The per-phase
// runners already no-op when emit is nil, so there is one code path regardless.
func (o *SuiteOrchestrator) execute(ctx context.Context, req SuiteExecutionRequest, emit ExecutionEventCallback) (*SuiteExecutionResult, error) {
	prepared, err := o.prepareExecution(req)
	if err != nil {
		return nil, err
	}
	readiness := o.checkProviderReadiness(ctx, prepared.env, prepared.plan.Selected, nil)
	prepared.result.ProviderReadiness = readiness.Outcomes

	env, runCtx, runtimeLease, runtimeManager, err := o.prepareTargetRuntime(ctx, prepared.env, readiness.Active, req, nil)
	if err != nil {
		return nil, err
	}
	prepared.env = env
	defer func() {
		if runtimeManager != nil {
			if cleanupErr := runtimeManager.Cleanup(context.Background(), runtimeLease, io.Discard); cleanupErr != nil {
				log.Printf("failed to clean up target runtime: %v", cleanupErr)
			}
		}
	}()

	phaseResults, anyFailure := o.runSelectedPhasesWithEvents(
		ctx,
		env,
		runCtx,
		prepared.runLogDir,
		prepared.plan.Selected,
		readiness.Blocked,
		req.FailFast,
		emit,
		buildPhaseWarningMap(prepared.plan),
	)

	return o.finalizeExecution(ctx, req, prepared, phaseResults, anyFailure, emit), nil
}

func (o *SuiteOrchestrator) prepareTargetRuntime(
	ctx context.Context,
	env workspacepkg.Environment,
	defs []phases.Definition,
	req SuiteExecutionRequest,
	logWriter io.Writer,
) (workspacepkg.Environment, runnability.RunContext, targetruntime.Lease, *targetruntime.Manager, error) {
	needs := runtimeNeeds(defs)
	newRuntime := o.newRuntime
	if newRuntime == nil {
		newRuntime = targetruntime.New
	}
	manager := newRuntime(env.ScenarioName, env.ScenarioDir)
	env.TargetRuntime = manager

	if req.UIURL != "" {
		env.UIURL = req.UIURL
		needs.UI = false
	}
	if req.APIURL != "" {
		env.APIURL = req.APIURL
		needs.API = false
	}

	env, lease, err := o.bringUpTargetSurfaces(ctx, env, manager, needs, logWriter)
	if err != nil {
		return env, runnability.RunContext{}, targetruntime.Lease{}, manager, err
	}

	// The run context is computed from the surfaces that ended up live (whether
	// reused from a self-target or started for another target). The per-phase
	// runnability gate reads it to decide RUN / RUN_DEGRADED / SKIP. The
	// resource map is probed from the selected phases' declared requirements so
	// BAS-dependent phases skip/degrade when BAS is unreachable rather than
	// failing hard.
	rc := resolveRunContext(env, targetruntime.URLs{}, false, "", resolveResources(ctx, defs))
	return env, rc, lease, manager, nil
}

// resolveResources probes availability for the local resources the selected
// phases declare as required, returning the name→available map the runnability
// gate consumes. Each resource is probed at most once and only when some
// selected phase needs it (so runs without that phase pay nothing).
func resolveResources(ctx context.Context, defs []phases.Definition) map[string]bool {
	required := make(map[string]struct{})
	for _, def := range defs {
		for _, r := range def.Capabilities.RequiredResources {
			required[r] = struct{}{}
		}
	}
	if len(required) == 0 {
		return nil
	}
	resources := make(map[string]bool, len(required))
	for name := range required {
		switch name {
		case runnability.ResourceBAS:
			resources[name] = basprobe.ProbeBAS(ctx)
		default:
			// Unknown resource: leave unset (treated as unavailable) so a
			// phase that requires it skips with a clear reason rather than
			// silently passing the gate.
		}
	}
	return resources
}

// bringUpTargetSurfaces resolves the target's UI/API URLs into env, honoring the
// self-host guard: a self-target is never started/restarted (that would SIGTERM
// the suite process) — only its already-live surfaces are reused. A different
// target is started on demand via EnsureRunning.
func (o *SuiteOrchestrator) bringUpTargetSurfaces(
	ctx context.Context,
	env workspacepkg.Environment,
	manager *targetruntime.Manager,
	needs targetruntime.Needs,
	logWriter io.Writer,
) (workspacepkg.Environment, targetruntime.Lease, error) {
	if selfidentity.Is(strings.TrimSpace(env.ScenarioName)) {
		live := manager.LiveSurfaces(ctx)
		if env.UIURL == "" {
			env.UIURL = live.UI
		}
		if env.APIURL == "" {
			env.APIURL = live.API
		}
		return env, targetruntime.Lease{}, nil
	}

	if !needs.UI && !needs.API {
		return env, targetruntime.Lease{}, nil
	}

	lease, err := manager.EnsureRunning(ctx, needs, logWriter)
	if err != nil {
		return env, targetruntime.Lease{}, err
	}
	if env.UIURL == "" {
		env.UIURL = lease.URLs.UI
	}
	if env.APIURL == "" {
		env.APIURL = lease.URLs.API
	}
	return env, lease, nil
}

// runtimeNeeds derives the union of required target surfaces directly from the
// per-phase capability manifest in the catalog SSOT. There is no hand-
// maintained phase→surface switch: a phase declares NeedsUI/NeedsAPI once, in
// the catalog, and both the runtime-needs computation and the runnability gate
// read the same field.
func runtimeNeeds(defs []phases.Definition) targetruntime.Needs {
	var needs targetruntime.Needs
	for _, def := range defs {
		if def.Capabilities.NeedsUI {
			needs.UI = true
		}
		if def.Capabilities.NeedsAPI {
			needs.API = true
		}
	}
	return needs
}

func newRunID() string {
	return sharedruns.NewRunID()
}

// resolveDiagnosticsPreset resolves the effective diagnostics preset for a run.
// An explicit DiagnosticsPreset always wins; otherwise the capture profile's
// paired preset applies (the baseline profile implies "full" diagnostics so its
// richer artifact capture is recorded). The default profile leaves the preset
// empty (cheap default), preserving routine comprehensive cost.
func resolveDiagnosticsPreset(req SuiteExecutionRequest) string {
	if p := strings.TrimSpace(req.DiagnosticsPreset); p != "" {
		return p
	}
	profile, _ := captureprofile.Resolve(req.CaptureProfile)
	return profile.DiagnosticsPreset()
}

// resolveRunDiagnostics maps the per-run diagnostics preset to the index's
// serialized diagnostics shape. An empty/unknown preset records the cheap
// default (console only), matching the playbooks config default.
func resolveRunDiagnostics(preset string) sharedruns.DiagnosticsConfig {
	d, ok := playbooksconfig.DiagnosticsPreset(strings.TrimSpace(preset))
	if !ok {
		d = playbooksconfig.DiagnosticsConfig{Console: true}
	}
	return sharedruns.DiagnosticsConfig{
		Video:   d.Video,
		Console: d.Console,
		Network: d.Network,
		HAR:     d.HAR,
		Trace:   d.Trace,
		DOM:     d.DOM,
	}
}

func (o *SuiteOrchestrator) prepareExecution(req SuiteExecutionRequest) (*preparedExecution, error) {
	planCtx, err := o.loadExecutionPlanContext(req)
	if err != nil {
		return nil, err
	}
	scenario := planCtx.env.ScenarioName

	// Honor a run id pre-minted by the run manager so the durable id is known
	// before execution begins; mint one only when running outside the manager.
	runID := strings.TrimSpace(req.RunID)
	if runID == "" {
		runID = newRunID()
	}
	planCtx.env.RunID = runID
	planCtx.env.CaptureProfile = strings.TrimSpace(req.CaptureProfile)
	planCtx.env.DiagnosticsPreset = resolveDiagnosticsPreset(req)
	if err := sharedartifacts.EnsureCoverageStructure(planCtx.env.ScenarioDir); err != nil {
		return nil, err
	}

	runLogDir := sharedartifacts.RunLogsDir(planCtx.env.ScenarioDir, runID)
	if err := os.MkdirAll(runLogDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create run log directory: %w", err)
	}

	// Record the run as in-progress so it is enumerable mid-flight; finalize
	// updates it to its terminal status. Git context and the tree digest are
	// captured at run START (not finalize) so they identify the byte-state
	// the phases actually executed against; both are best-effort and never
	// block the run.
	gitCtx := treedigest.CollectGitContext(planCtx.env.ScenarioDir)
	digest, digestErr := treedigest.Compute(planCtx.env.ScenarioDir)
	if digestErr != nil {
		log.Printf("tree digest unavailable for run %s: %v", runID, digestErr)
	}
	plannedPhases := phaseDefinitionNames(planCtx.plan.Selected)
	descriptorSnapshot, err := buildRunDescriptorSnapshot(planCtx.plan)
	if err != nil {
		return nil, fmt.Errorf("build run descriptor snapshot: %w", err)
	}
	if err := sharedruns.WriteDescriptorSnapshot(planCtx.env.ScenarioDir, runID, descriptorSnapshot); err != nil {
		return nil, fmt.Errorf("persist run descriptor snapshot: %w", err)
	}
	if err := sharedruns.NewIndex(planCtx.env.ScenarioDir).Append(sharedruns.RunRecord{
		RunID:                           runID,
		Scenario:                        scenario,
		StartedAt:                       time.Now().UTC(),
		Status:                          sharedruns.StatusInProgress,
		Diagnostics:                     resolveRunDiagnostics(planCtx.env.DiagnosticsPreset),
		GitSha:                          gitCtx.Sha,
		GitBranch:                       gitCtx.Branch,
		GitDirty:                        gitCtx.Dirty,
		GitDirtySummary:                 gitCtx.DirtySummary,
		TreeDigest:                      digest,
		Preset:                          strings.TrimSpace(req.Preset),
		CaptureProfile:                  strings.TrimSpace(req.CaptureProfile),
		PlannedPhases:                   plannedPhases,
		PhaseSetDigest:                  phases.PhaseSetDigest(plannedPhases),
		DescriptorSnapshotSchemaVersion: descriptorSnapshot.SchemaVersion,
		DescriptorSnapshotDigest:        descriptorSnapshot.Digest,
	}); err != nil {
		return nil, fmt.Errorf("record run %s in durable index: %w", runID, err)
	}

	return &preparedExecution{
		env:       planCtx.env,
		config:    planCtx.config,
		plan:      planCtx.plan,
		runID:     runID,
		runLogDir: runLogDir,
		result: &SuiteExecutionResult{
			RunID:                    runID,
			ArtifactDir:              sharedartifacts.RunDir(planCtx.env.ScenarioDir, runID),
			ScenarioName:             scenario,
			StartedAt:                time.Now().UTC(),
			PresetUsed:               planCtx.plan.PresetUsed,
			RequestedPreset:          phases.NormalizeKey(req.Preset),
			RequestedPhases:          normalizePhaseList(req.Phases),
			RequestedSkipPhases:      normalizePhaseList(req.Skip),
			PlannedPhases:            plannedPhases,
			PhaseSetDigest:           phases.PhaseSetDigest(plannedPhases),
			DescriptorSnapshotDigest: descriptorSnapshot.Digest,
			ConfigurationFingerprint: ExecutionConfigurationFingerprint(req, descriptorSnapshot.Digest),
			FailFast:                 req.FailFast,
			Warnings:                 buildPlanWarnings(planCtx.plan),
		},
	}, nil
}

// ExecutionConfigurationFingerprint captures the execution knobs that can
// materially change runtime without folding in volatile URLs or timestamps.
// The descriptor digest separately covers providers, policies, and descriptor
// revisions. Together with the selected phase-set digest these are the exact
// comparability key for full-run timing history.
func ExecutionConfigurationFingerprint(req SuiteExecutionRequest, descriptorDigest string) string {
	payload := strings.Join([]string{
		"v1",
		strings.TrimSpace(req.Preset),
		strings.TrimSpace(req.CaptureProfile),
		strings.TrimSpace(req.DiagnosticsPreset),
		fmt.Sprintf("fail-fast=%t", req.FailFast),
		strings.TrimSpace(descriptorDigest),
	}, "\n")
	sum := sha256.Sum256([]byte(payload))
	return "execution-config:" + hex.EncodeToString(sum[:])
}

func (o *SuiteOrchestrator) finalizeExecution(
	ctx context.Context,
	req SuiteExecutionRequest,
	prepared *preparedExecution,
	phaseResults []PhaseExecutionResult,
	anyFailure bool,
	emit ExecutionEventCallback,
) *SuiteExecutionResult {
	result := prepared.result
	result.RunID = prepared.runID
	result.CompletedAt = time.Now().UTC()
	// Tri-state verdict supersedes the binary flag. Success stays true for both
	// PASS and PARTIAL — only FAIL is a non-zero exit — so a self-test that
	// honestly skips an unrunnable phase does not fail CI.
	result.Verdict = computeSuiteVerdict(phaseResults, prepared.plan.Selected)
	result.Success = result.Verdict != SuiteVerdictFail
	_ = anyFailure // verdict derives failure from phase statuses (skips ≠ failures)
	result.Phases = phaseResults
	resultViews := buildPhaseResultViews(prepared.runLogDir, phaseResults)
	result.PhaseSummary = summarizePhaseViews(resultViews)
	result.WarningSummary = buildWarningSummaryFromViews(prepared.env.RunID, resultViews)

	// Persist the combined findings artifact BEFORE the nudge so the nudge can
	// point at a file that already exists on disk (the on-ramp the assessment
	// found broken). The path is run-deterministic regardless of write order.
	if err := o.writeFindingsArtifact(
		prepared.env.ScenarioDir,
		result.ScenarioName,
		prepared.runID,
		result.Verdict,
		result.CompletedAt,
		resultViews,
	); err != nil {
		log.Printf("failed to write findings artifact: %v", err)
	} else if err := writeEvidenceManifest(prepared.env.ScenarioDir, prepared.runID, result.ScenarioName, result.Verdict, result.CompletedAt, resultViews); err != nil {
		log.Printf("failed to write evidence manifest: %v", err)
	}
	// Inventory the bytes already owned by this run before publishing terminal
	// state. The catalog is metadata only: no provider artifact is copied into a
	// second store, and descriptor declarations assign producer metadata without
	// a phase-name registry.
	if err := writeArtifactCatalog(prepared.env.ScenarioDir, prepared.runID, result.CompletedAt); err != nil {
		warning := "artifact catalog unavailable: " + err.Error()
		result.Warnings = append(result.Warnings, warning)
		log.Printf("failed to write artifact catalog: %v", err)
	}

	artifactPath := sharedartifacts.RelativeRunFindingsArtifactPath(prepared.runID)
	if nudge := computeCampaignNudgeFromViews(result.ScenarioName, result.Verdict, artifactPath, resultViews); nudge != nil {
		result.CampaignNudge = nudge
		log.Printf("campaign nudge fired for %s: %d findings (%d blocker/error) — %s",
			result.ScenarioName, nudge.Total, nudge.Severe, nudge.Command)
	}

	if emit != nil {
		emit(ExecutionEvent{
			Type:      EventComplete,
			Timestamp: time.Now(),
			Result:    result,
		})
	}

	if err := o.writeLatestManifest(
		prepared.env.ScenarioDir,
		prepared.runID,
		result.StartedAt,
		result.CompletedAt,
		resultViews,
	); err != nil {
		log.Printf("failed to write latest manifest: %v", err)
	}

	result.Requirements = o.syncRequirementsIfNeeded(ctx, prepared.env, prepared.config, req, prepared.plan, phaseResults)
	o.finalizeRunRecord(prepared.env.ScenarioDir, prepared.runID, result, resultViews)

	return result
}

func writeArtifactCatalog(scenarioDir, runID string, generatedAt time.Time) error {
	var declarations []sharedartifacts.ArtifactPhaseDeclaration
	if snapshot, err := sharedruns.ReadDescriptorSnapshot(scenarioDir, runID); err == nil {
		declarations = make([]sharedartifacts.ArtifactPhaseDeclaration, 0, len(snapshot.Phases))
		for _, descriptor := range snapshot.Phases {
			declarations = append(declarations, sharedartifacts.ArtifactPhaseDeclaration{
				Phase: descriptor.Phase, EvidenceKinds: append([]string(nil), descriptor.EvidenceKinds...),
			})
		}
	}
	_, err := sharedartifacts.RefreshArtifactCatalog(scenarioDir, runID, declarations, generatedAt)
	return err
}

// finalizeRunRecord updates the run index entry with terminal status, per-phase
// summaries, and completion time. Pins set by external consumers are preserved
// because Update mutates the existing record in place.
func (o *SuiteOrchestrator) finalizeRunRecord(scenarioDir, runID string, result *SuiteExecutionResult, phaseResults []phaseResultView) {
	status := sharedruns.StatusPassed
	if !result.Success {
		status = sharedruns.StatusFailed
	}
	phaseRecords := make([]sharedruns.PhaseRecord, 0, len(phaseResults))
	descriptorByPhase := map[string]sharedruns.PhaseDescriptorSnapshot{}
	if snapshot, err := sharedruns.ReadDescriptorSnapshot(scenarioDir, runID); err == nil {
		for _, descriptor := range snapshot.Phases {
			descriptorByPhase[descriptor.Phase] = descriptor
		}
	}
	for _, p := range phaseResults {
		record := sharedruns.PhaseRecord{
			Name:            p.Name,
			Status:          p.Status,
			DurationSeconds: p.DurationSeconds,
			Comparable:      true,
		}
		if descriptor, ok := descriptorByPhase[p.Name]; ok {
			record.Advisory = descriptor.Policy.ResultGating == string(phasepolicy.ResultGatingAdvisory)
		}
		phaseRecords = append(phaseRecords, record)
	}
	err := sharedruns.NewIndex(scenarioDir).Finalize(runID, CompactTerminalSnapshot(result), func(r *sharedruns.RunRecord) error {
		r.Status = status
		r.CompletedAt = result.CompletedAt
		r.Phases = phaseRecords
		r.PlannedPhases = append([]string(nil), result.PlannedPhases...)
		r.PhaseSetDigest = phases.PhaseSetDigest(result.PlannedPhases)
		return nil
	})
	if err != nil {
		log.Printf("failed to finalize run %s in index: %v", runID, err)
	}
}

// CompactTerminalSnapshot is the terminal-snapshot persistence boundary. The
// in-memory result can contain provider findings and observations while a suite
// runs; the snapshot must retain only durable run metadata and phase summaries.
// Canonical detail is addressed through evidence-manifest.json instead.
func CompactTerminalSnapshot(result *SuiteExecutionResult) *SuiteExecutionResult {
	if result == nil {
		return nil
	}
	compact := *result
	compact.Phases = make([]PhaseExecutionResult, 0, len(result.Phases))
	for _, phase := range result.Phases {
		compact.Phases = append(compact.Phases, PhaseExecutionResult{
			Name:               phase.Name,
			Status:             phase.Status,
			DurationSeconds:    phase.DurationSeconds,
			Error:              phase.Error,
			Classification:     phase.Classification,
			Remediation:        phase.Remediation,
			RunnabilityVerdict: phase.RunnabilityVerdict,
			RunnabilityReason:  phase.RunnabilityReason,
			FindingSource:      phase.FindingSource,
			PhasePresentation:  phase.PhasePresentation,
			FindingsSummary:    phase.FindingsSummary,
		})
	}
	compact.Warnings = nil
	compact.WarningSummary = WarningSummary{Total: result.WarningSummary.Total}
	return &compact
}

// BuildWarningSummary converts WARNING observations into a deterministic
// execution-level summary while preserving the phase execution order.
func BuildWarningSummary(runID string, results []PhaseExecutionResult) WarningSummary {
	return buildWarningSummaryFromViews(runID, buildPhaseResultViews("", results))
}

func buildWarningSummaryFromViews(runID string, results []phaseResultView) WarningSummary {
	summary := WarningSummary{}
	for _, phase := range results {
		phaseSummary := PhaseWarningSummary{Name: phase.Name}
		for _, observation := range phase.Observations {
			if observation.Prefix != "WARNING" || strings.TrimSpace(observation.Text) == "" {
				continue
			}
			phaseSummary.Warnings = append(phaseSummary.Warnings, WarningDetail{
				Message:      strings.TrimSpace(observation.Text),
				Source:       "observation",
				LogPath:      phase.LogPath,
				ArtifactPath: phaseArtifactPath(runID, phase.Name),
			})
		}
		if len(phaseSummary.Warnings) == 0 {
			continue
		}
		phaseSummary.Count = len(phaseSummary.Warnings)
		summary.Total += phaseSummary.Count
		summary.Phases = append(summary.Phases, phaseSummary)
	}
	return summary
}

func phaseArtifactPath(runID, phaseName string) string {
	name := strings.TrimSpace(phaseName)
	if name == "" {
		return ""
	}
	return sharedartifacts.RelativePhaseResultsPath(runID, name+".json")
}

func (o *SuiteOrchestrator) runSelectedPhasesWithEvents(
	ctx context.Context,
	env workspacepkg.Environment,
	runCtx runnability.RunContext,
	runLogDir string,
	defs []phases.Definition,
	readiness map[string]providerreadiness.Outcome,
	failFast bool,
	emit ExecutionEventCallback,
	warnings map[string][]phases.Observation,
) ([]PhaseExecutionResult, bool) {
	if len(defs) == 0 {
		return nil, false
	}
	results := make([]PhaseExecutionResult, 0, len(defs))
	anyFailure := false
	total := len(defs)

	for idx, phase := range defs {
		// Emit phase start event
		if emit != nil {
			emit(ExecutionEvent{
				Type:       EventPhaseStart,
				Timestamp:  time.Now(),
				Phase:      phase.Name.String(),
				PhaseIndex: idx + 1,
				PhaseTotal: total,
			})
		}

		var phaseResult PhaseExecutionResult
		if outcome, blocked := readiness[phase.Name.Key()]; blocked {
			phaseResult = o.newProviderReadinessPhaseResult(phase, runLogDir, outcome)
		} else if verdict := resolvePhaseVerdict(phase, runCtx); verdict.IsSkip() {
			phaseResult = o.newSkippedPhaseResult(phase, runLogDir, verdict)
		} else {
			phaseResult = o.runPhaseWithEvents(ctx, env, runLogDir, phase, emit, mergeRunnabilityObservations(verdict, warnings[phase.Name.Key()]))
			annotatePhaseRunnability(&phaseResult, verdict)
		}

		// Emit phase end event
		if emit != nil {
			emit(ExecutionEvent{
				Type:              EventPhaseEnd,
				Timestamp:         time.Now(),
				Phase:             phase.Name.String(),
				Status:            phaseResult.Status,
				DurationSeconds:   phaseResult.DurationSeconds,
				Error:             phaseResult.Error,
				PhasePresentation: phaseResult.PhasePresentation,
				FindingsSummary:   phaseResult.FindingsSummary,
			})
		}

		if phaseResult.Status == phaseStatusFailed || phaseResult.Status == phaseStatusProviderUnavailable {
			anyFailure = true
		}
		results = append(results, phaseResult)
		if failFast && (phaseResult.Status == phaseStatusFailed || phaseResult.Status == phaseStatusProviderUnavailable) {
			break
		}
	}
	return results, anyFailure
}

// syncRequirementsIfNeeded synchronizes the PRD/requirements status from this
// run's evidence when the full suite ran, and otherwise reads back the last
// persisted counts so the report can always show requirement status. It returns
// a SyncOutcome (nil only when the scenario has no requirements/ tree) so the
// execute report can surface counts, deltas, and the skip reason on every run.
func (o *SuiteOrchestrator) syncRequirementsIfNeeded(
	ctx context.Context,
	env workspacepkg.Environment,
	cfg *workspacepkg.Config,
	req SuiteExecutionRequest,
	plan *phasePlan,
	phaseResults []PhaseExecutionResult,
) *requirements.SyncOutcome {
	if o.requirements == nil {
		return nil
	}
	history := buildCommandHistory(req, plan)
	input := requirements.SyncInput{
		ScenarioName:     env.ScenarioName,
		ScenarioDir:      env.ScenarioDir,
		PhaseDefinitions: planCoverageDefinitions(plan),
		PhaseResults:     phaseResults,
		CommandHistory:   history,
	}

	decision := newRequirementsSyncDecision(cfg, plan, phaseResults)
	if !decision.Execute {
		// Sync is gated (e.g. a partial/targeted run). Don't write, but read the
		// last persisted counts so the report still shows requirement status,
		// flagged stale with the skip reason. This is the common agent case.
		if decision.Reason != "" {
			log.Printf("requirements sync skipped: %s", decision.Reason)
		}
		outcome, err := o.requirements.Snapshot(ctx, input)
		if err != nil {
			log.Printf("requirements snapshot failed: %v", err)
			return nil
		}
		if outcome != nil {
			outcome.SkipReason = decision.Reason
			if outcome.SkipReason == "" {
				outcome.SkipReason = "partial run — requirements not updated"
			}
		}
		return outcome
	}
	if decision.Forced && decision.Reason != "" {
		log.Printf("forcing requirements sync despite: %s", decision.Reason)
	}
	// Allow partial sync on early abort (fail-fast). The syncer only updates
	// requirements for phases that actually ran; unaffected requirements keep
	// their previous status. This ensures PRD checkboxes stay honest even when
	// the test suite exits early.
	if len(phaseResults) < len(plan.Selected) {
		log.Printf("requirements sync proceeding with partial results: recorded %d of %d phases (fail-fast likely)", len(phaseResults), len(plan.Selected))
	}
	outcome, err := o.requirements.Sync(ctx, input)
	if err != nil {
		log.Printf("requirements sync skipped: %v", err)
		return nil
	}
	return outcome
}

func planCoverageDefinitions(plan *phasePlan) []phases.Definition {
	if plan == nil {
		return nil
	}
	if len(plan.Applicable) > 0 {
		return plan.Applicable
	}
	return plan.Definitions
}

func (o *SuiteOrchestrator) discoverPhaseDefinitions(_ workspacepkg.Environment) ([]phases.Definition, error) {
	definitions := make(map[string]phases.Definition)
	if o.registry != nil {
		for _, entry := range o.registry.All() {
			spec, ok := phases.SpecFromRegistryEntry(entry)
			if !ok {
				continue
			}
			definitions[spec.Name.Key()] = spec.ToDefinition()
		}
	}
	if o.catalog != nil {
		for _, spec := range o.catalog.All() {
			definitions[spec.Name.Key()] = spec.ToDefinition()
		}
	}
	var defs []phases.Definition
	for _, def := range definitions {
		defs = append(defs, def)
	}
	sort.Slice(defs, func(i, j int) bool {
		left := phaseSortValue(defs[i].Name, o.catalog)
		right := phaseSortValue(defs[j].Name, o.catalog)
		if left == right {
			return defs[i].Name.String() < defs[j].Name.String()
		}
		return left < right
	})
	return defs, nil
}

func (o *SuiteOrchestrator) descriptorEntry(phase string) (phaseregistry.Entry, bool) {
	if o == nil || o.registry == nil {
		return phaseregistry.Entry{}, false
	}
	return o.registry.Lookup(phase)
}

func (o *SuiteOrchestrator) descriptorPredicates() []providerdescriptor.Predicate {
	if o == nil || o.registry == nil {
		return nil
	}
	var predicates []providerdescriptor.Predicate
	for _, entry := range o.registry.All() {
		predicates = append(predicates, entry.Descriptor.Applicability.Any...)
		predicates = append(predicates, entry.Descriptor.Applicability.All...)
	}
	return predicates
}

func formatRegistryDiagnostics(diagnostics []phaseregistry.Diagnostic) string {
	if len(diagnostics) == 0 {
		return ""
	}
	parts := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		prefix := diagnostic.Code
		if diagnostic.Phase != "" {
			prefix = diagnostic.Phase + ":" + prefix
		}
		if diagnostic.Path != "" {
			prefix = diagnostic.Path + ":" + prefix
		}
		parts = append(parts, prefix+": "+diagnostic.Message)
	}
	return strings.Join(parts, "; ")
}

func looksLikeVrooliRepoRoot(root string) bool {
	if strings.TrimSpace(root) == "" {
		return false
	}
	for _, rel := range []string{
		"AGENTS.md",
		filepath.Join("scenarios", "test-genie"),
		filepath.Join("packages", "maturity-go"),
	} {
		if _, err := os.Stat(filepath.Join(root, rel)); err != nil {
			return false
		}
	}
	return true
}

type phaseSelectionNotices struct {
	Skipped  []phaseDisableNotice
	Explicit []phaseDisableNotice
}

func selectPhases(defs []phases.Definition, presets map[string][]string, req SuiteExecutionRequest, toggles PhaseToggleConfig) ([]phases.Definition, string, phaseSelectionNotices, error) {
	if len(defs) == 0 {
		return nil, "", phaseSelectionNotices{}, nil
	}
	index := make(map[string]phases.Definition, len(defs))
	for _, def := range defs {
		index[def.Name.Key()] = def
	}

	desired, presetUsed, err := resolveDesiredPhaseList(req, presets)
	if err != nil {
		return nil, "", phaseSelectionNotices{}, err
	}

	var notices phaseSelectionNotices
	var resolved []phases.Definition
	isDisabled := func(name string) (PhaseToggle, bool) {
		if toggles.Phases == nil {
			return PhaseToggle{}, false
		}
		toggle, ok := toggles.Phases[name]
		return toggle, ok && toggle.Disabled
	}
	isEnvDisabled := func(def phases.Definition) (string, bool) {
		envVar := strings.TrimSpace(def.SkipEnvVar)
		if envVar == "" {
			return "", false
		}
		return envVar, strings.TrimSpace(os.Getenv(envVar)) == "1"
	}

	if len(desired) == 0 {
		for _, def := range defs {
			if envVar, disabled := isEnvDisabled(def); disabled {
				notices.Skipped = append(notices.Skipped, phaseDisableNotice{Name: def.Name.String(), EnvVar: envVar})
				continue
			}
			if toggle, disabled := isDisabled(def.Name.Key()); disabled {
				notices.Skipped = append(notices.Skipped, phaseDisableNotice{Name: def.Name.String(), Toggle: toggle})
				continue
			}
			resolved = append(resolved, def)
		}
	} else {
		explicitRequest := len(req.Phases) > 0
		for _, phase := range desired {
			normalized := phases.NormalizeKey(phase)
			if normalized == "" {
				continue
			}
			def, ok := index[normalized]
			if !ok {
				return nil, "", phaseSelectionNotices{}, shared.NewValidationError(fmt.Sprintf("phase '%s' is not defined", phase))
			}
			if envVar, disabled := isEnvDisabled(def); disabled {
				notices.Skipped = append(notices.Skipped, phaseDisableNotice{Name: def.Name.String(), EnvVar: envVar})
				continue
			}
			if toggle, disabled := isDisabled(def.Name.Key()); disabled && !explicitRequest {
				notices.Skipped = append(notices.Skipped, phaseDisableNotice{Name: def.Name.String(), Toggle: toggle})
				continue
			}
			if toggle, disabled := isDisabled(def.Name.Key()); disabled && explicitRequest {
				notices.Explicit = append(notices.Explicit, phaseDisableNotice{Name: def.Name.String(), Toggle: toggle})
			}
			resolved = append(resolved, def)
		}
	}

	filtered, requestedSkipNotices := applySkipFilters(resolved, req.Skip)
	notices.Skipped = append(notices.Skipped, requestedSkipNotices...)
	return filtered, presetUsed, notices, nil
}

// runPhaseWithEvents runs a single phase, emitting observation events during
// execution when emit is non-nil (the per-phase writer no-ops emit otherwise).
func (o *SuiteOrchestrator) runPhaseWithEvents(ctx context.Context, env workspacepkg.Environment, runLogDir string, def phases.Definition, emit ExecutionEventCallback, preObservations []phases.Observation) PhaseExecutionResult {
	run, err := o.beginPhaseRun(ctx, env, runLogDir, def, emit, preObservations)
	if err != nil {
		return o.newPhaseSetupFailure(def.Name, runLogDir, err)
	}
	defer run.close()

	report := def.Runner(run.phaseCtx, env, run.logWriter)
	return o.completePhaseRun(run, report, preObservations)
}

func (o *SuiteOrchestrator) beginPhaseRun(
	ctx context.Context,
	env workspacepkg.Environment,
	runLogDir string,
	def phases.Definition,
	emit ExecutionEventCallback,
	preObservations []phases.Observation,
) (*phaseRunContext, error) {
	logPath := phaseLogPath(runLogDir, def.Name)
	logFile, err := os.Create(logPath)
	if err != nil {
		return nil, err
	}

	timeout := def.Timeout
	if timeout <= 0 {
		timeout = o.phaseTimeout
	}

	phaseCtx, cancel := context.WithTimeout(ctx, timeout)
	logWriter := io.Writer(logFile)
	if emit != nil {
		logWriter = io.MultiWriter(
			logFile,
			&observationEmitter{
				underlying: io.Discard,
				emit:       emit,
				phase:      def.Name.String(),
			},
		)
	}

	for _, obs := range preObservations {
		if obsStr := obs.String(); obsStr != "" {
			_, _ = fmt.Fprintln(logWriter, obsStr)
		}
	}

	return &phaseRunContext{
		start:       time.Now(),
		timeout:     timeout,
		phaseCtx:    phaseCtx,
		cancel:      cancel,
		definition:  def,
		logPath:     logPath,
		logFile:     logFile,
		logWriter:   logWriter,
		projectRoot: o.projectRoot,
	}, nil
}

func (p *phaseRunContext) close() {
	if p.cancel != nil {
		p.cancel()
	}
	if p.logFile != nil {
		_ = p.logFile.Close()
	}
}

func (o *SuiteOrchestrator) completePhaseRun(
	run *phaseRunContext,
	report phases.RunReport,
	preObservations []phases.Observation,
) PhaseExecutionResult {
	runErr := report.Err
	duration := int(math.Ceil(time.Since(run.start).Seconds()))
	if duration < 0 {
		duration = 0
	}

	status := phaseStatusPassed
	errMsg := ""
	classification := report.FailureClassification
	remediation := report.Remediation

	if runErr != nil {
		status = phaseStatusFailed
		errMsg = runErr.Error()
		if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(run.phaseCtx.Err(), context.DeadlineExceeded) {
			errMsg = fmt.Sprintf("phase timed out after %s", run.timeout)
			classification = phases.FailureClassTimeout
			if remediation == "" {
				remediation = "Increase the timeout or break the phase into smaller steps."
			}
		}
		if classification == "" {
			classification = phases.FailureClassSystem
		}
		if remediation == "" {
			remediation = "Refer to the phase logs to triage the failure."
		}
	}

	displayLogPath := run.logPath
	if relLog, err := filepath.Rel(run.projectRoot, run.logPath); err == nil {
		displayLogPath = relLog
	}

	result := PhaseExecutionResult{
		Name:              run.definition.Name.String(),
		Status:            status,
		DurationSeconds:   duration,
		LogPath:           displayLogPath,
		Error:             errMsg,
		Classification:    classification,
		Remediation:       remediation,
		Observations:      report.Observations,
		Findings:          report.Findings,
		Metrics:           report.Metrics,
		PhasePresentation: report.PhasePresentation,
		FindingsSummary:   report.FindingsSummary,
	}
	// Stamp the phase's finding-source token (empty for phases that emit no
	// findings) so a downstream campaign reaudit can derive which sources
	// this run covered — even when the phase produced zero findings.
	if run.definition.FindingSource != architecturev1.FindingSource_FINDING_SOURCE_UNSPECIFIED {
		result.FindingSource = findingid.SourceToken(run.definition.FindingSource)
	} else if report.FindingSource != "" {
		result.FindingSource = report.FindingSource
	}
	if len(preObservations) > 0 {
		result.Observations = append(preObservations, result.Observations...)
	}
	appendObservationsToLog(displayLogPath, run.projectRoot, result.Observations)
	return result
}

func (o *SuiteOrchestrator) newPhaseSetupFailure(name phases.Name, runLogDir string, err error) PhaseExecutionResult {
	return PhaseExecutionResult{
		Name:            name.String(),
		Status:          "failed",
		DurationSeconds: 0,
		LogPath:         phaseLogPath(runLogDir, name),
		Error:           fmt.Sprintf("failed to create log file: %v", err),
	}
}

func (o *SuiteOrchestrator) writeLatestManifest(scenarioDir, runID string, startedAt, completedAt time.Time, results []phaseResultView) error {
	latestDir := sharedartifacts.LatestDirPath(scenarioDir)
	if err := os.MkdirAll(latestDir, 0o755); err != nil {
		return fmt.Errorf("failed to create latest dir: %w", err)
	}

	logs := make(map[string]string, len(results))
	phaseEntries := make([]map[string]any, 0, len(results))

	for _, res := range results {
		logRel := sharedartifacts.RelPath(scenarioDir, res.LogAbs)
		logs[res.Name] = logRel

		phaseEntries = append(phaseEntries, map[string]any{
			"name":             res.Name,
			"status":           res.Status,
			"duration_seconds": res.DurationSeconds,
			"log":              logRel,
		})

		if err := updateLatestPointer(latestDir, phaseLogFileName(phases.Name(res.Name)), res.LogAbs); err != nil {
			return err
		}
	}

	manifest := map[string]any{
		"run_id":       runID,
		"started_at":   startedAt.UTC().Format(time.RFC3339),
		"completed_at": completedAt.UTC().Format(time.RFC3339),
		"logs":         logs,
		"phases":       phaseEntries,
	}

	writer := sharedartifacts.NewBaseWriter(scenarioDir, filepath.Base(scenarioDir), runID)
	return writer.WriteJSON(sharedartifacts.LatestManifestPath(scenarioDir), manifest)
}

// findingsArtifact is the per-run combined findings document. Its shape is a
// superset of the campaign `--from-audit` ingest contract (`phases[].findings`)
// so the nudge can point at a file that already exists on disk. Zero-finding
// phases are INCLUDED — their presence with a findingSource token is what lets
// a campaign reaudit derive which sources a partial run actually covered.
type findingsArtifact struct {
	Scenario    string                  `json:"scenario"`
	RunID       string                  `json:"runId"`
	Verdict     string                  `json:"verdict"`
	CompletedAt string                  `json:"completedAt"`
	Phases      []findingsArtifactPhase `json:"phases"`
}

type findingsArtifactPhase struct {
	Name          string                                `json:"name"`
	Status        string                                `json:"status"`
	FindingSource string                                `json:"findingSource,omitempty"`
	Findings      []*architecturev1.ArchitectureFinding `json:"findings"`
	// PhasePresentation + FindingsSummary carry the per-phase standing (Phase
	// Capability Contract) so `test-genie runs findings <run-id>` renders the same
	// standing on demand. Additive and omitempty — architecture-cartographer's
	// --from-audit ingest reads only phases[].findings, so this does not affect it.
	PhasePresentation *commonv1.PhasePresentation  `json:"phasePresentation,omitempty"`
	FindingsSummary   *runspb.PhaseFindingsSummary `json:"findingsSummary,omitempty"`
}

// writeFindingsArtifact persists the one canonical detailed findings document
// under coverage/runs/<runID>/findings.json. Encoding matches the suite
// `--json` report (encoding/json, enums as integers) so the cartographer ingest
// round-trips. The latest view is a lightweight manifest pointer, never a
// second findings copy.
func (o *SuiteOrchestrator) writeFindingsArtifact(scenarioDir, scenario, runID, verdict string, completedAt time.Time, results []phaseResultView) error {
	artifact := findingsArtifact{
		Scenario:    scenario,
		RunID:       runID,
		Verdict:     verdict,
		CompletedAt: completedAt.UTC().Format(time.RFC3339),
		Phases:      make([]findingsArtifactPhase, 0, len(results)),
	}
	for _, res := range results {
		artifact.Phases = append(artifact.Phases, findingsArtifactPhase{
			Name:              res.Name,
			Status:            res.Status,
			FindingSource:     res.FindingSource,
			Findings:          res.Findings,
			PhasePresentation: res.PhasePresentation,
			FindingsSummary:   res.FindingsSummary,
		})
	}
	writer := sharedartifacts.NewBaseWriter(scenarioDir, filepath.Base(scenarioDir), runID)
	if err := writer.EnsureDir(sharedartifacts.RunDir(scenarioDir, runID)); err != nil {
		return err
	}
	return writer.WriteJSON(sharedartifacts.RunFindingsArtifactPath(scenarioDir, runID), artifact)
}

// writeEvidenceManifest publishes the versioned canonical index only after its
// detailed findings owner is durable. All phase projections refer back to that
// one payload instead of embedding duplicate arrays.
func writeEvidenceManifest(scenarioDir, runID, scenario, verdict string, completedAt time.Time, results []phaseResultView) error {
	writer, err := executionevidence.NewWriter(sharedartifacts.RunDir(scenarioDir, runID))
	if err != nil {
		return err
	}
	findings, err := writer.ReferenceExisting("findings", "findings.document", sharedartifacts.FindingsArtifactFile, "application/json", "")
	if err != nil {
		return err
	}
	manifest := executionevidence.Manifest{
		SchemaVersion: executionevidence.SchemaVersion,
		RunID:         runID,
		Scenario:      scenario,
		CreatedAt:     completedAt.UTC(),
		Verdict:       verdict,
		Findings:      findings,
		Phases:        make([]executionevidence.PhaseSummary, 0, len(results)),
	}
	for _, result := range results {
		phase := executionevidence.PhaseSummary{
			Name:              result.Name,
			Status:            result.Status,
			DurationSeconds:   result.DurationSeconds,
			FindingCount:      len(result.Findings),
			ObservationCount:  len(result.Observations),
			FindingSource:     result.FindingSource,
			PhasePresentation: result.PhasePresentation,
			FindingsSummary:   result.FindingsSummary,
		}
		if phase.FindingCount > 0 {
			phase.Findings = &findings
		}
		manifest.Phases = append(manifest.Phases, phase)
	}
	return writer.WriteManifest(manifest)
}

func updateLatestPointer(latestDir, linkName, target string) error {
	linkPath := filepath.Join(latestDir, linkName)
	_ = os.Remove(linkPath)
	if err := os.Symlink(target, linkPath); err != nil {
		// Fallback: write target path into a small text file
		if writeErr := os.WriteFile(linkPath, []byte(target), 0o644); writeErr != nil {
			return fmt.Errorf("failed to create latest pointer for %s: %v (symlink error: %v)", linkName, writeErr, err)
		}
	}
	return nil
}

type phaseResultView struct {
	Name              string
	Status            string
	DurationSeconds   int
	LogPath           string
	LogAbs            string
	Observations      []phases.Observation
	FindingSource     string
	Findings          []*architecturev1.ArchitectureFinding
	PhasePresentation *commonv1.PhasePresentation
	FindingsSummary   *runspb.PhaseFindingsSummary
}

func buildPhaseResultViews(runLogDir string, results []PhaseExecutionResult) []phaseResultView {
	if len(results) == 0 {
		return nil
	}
	views := make([]phaseResultView, 0, len(results))
	for _, result := range results {
		findings := result.Findings
		if findings == nil {
			findings = []*architecturev1.ArchitectureFinding{}
		}
		name := phases.Name(result.Name)
		views = append(views, phaseResultView{
			Name:              result.Name,
			Status:            result.Status,
			DurationSeconds:   result.DurationSeconds,
			LogPath:           result.LogPath,
			LogAbs:            phaseLogPath(runLogDir, name),
			Observations:      result.Observations,
			FindingSource:     result.FindingSource,
			Findings:          findings,
			PhasePresentation: result.PhasePresentation,
			FindingsSummary:   result.FindingsSummary,
		})
	}
	return views
}

// observationEmitter wraps an io.Writer and emits observation events for lines with markers.
type observationEmitter struct {
	underlying io.Writer
	emit       ExecutionEventCallback
	phase      string
	buffer     []byte
}

// maxObservationLineBytes prevents an uncooperative phase process that emits a
// newline-free stream from retaining arbitrary log bytes in the orchestration
// process. Full logs remain owned by the file writer; observations are only a
// bounded diagnostic projection.
const maxObservationLineBytes = 4 * 1024

func (e *observationEmitter) Write(p []byte) (n int, err error) {
	// Write to underlying log
	n, err = e.underlying.Write(p)
	if err != nil || e.emit == nil {
		return n, err
	}

	// Buffer and scan for complete lines with observation markers
	e.buffer = append(e.buffer, p...)
	if len(e.buffer) > maxObservationLineBytes {
		// A partial line cannot be an event yet. Keep only its recent bounded
		// tail rather than allowing a malformed producer to grow run memory.
		e.buffer = append(e.buffer[:0], e.buffer[len(e.buffer)-maxObservationLineBytes:]...)
	}
	for {
		idx := -1
		for i, b := range e.buffer {
			if b == '\n' {
				idx = i
				break
			}
		}
		if idx < 0 {
			break
		}

		line := string(e.buffer[:idx])
		e.buffer = e.buffer[idx+1:]

		// Emit observation events for significant lines
		// Look for common markers that indicate progress
		if e.isSignificantLine(line) {
			e.emit(ExecutionEvent{
				Type:      EventObservation,
				Timestamp: time.Now(),
				Phase:     e.phase,
				Message:   line,
			})
		}
	}

	return n, nil
}

// isSignificantLine determines if a log line should be emitted as an observation.
func (e *observationEmitter) isSignificantLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}

	// Only emit lines that use explicit structured markers intended for the stream.
	// Raw command output still goes to the log file but will not be streamed.
	significantPrefixes := []string{
		"[SUCCESS]", "[ERROR]",
		"✅", "❌", "🧪", "SECTION",
	}
	for _, prefix := range significantPrefixes {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}

	return false
}

func SummarizePhases(phases []PhaseExecutionResult) PhaseSummary {
	return summarizePhaseViews(buildPhaseResultViews("", phases))
}

func summarizePhaseViews(phases []phaseResultView) PhaseSummary {
	summary := PhaseSummary{}
	for _, phase := range phases {
		summary.Total++
		if phase.DurationSeconds > 0 {
			summary.DurationSeconds += phase.DurationSeconds
		}
		summary.ObservationCount += len(phase.Observations)
		switch strings.ToLower(phase.Status) {
		case phaseStatusPassed:
			summary.Passed++
		case phaseStatusFailed, phaseStatusProviderUnavailable:
			summary.Failed++
		case phaseStatusSkipped:
			summary.Skipped++
		}
	}
	return summary
}

func phaseSortValue(name phases.Name, catalog *phases.Catalog) int {
	if catalog != nil {
		if index, ok := catalog.Order(name); ok {
			return index
		}
	}
	return defaultPhaseSortFallback
}

func phaseLogPath(runLogDir string, name phases.Name) string {
	return filepath.Join(runLogDir, phaseLogFileName(name))
}

func phaseLogFileName(name phases.Name) string {
	return fmt.Sprintf("%s.log", name.String())
}

func (o *SuiteOrchestrator) loadPresets(testDir string, cfg *workspacepkg.Config, allowed map[string]struct{}) map[string][]string {
	configPath := filepath.Join(testDir, "presets.json")
	var fileOverrides map[string][]string
	if raw, err := os.ReadFile(configPath); err == nil {
		_ = json.Unmarshal(raw, &fileOverrides)
	}

	var configOverrides map[string][]string
	if cfg != nil && len(cfg.Presets) > 0 {
		configOverrides = cfg.Presets
	}

	return phases.MergePresets(defaultExecutionPresets, fileOverrides, configOverrides, allowed)
}

// DescribePhases exposes catalog phase descriptors for HTTP clients.
func (o *SuiteOrchestrator) DescribePhases() []phases.Descriptor {
	if o == nil || o.catalog == nil {
		return nil
	}
	descriptors := o.catalog.Descriptors()
	indexByName := make(map[string]int, len(descriptors))
	for i, descriptor := range descriptors {
		indexByName[phases.NormalizeKey(descriptor.Name)] = i
	}
	if o.registry != nil {
		for _, entry := range o.registry.All() {
			descriptor := descriptorFromRegistryEntry(entry)
			key := phases.NormalizeKey(descriptor.Name)
			if index, ok := indexByName[key]; ok {
				descriptors[index] = descriptor
				continue
			}
			indexByName[key] = len(descriptors)
			descriptors = append(descriptors, descriptor)
		}
	}
	return descriptors
}

func descriptorFromRegistryEntry(entry phaseregistry.Entry) phases.Descriptor {
	spec, ok := phases.SpecFromRegistryEntry(entry)
	if !ok {
		return phases.Descriptor{}
	}
	provider := ""
	if spec.Delegated != nil {
		provider = spec.Delegated.ProviderScenario
	}
	return phases.Descriptor{
		Name:                  spec.Name.String(),
		DisplayName:           spec.DisplayName,
		Optional:              spec.Optional,
		Description:           spec.Description,
		Source:                spec.Source,
		Provider:              provider,
		DefaultTimeoutSeconds: int(spec.DefaultTimeout.Seconds()),
		DocPath:               spec.Doc,
		DescriptorPath:        entry.Descriptor.Path,
		SkipEnvVar:            spec.SkipEnvVar,
		Comparable:            spec.Comparable(),
		Advisory:              spec.Advisory,
		ArtifactBacked:        spec.ArtifactBacked,
		NonComparable:         spec.NonComparable,
		Policy:                spec.Policy,
		Runnability:           spec.Capabilities,
		FindingSource:         findingid.SourceToken(spec.FindingSource),
		ProfileMembership:     append([]string(nil), spec.ProfileMembership...),
		FreshnessRequirement:  spec.FreshnessRequirement,
		PhaseClass:            spec.PhaseClass,
		RuntimeClass:          spec.RuntimeClass,
		Dimensions:            append([]string(nil), spec.Dimensions...),
	}
}

// GlobalPhaseToggles returns the persisted global phase toggle configuration.
func (o *SuiteOrchestrator) GlobalPhaseToggles() (PhaseToggleConfig, error) {
	if o == nil || o.phaseToggles == nil {
		return PhaseToggleConfig{Phases: map[string]PhaseToggle{}}, nil
	}
	return o.phaseToggles.Load()
}

// SaveGlobalPhaseToggles persists the provided toggle configuration, preserving
// the original AddedAt timestamp when a phase remains disabled.
func (o *SuiteOrchestrator) SaveGlobalPhaseToggles(cfg PhaseToggleConfig) (PhaseToggleConfig, error) {
	if o == nil || o.phaseToggles == nil {
		return normalizePhaseToggleConfig(cfg, time.Now().UTC()), nil
	}
	current, err := o.phaseToggles.Load()
	if err != nil {
		return PhaseToggleConfig{}, err
	}

	now := time.Now().UTC()
	normalized := normalizePhaseToggleConfig(cfg, now)
	if normalized.Phases == nil {
		normalized.Phases = map[string]PhaseToggle{}
	}

	for name, existing := range current.Phases {
		updated, ok := normalized.Phases[name]
		if !ok {
			// Missing entries mean "enabled by default"—skip
			continue
		}
		if updated.Disabled && updated.AddedAt.IsZero() {
			if !existing.AddedAt.IsZero() {
				updated.AddedAt = existing.AddedAt
			} else {
				updated.AddedAt = now
			}
		}
		normalized.Phases[name] = updated
	}

	return o.phaseToggles.Save(normalized)
}

func (o *SuiteOrchestrator) applyTestingConfig(defs []phases.Definition, cfg *workspacepkg.Config) []phases.Definition {
	if cfg == nil || len(cfg.Phases) == 0 {
		return defs
	}
	var configured []phases.Definition
	for _, def := range defs {
		name := def.Name.Key()
		settings, ok := cfg.Phases[name]
		if ok {
			if settings.Enabled != nil && !*settings.Enabled {
				continue
			}
			if settings.Timeout > 0 {
				def.Timeout = settings.Timeout
			}
		}
		configured = append(configured, def)
	}
	return configured
}

func buildCommandHistory(req SuiteExecutionRequest, plan *phasePlan) []string {
	var history []string
	var descriptor []string
	if req.ScenarioName != "" {
		descriptor = append(descriptor, fmt.Sprintf("scenario=%s", req.ScenarioName))
	}
	if plan != nil && plan.PresetUsed != "" {
		descriptor = append(descriptor, fmt.Sprintf("preset=%s", plan.PresetUsed))
	}
	if len(req.Phases) > 0 {
		descriptor = append(descriptor, fmt.Sprintf("phases=%s", strings.Join(req.Phases, ",")))
	}
	if len(req.Skip) > 0 {
		descriptor = append(descriptor, fmt.Sprintf("skip=%s", strings.Join(req.Skip, ",")))
	}
	if req.FailFast {
		descriptor = append(descriptor, "failFast=true")
	}
	if len(descriptor) > 0 {
		history = append(history, strings.Join(descriptor, " "))
	}
	if plan != nil && len(plan.Selected) > 0 {
		var names []string
		for _, def := range plan.Selected {
			if def.Name.IsZero() {
				continue
			}
			names = append(names, def.Name.String())
		}
		if len(names) > 0 {
			history = append(history, fmt.Sprintf("phase-order:%s", strings.Join(names, ",")))
		}
	}
	return history
}

func normalizePhaseList(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	normalized := make([]string, 0, len(values))
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		name := phases.NormalizeKey(value)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		normalized = append(normalized, name)
	}
	if len(normalized) == 0 {
		return nil
	}
	return normalized
}

func phaseDefinitionNames(defs []phases.Definition) []string {
	if len(defs) == 0 {
		return nil
	}

	names := make([]string, 0, len(defs))
	for _, def := range defs {
		name := phases.NormalizeKey(def.Name.String())
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil
	}
	return names
}

func resolveDesiredPhaseList(req SuiteExecutionRequest, presets map[string][]string) ([]string, string, error) {
	if len(req.Phases) > 0 {
		return req.Phases, "", nil
	}
	if req.Preset == "" {
		return nil, "", nil
	}
	name := phases.NormalizeKey(req.Preset)
	if name == "" {
		return nil, "", shared.NewValidationError(fmt.Sprintf("preset '%s' is not defined", req.Preset))
	}
	phases, ok := presets[name]
	if !ok {
		return nil, "", shared.NewValidationError(fmt.Sprintf("preset '%s' is not defined", req.Preset))
	}
	return phases, name, nil
}

func applySkipFilters(selected []phases.Definition, skip []string) ([]phases.Definition, []phaseDisableNotice) {
	if len(selected) == 0 || len(skip) == 0 {
		return selected, nil
	}
	skipSet := make(map[string]struct{}, len(skip))
	for _, phase := range skip {
		if normalized := phases.NormalizeKey(phase); normalized != "" {
			skipSet[normalized] = struct{}{}
		}
	}
	var filtered []phases.Definition
	var notices []phaseDisableNotice
	for _, def := range selected {
		if _, skip := skipSet[def.Name.Key()]; skip {
			notices = append(notices, phaseDisableNotice{Name: def.Name.String(), Requested: true})
			continue
		}
		filtered = append(filtered, def)
	}
	return filtered, notices
}

func buildPlanWarnings(plan *phasePlan) []string {
	if plan == nil {
		return nil
	}
	var warnings []string
	for _, notice := range plan.DisabledByDefault {
		warnings = append(warnings, formatSkipWarning(notice))
	}
	for _, notice := range plan.ExplicitDisabled {
		warnings = append(warnings, formatExplicitWarning(notice))
	}
	return warnings
}

func buildPhaseWarningMap(plan *phasePlan) map[string][]phases.Observation {
	warnings := make(map[string][]phases.Observation)
	if plan == nil {
		return warnings
	}
	for _, notice := range plan.ExplicitDisabled {
		text := formatExplicitWarning(notice)
		warnings[phases.NormalizeKey(notice.Name)] = []phases.Observation{phases.NewWarningObservation(text)}
	}
	return warnings
}

func formatSkipWarning(notice phaseDisableNotice) string {
	if notice.EnvVar != "" {
		return fmt.Sprintf("Phase '%s' is disabled via %s=1 and was skipped by default.", notice.Name, notice.EnvVar)
	}
	if notice.Requested {
		return fmt.Sprintf("Phase '%s' was skipped by request.", notice.Name)
	}
	base := fmt.Sprintf("Phase '%s' is globally disabled and was skipped by default.", notice.Name)
	return base + formatToggleContext(notice.Toggle)
}

func formatExplicitWarning(notice phaseDisableNotice) string {
	base := fmt.Sprintf("Phase '%s' is globally disabled but was explicitly requested; results may be unstable.", notice.Name)
	return base + formatToggleContext(notice.Toggle)
}

func formatToggleContext(toggle PhaseToggle) string {
	var parts []string
	if toggle.Reason != "" {
		parts = append(parts, fmt.Sprintf("reason: %s", toggle.Reason))
	}
	if toggle.Owner != "" {
		parts = append(parts, fmt.Sprintf("owner: %s", toggle.Owner))
	}
	if !toggle.AddedAt.IsZero() {
		parts = append(parts, fmt.Sprintf("disabledAt: %s", toggle.AddedAt.UTC().Format("2006-01-02")))
	}
	if len(parts) == 0 {
		return ""
	}
	return " (" + strings.Join(parts, "; ") + ")"
}

func appendObservationsToLog(logPath string, projectRoot string, observations []phases.Observation) {
	if logPath == "" || len(observations) == 0 {
		return
	}
	target := logPath
	if !filepath.IsAbs(target) {
		target = filepath.Join(projectRoot, target)
	}
	f, err := os.OpenFile(target, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()

	fmt.Fprintln(f, "OBSERVATIONS:")
	for _, obs := range observations {
		if text := obs.String(); strings.TrimSpace(text) != "" {
			fmt.Fprintln(f, " -", text)
		}
	}
}
