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
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"test-genie/internal/captureprofile"
	"test-genie/internal/executionevidence"
	"test-genie/internal/orchestrator/phasecache"
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
	sharedcapacity "github.com/vrooli/vrooli/packages/capacity"

	workspacepkg "test-genie/internal/orchestrator/workspace"

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
	capacity      PhaseCapacityBroker
	costEstimator PhaseCostEstimator
}

// PhaseCapacityBroker is the narrow admission seam used by the scheduler.
// Implementations own the shared host ledger; the orchestrator only decides
// whether a measured phase may join a concurrent batch.
type PhaseCapacityBroker interface {
	Acquire(context.Context, string, int64, int64) (sharedcapacity.Lease, sharedcapacity.Verdict, error)
}

// PhaseCostEstimator supplies reliable historical sizing for admission.
type PhaseCostEstimator interface {
	PhaseCostEstimate(context.Context, string, string) (ramBytes, cpuMilli int64, reliable bool)
}

// PhaseDurationEstimator supplies measured wall-clock history for the deadline
// guard. It is optional: an estimator that does not implement it falls back to
// the planner's predicted durations, which are available on every run but are
// rounded and biased upward.
type PhaseDurationEstimator interface {
	PhaseDurationEstimate(context.Context, string, string) (p90Milliseconds int64, ok bool)
}

// PhaseCalibrationPlanner owns the age and descriptor-history policy for
// reliable samples. The orchestrator consumes the decision and records its
// reason; it does not infer freshness from elapsed test time.
type PhaseCalibrationPlanner interface {
	CalibrationDecision(context.Context, string, []string, string) (forceSerial bool, reason string)
}

// SetClaims wires the playbooks-claims service used by the playbooks phase
// to guard against concurrent runs against the same scenario.
func (o *SuiteOrchestrator) SetClaims(svc *playbooksclaims.Service) {
	if o == nil {
		return
	}
	o.claims = svc
}

// SetCapacityBroker wires shared host-capacity admission into the scheduler.
func (o *SuiteOrchestrator) SetCapacityBroker(broker PhaseCapacityBroker) {
	if o != nil {
		o.capacity = broker
	}
}

// SetPhaseCostEstimator wires persisted cost history into scheduler sizing.
func (o *SuiteOrchestrator) SetPhaseCostEstimator(estimator PhaseCostEstimator) {
	if o != nil {
		o.costEstimator = estimator
	}
}

// SuiteExecutionRequest configures a single test execution run.
type SuiteExecutionRequest struct {
	ScenarioName string `json:"scenarioName"`
	// Target accepts the generalized kind:id notation. ScenarioName remains
	// required by legacy callers and is the display alias for scenario targets.
	Target   string   `json:"target,omitempty"`
	Preset   string   `json:"preset,omitempty"`
	Phases   []string `json:"phases,omitempty"`
	Skip     []string `json:"skip,omitempty"`
	FailFast bool     `json:"failFast"`

	// RunID, when set, is the durable run identifier the run manager mints up
	// front so it can register and return the id synchronously (before the
	// suite actually starts executing). When empty, prepareExecution mints one.
	// Threading the id keeps a single run-id scheme and makes the start→finalize
	// index upsert idempotent under the pre-minted id.
	RunID string `json:"runId,omitempty"`

	// DiagnosticsPreset ("none"|"light"|"full"), when set, records the
	// requested diagnostic depth for provider-owned evidence capture.
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
	// Admission identity is resolved before a request can coalesce and checked
	// again immediately before execution persists run evidence.
	AdmissionTreeDigest          string `json:"admissionTreeDigest,omitempty"`
	AdmissionPhaseSetDigest      string `json:"admissionPhaseSetDigest,omitempty"`
	AdmissionDescriptorDigest    string `json:"admissionDescriptorDigest,omitempty"`
	AdmissionConfigurationDigest string `json:"admissionConfigurationDigest,omitempty"`
	// AdmissionQueued is set by the run manager when this request waited for a
	// global execution slot. A queued request may outlive the plan preview that
	// admitted it, so prepareExecution is allowed one explicit rebase against
	// the current source and descriptor snapshot. Immediate requests remain
	// fail-closed when their admission identity changes.
	AdmissionQueued bool `json:"admissionQueued,omitempty"`
	// AdmissionResources is populated from selected phase descriptors so the
	// durable run manager can serialize suites sharing a singleton service.
	AdmissionResources                  []string         `json:"admissionResources,omitempty"`
	RequireGateQuality                  bool             `json:"requireGateQuality,omitempty"`
	PredictedPhaseDurationsMilliseconds map[string]int64 `json:"predictedPhaseDurationsMilliseconds,omitempty"`
	// ResolvedPhases is the phase set the planner selected for this request —
	// notably an adaptive profile's budget-trimmed subset, which the executor
	// cannot re-derive because profile fitting happens in the plan service.
	//
	// It is deliberately separate from Phases. Phases means "the operator named
	// these", which is exact intent and carries no preset; ResolvedPhases means
	// "the planner resolved the operator's preset to these", which still has a
	// preset. Collapsing the two is what recorded preset_used=NULL on every
	// durable run and made each one ineligible for the baseline reuse it had
	// just earned.
	ResolvedPhases []string `json:"resolvedPhases,omitempty"`
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
	TargetKind   string    `json:"targetKind,omitempty"`
	TargetID     string    `json:"targetId,omitempty"`
	// Verdict is the tri-state outcome (PASS/PARTIAL/FAIL). Success is kept for
	// backward compatibility and is true for both PASS and PARTIAL (only FAIL is
	// a non-zero exit), so a self-test that skips an unrunnable phase is honestly
	// reported without failing CI.
	Verdict string `json:"verdict,omitempty"`
	// FailureReason records a suite-level error that occurred before a phase
	// could emit its own result, such as target-runtime startup failure. It is
	// durable terminal evidence, not a warning, and lets waiters distinguish an
	// empty zero-phase failure from a scenario validation result.
	FailureReason            string   `json:"failureReason,omitempty"`
	PresetUsed               string   `json:"preset,omitempty"`
	RequestedPreset          string   `json:"requestedPreset,omitempty"`
	RequestedPhases          []string `json:"requestedPhases,omitempty"`
	RequestedSkipPhases      []string `json:"requestedSkipPhases,omitempty"`
	PlannedPhases            []string `json:"plannedPhases,omitempty"`
	PhaseSetDigest           string   `json:"phaseSetDigest,omitempty"`
	DescriptorSnapshotDigest string   `json:"descriptorSnapshotDigest,omitempty"`
	ConfigurationFingerprint string   `json:"configurationFingerprint,omitempty"`
	// SourceFingerprint identifies the declared relevant source scope at run
	// admission. It is deliberately separate from Git dirtiness: ordinary
	// concurrent edits outside the scope cannot invalidate this evidence.
	SourceFingerprint string                      `json:"sourceFingerprint,omitempty"`
	SourceScope       string                      `json:"sourceScope,omitempty"`
	SourceStable      bool                        `json:"sourceStable"`
	GateQuality       bool                        `json:"gateQuality,omitempty"`
	FailFast          bool                        `json:"failFast"`
	Phases            []PhaseExecutionResult      `json:"phases"`
	PhaseSummary      PhaseSummary                `json:"phaseSummary"`
	PreparationStages []PreparationStage          `json:"preparationStages,omitempty"`
	ProviderReadiness []providerreadiness.Outcome `json:"providerReadiness,omitempty"`
	SchedulerDecision string                      `json:"schedulerDecision,omitempty"`
	// SchedulerAdmissionAttempts and SchedulerEstimatedAdmissions make the
	// scheduler's fallback usage durable. The latter is the revisit signal for
	// the fleet-wide reservation constant; without persistence, the scheduler
	// could silently drift back to guessed capacity forever.
	SchedulerAdmissionAttempts   int            `json:"schedulerAdmissionAttempts,omitempty"`
	SchedulerEstimatedAdmissions int            `json:"schedulerEstimatedAdmissions,omitempty"`
	Warnings                     []string       `json:"warnings,omitempty"`
	WarningSummary               WarningSummary `json:"warningSummary"`
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

// PreparationStage is a bounded timing span for work that happens before test
// phases begin. It is intentionally distinct from PhaseExecutionResult: stage
// timings explain orchestration cost but must never affect phase reliability or
// adaptive phase-selection estimates.
type PreparationStage struct {
	Name                 string `json:"name"`
	Parent               string `json:"parent,omitempty"`
	Subject              string `json:"subject,omitempty"`
	Status               string `json:"status,omitempty"`
	DurationMilliseconds int64  `json:"durationMilliseconds"`
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
	Status               string `json:"status,omitempty"`
	DurationSeconds      int    `json:"durationSeconds,omitempty"`
	DurationMilliseconds int64  `json:"durationMilliseconds,omitempty"`
	Error                string `json:"error,omitempty"`
	// PhasePresentation and FindingsSummary carry the provider-computed per-phase
	// standing (Phase Capability Contract) on phase_end events, so a follower
	// derives the standing from the same payload the terminal result carries. nil
	// for native phases / providers with no ladder.
	PhasePresentation *commonv1.PhasePresentation  `json:"-"`
	FindingsSummary   *runspb.PhaseFindingsSummary `json:"-"`
	Assessment        *commonv1.MaturityAssessment `json:"-"`

	// For observation events
	Message string `json:"message,omitempty"`

	// For complete events
	Result *SuiteExecutionResult `json:"result,omitempty"`
}

// ExecutionEventCallback is called for each event during streaming execution.
type ExecutionEventCallback func(event ExecutionEvent)

type preparedExecution struct {
	env               workspacepkg.Environment
	config            *workspacepkg.Config
	plan              *phasePlan
	request           SuiteExecutionRequest
	runID             string
	runLogDir         string
	result            *SuiteExecutionResult
	preparationStages []PreparationStage
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
	prepareStarted := time.Now()
	emitPreparationProgress(emit, "planning", "starting")
	prepared, err := o.prepareExecution(req)
	if err != nil {
		return nil, err
	}
	// A queued request may have been rebased against a newer plan while it
	// waited for a global slot. Carry that effective request through the rest
	// of execution so runtime setup, scheduling, and final evidence agree on
	// the same current contract.
	req = prepared.request
	prepared.result.PreparationStages = append(prepared.result.PreparationStages, PreparationStage{Name: "planning", Status: "completed", DurationMilliseconds: time.Since(prepareStarted).Milliseconds()})
	prepared.result.PreparationStages = append(prepared.result.PreparationStages, prepared.preparationStages...)
	emitPreparationProgress(emit, "planning", fmt.Sprintf("completed in %s", time.Since(prepareStarted).Round(time.Millisecond)))
	log.Printf("test-genie preflight planning for %s completed in %s", prepared.env.ScenarioName, time.Since(prepareStarted).Round(time.Millisecond))

	readinessStarted := time.Now()
	emitPreparationProgress(emit, "provider_readiness", "starting")
	readiness := o.checkProviderReadiness(ctx, prepared.env, prepared.plan.Selected, nil, emit)
	prepared.result.ProviderReadiness = readiness.Outcomes
	prepared.result.PreparationStages = append(prepared.result.PreparationStages, PreparationStage{Name: "provider_readiness", Status: "completed", DurationMilliseconds: time.Since(readinessStarted).Milliseconds()})
	prepared.result.PreparationStages = append(prepared.result.PreparationStages, readiness.Stages...)
	emitPreparationProgress(emit, "provider_readiness", fmt.Sprintf("completed in %s", time.Since(readinessStarted).Round(time.Millisecond)))
	log.Printf("test-genie preflight provider readiness for %s completed in %s", prepared.env.ScenarioName, time.Since(readinessStarted).Round(time.Millisecond))

	runtimeStarted := time.Now()
	emitPreparationProgress(emit, "target_runtime", "starting")
	env, runCtx, runtimeLease, runtimeManager, err := o.prepareTargetRuntime(ctx, prepared.env, readiness.Active, req, nil)
	if err != nil {
		return nil, err
	}
	prepared.result.PreparationStages = append(prepared.result.PreparationStages, PreparationStage{Name: "target_runtime", Status: "completed", DurationMilliseconds: time.Since(runtimeStarted).Milliseconds()})
	emitPreparationProgress(emit, "target_runtime", fmt.Sprintf("completed in %s", time.Since(runtimeStarted).Round(time.Millisecond)))
	log.Printf("test-genie preflight target runtime for %s completed in %s", prepared.env.ScenarioName, time.Since(runtimeStarted).Round(time.Millisecond))
	if planner, ok := o.costEstimator.(PhaseCalibrationPlanner); ok {
		forceSerial, reason := planner.CalibrationDecision(ctx, env.ScenarioName, phaseDefinitionNames(prepared.plan.Selected), prepared.result.DescriptorSnapshotDigest)
		if forceSerial && strings.TrimSpace(reason) != "" {
			// Assign to env, not to prepared.env. workspace.Environment is a
			// value type, so writing through the copy recorded the decision in
			// run evidence while the scheduler — which is handed env below —
			// never saw it. Every run reported a serial calibration it did not
			// perform.
			env.SchedulerDecision = strings.TrimSpace(reason)
			prepared.result.SchedulerDecision = env.SchedulerDecision
		}
	}
	prepared.env = env
	defer func() {
		if runtimeManager != nil {
			if cleanupErr := runtimeManager.Cleanup(context.Background(), runtimeLease, io.Discard); cleanupErr != nil {
				log.Printf("failed to clean up target runtime: %v", cleanupErr)
			}
		}
	}()

	phaseResults, anyFailure, loopMetrics := o.runSelectedPhasesWithRunID(
		ctx,
		env,
		runCtx,
		prepared.result.RunID,
		prepared.runLogDir,
		prepared.plan.Selected,
		readiness.Blocked,
		req.FailFast,
		emit,
		buildPhaseWarningMap(prepared.plan),
		req.PredictedPhaseDurationsMilliseconds,
	)
	// Keep phase execution wall time separate from provider readiness and other
	// preflight. The scheduler can control overlap only inside this interval;
	// folding a slow legacy readiness probe into its denominator makes a faster
	// scheduler look worse and hides the actual readiness tax from operators.
	prepared.result.PreparationStages = append(prepared.result.PreparationStages, PreparationStage{
		Name:                 "phase_execution",
		Status:               "completed",
		DurationMilliseconds: loopMetrics.ExecutionMilliseconds,
	})
	// Recorded as a stage so `runs cost` and any consumer summing stages sees
	// scheduling overhead beside phase cost instead of losing it.
	prepared.result.PreparationStages = append(prepared.result.PreparationStages, PreparationStage{
		Name:                 "phase_scheduling",
		Status:               "completed",
		Subject:              fmt.Sprintf("%d admission attempts; %d estimated admissions; %d batches; max batch %d", loopMetrics.AdmissionAttempts, loopMetrics.EstimatedAdmissions, loopMetrics.BatchCount, loopMetrics.MaxBatchSize),
		DurationMilliseconds: loopMetrics.SchedulingMilliseconds,
	})
	prepared.result.SchedulerAdmissionAttempts = loopMetrics.AdmissionAttempts
	prepared.result.SchedulerEstimatedAdmissions = loopMetrics.EstimatedAdmissions

	return o.finalizeExecution(ctx, req, prepared, phaseResults, anyFailure, emit), nil
}

// emitPreparationProgress exposes work that precedes the first test phase.
// The run manager keeps this stage in its live snapshot, so callers can
// distinguish legitimate preflight from a run with no recent progress.
func emitPreparationProgress(emit ExecutionEventCallback, stage, message string) {
	if emit == nil {
		return
	}
	emit(ExecutionEvent{
		Type:      EventProgress,
		Timestamp: time.Now(),
		Phase:     "preparing:" + stage,
		Message:   message,
	})
}

func (o *SuiteOrchestrator) prepareTargetRuntime(
	ctx context.Context,
	env workspacepkg.Environment,
	defs []phases.Definition,
	req SuiteExecutionRequest,
	logWriter io.Writer,
) (workspacepkg.Environment, runnability.RunContext, targetruntime.Lease, *targetruntime.Manager, error) {
	if strings.TrimSpace(env.TargetKind) != "" && strings.TrimSpace(env.TargetKind) != "scenario" {
		// Non-scenario targets are source trees, not deployable services. Their
		// phases can inspect and validate files, but Test Genie must never try to
		// start a scenario named after a package/resource/tool.
		rc := resolveRunContext(env, targetruntime.URLs{}, nil)
		env.TargetRuntime = targetruntime.NewNoOp(fmt.Sprintf("target kind %q is source-only; lifecycle operations are not applicable", env.TargetKind))
		return env, rc, targetruntime.Lease{}, nil, nil
	}
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

	// The run context is computed from the surfaces that ended up live. Provider
	// phases own their dependency/readiness checks and return a provider verdict;
	// Test Genie does not probe provider-specific scenarios on their behalf.
	rc := resolveRunContext(env, targetruntime.URLs{}, nil)
	return env, rc, lease, manager, nil
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

// resolveRunDiagnostics maps the per-run diagnostics preset to the generic
// serialized diagnostics shape. Providers decide which of these capabilities
// they can actually collect; Test Genie only records the requested contract.
func resolveRunDiagnostics(preset string) sharedruns.DiagnosticsConfig {
	switch strings.TrimSpace(strings.ToLower(preset)) {
	case "none":
		return sharedruns.DiagnosticsConfig{}
	case "full":
		return sharedruns.DiagnosticsConfig{
			Video: true, Console: true, Network: true, HAR: true, Trace: true, DOM: true,
		}
	case "light", "":
		return sharedruns.DiagnosticsConfig{Console: true}
	default:
		// Unknown presets fail soft to the cheapest useful evidence shape. The
		// provider remains the authority for accepted preset names.
		return sharedruns.DiagnosticsConfig{Console: true}
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
	if err := sharedartifacts.EnsureCoverageStructure(planCtx.env.ArtifactRoot); err != nil {
		return nil, err
	}

	runLogDir := sharedartifacts.RunLogsDir(planCtx.env.ArtifactRoot, runID)
	if err := os.MkdirAll(runLogDir, 0o755); err != nil {
		return nil, fmt.Errorf("failed to create run log directory: %w", err)
	}

	// Record the run as in-progress so it is enumerable mid-flight; finalize
	// updates it to its terminal status. Git context and the tree digest are
	// captured at run START (not finalize) so they identify the byte-state
	// the phases actually executed against; both are best-effort and never
	// block the run.
	gitContextStarted := time.Now()
	gitCtx := treedigest.CollectGitContext(planCtx.env.ScenarioDir)
	gitContextDuration := time.Since(gitContextStarted)
	log.Printf("test-genie preflight git context for %s completed in %s", scenario, gitContextDuration.Round(time.Millisecond))
	digestStarted := time.Now()
	digest, digestErr := treedigest.Compute(planCtx.env.ScenarioDir)
	digestDuration := time.Since(digestStarted)
	log.Printf("test-genie preflight source digest for %s completed in %s", scenario, digestDuration.Round(time.Millisecond))
	if digestErr != nil {
		log.Printf("tree digest unavailable for run %s: %v", runID, digestErr)
	}
	plannedPhases := phaseDefinitionNames(planCtx.plan.Selected)
	descriptorSnapshot, err := buildRunDescriptorSnapshot(planCtx.plan)
	if err != nil {
		return nil, fmt.Errorf("build run descriptor snapshot: %w", err)
	}
	phaseSetDigest := phases.PhaseSetDigest(plannedPhases)
	configurationDigest := ExecutionConfigurationFingerprint(req, descriptorSnapshot.Digest)
	planCtx.env.DescriptorSnapshotDigest = descriptorSnapshot.Digest
	planCtx.env.ExecutionConfigurationDigest = configurationDigest
	admissionMismatches := admissionIdentityMismatches(req, digest, phaseSetDigest, descriptorSnapshot.Digest, configurationDigest)
	admissionRebased := false
	if len(admissionMismatches) > 0 {
		if !req.AdmissionQueued {
			return nil, fmt.Errorf("admission identity changed (%s); retry to measure the current validation contract", strings.Join(admissionMismatches, ", "))
		}

		// A queued request was previewed before it acquired a slot. Rebuild the
		// adaptive selection from the current preset instead of executing the
		// old ResolvedPhases set. Explicit phase requests retain their exact
		// operator intent; only planner-resolved selections are cleared.
		if len(req.Phases) == 0 && len(req.ResolvedPhases) > 0 {
			req.ResolvedPhases = nil
			req.PredictedPhaseDurationsMilliseconds = nil
			planCtx, err = o.loadExecutionPlanContext(req)
			if err != nil {
				return nil, fmt.Errorf("rebase queued execution plan: %w", err)
			}
			scenario = planCtx.env.ScenarioName
			plannedPhases = phaseDefinitionNames(planCtx.plan.Selected)
			descriptorSnapshot, err = buildRunDescriptorSnapshot(planCtx.plan)
			if err != nil {
				return nil, fmt.Errorf("build rebased run descriptor snapshot: %w", err)
			}
			phaseSetDigest = phases.PhaseSetDigest(plannedPhases)
			configurationDigest = ExecutionConfigurationFingerprint(req, descriptorSnapshot.Digest)
			planCtx.env.DescriptorSnapshotDigest = descriptorSnapshot.Digest
			planCtx.env.ExecutionConfigurationDigest = configurationDigest
		}

		// The current source and plan were just measured together. Replace the
		// stale admission identity with that measurement and retain an explicit
		// warning in the durable result. A later source change is still captured
		// by finalizeExecution's source-stability check.
		req.AdmissionTreeDigest = digest
		req.AdmissionPhaseSetDigest = phaseSetDigest
		req.AdmissionDescriptorDigest = descriptorSnapshot.Digest
		req.AdmissionConfigurationDigest = configurationDigest
		admissionRebased = true
	}
	if req.RequireGateQuality && (digest == "" || digestErr != nil || gitCtx.Dirty || !isLinkedWorktree(planCtx.env.ScenarioDir)) {
		return nil, fmt.Errorf("gate-quality execution requires an isolated linked Git worktree with a clean, digest-stamped source tree")
	}
	if err := sharedruns.WriteDescriptorSnapshot(planCtx.env.ArtifactRoot, runID, descriptorSnapshot); err != nil {
		return nil, fmt.Errorf("persist run descriptor snapshot: %w", err)
	}
	if err := sharedruns.NewIndex(planCtx.env.ArtifactRoot).Append(sharedruns.RunRecord{
		RunID:                           runID,
		Scenario:                        scenario,
		TargetKind:                      planCtx.env.TargetKind,
		TargetID:                        planCtx.env.TargetID,
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
		PhaseSetDigest:                  phaseSetDigest,
		DescriptorSnapshotSchemaVersion: descriptorSnapshot.SchemaVersion,
		DescriptorSnapshotDigest:        descriptorSnapshot.Digest,
	}); err != nil {
		return nil, fmt.Errorf("record run %s in durable index: %w", runID, err)
	}

	warnings := buildPlanWarnings(planCtx.plan)
	if admissionRebased {
		warnings = append(warnings, "queued admission identity changed; execution was rebased onto the current validation contract")
	}

	return &preparedExecution{
		env:       planCtx.env,
		config:    planCtx.config,
		plan:      planCtx.plan,
		request:   req,
		runID:     runID,
		runLogDir: runLogDir,
		result: &SuiteExecutionResult{
			RunID:                    runID,
			ArtifactDir:              sharedartifacts.RunDir(planCtx.env.ArtifactRoot, runID),
			ScenarioName:             scenario,
			TargetKind:               planCtx.env.TargetKind,
			TargetID:                 planCtx.env.TargetID,
			StartedAt:                time.Now().UTC(),
			PresetUsed:               planCtx.plan.PresetUsed,
			RequestedPreset:          phases.NormalizeKey(req.Preset),
			RequestedPhases:          normalizePhaseList(req.Phases),
			RequestedSkipPhases:      normalizePhaseList(req.Skip),
			PlannedPhases:            plannedPhases,
			PhaseSetDigest:           phaseSetDigest,
			DescriptorSnapshotDigest: descriptorSnapshot.Digest,
			ConfigurationFingerprint: configurationDigest,
			SourceFingerprint:        digest,
			SourceScope:              planCtx.env.TargetKind + ":" + planCtx.env.TargetID,
			SourceStable:             true,
			GateQuality:              req.RequireGateQuality,
			FailFast:                 req.FailFast,
			Warnings:                 warnings,
		},
		preparationStages: []PreparationStage{
			{Name: "git_context", Parent: "planning", Status: "completed", DurationMilliseconds: gitContextDuration.Milliseconds()},
			{Name: "source_digest", Parent: "planning", Status: "completed", DurationMilliseconds: digestDuration.Milliseconds()},
		},
	}, nil
}

func admissionIdentityMismatches(req SuiteExecutionRequest, treeDigest, phaseSetDigest, descriptorDigest, configurationDigest string) []string {
	var mismatches []string
	if req.AdmissionTreeDigest != "" && req.AdmissionTreeDigest != treeDigest {
		mismatches = append(mismatches, "source")
	}
	if req.AdmissionPhaseSetDigest != "" && req.AdmissionPhaseSetDigest != phaseSetDigest {
		mismatches = append(mismatches, "phase-set")
	}
	if req.AdmissionDescriptorDigest != "" && req.AdmissionDescriptorDigest != descriptorDigest {
		mismatches = append(mismatches, "descriptor")
	}
	if req.AdmissionConfigurationDigest != "" && req.AdmissionConfigurationDigest != configurationDigest {
		mismatches = append(mismatches, "configuration")
	}
	return mismatches
}

// isLinkedWorktree requires a separate Git worktree rather than merely a
// clean main checkout. The .git entry at the repository root is a file for a
// linked worktree and a directory for the shared primary checkout.
func isLinkedWorktree(scenarioDir string) bool {
	command := exec.Command("git", "-C", scenarioDir, "rev-parse", "--show-toplevel")
	root, err := command.Output()
	if err != nil {
		return false
	}
	info, err := os.Stat(filepath.Join(strings.TrimSpace(string(root)), ".git"))
	return err == nil && !info.IsDir()
}

// ExecutionConfigurationFingerprint captures the execution knobs that can
// materially change runtime without folding in volatile URLs or timestamps.
// The descriptor digest separately covers providers, policies, and descriptor
// revisions. Capture depth is deliberately excluded: it controls optional
// evidence recorded by a run, not the validation contract being compared.
// Together with the selected phase-set digest these are the exact comparability
// key for full-run timing history.
func ExecutionConfigurationFingerprint(req SuiteExecutionRequest, descriptorDigest string) string {
	payload := strings.Join([]string{
		"v1",
		strings.TrimSpace(req.Preset),
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
	// A run may spend long enough executing that another agent changes relevant
	// inputs. That does not invalidate any stored baseline; it simply means this
	// particular current attempt must not enter the reusable-result cache.
	// Keep the terminal result for diagnosis and make the retry reason explicit.
	if after, err := treedigest.Compute(prepared.env.ScenarioDir); err != nil {
		result.SourceStable = false
		result.Warnings = append(result.Warnings, "could not recheck source fingerprint after execution: "+err.Error())
	} else if after != result.SourceFingerprint {
		result.SourceStable = false
		result.Warnings = append(result.Warnings, "relevant source inputs changed during this test attempt; rerun to measure the current inputs")
	}

	// Persist the combined findings artifact BEFORE the nudge so the nudge can
	// point at a file that already exists on disk (the on-ramp the assessment
	// found broken). The path is run-deterministic regardless of write order.
	if err := o.writeFindingsArtifact(
		prepared.env.ArtifactRoot,
		result.ScenarioName,
		prepared.runID,
		result.Verdict,
		result.CompletedAt,
		resultViews,
	); err != nil {
		log.Printf("failed to write findings artifact: %v", err)
	} else if err := writeEvidenceManifest(prepared.env.ArtifactRoot, prepared.runID, result.ScenarioName, result.Verdict, result.CompletedAt, resultViews); err != nil {
		log.Printf("failed to write evidence manifest: %v", err)
	}
	// Inventory the bytes already owned by this run before publishing terminal
	// state. The catalog is metadata only: no provider artifact is copied into a
	// second store, and descriptor declarations assign producer metadata without
	// a phase-name registry.
	if err := writeArtifactCatalog(prepared.env.ArtifactRoot, prepared.runID, result.CompletedAt); err != nil {
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
		prepared.env.ArtifactRoot,
		prepared.runID,
		result.StartedAt,
		result.CompletedAt,
		resultViews,
	); err != nil {
		log.Printf("failed to write latest manifest: %v", err)
	}

	result.Requirements = o.syncRequirementsIfNeeded(ctx, prepared.env, prepared.config, req, prepared.plan, phaseResults)
	o.finalizeRunRecord(prepared.env.ArtifactRoot, prepared.runID, result, resultViews)

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
			Name:             p.Name,
			Status:           p.Status,
			DurationSeconds:  p.DurationSeconds,
			Comparable:       true,
			CacheHit:         p.CacheHit,
			CacheSourceRunID: p.CacheSourceRunID,
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
			Name:                 phase.Name,
			Status:               phase.Status,
			DurationSeconds:      phase.DurationSeconds,
			Error:                phase.Error,
			Classification:       phase.Classification,
			ClassificationSource: phase.ClassificationSource,
			Remediation:          phase.Remediation,
			RunnabilityVerdict:   phase.RunnabilityVerdict,
			RunnabilityReason:    phase.RunnabilityReason,
			FindingSource:        phase.FindingSource,
			PhasePresentation:    phase.PhasePresentation,
			FindingsSummary:      phase.FindingsSummary,
			Assessment:           phase.Assessment,
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

// runSelectedPhasesWithEvents preserves the focused unit-test seam used by
// callers that do not have a durable run id yet.
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
	predicted ...map[string]int64,
) ([]PhaseExecutionResult, bool) {
	var predictions map[string]int64
	if len(predicted) > 0 {
		predictions = predicted[0]
	}
	results, anyFailure, _ := o.runSelectedPhasesWithRunID(ctx, env, runCtx, env.ScenarioName, runLogDir, defs, readiness, failFast, emit, warnings, predictions)
	return results, anyFailure
}

// phaseLoopMetrics attributes the wall-clock spent in the phase loop and the
// scheduler work inside it, separately from preflight.
//
// It exists because preflight and phase-loop cost were previously mixed in the
// run wall time while the cost surface only retained individual phase rows.
// A scheduler metric that cannot name its own interval is not auditable, so
// the execution and scheduling intervals are recorded as preparation stages.
type phaseLoopMetrics struct {
	// ExecutionMilliseconds is wall time spent in the phase loop, including
	// phase execution and its inter-batch waits, but excluding preflight.
	ExecutionMilliseconds int64
	// SchedulingMilliseconds is time spent choosing a batch and admitting it
	// against the capacity broker — work that belongs to no phase.
	SchedulingMilliseconds int64
	// AdmissionAttempts counts calls into batch admission. A failed batch
	// re-proposes its remaining phases on the next iteration, so this grows
	// faster than the phase count and is the signal that it is doing so.
	AdmissionAttempts int
	// EstimatedAdmissions counts phases admitted with the fleet-wide fallback
	// reservation. It is the revisit signal for the metrics adoption contract.
	EstimatedAdmissions int
	// BatchCount counts the batches actually executed after capacity admission
	// may have shortened the proposed batch.
	BatchCount int
	// MaxBatchSize is the largest admitted batch in this execution.
	MaxBatchSize int
}

func (o *SuiteOrchestrator) runSelectedPhasesWithRunID(
	ctx context.Context,
	env workspacepkg.Environment,
	runCtx runnability.RunContext,
	runID string,
	runLogDir string,
	defs []phases.Definition,
	readiness map[string]providerreadiness.Outcome,
	failFast bool,
	emit ExecutionEventCallback,
	warnings map[string][]phases.Observation,
	predicted map[string]int64,
) ([]PhaseExecutionResult, bool, phaseLoopMetrics) {
	var metrics phaseLoopMetrics
	if len(defs) == 0 {
		return nil, false, metrics
	}
	if emit != nil {
		underlyingEmit := emit
		var emitMu sync.Mutex
		emit = func(event ExecutionEvent) {
			emitMu.Lock()
			defer emitMu.Unlock()
			underlyingEmit(event)
		}
	}
	results := make([]PhaseExecutionResult, 0, len(defs))
	anyFailure := false
	total := len(defs)
	forceSerial := phaseSchedulerForcedSerial()
	// Fail-fast cannot honor its contract if phases are admitted concurrently:
	// a later phase could start before an earlier failure is observable. Keep
	// fail-fast runs serial so the first failure is a hard scheduling boundary.
	if failFast {
		forceSerial = true
	}
	if strings.TrimSpace(env.SchedulerDecision) != "" {
		forceSerial = true
	}
	policy := o.phaseBatchPolicy(ctx, env.ScenarioName, forceSerial, predicted)
	executionStarted := time.Now()
	for start := 0; start < len(defs); {
		// A reusable verdict is not phase work and must not consume a host
		// reservation or an admission attempt. Besides saving the broker round
		// trip, this keeps cache-heavy runs from reporting scheduler overhead as
		// execution pressure. Audited hits intentionally stay on the normal path
		// so the provider still revalidates the cached verdict at its sample rate.
		if _, blocked := readiness[defs[start].Name.Key()]; !blocked {
			verdict := resolvePhaseVerdict(defs[start], runCtx)
			if !verdict.IsSkip() {
				if cached, audit, found, _ := o.loadCachedPhaseResult(env, runID, runLogDir, defs[start], readiness); found && !audit {
					cached.PredictedDurationMilliseconds = predicted[strings.ToLower(strings.TrimSpace(defs[start].Name.String()))]
					if emit != nil {
						emit(ExecutionEvent{Type: EventPhaseStart, Timestamp: time.Now(), Phase: defs[start].Name.String(), PhaseIndex: start + 1, PhaseTotal: total})
					}
					results = append(results, cached)
					if emit != nil {
						emit(ExecutionEvent{Type: EventPhaseEnd, Timestamp: time.Now(), Phase: cached.Name, Status: cached.Status, DurationSeconds: cached.DurationSeconds, DurationMilliseconds: cached.DurationMilliseconds, PhasePresentation: cached.PhasePresentation, FindingsSummary: cached.FindingsSummary, Assessment: cached.Assessment})
					}
					start++
					continue
				}
			}
		}
		// Everything from here to the goroutine launch is scheduling, not
		// phase work. It is timed because it used to be invisible.
		admissionStarted := time.Now()
		end := nextPhaseBatch(defs, start, policy)
		// A batcher policy may defer a phase by moving it to the pending tail.
		// Even when that phase is the last remaining member, the scheduler must
		// consume one item; otherwise a malformed/changed predicate can return
		// the same index forever and strand the run. The singleton fallback is
		// safe because admission and provider concurrency still guard it.
		if end <= start {
			end = start + 1
		}
		batch := defs[start:end]
		metrics.AdmissionAttempts++
		leases, admissionReason, admitted, estimated := o.admitPhaseBatch(ctx, env.ScenarioName, runID, batch)
		metrics.EstimatedAdmissions += estimated
		if admitted < len(batch) {
			// The host granted a shorter prefix than the batcher proposed. The
			// phases past it are re-proposed on the next iteration rather than
			// dropped, so a full host slows the run instead of changing what it
			// validates.
			end = start + admitted
			batch = defs[start:end]
		}
		metrics.BatchCount++
		if len(batch) > metrics.MaxBatchSize {
			metrics.MaxBatchSize = len(batch)
		}
		metrics.SchedulingMilliseconds += time.Since(admissionStarted).Milliseconds()
		for offset, phase := range batch {
			if emit != nil {
				emit(ExecutionEvent{Type: EventPhaseStart, Timestamp: time.Now(), Phase: phase.Name.String(), PhaseIndex: start + offset + 1, PhaseTotal: total})
			}
		}

		type phaseOutcome struct {
			index  int
			result PhaseExecutionResult
		}
		outcomes := make(chan phaseOutcome, len(batch))
		for offset, phase := range batch {
			go func(offset int, phase phases.Definition) {
				var result PhaseExecutionResult
				phaseWarnings := append([]phases.Observation(nil), warnings[phase.Name.Key()]...)
				if env.SchedulerDecision != "" {
					phaseWarnings = append(phaseWarnings, phases.NewWarningObservation("scheduler serial calibration: "+env.SchedulerDecision))
				}
				if admissionReason != "" {
					phaseWarnings = append(phaseWarnings, phases.NewWarningObservation("scheduler serial fallback: "+admissionReason))
				}
				if outcome, blocked := readiness[phase.Name.Key()]; blocked {
					result = o.newProviderReadinessPhaseResult(phase, runLogDir, outcome)
				} else if verdict := resolvePhaseVerdict(phase, runCtx); verdict.IsSkip() {
					result = o.newSkippedPhaseResult(phase, runLogDir, verdict)
				} else {
					if cached, audit, ok, cachedDuration := o.loadCachedPhaseResult(env, runID, runLogDir, phase, readiness); ok && !audit {
						result = cached
					} else {
						result = o.runPhaseWithEvents(ctx, env, runLogDir, phase, emit, mergeRunnabilityObservations(verdict, phaseWarnings))
						annotatePhaseRunnability(&result, verdict)
						cacheResult := true
						if cached.Status == phaseStatusPassed && audit {
							result.CacheAudit = true
							identity, identityOK := o.phaseCacheIdentity(env, phase, readiness)
							if identityOK {
								key := phasecache.Key(identity)
								store := phasecache.New(env.ArtifactRoot)
								if !phasecache.Equivalent(cached, result) {
									result.CacheAuditMismatch = true
									appendPhaseCacheFinding(&result, env.ScenarioName, phase.Name.Key(), "test_genie.phase_cache_audit_mismatch", "cached phase result differed from the freshly executed capability result; the cache entry was demoted")
									_ = store.Demote(key, "audit result differed from cached passed result")
									cacheResult = false
								} else {
									result.Observations = append(result.Observations, phases.NewObservation("phase cache audit matched cached passed result"))
									if cachedDuration > 0 && result.DurationMilliseconds >= cachedDuration*9/10 {
										result.CacheNoSaving = true
										appendPhaseCacheFinding(&result, env.ScenarioName, phase.Name.Key(), "test_genie.phase_cache_audit_no_saving", "cache audit found no measured saving; the provider may be filtering its response instead of skipping the work")
										cacheResult = false
									}
								}
							}
						}
						if cacheResult {
							o.saveCachedPhaseResult(env, runID, phase, readiness, result)
						}
					}
				}
				result.PredictedDurationMilliseconds = predicted[strings.ToLower(strings.TrimSpace(phase.Name.String()))]
				if result.PredictedDurationMilliseconds < 0 {
					result.PredictedDurationMilliseconds = 0
				}
				if offset < len(leases) && leases[offset] != nil {
					if err := leases[offset].Release(context.Background()); err != nil {
						log.Printf("release phase capacity claim for %s: %v", phase.Name, err)
					}
				}
				outcomes <- phaseOutcome{index: offset, result: result}
			}(offset, phase)
		}
		batchResults := make([]PhaseExecutionResult, len(batch))
		for range batch {
			outcome := <-outcomes
			batchResults[outcome.index] = outcome.result
		}
		close(outcomes)
		for offset, phaseResult := range batchResults {
			phase := batch[offset]
			if emit != nil {
				emit(ExecutionEvent{Type: EventPhaseEnd, Timestamp: time.Now(), Phase: phase.Name.String(), Status: phaseResult.Status, DurationSeconds: phaseResult.DurationSeconds, DurationMilliseconds: phaseResult.DurationMilliseconds, Error: phaseResult.Error, PhasePresentation: phaseResult.PhasePresentation, FindingsSummary: phaseResult.FindingsSummary, Assessment: phaseResult.Assessment})
			}
			if phaseResult.Status == phaseStatusFailed || phaseResult.Status == phaseStatusProviderUnavailable {
				anyFailure = true
			}
			results = append(results, phaseResult)
		}
		if failFast && anyPhaseFailure(batchResults) {
			break
		}
		start = end
	}
	metrics.ExecutionMilliseconds = time.Since(executionStarted).Milliseconds()
	return results, anyFailure, metrics
}

func appendPhaseCacheFinding(result *PhaseExecutionResult, scenario, phase, code, message string) {
	if result == nil {
		return
	}
	capability := strings.TrimSpace(phase)
	if result.PhasePresentation != nil && strings.TrimSpace(result.PhasePresentation.GetFocusCapabilityId()) != "" {
		capability = strings.TrimSpace(result.PhasePresentation.GetFocusCapabilityId())
	}
	message = fmt.Sprintf("phase %s capability %s: %s", phase, capability, message)
	finding := &architecturev1.ArchitectureFinding{
		Scenario:     scenario,
		Source:       architecturev1.FindingSource_FINDING_SOURCE_MEASURES,
		Code:         code,
		Severity:     architecturev1.FindingSeverity_FINDING_SEVERITY_WARNING,
		Locations:    []string{"phase:" + phase, "capability:" + capability},
		Message:      message,
		Suggestion:   "Review the provider determinism declaration and its cache-audit behavior before relying on phase reuse.",
		FindingClass: architecturev1.FindingClass_FINDING_CLASS_HEURISTIC,
	}
	findingid.Stamp(finding)
	result.Findings = append(result.Findings, finding)
	if result.FindingsSummary == nil {
		result.FindingsSummary = &runspb.PhaseFindingsSummary{}
	}
	result.FindingsSummary.Warnings++
	result.FindingsSummary.Total++
	result.Observations = append(result.Observations, phases.NewWarningObservation(message))
}

func (o *SuiteOrchestrator) phaseCacheIdentity(env workspacepkg.Environment, phase phases.Definition, readiness map[string]providerreadiness.Outcome) (phasecache.Identity, bool) {
	if strings.ToLower(strings.TrimSpace(phase.Determinism.Default)) != "file-determined" || len(phase.Determinism.Inputs) == 0 {
		return phasecache.Identity{}, false
	}
	digest, err := phasecache.ScopedDigest(env.ScenarioDir, phase.Determinism.Inputs)
	if err != nil {
		return phasecache.Identity{}, false
	}
	providerIdentity := "native:" + phase.Name.Key()
	if phase.ProviderScenario != "" {
		outcome := readiness[phase.Name.Key()]
		provider := strings.TrimSpace(outcome.ProviderScenario)
		if provider == "" {
			provider = strings.TrimSpace(phase.ProviderScenario)
		}
		if provider == "" {
			return phasecache.Identity{}, false
		}
		providerIdentity = strings.Join([]string{
			provider,
			strings.TrimSpace(outcome.SpecVersion),
			strings.TrimSpace(outcome.BuildRevision),
			strings.TrimSpace(outcome.FreshnessDigest),
		}, "|")
		// A provider with no readiness policy has no live build identity to
		// report. The descriptor snapshot is still part of the cache key, so
		// bind this safe file-determined fallback to the provider contract rather
		// than silently treating an empty readiness outcome as an identity.
		if strings.Trim(providerIdentity, "|") == provider {
			providerIdentity = "descriptor-contract:" + env.DescriptorSnapshotDigest + "|" + provider
		}
	}
	if env.DescriptorSnapshotDigest == "" || env.ExecutionConfigurationDigest == "" {
		return phasecache.Identity{}, false
	}
	return phasecache.Identity{
		ScopedInputDigest:      digest,
		ProviderBuildIdentity:  providerIdentity,
		DescriptorSnapshotHash: env.DescriptorSnapshotDigest,
		ExecutionConfiguration: env.ExecutionConfigurationDigest,
	}, true
}

func (o *SuiteOrchestrator) loadCachedPhaseResult(env workspacepkg.Environment, runID, runLogDir string, phase phases.Definition, readiness map[string]providerreadiness.Outcome) (PhaseExecutionResult, bool, bool, int64) {
	identity, ok := o.phaseCacheIdentity(env, phase, readiness)
	if !ok {
		return PhaseExecutionResult{}, false, false, 0
	}
	key := phasecache.Key(identity)
	store := phasecache.New(env.ArtifactRoot)
	entry, found, err := store.Load(key)
	if err != nil || !found {
		return PhaseExecutionResult{}, false, false, 0
	}
	result := entry.Phase
	cachedDuration := result.DurationMilliseconds
	result.Name = phase.Name.String()
	result.DurationMilliseconds = 0
	result.DurationSeconds = 0
	result.CacheHit = true
	result.CacheSourceRunID = entry.RunID
	result.LogPath = phaseLogPath(runLogDir, phase.Name)
	if err := os.WriteFile(result.LogPath, []byte(fmt.Sprintf("[INFO] cache hit: reused passed result from run %s\n", entry.RunID)), 0o644); err != nil {
		return PhaseExecutionResult{}, false, false, 0
	}
	if o.projectRoot != "" {
		if rel, err := filepath.Rel(o.projectRoot, result.LogPath); err == nil {
			result.LogPath = rel
		}
	}
	return result, store.ShouldAudit(key, runID), true, cachedDuration
}

func (o *SuiteOrchestrator) saveCachedPhaseResult(env workspacepkg.Environment, runID string, phase phases.Definition, readiness map[string]providerreadiness.Outcome, result PhaseExecutionResult) {
	identity, ok := o.phaseCacheIdentity(env, phase, readiness)
	if !ok || result.Status != phaseStatusPassed {
		return
	}
	// Recompute immediately after the phase. A source edit during execution is
	// never allowed to publish a result under the digest from before the edit.
	after, err := phasecache.ScopedDigest(env.ScenarioDir, phase.Determinism.Inputs)
	if err != nil || after != identity.ScopedInputDigest {
		return
	}
	_ = phasecache.New(env.ArtifactRoot).Save(phasecache.Key(identity), runID, result)
}

// phaseBatchPolicy resolves the batching predicates once per run.
//
// The duration predicate queries durable history, and the batcher consults it
// repeatedly as it re-proposes the tail of the phase list, so the results are
// memoized per phase. Without that, admission cost grows with the square of the
// phase count — the shape that put 401 s of unattributed scheduling into a run
// whose phases totalled 72.5 s on 2026-08-08.
func (o *SuiteOrchestrator) phaseBatchPolicy(ctx context.Context, scenario string, forceSerial bool, predicted map[string]int64) phaseBatchPolicy {
	policy := phaseBatchPolicy{forceSerial: forceSerial}
	if forceSerial || o.capacity == nil || o.costEstimator == nil {
		// Nothing is batchable, so the predicates are never consulted and the
		// run walks the phase list one at a time.
		return policy
	}
	policy.admissionEnabled = true
	var measured func(phases.Definition) (int64, bool)
	if estimator, ok := o.costEstimator.(PhaseDurationEstimator); ok {
		type durationSample struct {
			ms int64
			ok bool
		}
		durationCache := make(map[string]durationSample, len(predicted))
		measured = func(def phases.Definition) (int64, bool) {
			key := def.Name.Key()
			if cached, hit := durationCache[key]; hit {
				return cached.ms, cached.ok
			}
			ms, found := estimator.PhaseDurationEstimate(ctx, scenario, key)
			durationCache[key] = durationSample{ms: ms, ok: found}
			return ms, found
		}
	}
	policy.timeoutRisk = func(def phases.Definition) bool {
		return phaseTimeoutRisk(def, predicted, measured)
	}
	return policy
}

// admitPhaseBatch acquires capacity for the longest prefix of batch the host
// grants, and returns that prefix length along with the leases backing it.
//
// It admits a prefix rather than all-or-nothing because the previous behaviour
// let one phase veto every phase beside it: a broker denial on the last member
// released the grants already held for the others and dropped the whole batch
// to serial, and the caller then re-proposed the remainder, denied again, and
// walked the run one phase at a time. Sizing is no longer a failure mode here
// at all — missing estimates use the named fallback reservation — so the only
// reason a prefix is short now is that the host is genuinely full, which is a
// reason to run what fits rather than to run nothing.
//
// The returned length is always at least 1, so the caller always makes
// progress. A prefix of exactly 1 carries the broker's reason so the run
// records why it serialized.
func (o *SuiteOrchestrator) admitPhaseBatch(ctx context.Context, scenario, runID string, batch []phases.Definition) ([]sharedcapacity.Lease, string, int, int) {
	if len(batch) <= 1 || o.capacity == nil || o.costEstimator == nil {
		return nil, "", len(batch), 0
	}
	leases := make([]sharedcapacity.Lease, 0, len(batch))
	estimated := 0
	release := func() {
		for _, acquired := range leases {
			if acquired != nil {
				_ = acquired.Release(context.Background())
			}
		}
	}
	for _, phase := range batch {
		ramBytes, cpuMilli, reliable := o.costEstimator.PhaseCostEstimate(ctx, scenario, phase.Name.Key())
		var (
			lease   sharedcapacity.Lease
			verdict sharedcapacity.Verdict
			err     error
			reason  string
		)
		if !reliable || ramBytes <= 0 || cpuMilli <= 0 {
			ramBytes = defaultPhaseReservationRAMBytes
			cpuMilli = defaultPhaseReservationCPUMilli
			estimated++
		}
		ownerID := fmt.Sprintf("test-genie:%s:%s", strings.TrimSpace(runID), phase.Name.Key())
		lease, verdict, err = o.capacity.Acquire(ctx, ownerID, ramBytes, cpuMilli)
		switch {
		case err != nil:
			reason = err.Error()
		case verdict.Kind != "grant" && verdict.Kind != "degrade":
			reason = verdict.Reason
			if reason == "" {
				reason = verdict.Kind
			}
		}
		if reason != "" {
			if len(leases) >= 2 {
				// A partial batch still beats a serial walk. The phases past
				// the stopping point are re-proposed on the next iteration.
				return leases, reason, len(leases), estimated
			}
			release()
			return nil, reason, 1, 0
		}
		leases = append(leases, lease)
	}
	return leases, "", len(leases), estimated
}

func phaseSchedulerForcedSerial() bool {
	if strings.EqualFold(strings.TrimSpace(os.Getenv("TEST_GENIE_PHASE_SCHEDULER_ENABLED")), "false") || strings.TrimSpace(os.Getenv("TEST_GENIE_PHASE_SCHEDULER_ENABLED")) == "0" {
		return true
	}
	value := strings.ToLower(strings.TrimSpace(os.Getenv("TEST_GENIE_FORCE_SERIAL")))
	return value == "1" || value == "true" || value == "yes" || value == "on"
}

// phaseBatchPolicy carries the per-run predicates the batcher consults. They
// are injected rather than looked up inline so the batcher stays a pure
// function of the phase list, and so a run resolves each phase's duration
// history once instead of once per batch proposal.
type phaseBatchPolicy struct {
	forceSerial bool
	// timeoutRisk reports whether the phase runs close enough to its own
	// deadline that contention could turn a pass into a timeout.
	timeoutRisk func(phases.Definition) bool
	// admissionEnabled distinguishes an unavailable broker from a phase whose
	// individual measurement is missing. The latter uses a conservative
	// reservation and remains eligible for a batch.
	admissionEnabled bool
}

func (p phaseBatchPolicy) canBatch(def phases.Definition) bool {
	if !p.admissionEnabled {
		return false
	}
	return p.timeoutRisk == nil || !p.timeoutRisk(def)
}

// nextPhaseBatch returns the exclusive end of the next contiguous batch.
//
// It excludes two kinds of phase from a batch. A phase whose provider
// declares `exclusive` never shares a batch. A phase whose duration sits close
// to its own timeout is kept alone, because a phase that already consumes most
// of its budget has no contention headroom and concurrency could turn a passing
// run into a timeout with no source change. A phase with no resource estimate is
// not excluded: admitPhaseBatch uses the documented fallback reservation.
func nextPhaseBatch(defs []phases.Definition, start int, policy phaseBatchPolicy) int {
	if start >= len(defs) {
		return start
	}
	if policy.forceSerial || phaseConcurrencyMode(defs[start]) == "exclusive" {
		return start + 1
	}
	// Collect deferred phases in one pass. Repeatedly moving a deferred phase
	// and retrying the same index can cycle forever when all remaining phases
	// are non-batchable. Reorder once after the scan so the scheduler always
	// consumes work.
	deferred := make([]phases.Definition, 0)
	providers := map[string]struct{}{}
	batch := make([]phases.Definition, 0, len(defs)-start)
	boundary := len(defs)
	for end := start; end < len(defs); {
		def := defs[end]
		mode := phaseConcurrencyMode(def)
		if mode == "exclusive" {
			boundary = end
			break
		}
		if !policy.canBatch(def) {
			deferred = append(deferred, def)
			end++
			continue
		}
		if mode == "provider-serial" {
			provider := strings.TrimSpace(def.ProviderScenario)
			if provider == "" {
				provider = def.Name.Key()
			}
			if _, exists := providers[provider]; exists {
				boundary = end
				break
			}
			providers[provider] = struct{}{}
		}
		batch = append(batch, def)
		end++
	}
	if len(deferred) > 0 {
		reordered := append(append([]phases.Definition(nil), batch...), deferred...)
		copy(defs[start:boundary], reordered)
		if len(batch) == 0 {
			return start + 1
		}
		return start + len(batch)
	}
	if boundary < len(defs) {
		return boundary
	}
	return start + len(batch)
}

// contentionAllowance is how much slower a phase is assumed to run when it
// shares the host with its batch. A phase is kept out of a batch when its
// measured duration times this allowance would reach its timeout.
//
// The original guard used 2x because it compared an upward-biased planner
// prediction against half the timeout. The scheduler now uses observed p90
// wall-clock, and the post-change full-suite evidence showed that 2x still
// serialized a roughly 100 s security phase against its 180 s timeout despite
// the capacity broker granting the batch. 1.5x preserves a meaningful timeout
// margin while allowing measured p90 phases with genuine headroom to overlap.
// Revisit if a fresh 200-run window records timeout escapes under contention or
// remains below the 2.5x parallelism target.
const contentionAllowance = 1.5

// These reservations approximate the fleet p90 of observed phase resource
// claims. They keep an unmeasured phase eligible for a safe batch without
// pretending its cost is known. Revisit when durable fallback admissions
// exceed 10% for two consecutive weeks or the host capacity profile changes.
const (
	defaultPhaseReservationRAMBytes int64 = 512 * 1024 * 1024
	defaultPhaseReservationCPUMilli int64 = 500
)

// phaseTimeoutRisk reports whether concurrency could push the phase past its
// own deadline. It prefers measured history and falls back to the planner's
// prediction when a phase has none, so a phase nobody has run yet is still
// guarded — conservatively, since the fallback input is the biased one.
func phaseTimeoutRisk(def phases.Definition, predicted map[string]int64, measured func(phases.Definition) (int64, bool)) bool {
	timeout := def.Timeout
	if timeout <= 0 {
		timeout = phases.DefaultTimeout
	}
	if measured != nil {
		if observed, ok := measured(def); ok && observed > 0 {
			return float64(observed)*contentionAllowance >= float64(timeout.Milliseconds())
		}
	}
	if len(predicted) == 0 {
		return false
	}
	prediction := predicted[strings.ToLower(strings.TrimSpace(def.Name.String()))]
	if prediction <= 0 {
		return false
	}
	return float64(prediction)*contentionAllowance >= float64(timeout.Milliseconds())
}

func phaseConcurrencyMode(def phases.Definition) string {
	mode := strings.ToLower(strings.TrimSpace(def.Concurrency.Mode))
	if mode == "parallel-safe" || mode == "provider-serial" {
		return mode
	}
	return "exclusive"
}

func anyPhaseFailure(results []PhaseExecutionResult) bool {
	for _, result := range results {
		if result.Status == phaseStatusFailed || result.Status == phaseStatusProviderUnavailable {
			return true
		}
	}
	return false
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
	durationMs := time.Since(run.start).Milliseconds()
	if durationMs < 0 {
		durationMs = 0
	}
	duration := int((durationMs + 999) / 1000)

	status := phaseStatusPassed
	errMsg := ""
	classification := report.FailureClassification
	classificationSource := ""
	if classification != "" && classification != phases.FailureClassSystem {
		classificationSource = phases.ClassificationSourceProvider
	}
	remediation := report.Remediation

	if runErr != nil {
		status = phaseStatusFailed
		errMsg = runErr.Error()
		if errors.Is(runErr, context.DeadlineExceeded) || errors.Is(run.phaseCtx.Err(), context.DeadlineExceeded) {
			errMsg = fmt.Sprintf("phase timed out after %s", run.timeout)
			classification = phases.FailureClassTimeout
			classificationSource = phases.ClassificationSourceHarness
			if remediation == "" {
				remediation = "Increase the timeout or break the phase into smaller steps."
			}
		}
		if classification == "" {
			classification = phases.FailureClassSystem
			classificationSource = phases.ClassificationSourceHarness
		}
		if classificationSource == "" {
			classificationSource = phases.ClassificationSourceHarness
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
		Name:                 run.definition.Name.String(),
		Status:               status,
		DurationSeconds:      duration,
		DurationMilliseconds: durationMs,
		LogPath:              displayLogPath,
		Error:                errMsg,
		Classification:       classification,
		ClassificationSource: classificationSource,
		Remediation:          remediation,
		Observations:         report.Observations,
		Findings:             report.Findings,
		Assessment:           report.Assessment,
		Metrics:              report.Metrics,
		PhasePresentation:    report.PhasePresentation,
		FindingsSummary:      report.FindingsSummary,
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
		Name:                 name.String(),
		Status:               "failed",
		DurationSeconds:      0,
		DurationMilliseconds: 0,
		LogPath:              phaseLogPath(runLogDir, name),
		Error:                fmt.Sprintf("failed to create log file: %v", err),
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
	Assessment    *commonv1.MaturityAssessment          `json:"assessment,omitempty"`
	// PhasePresentation + FindingsSummary carry the per-phase standing (Phase
	// Capability Contract) so `test-genie runs findings <run-id>` renders the same
	// standing on demand. Additive and omitempty — architecture-cartographer's
	// --from-audit ingest reads only phases[].findings, so this does not affect it.
	PhasePresentation *commonv1.PhasePresentation  `json:"phasePresentation,omitempty"`
	FindingsSummary   *runspb.PhaseFindingsSummary `json:"findingsSummary,omitempty"`
	CacheHit          bool                         `json:"cacheHit,omitempty"`
	CacheSourceRunID  string                       `json:"cacheSourceRunId,omitempty"`
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
			Assessment:        res.Assessment,
			PhasePresentation: res.PhasePresentation,
			FindingsSummary:   res.FindingsSummary,
			CacheHit:          res.CacheHit,
			CacheSourceRunID:  res.CacheSourceRunID,
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
			CacheHit:          result.CacheHit,
			CacheSourceRunID:  result.CacheSourceRunID,
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
	Name                 string
	Status               string
	DurationSeconds      int
	DurationMilliseconds int64
	LogPath              string
	LogAbs               string
	Observations         []phases.Observation
	FindingSource        string
	Findings             []*architecturev1.ArchitectureFinding
	Assessment           *commonv1.MaturityAssessment
	PhasePresentation    *commonv1.PhasePresentation
	FindingsSummary      *runspb.PhaseFindingsSummary
	Metrics              *commonv1.ExecutionMetrics
	CacheHit             bool
	CacheSourceRunID     string
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
			Name:                 result.Name,
			Status:               result.Status,
			DurationSeconds:      result.DurationSeconds,
			DurationMilliseconds: result.DurationMilliseconds,
			LogPath:              result.LogPath,
			LogAbs:               phaseLogPath(runLogDir, name),
			Observations:         result.Observations,
			FindingSource:        result.FindingSource,
			Findings:             findings,
			Assessment:           result.Assessment,
			PhasePresentation:    result.PhasePresentation,
			FindingsSummary:      result.FindingsSummary,
			Metrics:              result.Metrics,
			CacheHit:             result.CacheHit,
			CacheSourceRunID:     result.CacheSourceRunID,
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
		Concurrency:           spec.Concurrency,
		Determinism:           spec.Determinism,
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
	// A planner-resolved set narrows selection without being user intent, so it
	// keeps the preset name. This is the adaptive-profile path: `quick` and
	// `smoke` are budget-fitted in the plan service, and the executor cannot
	// re-derive that trim on its own.
	presetName := phases.NormalizeKey(req.Preset)
	if presetName == "" {
		// An unspecified preset selects every applicable phase — see the
		// len(desired)==0 branch in selectPhases — which is precisely what the
		// comprehensive preset names. Recording "" for it was not a smaller
		// claim, it was a wrong one: Git Control Tower's FindReusableRun keys
		// reuse on Preset=="comprehensive", so a run that had done the full
		// comprehensive work advertised itself as ineligible and forced a
		// second, more expensive baseline run.
		presetName = phases.PresetComprehensive.String()
	}
	if len(req.ResolvedPhases) > 0 {
		return req.ResolvedPhases, presetName, nil
	}
	if req.Preset == "" {
		// nil desired is returned deliberately. Naming the preset must not
		// change which phases run, only what the run says it did.
		return nil, presetName, nil
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
