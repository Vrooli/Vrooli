package execution

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/handoff"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptcatalog"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/storage"
	"swarm-manager/internal/workshop"
)

var (
	errNotFound        = errors.New("execution not found")
	errSessionExpired  = errors.New("agent session expired")
	errAtCapacity      = errors.New("concurrency limit reached")
	errQueueFull       = errors.New("queue depth limit exceeded")
	errCircuitBroken   = errors.New("circuit breaker tripped")
	errCostCapExceeded = errors.New("estimated cost exceeds cap")
)

// Backlog status values used by execution to update backlog items.
// Defined locally to avoid a circular import with the backlog package.
const (
	backlogStatusQueued      = "queued"
	backlogStatusCompleted   = "completed"
	backlogStatusFailed      = "failed"
	backlogStatusArchived    = "archived"
	backlogStatusBacklog     = "backlog"
	backlogStatusResearching = "researching"
	backlogStatusReady       = "ready"
)

// DOC: docs/concepts/ARCHITECTURE.md#key-flows
// DOC: docs/reference/operational-targets.md
// DOC: docs/internal/TEMPORAL-FLOWS.md

type agentSpawner interface {
	IsEnabled() bool
	SpawnBacklog(ctx context.Context, req agentmanager.BacklogSpawnRequest) (agentmanager.RunResult, error)
}

type runInspector interface {
	GetRunState(ctx context.Context, runID string) (agentmanager.RunState, error)
}

type runStopper interface {
	StopRun(ctx context.Context, runID string) error
}

type runContinuer interface {
	ContinueRun(ctx context.Context, runID string, message string) error
}

// Archiver performs scenario archive operations after spec-sync completes.
type Archiver interface {
	ArchiveScenario(ctx context.Context, ac ArchiveContext) error
}

// PolicyProvider reads execution policy defaults from the unified settings store.
type PolicyProvider interface {
	LoadPolicy() (Policy, error)
}

// ReviewThresholdsProvider reads review threshold settings.
type ReviewThresholdsProvider interface {
	LoadReviewThresholds() (*ReviewThresholds, error)
}

// GovernanceProvider reads governance settings from the unified settings store.
type GovernanceProvider interface {
	LoadGovernance() (GovernanceSettings, error)
}

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

func executionActivityPurpose(runType string) agentactivity.Purpose {
	switch strings.ToLower(strings.TrimSpace(runType)) {
	case "initialize":
		return agentactivity.PurposeInitialize
	case "workshop":
		return agentactivity.PurposeWorkshop
	case "finalize":
		return agentactivity.PurposeFinalize
	case "research":
		return agentactivity.PurposeResearch
	case "process":
		return agentactivity.PurposeProcess
	case "fixup":
		return agentactivity.PurposeFixup
	case "followup", "custom":
		return agentactivity.PurposeFollowUp
	case "spec-sync", "spec_sync":
		return agentactivity.PurposeSpecSync
	case "classify":
		return agentactivity.PurposeClassify
	default:
		return agentactivity.PurposeProcess
	}
}

func backlogActivitySpec(
	item backlogItem,
	executionID string,
	purpose agentactivity.Purpose,
	requestedBy string,
	metadata map[string]string,
) agentactivity.Spec {
	return agentactivity.Spec{
		OwnerType:   agentactivity.OwnerBacklog,
		OwnerKind:   item.Kind,
		OwnerName:   item.Name,
		OwnerTitle:  item.Title,
		ExecutionID: executionID,
		Purpose:     purpose,
		RequestedBy: requestedBy,
		Metadata:    metadata,
	}
}

func scenarioActivitySpec(
	ac ArchiveContext,
	executionID string,
	requestedBy string,
	metadata map[string]string,
) agentactivity.Spec {
	return agentactivity.Spec{
		OwnerType:   agentactivity.OwnerScenario,
		OwnerName:   ac.ScenarioName,
		OwnerTitle:  ac.ScenarioName,
		ExecutionID: executionID,
		Purpose:     agentactivity.PurposeSpecSync,
		RequestedBy: requestedBy,
		Metadata:    metadata,
	}
}

// ServiceConfig configures execution service dependencies.
type ServiceConfig struct {
	RootDir                  string
	StorePath                string
	PolicyProvider           PolicyProvider
	GovernanceProvider       GovernanceProvider
	ReviewThresholdsProvider ReviewThresholdsProvider
	AgentService             agentSpawner
	ScenarioLifecycle        ScenarioLifecycle
	ScenarioHealthChecker    ScenarioHealthChecker
	PromptClient             promptmanager.Client
	Archiver                 Archiver
	ReviewClient             ReviewClient
	Finalization             FinalizationConfig
}

// EventDispatcher emits graph change events for real-time WebSocket updates.
type EventDispatcher interface {
	DispatchNodeUpdate(nodeType, nodeID string, data any)
	DispatchInvalidate(lenses ...string)
}

// ReviewServiceIntegration is the subset of the review service that finalization
// needs to trigger evidence gathering. Uses a callback signature to avoid
// import cycles between the execution and review packages.
type ReviewServiceIntegration interface {
	StartReviewForExecution(ctx context.Context, executionID, backlogKind, backlogName, itemTitle, itemDir string, affectedScenarios []string, changedPathsByScenario map[string][]string) error
}

// EventLogger records execution state-change events for analytics.
type EventLogger interface {
	EmitExecutionCreated(execID, backlogKind, backlogName, mode string)
	EmitExecutionStatusChanged(execID, from, to string)
	EmitExecutionCompleted(execID string, durationSecs float64, hadFixups bool)
	EmitExecutionFailed(execID, reason string, durationSecs float64)
	EmitExecutionCanceled(execID, reason string)
	EmitExecutionViewed(execID string)
}

// Service owns execution lifecycle logic.
type Service struct {
	rootDir                  string
	finalizationCfg          FinalizationConfig
	store                    Store
	policyProvider           PolicyProvider
	governanceProvider       GovernanceProvider
	reviewThresholdsProvider ReviewThresholdsProvider
	agentService             agentSpawner
	promptClient             promptmanager.Client
	archiver                 Archiver
	reviewClient             ReviewClient
	inspector                runInspector
	differ                   RunDiffer
	stopper                  runStopper
	continuer                runContinuer
	scenarioLifecycle        ScenarioLifecycle
	scenarioHealth           ScenarioHealthChecker
	reviewService            ReviewServiceIntegration
	eventDispatcher          EventDispatcher
	eventLogger              EventLogger
	circuitBreaker           *CircuitBreaker
	processingFinalizations  map[string]struct{}
	mu                       sync.Mutex
}

// NewService creates a new execution service.
func NewService(cfg ServiceConfig) *Service {
	rootDir := strings.TrimSpace(cfg.RootDir)
	if rootDir == "" {
		rootDir = pathutil.ResolveScenarioRoot("swarm-manager")
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
	service := &Service{
		rootDir:                  rootDir,
		finalizationCfg:          fc,
		store:                    NewStore(cfg.StorePath),
		policyProvider:           pp,
		governanceProvider:       gp,
		reviewThresholdsProvider: rtp,
		agentService:             cfg.AgentService,
		promptClient:             pc,
		archiver:                 cfg.Archiver,
		reviewClient:             cfg.ReviewClient,
		scenarioLifecycle:        cfg.ScenarioLifecycle,
		scenarioHealth:           cfg.ScenarioHealthChecker,
		circuitBreaker:           NewCircuitBreaker(filepath.Join(rootDir, ".vrooli", "circuit-breaker.json")),
		processingFinalizations:  map[string]struct{}{},
	}
	if inspector, ok := cfg.AgentService.(runInspector); ok {
		service.inspector = inspector
	}
	if differ, ok := cfg.AgentService.(RunDiffer); ok {
		service.differ = differ
	}
	if continuer, ok := cfg.AgentService.(runContinuer); ok {
		service.continuer = continuer
	}
	if stopper, ok := cfg.AgentService.(runStopper); ok {
		service.stopper = stopper
	}
	return service
}

// SetEventDispatcher sets an optional event dispatcher for real-time graph updates.
func (s *Service) SetEventDispatcher(d EventDispatcher) {
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

// RecordView emits a view event for analytics.
func (s *Service) RecordView(execID string) {
	if s.eventLogger != nil {
		s.eventLogger.EmitExecutionViewed(execID)
	}
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
	s.eventDispatcher.DispatchInvalidate("topology", "flow", "operations")
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
		s.eventLogger.EmitExecutionCompleted(record.ExecutionID, dur, record.FixupAttempt > 0)
	case StatusFailed:
		dur := executionDuration(record)
		s.eventLogger.EmitExecutionFailed(record.ExecutionID, record.FailureReason, dur)
	case StatusCanceled:
		s.eventLogger.EmitExecutionCanceled(record.ExecutionID, "user canceled")
	}
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

// QueueBacklog creates an execution record and optionally starts it.
func (s *Service) QueueBacklog(ctx context.Context, req CreateRequest) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy, err := s.policyProvider.LoadPolicy()
	if err != nil {
		return Record{}, err
	}

	if strings.TrimSpace(req.BacklogKind) == "" {
		return Record{}, fmt.Errorf("backlog_kind is required")
	}
	if strings.TrimSpace(req.BacklogName) == "" {
		return Record{}, fmt.Errorf("backlog_name is required")
	}
	mode := normalizeMode(req.Mode)
	if mode == "" {
		mode = policy.DefaultMode
	}
	if mode == "" {
		return Record{}, fmt.Errorf("mode must be manual or yolo")
	}

	item, err := s.loadBacklogItem(req.BacklogKind, req.BacklogName)
	if err != nil {
		return Record{}, err
	}
	if !isQueueableStatus(item.Kind, item.Status) {
		return Record{}, fmt.Errorf("backlog item cannot be queued from current status: %s", item.Status)
	}
	preflight := s.processPreflightForItem(item, true)
	if !preflight.Ready && (!req.Force || hasNonForceableExecutionReasons(preflight.BlockingReasons)) {
		return Record{}, fmt.Errorf("process preflight failed: %s", strings.Join(preflight.BlockingReasons, "; "))
	}

	// Load governance settings for enforcement checks.
	gov, govErr := s.governanceProvider.LoadGovernance()
	if govErr != nil {
		return Record{}, govErr
	}

	// Circuit breaker check.
	itemKey := strings.ToLower(strings.TrimSpace(req.BacklogKind)) + "/" + strings.TrimSpace(req.BacklogName)
	if broken, remaining, cbErr := s.circuitBreaker.IsBroken(itemKey, gov.CircuitBreakerCooldownMinutes); cbErr != nil {
		return Record{}, cbErr
	} else if broken && !req.Force {
		return Record{}, fmt.Errorf("%w: %s (cooldown remaining: %s)", errCircuitBroken, itemKey, remaining.Truncate(time.Second))
	}

	records, err := s.store.Load()
	if err != nil {
		return Record{}, err
	}

	// Queue depth enforcement.
	if gov.MaxQueueDepth > 0 {
		queued := countQueuedExecutions(records)
		if queued >= gov.MaxQueueDepth {
			return Record{}, fmt.Errorf("%w (%d/%d)", errQueueFull, queued, gov.MaxQueueDepth)
		}
	}

	// Cost cap enforcement.
	if gov.ExecutionCostCapPerRun > 0 && gov.CostPerTurnEstimate > 0 {
		agentMaxTurns := gov.AgentMaxTurns
		if agentMaxTurns <= 0 {
			agentMaxTurns = 60
		}
		estimatedCost := gov.CostPerTurnEstimate * float64(agentMaxTurns)
		if estimatedCost > gov.ExecutionCostCapPerRun && !req.Force {
			return Record{}, fmt.Errorf("%w: estimated $%.2f exceeds cap $%.2f (use force to override)", errCostCapExceeded, estimatedCost, gov.ExecutionCostCapPerRun)
		}
	}

	now := nowRFC3339()
	record := Record{
		ExecutionID:    idgen.Generate(),
		BacklogKind:    strings.ToLower(strings.TrimSpace(req.BacklogKind)),
		BacklogName:    strings.TrimSpace(req.BacklogName),
		PreviousStatus: strings.ToLower(strings.TrimSpace(item.Status)),
		Mode:           mode,
		Status:         StatusPending,
		StartedBy:      strings.TrimSpace(req.StartedBy),
		Operation:      normalizeOperation(req.Operation),
		Force:          req.Force,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if record.StartedBy == "" {
		record.StartedBy = "swarm-manager"
	}
	if record.Operation == "" {
		record.Operation = "generator"
	}

	if err := s.updateBacklogStatus(item, backlogStatusQueued); err != nil {
		return Record{}, err
	}

	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
	s.logExecutionEvent(record, "")

	if mode == ModeYOLO {
		started, startErr := s.startLocked(ctx, record.ExecutionID)
		if startErr == nil {
			return started, nil
		}

		// At capacity: leave as pending for the poller to drain later.
		if errors.Is(startErr, errAtCapacity) {
			log.Printf("[execution] at capacity (%s); leaving %s as pending", record.ExecutionID, itemKey)
			s.dispatchStatusUpdate(record)
			return record, nil
		}

		// Roll back queue side-effects when immediate start fails.
		rolledBack, rbErr := s.store.Load()
		if rbErr == nil {
			filtered := make([]Record, 0, len(rolledBack))
			for _, candidate := range rolledBack {
				if candidate.ExecutionID != record.ExecutionID {
					filtered = append(filtered, candidate)
				}
			}
			_ = s.store.Save(filtered)
		}
		_ = s.updateBacklogStatus(item, restoreBacklogStatus(record))
		return Record{}, startErr
	}

	s.dispatchStatusUpdate(record)
	return record, nil
}

// QueueSpecSyncArchive creates an execution that runs spec-sync, then archives on completion.
func (s *Service) QueueSpecSyncArchive(ctx context.Context, ac ArchiveContext) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(ac.ScenarioName) == "" {
		return Record{}, fmt.Errorf("scenario_name is required")
	}
	if strings.TrimSpace(ac.ScenarioPath) == "" {
		return Record{}, fmt.Errorf("scenario_path is required")
	}
	if _, err := os.Stat(ac.ScenarioPath); err != nil {
		return Record{}, fmt.Errorf("scenario path does not exist: %s", ac.ScenarioPath)
	}

	if s.agentService == nil || !s.agentService.IsEnabled() {
		return Record{}, agentmanager.ErrNotAvailable
	}

	records, err := s.store.Load()
	if err != nil {
		return Record{}, err
	}

	now := nowRFC3339()
	record := Record{
		ExecutionID:    idgen.Generate(),
		BacklogKind:    "spec-sync",
		BacklogName:    ac.ScenarioName,
		Mode:           ModeYOLO,
		Status:         StatusPending,
		StartedBy:      "swarm-manager",
		Operation:      "spec-sync-archive",
		ArchiveContext: &ac,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	specSyncEntry, ok := promptcatalog.ResolveSpecSyncSkill()
	if !ok {
		return Record{}, fmt.Errorf("spec-sync prompt catalog entry missing")
	}

	// Fetch spec-sync prompt from prompt-manager
	specSyncVars := map[string]string{
		"TARGET": ac.ScenarioName,
	}
	prompt, promptErr := s.promptClient.ReadSkill(ctx, specSyncEntry.SkillID, specSyncVars, false)
	if promptErr != nil {
		log.Printf("[execution] spec-sync prompt fetch failed: %v", promptErr)
		prompt = "Read the implementation code in this scenario and update all spec artifacts (PRD.md, requirements/, README.md, docs/) to match the actual behavior."
	}
	record.PromptTrace = &PromptTrace{
		Purpose:        "spec-sync",
		Prompt:         prompt,
		PromptRevision: promptRevision(prompt),
		UsedFallback:   promptErr != nil,
		CapturedAt:     now,
	}

	// Spawn agent targeting the scenario directory
	activityCtx := agentactivity.WithSpec(ctx, scenarioActivitySpec(
		ac,
		record.ExecutionID,
		record.StartedBy,
		map[string]string{
			"entrypoint": "execution.spec_sync_archive",
		},
	))

	runResult, err := s.agentService.SpawnBacklog(activityCtx, agentmanager.BacklogSpawnRequest{
		Kind:        "spec-sync",
		Name:        ac.ScenarioName,
		Title:       "Spec sync: " + ac.ScenarioName,
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   ac.ScenarioPath,
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager",
		Purpose:     "spec-sync",
		Environment: map[string]string{"VROOLI_SPAWN_SOURCE": record.BacklogKind + "/" + record.BacklogName},
	})
	if err != nil {
		return Record{}, err
	}

	record.TaskID = runResult.TaskID
	record.RunID = runResult.RunID
	record.StartedAt = nowRFC3339()
	record.Status = StatusStarting
	record.UpdatedAt = nowRFC3339()

	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}

	s.dispatchStatusUpdate(record)
	return record, nil
}

// Start starts a pending/failed execution now.
func (s *Service) Start(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.startLocked(ctx, executionID)
}

func (s *Service) startLocked(ctx context.Context, executionID string) (Record, error) {
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := records[idx]
	if record.Status == StatusStarting || record.Status == StatusRunning || record.Status == StatusNeedsReview || record.Status == StatusCompleted {
		return record, nil
	}
	if record.Status == StatusCanceled {
		return Record{}, fmt.Errorf("cannot start canceled execution")
	}

	// Concurrency gate.
	if gov, govErr := s.governanceProvider.LoadGovernance(); govErr == nil {
		active := countActiveExecutions(records)
		if active >= gov.MaxConcurrentExecutions {
			return Record{}, errAtCapacity
		}
	}

	if s.agentService == nil || !s.agentService.IsEnabled() {
		return Record{}, agentmanager.ErrNotAvailable
	}

	item, err := s.loadBacklogItem(record.BacklogKind, record.BacklogName)
	if err != nil {
		return Record{}, err
	}
	preflight := s.processPreflightForItem(item, false)
	if !preflight.Ready && (!record.Force || hasNonForceableExecutionReasons(preflight.BlockingReasons)) {
		return Record{}, fmt.Errorf("process preflight failed: %s", strings.Join(preflight.BlockingReasons, "; "))
	}

	itemDir := s.itemDir(item.Kind, item.Name)
	deliverablePath := deliverableForKind(item.Kind)
	deliverableContent := workshop.LoadPlanContentByName(itemDir, deliverablePath)
	usedFallback := strings.TrimSpace(deliverableContent) == ""
	if usedFallback {
		log.Printf("[execution] %s empty or missing in %s", deliverablePath, itemDir)
	}
	ideaHandoff, handoffErr := s.buildIdeaHandoffPackage(item, itemDir, preflight)
	if handoffErr != nil {
		return Record{}, handoffErr
	}
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               item.Kind,
		Name:               item.Name,
		Title:              item.Title,
		ItemFolder:         itemDir,
		RunType:            "process",
		DeliverablePath:    deliverablePath,
		DeliverableContent: deliverableContent,
		IdeaHandoff:        ideaHandoff,
	})
	record.PromptTrace = &PromptTrace{
		Purpose:        "process",
		Prompt:         prompt,
		PromptRevision: promptRevision(prompt),
		UsedFallback:   usedFallback,
		CapturedAt:     nowRFC3339(),
	}

	activityCtx := agentactivity.WithSpec(ctx, backlogActivitySpec(
		item,
		record.ExecutionID,
		agentactivity.PurposeProcess,
		record.StartedBy,
		map[string]string{
			"entrypoint": "execution.start",
			"mode":       string(record.Mode),
			"operation":  record.Operation,
		},
	))

	runResult, err := s.agentService.SpawnBacklog(activityCtx, agentmanager.BacklogSpawnRequest{
		Kind:            item.Kind,
		Name:            item.Name,
		Title:           buildProcessingTitle(item),
		Description:     prompt,
		Prompt:          prompt,
		ScopePath:       ".",
		ProjectRoot:     ".",
		CreatedBy:       record.StartedBy,
		Purpose:         "process",
		AcceptanceAllow: item.AcceptanceAllow,
		AcceptanceDeny:  item.AcceptanceDeny,
		Environment:     map[string]string{"VROOLI_SPAWN_SOURCE": item.Kind + "/" + item.Name},
	})
	if err != nil {
		return Record{}, err
	}

	record.TaskID = runResult.TaskID
	record.RunID = runResult.RunID
	record.StartedAt = nowRFC3339()
	record.FinishedAt = ""
	record.FailureReason = ""
	record.Status = StatusStarting
	record.UpdatedAt = nowRFC3339()
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
	s.dispatchStatusUpdate(record)
	return record, nil
}

// Cancel cancels a scheduled record before it starts.
func (s *Service) Cancel(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := records[idx]

	prevStatus := record.Status
	switch record.Status {
	case StatusPending:
		record.Status = StatusCanceled
		record.UpdatedAt = nowRFC3339()
		record.FinishedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		s.logExecutionEvent(record, prevStatus)
		s.dispatchStatusUpdate(record)
		if err := s.restoreBacklogStatusForRecord(record); err != nil {
			return Record{}, err
		}
		return record, nil
	case StatusStarting, StatusRunning, StatusNeedsReview:
		if s.stopper == nil {
			return Record{}, fmt.Errorf("cancel is not supported by current agent service")
		}
		if strings.TrimSpace(record.RunID) == "" {
			return Record{}, fmt.Errorf("execution has no run id")
		}
		if err := s.stopper.StopRun(ctx, record.RunID); err != nil {
			return Record{}, err
		}
		record.Status = StatusCanceled
		record.UpdatedAt = nowRFC3339()
		record.FinishedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		if err := s.restoreBacklogStatusForRecord(record); err != nil {
			return Record{}, err
		}
		s.logExecutionEvent(record, prevStatus)
		s.dispatchStatusUpdate(record)
		return record, nil
	case StatusValidating, StatusNeedsFixup:
		record.Status = StatusCanceled
		record.UpdatedAt = nowRFC3339()
		record.FinishedAt = nowRFC3339()
		records[idx] = record
		if err := s.store.Save(records); err != nil {
			return Record{}, err
		}
		if err := s.restoreBacklogStatusForRecord(record); err != nil {
			return Record{}, err
		}
		s.logExecutionEvent(record, prevStatus)
		s.dispatchStatusUpdate(record)
		return record, nil
	default:
		return Record{}, fmt.Errorf("only pending/starting/running/needs_review/validating/needs_fixup executions can be canceled")
	}
}

func (s *Service) restoreBacklogStatusForRecord(record Record) error {
	item, err := s.loadBacklogItem(record.BacklogKind, record.BacklogName)
	if err != nil {
		return fmt.Errorf("failed to load backlog item for cancel restore: %w", err)
	}
	if err := s.updateBacklogStatus(item, restoreBacklogStatus(record)); err != nil {
		return fmt.Errorf("failed to restore backlog status after cancel: %w", err)
	}
	return nil
}

// TriggerReview reruns the unified post-run finalization flow for a terminal
// execution.
// DOC: docs/internal/SEAMS.md#trigger-review-api
func (s *Service) TriggerReview(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	record := &records[idx]

	switch record.Status {
	case StatusCompleted, StatusNeedsFixup, StatusFailed:
		// Valid terminal statuses for triggering review
	default:
		return Record{}, fmt.Errorf("cannot trigger post-run checks for execution in %q status", record.Status)
	}

	if _, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr != nil {
		return Record{}, fmt.Errorf("load backlog item for post-run checks: %w", loadErr)
	}
	if !isFinalizationEligible(*record) {
		return Record{}, fmt.Errorf("execution type %q does not support post-run checks", record.effectiveRunType())
	}

	record.Status = StatusValidating
	record.Finalization = &Finalization{
		Eligible:          true,
		Status:            FinalizationStatusPending,
		Phase:             FinalizationPhaseScopeDetection,
		ScopeSource:       FinalizationScopeNone,
		Warnings:          []FinalizationWarning{},
		AffectedScenarios: []string{},
		Scenarios:         []ScenarioFinalization{},
		StartedAt:         nowRFC3339(),
	}
	record.LegacyReviewResult = nil
	record.LegacyReviewJobID = ""
	record.LegacyReviewSkipReason = ""
	record.LegacyReviewStartedAt = ""
	record.FinishedAt = ""
	record.UpdatedAt = nowRFC3339()
	record.FailureReason = ""

	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
	s.dispatchStatusUpdate(*record)
	return *record, nil
}

// Retry retries a failed run immediately.
func (s *Service) Retry(ctx context.Context, executionID string) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	if records[idx].Status != StatusFailed {
		return Record{}, fmt.Errorf("only failed executions can be retried")
	}
	return s.startLocked(ctx, executionID)
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
	s.mu.Lock()
	defer s.mu.Unlock()
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

func (s *Service) loadBacklogItemByRecord(record *Record) (backlogItem, error) {
	return s.loadBacklogItem(record.BacklogKind, record.BacklogName)
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
	return nil, -1, errNotFound
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

func restoreBacklogStatus(record Record) string {
	previous := strings.ToLower(strings.TrimSpace(record.PreviousStatus))
	switch previous {
	case backlogStatusArchived:
		if strings.TrimSpace(record.BacklogKind) == "idea" {
			return backlogStatusArchived
		}
	case backlogStatusBacklog, backlogStatusResearching, backlogStatusReady:
		return previous
	}
	return backlogStatusReady
}

type backlogItem struct {
	Name               string   `json:"name"`
	Title              string   `json:"title"`
	Description        string   `json:"description"`
	Status             string   `json:"status"`
	Priority           int      `json:"priority"`
	Tags               []string `json:"tags"`
	Created            string   `json:"created"`
	Updated            string   `json:"updated"`
	Kind               string   `json:"kind"`
	SourceScenarioName string   `json:"sourceScenarioName,omitempty"`
	AcceptanceAllow    []string `json:"acceptance_allow,omitempty"`
	AcceptanceDeny     []string `json:"acceptance_deny,omitempty"`
}

func (s *Service) loadBacklogItem(kind, name string) (backlogItem, error) {
	specPath := filepath.Join(s.itemDir(kind, name), "spec.json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		return backlogItem{}, err
	}
	var item backlogItem
	if err := json.Unmarshal(data, &item); err != nil {
		return backlogItem{}, err
	}
	item.Name = strings.TrimSpace(name)
	item.Kind = strings.ToLower(strings.TrimSpace(kind))
	if item.Tags == nil {
		item.Tags = []string{}
	}
	return item, nil
}

func (s *Service) updateBacklogStatus(item backlogItem, status string) error {
	item.Status = status
	item.Updated = nowRFC3339()
	specPath := filepath.Join(s.itemDir(item.Kind, item.Name), "spec.json")
	merged := map[string]any{}
	if existing, err := os.ReadFile(specPath); err == nil {
		_ = json.Unmarshal(existing, &merged)
	} else if !os.IsNotExist(err) {
		return err
	}

	merged["name"] = item.Name
	merged["title"] = item.Title
	merged["description"] = item.Description
	merged["status"] = item.Status
	merged["priority"] = item.Priority
	merged["tags"] = item.Tags
	merged["created"] = item.Created
	merged["updated"] = item.Updated
	merged["kind"] = item.Kind
	delete(merged, "research_target")
	return storage.WriteJSONAtomic(specPath, merged)
}

func (s *Service) kindDir(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "idea":
		return filepath.Join(s.rootDir, "ideas")
	case "research":
		return filepath.Join(s.rootDir, "research")
	case "fix":
		return filepath.Join(s.rootDir, "fix")
	case "execute":
		return filepath.Join(s.rootDir, "execute")
	case "chore":
		return filepath.Join(s.rootDir, "chore")
	default:
		return filepath.Join(s.rootDir, "ideas")
	}
}

func (s *Service) itemDir(kind, name string) string {
	return filepath.Join(s.kindDir(kind), strings.TrimSpace(name))
}

func (s *Service) scenariosRootDir() string {
	return filepath.Dir(s.rootDir)
}

func isQueueableStatus(kind, status string) bool {
	normalizedKind := strings.ToLower(strings.TrimSpace(kind))
	switch strings.ToLower(strings.TrimSpace(status)) {
	case backlogStatusBacklog, backlogStatusResearching, backlogStatusReady:
		return true
	case backlogStatusArchived:
		return normalizedKind == "idea"
	default:
		return false
	}
}

func buildProcessingTitle(item backlogItem) string {
	label := strings.TrimSpace(item.Title)
	if label == "" {
		label = strings.TrimSpace(item.Name)
	}
	if label == "" {
		label = "backlog item"
	}
	switch item.Kind {
	case "fix":
		return "Apply fix: " + label
	case "execute":
		return "Execute task: " + label
	case "chore":
		return "Run chore: " + label
	default:
		return "Generate scenario: " + label
	}
}

func deliverableForKind(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "research":
		return "conclusion.md"
	default:
		return "plan.md"
	}
}

func deliverablePromptTag(kind string) string {
	switch deliverableForKind(kind) {
	case "conclusion.md":
		return "research-conclusion"
	default:
		return "implementation-plan"
	}
}

func missingDeliverableReason(kind, deliverablePath string) string {
	switch deliverableForKind(kind) {
	case "conclusion.md":
		return fmt.Sprintf("no research conclusion (%s) exists — run workshop first", deliverablePath)
	default:
		return fmt.Sprintf("no implementation plan (%s) exists — run workshop first", deliverablePath)
	}
}

// executionPromptParams holds all inputs for building a unified execution prompt.
type executionPromptParams struct {
	Kind               string // backlog item kind (idea, fix, execute, etc.)
	Name               string // backlog item name
	Title              string // human-readable title
	ItemFolder         string // absolute path to the backlog item directory
	RunType            string // process, fixup, followup, custom
	DeliverablePath    string // primary workshop artifact path (plan.md or conclusion.md)
	DeliverableContent string // full primary workshop artifact text (empty if missing)
	ReviewFeedback     string // review summary for fixup runs
	FollowUpNote       string // user-provided context for follow-up/custom runs
	IdeaHandoff        *handoff.Package
}

// buildExecutionPrompt constructs a single unified prompt for all execution
// run types. The prompt uses XML tags to clearly delineate context sections.
func buildExecutionPrompt(p executionPromptParams) string {
	var b strings.Builder

	// Execution context header — always present.
	b.WriteString("<execution-context>\n")
	b.WriteString(fmt.Sprintf("Backlog item: %s/%s\n", p.Kind, p.Name))
	if strings.TrimSpace(p.Title) != "" {
		b.WriteString(fmt.Sprintf("Title: %s\n", p.Title))
	}
	b.WriteString(fmt.Sprintf("Item folder: %s\n", p.ItemFolder))
	b.WriteString(fmt.Sprintf("Run type: %s\n", p.RunType))
	b.WriteString("</execution-context>\n")

	// Review feedback — only for fixup runs.
	if strings.TrimSpace(p.ReviewFeedback) != "" {
		b.WriteString("\n<review-feedback>\n")
		b.WriteString(p.ReviewFeedback)
		b.WriteString("\n</review-feedback>\n")
	}

	// Follow-up context — for follow-up and custom runs with user-provided notes.
	if strings.TrimSpace(p.FollowUpNote) != "" {
		b.WriteString("\n<follow-up-context>\n")
		b.WriteString(p.FollowUpNote)
		b.WriteString("\n</follow-up-context>\n")
	}

	// Primary workshop deliverable — always present when available.
	if strings.TrimSpace(p.DeliverableContent) != "" {
		tag := deliverablePromptTag(p.Kind)
		b.WriteString(fmt.Sprintf("\n<%s path=\"%s\">\n", tag, p.DeliverablePath))
		b.WriteString(p.DeliverableContent)
		b.WriteString(fmt.Sprintf("\n</%s>\n", tag))
	}

	if p.IdeaHandoff != nil {
		b.WriteString("\n<idea-handoff>\n")
		b.WriteString(fmt.Sprintf("Handoff directory: %s\n", p.IdeaHandoff.Dir))
		b.WriteString(fmt.Sprintf("Brief path: %s\n", p.IdeaHandoff.BriefPath))
		b.WriteString(fmt.Sprintf("Manifest path: %s\n", p.IdeaHandoff.ManifestPath))
		b.WriteString(fmt.Sprintf("Source index path: %s\n", p.IdeaHandoff.SourceIndexPath))
		b.WriteString("Use brief.md as the ecosystem-manager task notes when creating the downstream task.\n")
		b.WriteString("Preserve the handoff origin metadata on the ecosystem-manager task so later loops can trace back to swarm-manager.\n")
		b.WriteString("When creating the downstream task, pass: --handoff-dir, --origin-source swarm-manager, --origin-backlog-item, and --origin-item-folder.\n")
		b.WriteString("</idea-handoff>\n")

		if strings.TrimSpace(p.IdeaHandoff.BriefMarkdown) != "" {
			b.WriteString(fmt.Sprintf("\n<idea-handoff-brief path=\"%s\">\n", p.IdeaHandoff.BriefPath))
			b.WriteString(p.IdeaHandoff.BriefMarkdown)
			b.WriteString("\n</idea-handoff-brief>\n")
		}
	}

	return b.String()
}

// buildFinalizationFeedback formats multi-scenario finalization output into a
// readable prompt block for fixup/follow-up runs.
func buildFinalizationFeedback(finalization *Finalization) string {
	if finalization == nil {
		return ""
	}
	var b strings.Builder
	if strings.TrimSpace(finalization.AggregateSummary) != "" {
		b.WriteString(finalization.AggregateSummary)
	}
	for _, warning := range finalization.Warnings {
		if b.Len() > 0 {
			b.WriteString("\n")
		}
		if warning.ScenarioName != "" {
			b.WriteString(fmt.Sprintf("- warning [%s] %s: %s", warning.Code, warning.ScenarioName, warning.Message))
		} else {
			b.WriteString(fmt.Sprintf("- warning [%s]: %s", warning.Code, warning.Message))
		}
	}
	for _, scenario := range finalization.Scenarios {
		if scenario.Restart.Status != "" && scenario.Restart.Status != FinalizationStatusCompleted {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("- %s restart: %s", scenario.ScenarioName, scenario.Restart.LastError))
		}
		if scenario.Health.Status != "" && scenario.Health.Status != FinalizationStatusCompleted {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("- %s health: %s", scenario.ScenarioName, scenario.Health.Details))
		}
		if scenario.Review.Result == nil {
			if scenario.Review.SkipReason == "" {
				continue
			}
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("- %s review: %s", scenario.ScenarioName, scenario.Review.SkipReason))
			continue
		}
		if strings.TrimSpace(scenario.Review.Result.Summary) != "" {
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			b.WriteString(fmt.Sprintf("- %s review summary: %s", scenario.ScenarioName, scenario.Review.Result.Summary))
		}
		for _, dim := range scenario.Review.Result.Dimensions {
			if dim.Status == "green" || dim.Status == "skipped" {
				continue
			}
			if b.Len() > 0 {
				b.WriteString("\n")
			}
			if dim.Details != "" {
				b.WriteString(fmt.Sprintf("- %s %s (%s): %s", scenario.ScenarioName, dim.Name, dim.Status, dim.Details))
			} else {
				b.WriteString(fmt.Sprintf("- %s %s (%s)", scenario.ScenarioName, dim.Name, dim.Status))
			}
		}
	}
	return b.String()
}

// ProcessPreflight evaluates whether a backlog item is ready for processing.
func (s *Service) ProcessPreflight(_ context.Context, backlogKind, backlogName string) (ProcessPreflight, error) {
	item, err := s.loadBacklogItem(backlogKind, backlogName)
	if err != nil {
		return ProcessPreflight{}, err
	}
	return s.processPreflightForItem(item, true), nil
}

func (s *Service) processPreflightForItem(item backlogItem, checkQueueable bool) ProcessPreflight {
	targetScenarioID, archivedRevival := resolveTargetScenario(item)
	targetScenarioExists := false
	if strings.TrimSpace(targetScenarioID) != "" {
		targetScenarioExists = scenarioExists(filepath.Join(s.scenariosRootDir(), targetScenarioID))
	}

	preflight := ProcessPreflight{
		BacklogKind:              strings.TrimSpace(item.Kind),
		BacklogName:              strings.TrimSpace(item.Name),
		Ready:                    true,
		ArchivedRevival:          archivedRevival,
		ResolvedTargetScenarioID: targetScenarioID,
		TargetScenarioExists:     targetScenarioExists,
		SuggestedOperation:       "generator",
		SuggestedSteerProfileID:  "rapid-mvp",
	}
	if targetScenarioExists {
		preflight.SuggestedOperation = "improver"
		preflight.SuggestedSteerProfileID = "production-ready"
	}

	if checkQueueable && !isQueueableStatus(item.Kind, item.Status) {
		preflight.BlockingReasons = append(preflight.BlockingReasons, fmt.Sprintf("backlog item cannot be queued from current status: %s", item.Status))
	}

	// Check workshop readiness instead of clarify questions.
	itemDir := s.itemDir(item.Kind, item.Name)
	deliverablePath := deliverableForKind(item.Kind)
	if !workshop.HasPlanByName(itemDir, deliverablePath) {
		preflight.BlockingReasons = append(preflight.BlockingReasons, missingDeliverableReason(item.Kind, deliverablePath))
	}
	rounds, _ := workshop.LoadRounds(itemDir)
	if len(rounds) > 0 {
		latest := rounds[len(rounds)-1]
		rawScores := make(map[string]int, len(workshop.ReadinessDimensions))
		for _, dim := range workshop.ReadinessDimensions {
			if v, ok := latest.Readiness[dim]; ok {
				rawScores[dim] = v
			}
		}
		effective := workshop.ComputeEffectiveScores(rawScores, len(rounds), item.Kind)
		for _, dim := range workshop.ReadinessDimensions {
			if effective[dim] < 3 {
				preflight.BlockingReasons = append(preflight.BlockingReasons, fmt.Sprintf("readiness dimension %q is %d/3 — needs more workshop refinement", dim, effective[dim]))
			}
		}
	} else if workshop.HasPlanByName(itemDir, deliverablePath) {
		// Primary deliverable exists but no workshop rounds — allow execution
		// (manually created artifact).
	} else {
		preflight.BlockingReasons = append(preflight.BlockingReasons, "no workshop rounds completed — run workshop or initialize first")
	}

	preflight.Ready = len(preflight.BlockingReasons) == 0
	return preflight
}

// Old clarify-based blocking question types and loading have been removed.
// The execution preflight now uses workshop readiness from backlog.LoadWorkshopRounds
// and backlog.ComputeEffectiveScores instead.

func resolveTargetScenario(item backlogItem) (string, bool) {
	source := strings.TrimSpace(item.SourceScenarioName)
	if source != "" {
		return source, true
	}
	return strings.TrimSpace(item.Name), strings.EqualFold(strings.TrimSpace(item.Status), backlogStatusArchived)
}

func (s *Service) buildIdeaHandoffPackage(item backlogItem, itemDir string, preflight ProcessPreflight) (*handoff.Package, error) {
	if strings.TrimSpace(item.Kind) != "idea" {
		return nil, nil
	}
	targetScenarioID, _ := resolveTargetScenario(item)
	return handoff.BuildIdeaPackage(handoff.BuildRequest{
		BacklogKind:             item.Kind,
		BacklogName:             item.Name,
		BacklogTitle:            item.Title,
		BacklogDescription:      item.Description,
		ItemFolder:              itemDir,
		DeliverableFileName:     deliverableForKind(item.Kind),
		TargetScenario:          targetScenarioID,
		Operation:               preflight.SuggestedOperation,
		SuggestedSteerProfileID: preflight.SuggestedSteerProfileID,
		AcceptanceAllow:         item.AcceptanceAllow,
		AcceptanceDeny:          item.AcceptanceDeny,
	})
}

func hasNonForceableExecutionReasons(reasons []string) bool {
	for _, reason := range reasons {
		normalized := strings.ToLower(strings.TrimSpace(reason))
		if normalized == "" {
			continue
		}
		if strings.Contains(normalized, "workshop decision") || strings.Contains(normalized, "pending decision") {
			continue
		}
		return true
	}
	return false
}

func scenarioExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func promptRevision(prompt string) string {
	sum := sha256.Sum256([]byte(prompt))
	return "sha256:" + hex.EncodeToString(sum[:8])
}
