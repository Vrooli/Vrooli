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
	"swarm-manager/internal/dispatch"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
)

var (
	errNotFound       = apierr.ErrNotFound
	errSessionExpired = apierr.ErrSessionExpired
	errAtCapacity     = apierr.ErrAtCapacity
)

// Backlog status values used by execution to update backlog items.
// Defined locally to avoid a circular import with the backlog package.
const (
	backlogStatusQueued      = "queued"
	backlogStatusCompleted   = "completed"
	backlogStatusFailed      = "failed"
	backlogStatusBacklog     = "backlog"
	backlogStatusResearching = "researching"
	backlogStatusReady       = "ready"
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
	SelfScenarioName         string
	PolicyProvider           PolicyProvider
	GovernanceProvider       GovernanceProvider
	ReviewThresholdsProvider ReviewThresholdsProvider
	AgentService             AgentSpawner
	ScenarioLifecycle        ScenarioLifecycle
	ScenarioHealthChecker    ScenarioHealthChecker
	PromptClient             promptmanager.Client
	ExperimentClient         promptmanager.ExperimentClient
	Archiver                 Archiver
	ReviewClient             ReviewClient
	Finalization             FinalizationConfig
}

// Service owns execution lifecycle logic.
type Service struct {
	rootDir                  string
	selfScenarioName         string
	finalizationCfg          FinalizationConfig
	store                    Store
	policyProvider           PolicyProvider
	governanceProvider       GovernanceProvider
	reviewThresholdsProvider ReviewThresholdsProvider
	agentService             AgentSpawner
	promptClient             promptmanager.Client
	experimentClient         promptmanager.ExperimentClient
	archiver                 Archiver
	reviewClient             ReviewClient
	inspector                RunInspector
	differ                   RunDiffer
	stopper                  RunStopper
	continuer                RunContinuer
	scenarioLifecycle        ScenarioLifecycle
	scenarioHealth           ScenarioHealthChecker
	reviewService            ReviewServiceIntegration
	eventDispatcher          dispatch.NodeDispatcher
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
	selfName := strings.TrimSpace(cfg.SelfScenarioName)
	if selfName == "" {
		selfName = filepath.Base(rootDir)
	}

	service := &Service{
		rootDir:                  rootDir,
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
		scenarioLifecycle:        cfg.ScenarioLifecycle,
		scenarioHealth:           cfg.ScenarioHealthChecker,
		circuitBreaker:           NewCircuitBreaker(filepath.Join(rootDir, ".vrooli", "circuit-breaker.json")),
		processingFinalizations:  map[string]struct{}{},
	}
	if inspector, ok := cfg.AgentService.(RunInspector); ok {
		service.inspector = inspector
	}
	if differ, ok := cfg.AgentService.(RunDiffer); ok {
		service.differ = differ
	}
	if continuer, ok := cfg.AgentService.(RunContinuer); ok {
		service.continuer = continuer
	}
	if stopper, ok := cfg.AgentService.(RunStopper); ok {
		service.stopper = stopper
	}
	return service
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
	if errors.Is(err, agentmanager.ErrRequestFailed) {
		return apierr.BadGateway("agent-manager request failed; check agent-manager health/logs and retry")
	}
	return err
}
