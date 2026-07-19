package execution

import (
	"context"
	"errors"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/backlogstatus"
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/planclient"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/runtimepaths"
	"swarm-manager/internal/transitions"
)

var (
	errNotFound   = apierr.ErrNotFound
	errAtCapacity = apierr.ErrAtCapacity
)

// Backlog status values referenced by execution when writing backlog items.
// Aliased from the shared backlogstatus package (which has no dependencies
// and is imported by both backlog and execution to break the cycle). Using
// named locals rather than qualifying every call site keeps the hot paths
// readable.
const (
	backlogStatusQueued        = backlogstatus.Queued
	backlogStatusInReview      = backlogstatus.InReview
	backlogStatusReviewPending = backlogstatus.ReviewPending
	backlogStatusFailed        = backlogstatus.Failed
	backlogStatusBacklog       = backlogstatus.Backlog
	backlogStatusResearching   = backlogstatus.Researching
	backlogStatusReady         = backlogstatus.Ready
)

// DOC: docs/concepts/ARCHITECTURE.md#key-flows
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/TEMPORAL-FLOWS.md

// defaultPolicyProvider returns hardcoded defaults when no provider is configured.
type defaultPolicyProvider struct{}

func (d *defaultPolicyProvider) LoadPolicy() (Policy, error) {
	return Policy{
		DefaultMode:      ModeYOLO,
		MaxFixupAttempts: 2,
		AutoFixup:        false,
	}, nil
}

// defaultGovernanceProvider returns hardcoded defaults when no provider is configured.
type defaultGovernanceProvider struct{}

func (d *defaultGovernanceProvider) LoadGovernance() (GovernanceSettings, error) {
	return DefaultGovernanceSettings(), nil
}

func backlogActivitySpec(
	item backlogItem,
	executionID string,
	purpose agentactivity.Purpose,
	requestedBy string,
	metadata map[string]string,
) agentactivity.Spec {
	// Resolve PhaseKind explicitly here so every execution-package spawn
	// site sees a fully-typed lane intent. Falls back to the per-Purpose
	// default in agentactivity.LaneOf — this assignment exists so the wire
	// shape carries phase_kind for Operations Center utilization without
	// relying on the default-resolution path.
	lane, err := agentactivity.LaneOf(purpose, "")
	phaseKind := ""
	if err == nil {
		phaseKind = string(lane)
	}
	return agentactivity.Spec{
		OwnerType:   agentactivity.OwnerBacklog,
		OwnerKind:   item.Kind,
		OwnerName:   item.Name,
		OwnerTitle:  item.Title,
		ExecutionID: executionID,
		Purpose:     purpose,
		PhaseKind:   phaseKind,
		RequestedBy: requestedBy,
		Metadata:    metadata,
	}
}

// ServiceConfig configures execution service dependencies.
//
// DataRoot is the runtime-home data directory where backlog item folders
// live (`~/.vrooli/data/vrooli/swarm-manager/<kind>/<name>/...`). RepoRoot
// is the scenario source path, used only as a repo anchor for resolving
// the sibling scenarios/ directory in preflight.
type ServiceConfig struct {
	DataRoot                 string
	RepoRoot                 string
	StorePath                string
	CircuitBreakerPath       string
	SelfScenarioName         string
	PolicyProvider           PolicyProvider
	GovernanceProvider       GovernanceProvider
	ReviewThresholdsProvider ReviewThresholdsProvider
	AgentService             AgentManagerAvailability
	ScenarioLifecycle        ScenarioLifecycle
	ScenarioHealthChecker    ScenarioHealthChecker
	PromptClient             promptmanager.Client
	ExperimentClient         promptmanager.ExperimentClient
	Archiver                 Archiver
	ReviewClient             ReviewClient
	BaselineClient           BaselineClient
	BaselineEngagementRunner BaselineEngagementRunner
	PlanRenderer             planclient.MarkdownRenderer
	PhasedPlanWorkflow       agentmanager.WorkflowInvoker
	ConclusionWorkflow       agentmanager.WorkflowInvoker
	WorkWorkflow             agentmanager.WorkflowInvoker
	SpecSyncWorkflow         agentmanager.WorkflowInvoker
	TransitionRegistry       transitions.Registry
	Finalization             FinalizationConfig
}

// Service owns execution lifecycle logic.
type Service struct {
	dataRoot                 string
	repoRoot                 string
	selfScenarioName         string
	finalizationCfg          FinalizationConfig
	store                    Store
	policyProvider           PolicyProvider
	governanceProvider       GovernanceProvider
	reviewThresholdsProvider ReviewThresholdsProvider
	agentService             AgentManagerAvailability
	operationStarter         OperationStarter
	promptClient             promptmanager.Client
	experimentClient         promptmanager.ExperimentClient
	archiver                 Archiver
	reviewClient             ReviewClient
	baselineClient           BaselineClient
	baselineEngagementRunner BaselineEngagementRunner
	planRenderer             planclient.MarkdownRenderer
	phasedPlanWorkflow       agentmanager.WorkflowInvoker
	conclusionWorkflow       agentmanager.WorkflowInvoker
	workWorkflow             agentmanager.WorkflowInvoker
	specSyncWorkflow         agentmanager.WorkflowInvoker
	transitionRegistry       transitions.Registry
	engagementStore          *EngagementStore
	differ                   RunDiffer
	stopper                  RunStopper
	approver                 RunApprover
	scenarioLifecycle        ScenarioLifecycle
	scenarioHealth           ScenarioHealthChecker
	reviewService            ReviewServiceIntegration
	eventDispatcher          dispatch.NodeDispatcher
	eventLogger              EventLogger
	circuitBreaker           *CircuitBreaker
	activityLaneReader       ActivityLaneReader
	goalPriorityProvider     GoalPriorityProvider
	goalReadyProvider        GoalReadyProvider
	autoDrainProvider        AutoDrainProvider
	autoFilerWaker           AutoFilerWaker
	processingFinalizations  map[string]struct{}
	processingHolds          map[string]struct{}
	mu                       sync.Mutex
}

// SetActivityLaneReader wires the agentactivity-backed lane reader after
// construction. The wiring layer (server bootstrap) calls this once both
// services exist; tests can leave it unset and GovernanceStatus will
// report zero for non-Execute lanes.
func (s *Service) SetActivityLaneReader(r ActivityLaneReader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.activityLaneReader = r
}

// NewService creates a new execution service.
func NewService(cfg ServiceConfig) *Service {
	dataRoot := strings.TrimSpace(cfg.DataRoot)
	if dataRoot == "" {
		if p, err := runtimepaths.DataPath(""); err == nil {
			dataRoot = p
		} else {
			dataRoot = pathutil.ResolveScenarioRoot("swarm-manager")
		}
	}
	repoRoot := strings.TrimSpace(cfg.RepoRoot)
	if repoRoot == "" {
		repoRoot = pathutil.ResolveScenarioRoot("swarm-manager")
	}
	if len(cfg.TransitionRegistry.Definitions()) == 0 {
		// Direct service users (including focused package tests) still resolve
		// the scenario declaration rather than reintroducing workflow-key
		// defaults. Server bootstrap replaces this with its already-validated
		// registry instance.
		if registry, err := transitions.LoadDir(filepath.Join(repoRoot, ".vrooli", "swarm-transitions")); err == nil {
			cfg.TransitionRegistry = registry
		}
	}

	pc := cfg.PromptClient
	if pc == nil {
		pc = promptmanager.NewHTTPClient()
	}
	pp := cfg.PolicyProvider
	if pp == nil {
		pp = &defaultPolicyProvider{}
	}
	gp := cfg.GovernanceProvider
	if gp == nil {
		gp = &defaultGovernanceProvider{}
	}
	rtp := cfg.ReviewThresholdsProvider
	fc := cfg.Finalization
	if fc == (FinalizationConfig{}) {
		fc = DefaultFinalizationConfig()
	}
	selfName := strings.TrimSpace(cfg.SelfScenarioName)
	if selfName == "" {
		selfName = filepath.Base(repoRoot)
	}
	circuitBreakerPath := strings.TrimSpace(cfg.CircuitBreakerPath)
	if circuitBreakerPath == "" {
		circuitBreakerPath = defaultCircuitBreakerPath(cfg.StorePath)
	}

	service := &Service{
		dataRoot:                 dataRoot,
		repoRoot:                 repoRoot,
		selfScenarioName:         selfName,
		finalizationCfg:          fc,
		store:                    NewStore(cfg.StorePath),
		policyProvider:           pp,
		governanceProvider:       gp,
		reviewThresholdsProvider: rtp,
		agentService:             cfg.AgentService,
		promptClient:             pc,
		experimentClient:         cfg.ExperimentClient,
		archiver:                 cfg.Archiver,
		reviewClient:             cfg.ReviewClient,
		baselineClient:           cfg.BaselineClient,
		baselineEngagementRunner: cfg.BaselineEngagementRunner,
		planRenderer:             cfg.PlanRenderer,
		phasedPlanWorkflow:       cfg.PhasedPlanWorkflow,
		conclusionWorkflow:       cfg.ConclusionWorkflow,
		workWorkflow:             cfg.WorkWorkflow,
		specSyncWorkflow:         cfg.SpecSyncWorkflow,
		transitionRegistry:       cfg.TransitionRegistry,
		engagementStore:          NewEngagementStore(engagementStorePath(cfg.StorePath)),
		scenarioLifecycle:        cfg.ScenarioLifecycle,
		scenarioHealth:           cfg.ScenarioHealthChecker,
		circuitBreaker:           NewCircuitBreaker(circuitBreakerPath),
		processingFinalizations:  map[string]struct{}{},
		processingHolds:          map[string]struct{}{},
	}
	if service.phasedPlanWorkflow == nil {
		service.phasedPlanWorkflow = agentmanager.NewWorkflowService()
	}
	if service.conclusionWorkflow == nil {
		service.conclusionWorkflow = agentmanager.NewWorkflowService()
	}
	if service.workWorkflow == nil {
		service.workWorkflow = agentmanager.NewWorkflowService()
	}
	if service.specSyncWorkflow == nil {
		service.specSyncWorkflow = agentmanager.NewWorkflowService()
	}
	if differ, ok := cfg.AgentService.(RunDiffer); ok {
		service.differ = differ
	}
	if stopper, ok := cfg.AgentService.(RunStopper); ok {
		service.stopper = stopper
	}
	if approver, ok := cfg.AgentService.(RunApprover); ok {
		service.approver = approver
	}
	return service
}

// SetTransitionRegistry installs the immutable, scenario-owned transition
// declarations used to resolve Agent Manager workflow locators at runtime.
func (s *Service) SetTransitionRegistry(registry transitions.Registry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.transitionRegistry = registry
}

func defaultCircuitBreakerPath(storePath string) string {
	if trimmed := strings.TrimSpace(storePath); trimmed != "" {
		return filepath.Join(filepath.Dir(trimmed), "circuit-breaker.json")
	}
	path, err := runtimepaths.StatePath("circuit-breaker.json")
	if err != nil {
		panic(err)
	}
	return path
}

// engagementStorePath places the Baseline Modes engagement-owner index next to
// the execution records store so they share a lifecycle/backup boundary. Empty
// store path ⇒ the engagement store resolves the runtime state dir itself.
func engagementStorePath(storePath string) string {
	if trimmed := strings.TrimSpace(storePath); trimmed != "" {
		return filepath.Join(filepath.Dir(trimmed), "engagement-owners.json")
	}
	return ""
}

// SetEventDispatcher sets an optional event dispatcher for real-time graph updates.
func (s *Service) SetEventDispatcher(d dispatch.NodeDispatcher) {
	s.eventDispatcher = d
}

// SetEventLogger injects an optional event logger for analytics tracking.
func (s *Service) SetEventLogger(l EventLogger) {
	s.eventLogger = l
}

// SetReviewService injects an optional review service for post-finalization
// evidence gathering. Set after construction to avoid import cycles.
func (s *Service) SetReviewService(rs ReviewServiceIntegration) {
	s.reviewService = rs
}

// SetGoalDirectedProviders wires the optional goal-directed drain dependencies:
// a per-item goal-priority source for the drain comparator, a ready-items
// source and an enablement flag for continuous auto-enqueue. Any may be nil,
// leaving pure FIFO / no continuous drain. Set after construction to avoid an
// import cycle with the goals package.
func (s *Service) SetGoalDirectedProviders(priorities GoalPriorityProvider, ready GoalReadyProvider, autoDrain AutoDrainProvider) {
	s.goalPriorityProvider = priorities
	s.goalReadyProvider = ready
	s.autoDrainProvider = autoDrain
}

// RecordView emits a view event for analytics.
func (s *Service) RecordView(execID string) {
	if s.eventLogger != nil {
		s.eventLogger.EmitExecutionViewed(execID)
	}
}

// dispatchStatusAndLog is the canonical "after you mutate record.Status" helper.
// Every site that transitions an execution's status must call this (not just
// dispatchStatusUpdate) so the event log captures the transition.
func (s *Service) dispatchStatusAndLog(record Record, prevStatus Status) {
	s.logExecutionEvent(record, prevStatus)
	s.dispatchStatusUpdate(record)
}

// dispatchStatusUpdate emits a node-update event for an execution record status change.
func (s *Service) dispatchStatusUpdate(record Record) {
	if s.eventDispatcher == nil {
		return
	}
	s.eventDispatcher.DispatchNodeUpdate("ExecutionRecord", "execution-record/"+record.ExecutionID, map[string]any{
		"execution_id": record.ExecutionID,
		"backlog_kind": record.BacklogKind,
		"backlog_name": record.BacklogName,
		"status":       string(record.Status),
		"mode":         string(record.Mode),
		"run_id":       record.RunID,
	})
	s.eventDispatcher.DispatchInvalidate("topology", "plan")
}

// logExecutionEvent emits an event log entry for an execution status transition.
func (s *Service) logExecutionEvent(record Record, prevStatus Status) {
	if s.eventLogger == nil {
		return
	}
	if prevStatus == "" {
		// New record — emit created event.
		s.eventLogger.EmitExecutionCreated(record.ExecutionID, record.BacklogKind, record.BacklogName, string(record.Mode))
		return
	}
	if prevStatus == record.Status {
		return
	}
	s.eventLogger.EmitExecutionStatusChanged(record.ExecutionID, string(prevStatus), string(record.Status))

	switch record.Status {
	case StatusCompleted:
		dur := executionDuration(record)
		if record.ManuallyAccepted {
			s.eventLogger.EmitExecutionManuallyAccepted(
				record.ExecutionID,
				record.AcceptedBy,
				record.AcceptedReason,
				string(record.AcceptedPreviousStatus),
			)
		}
		s.eventLogger.EmitExecutionCompleted(record.ExecutionID, dur, record.FixupAttempt > 0)
	case StatusFailed:
		dur := executionDuration(record)
		s.eventLogger.EmitExecutionFailed(record.ExecutionID, record.FailureReason, dur)
	case StatusCanceled:
		s.eventLogger.EmitExecutionCanceled(record.ExecutionID, "user canceled")
	}
}

// ManuallyAcceptLatestForBacklog finds the most recent non-cancelled execution
// for the given backlog item and flips it to StatusCompleted with
// ManuallyAccepted=true. Intended to be called when the user manually
// transitions a backlog item from failed → completed, overriding the agent's
// own verdict. Returns (accepted execution ID, true) when a record was
// flipped, or ("", false) if no eligible execution was found.
func (s *Service) ManuallyAcceptLatestForBacklog(ctx context.Context, backlogKind, backlogName, acceptor, reason string) (string, bool, error) {
	_ = s.ProcessActiveExecutions(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return "", false, err
	}

	idx := -1
	for i := range records {
		r := &records[i]
		if r.BacklogKind != backlogKind || r.BacklogName != backlogName {
			continue
		}
		switch r.Status {
		case StatusFailed, StatusNeedsFixup:
			// eligible
		default:
			continue
		}
		if idx == -1 || r.CreatedAt > records[idx].CreatedAt {
			idx = i
		}
	}
	if idx == -1 {
		return "", false, nil
	}

	record := &records[idx]
	prev := record.Status
	now := nowRFC3339()
	record.AcceptedPreviousStatus = prev
	record.Status = StatusCompleted
	record.ManuallyAccepted = true
	record.AcceptedBy = strings.TrimSpace(acceptor)
	record.AcceptedReason = strings.TrimSpace(reason)
	record.FailureReason = ""
	if record.FinishedAt == "" {
		record.FinishedAt = now
	}
	record.UpdatedAt = now
	if err := s.store.Save(records); err != nil {
		return "", false, err
	}
	s.dispatchStatusAndLog(*record, prev)
	return record.ExecutionID, true, nil
}

func executionDuration(r Record) float64 {
	if r.StartedAt == "" || r.FinishedAt == "" {
		return 0
	}
	start, err1 := time.Parse(time.RFC3339, r.StartedAt)
	end, err2 := time.Parse(time.RFC3339, r.FinishedAt)
	if err1 != nil || err2 != nil {
		return 0
	}
	return end.Sub(start).Seconds()
}

// Get returns a single execution by ID after status refresh.
func (s *Service) Get(ctx context.Context, executionID string) (Record, error) {
	_ = s.ProcessActiveExecutions(ctx)
	s.mu.Lock()
	defer s.mu.Unlock()
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	return records[idx], nil
}

// List returns executions ordered by created_at descending.
func (s *Service) List(ctx context.Context, filters ListFilters) ([]Record, error) {
	_ = s.ProcessActiveExecutions(ctx)
	return s.ListSnapshot(ctx, filters)
}

// ListSnapshot returns executions from the persisted store without polling
// agent-manager or draining pending work. Use this for read-only aggregate
// projections where freshness is provided by the regular execution poller and
// graph invalidation events; blocking those projections on remote run-state
// refresh makes unrelated UI surfaces slow to open.
func (s *Service) ListSnapshot(_ context.Context, filters ListFilters) ([]Record, error) {
	records, err := s.store.Load()
	if err != nil {
		return nil, err
	}

	filtered := make([]Record, 0, len(records))
	for _, record := range records {
		if !matchesFilters(record, filters) {
			continue
		}
		filtered = append(filtered, record)
	}

	sort.Slice(filtered, func(i, j int) bool {
		if filtered[i].CreatedAt == filtered[j].CreatedAt {
			return filtered[i].ExecutionID > filtered[j].ExecutionID
		}
		return filtered[i].CreatedAt > filtered[j].CreatedAt
	})

	return filtered, nil
}

func (s *Service) loadRecordLocked(executionID string) ([]Record, int, error) {
	records, err := s.store.Load()
	if err != nil {
		return nil, -1, err
	}
	trimmed := strings.TrimSpace(executionID)
	for i := range records {
		if records[i].ExecutionID == trimmed {
			return records, i, nil
		}
	}
	return nil, -1, apierr.NotFound("execution not found")
}

func normalizeMode(mode Mode) Mode {
	switch Mode(strings.ToLower(strings.TrimSpace(string(mode)))) {
	case ModeManual:
		return ModeManual
	case ModeYOLO:
		return ModeYOLO
	default:
		return ""
	}
}

func normalizeOperation(operation string) string {
	switch strings.ToLower(strings.TrimSpace(operation)) {
	case "", "generator":
		return "generator"
	case "improver":
		return "improver"
	default:
		return "generator"
	}
}

// wrapAgentError translates agentmanager errors into DomainErrors with
// appropriate HTTP semantics.
func wrapAgentError(err error) error {
	if errors.Is(err, agentmanager.ErrNotAvailable) {
		return apierr.Unavailable("agent-manager is not available")
	}
	if errors.Is(err, agentactivity.ErrBacklogItemBusy) {
		return apierr.Conflict("an agent is already active for this backlog item")
	}
	if errors.Is(err, agentmanager.ErrRequestFailed) {
		return apierr.BadGateway("agent-manager request failed; check agent-manager health/logs and retry")
	}
	if spe := agentmanager.AsStalePlanError(err); spe != nil {
		return apierr.PlanStale(
			"this plan references paths that no longer exist; re-workshop required",
			map[string]any{
				"missingPaths": spe.MissingPaths,
				"projectRoot":  spe.ProjectRoot,
			},
		)
	}
	return err
}
