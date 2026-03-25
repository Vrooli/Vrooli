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

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/pathutil"
	"swarm-manager/internal/promptmanager"
	"swarm-manager/internal/storage"
	"swarm-manager/internal/workshop"
)

var (
	errNotFound       = errors.New("execution not found")
	errSessionExpired = errors.New("agent session expired")
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

// ServiceConfig configures execution service dependencies.
type ServiceConfig struct {
	RootDir      string
	StorePath    string
	PolicyPath   string
	AgentService agentSpawner
	PromptClient promptmanager.Client
	Archiver     Archiver
	ReviewClient ReviewClient
}

// Service owns execution lifecycle logic.
type Service struct {
	rootDir      string
	store        Store
	policyStore  *PolicyStore
	agentService agentSpawner
	promptClient promptmanager.Client
	archiver     Archiver
	reviewClient ReviewClient
	inspector  runInspector
	stopper    runStopper
	continuer  runContinuer
	mu         sync.Mutex
}

type promptSelection struct {
	SkillID   string
	Variables map[string]string
	Prompt    string
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
	service := &Service{
		rootDir:      rootDir,
		store:        NewStore(cfg.StorePath),
		policyStore:  newPolicyStore(cfg.PolicyPath),
		agentService: cfg.AgentService,
		promptClient: pc,
		archiver:     cfg.Archiver,
		reviewClient: cfg.ReviewClient,
	}
	if inspector, ok := cfg.AgentService.(runInspector); ok {
		service.inspector = inspector
	}
	if continuer, ok := cfg.AgentService.(runContinuer); ok {
		service.continuer = continuer
	}
	if stopper, ok := cfg.AgentService.(runStopper); ok {
		service.stopper = stopper
	}
	return service
}

// QueueBacklog creates an execution record and optionally starts it.
func (s *Service) QueueBacklog(ctx context.Context, req CreateRequest) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy, err := s.policyStore.Load()
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
		return Record{}, fmt.Errorf("mode must be manual, scheduled, or yolo")
	}
	if err := validateModeDelayInputs(mode, req.DelaySeconds); err != nil {
		return Record{}, err
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

	records, err := s.store.Load()
	if err != nil {
		return Record{}, err
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

	delaySeconds := req.DelaySeconds
	if mode == ModeScheduled && delaySeconds == 0 {
		delaySeconds = policy.DefaultDelaySeconds
	}
	if mode == ModeScheduled && delaySeconds <= 0 {
		return Record{}, fmt.Errorf("scheduled mode requires delay_seconds > 0 (or policy default > 0)")
	}
	scheduledAt, status := plannedSchedule(mode, delaySeconds)
	record.Status = status
	record.ScheduledAt = scheduledAt

	if err := s.updateBacklogStatus(item, "queued"); err != nil {
		return Record{}, err
	}

	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}

	if mode == ModeYOLO {
		started, startErr := s.startLocked(ctx, record.ExecutionID)
		if startErr == nil {
			return started, nil
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

	// Fetch spec-sync prompt from prompt-manager
	skillID := "spec-sync"
	vars := map[string]string{
		"TARGET": ac.ScenarioName,
	}
	prompt, promptErr := s.promptClient.ReadSkill(ctx, skillID, vars, false)
	if promptErr != nil {
		log.Printf("[execution] spec-sync prompt fetch failed: %v", promptErr)
		prompt = "Read the implementation code in this scenario and update all spec artifacts (PRD.md, requirements/, README.md, docs/) to match the actual behavior."
	}
	record.PromptTrace = &PromptTrace{
		SkillID:        skillID,
		Purpose:        "spec-sync",
		Variables:      vars,
		Prompt:         prompt,
		PromptRevision: promptRevision(prompt),
		UsedFallback:   promptErr != nil,
		CapturedAt:     now,
	}

	// Spawn agent targeting the scenario directory
	runResult, err := s.agentService.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind:        "spec-sync",
		Name:        ac.ScenarioName,
		Title:       "Spec sync: " + ac.ScenarioName,
		Description: prompt,
		Prompt:      prompt,
		ScopePath:   ac.ScenarioPath,
		ProjectRoot: ".",
		CreatedBy:   "swarm-manager",
		Purpose:     "spec-sync",
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

	return record, nil
}

// Policy returns current execution policy.
func (s *Service) Policy(_ context.Context) (Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.policyStore.Load()
}

// UpdatePolicy persists execution policy.
func (s *Service) UpdatePolicy(_ context.Context, policy Policy) (Policy, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := validatePolicyInputs(policy); err != nil {
		return Policy{}, err
	}
	normalized := normalizePolicy(policy)
	if err := s.policyStore.Save(normalized); err != nil {
		return Policy{}, err
	}
	return normalized, nil
}

// Start starts a pending/scheduled/failed execution now.
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

	selection, promptErr := s.fetchProcessingPrompt(ctx, item, record.Operation)
	prompt := selection.Prompt
	if promptErr != nil {
		log.Printf("[execution] prompt fetch failed: %v", promptErr)
		prompt = "Use the backlog item folder as context and complete the requested work."
	}
	record.PromptTrace = &PromptTrace{
		SkillID:        selection.SkillID,
		Purpose:        "process",
		Variables:      selection.Variables,
		Prompt:         prompt,
		PromptRevision: promptRevision(prompt),
		UsedFallback:   promptErr != nil,
		CapturedAt:     nowRFC3339(),
	}

	scopePath := s.itemDir(item.Kind, item.Name)
	if item.Scope != "" {
		scopePath = item.Scope
		if !filepath.IsAbs(scopePath) {
			scopePath = filepath.Join(s.rootDir, scopePath)
		}
	}

	runResult, err := s.agentService.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind:            item.Kind,
		Name:            item.Name,
		Title:           buildProcessingTitle(item),
		Description:     prompt,
		Prompt:          prompt,
		ScopePath:       scopePath,
		ProjectRoot:     ".",
		CreatedBy:       record.StartedBy,
		Purpose:         "process",
		AcceptanceAllow: item.AcceptanceAllow,
		AcceptanceDeny:  item.AcceptanceDeny,
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
	record.ScheduledAt = ""
	records[idx] = record
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
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

	switch record.Status {
	case StatusPending, StatusScheduled:
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
		return record, nil
	default:
		return Record{}, fmt.Errorf("only pending/scheduled/starting/running/needs_review/validating/needs_fixup executions can be canceled")
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

// shouldTriggerReview returns true if the completed execution should undergo review.
func (s *Service) shouldTriggerReview(item backlogItem, record Record) bool {
	if s.reviewClient == nil {
		return false
	}
	if strings.TrimSpace(item.Scope) == "" {
		return false
	}
	if !strings.HasPrefix(item.Scope, "scenarios/") {
		return false
	}
	if record.ArchiveContext != nil {
		return false
	}
	return true
}

func scenarioNameFromScope(scope string) string {
	// "scenarios/web-console" -> "web-console"
	// "scenarios/web-console/api" -> "web-console"
	parts := strings.SplitN(strings.TrimPrefix(scope, "scenarios/"), "/", 2)
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
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
	s.mu.Lock()
	defer s.mu.Unlock()
	_ = s.refreshRunningLocked(ctx)
	records, idx, err := s.loadRecordLocked(executionID)
	if err != nil {
		return Record{}, err
	}
	return records[idx], nil
}

// List returns executions ordered by created_at descending.
func (s *Service) List(ctx context.Context, filters ListFilters) ([]Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	_ = s.refreshRunningLocked(ctx)
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

// ProcessScheduledStarts starts due scheduled executions.
func (s *Service) ProcessScheduledStarts(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	records, err := s.store.Load()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	started := false
	for i := range records {
		record := records[i]
		if record.Status != StatusScheduled {
			continue
		}
		dueAt, err := time.Parse(time.RFC3339, strings.TrimSpace(record.ScheduledAt))
		if err != nil || dueAt.After(now) {
			continue
		}
		if _, err := s.startLocked(ctx, record.ExecutionID); err == nil {
			started = true
		}
	}
	if started {
		return nil
	}
	return nil
}

func (s *Service) refreshRunningLocked(ctx context.Context) error {
	records, err := s.store.Load()
	if err != nil {
		return err
	}

	changed := false

	// Handle validating records (poll review jobs).
	for i := range records {
		record := &records[i]
		if record.Status != StatusValidating || strings.TrimSpace(record.ReviewJobID) == "" {
			continue
		}
		if s.reviewClient == nil {
			continue
		}
		result, done, pollErr := s.reviewClient.PollReview(ctx, record.ReviewJobID)
		if pollErr != nil {
			log.Printf("[execution] review poll error for %s: %v", record.ExecutionID, pollErr)
			continue
		}
		if !done {
			continue
		}
		record.ReviewResult = result
		record.UpdatedAt = nowRFC3339()

		if result.Classification == "ready" || result.Classification == "ready_with_notes" {
			record.Status = StatusCompleted
			record.FinishedAt = nowRFC3339()
			if item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr == nil {
				_ = s.updateBacklogStatus(item, "completed")
			}
		} else {
			policy, _ := s.policyStore.Load()
			if policy.AutoFixup && record.FixupAttempt < policy.MaxFixupAttempts {
				if item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr == nil {
					s.spawnFixupRun(ctx, record, item)
				}
			} else {
				record.Status = StatusNeedsFixup
				record.FinishedAt = nowRFC3339()
				if item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr == nil {
					_ = s.updateBacklogStatus(item, "failed")
				}
			}
		}
		changed = true
	}

	// Handle running/starting/needs_review records.
	if s.inspector == nil {
		if changed {
			return s.store.Save(records)
		}
		return nil
	}

	for i := range records {
		record := &records[i]
		if (record.Status != StatusStarting && record.Status != StatusRunning && record.Status != StatusNeedsReview) || strings.TrimSpace(record.RunID) == "" {
			continue
		}
		runState, err := s.inspector.GetRunState(ctx, record.RunID)
		if err != nil {
			continue
		}
		nextStatus, reason := mapRunStatus(runState.Status, runState.ErrorMsg)
		if nextStatus == record.Status {
			continue
		}
		record.Status = nextStatus
		record.FailureReason = reason
		record.UpdatedAt = nowRFC3339()
		// Only set FinishedAt for terminal statuses
		if nextStatus == StatusCompleted || nextStatus == StatusFailed || nextStatus == StatusCanceled {
			if strings.TrimSpace(runState.FinishedAt) != "" {
				record.FinishedAt = runState.FinishedAt
			} else {
				record.FinishedAt = nowRFC3339()
			}
			// Post-completion hook: archive scenario after successful spec-sync
			if record.ArchiveContext != nil {
				if nextStatus == StatusCompleted {
					s.handleSpecSyncComplete(ctx, record)
				}
				// For spec-sync failures, leave status as failed for UI recovery
			} else if item, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr == nil {
				if nextStatus == StatusCompleted {
					// Check if this execution should trigger a review
					if s.shouldTriggerReview(item, *record) {
						scenarioName := scenarioNameFromScope(item.Scope)
						jobID, triggerErr := s.reviewClient.TriggerReview(ctx, ReviewRequest{
							ScenarioName:  scenarioName,
							ExpectedPaths: item.AcceptanceAllow,
						})
						if triggerErr != nil {
							log.Printf("[execution] review trigger failed for %s: %v", record.ExecutionID, triggerErr)
							// Fall through to normal completion
							_ = s.updateBacklogStatus(item, "completed")
						} else {
							record.Status = StatusValidating
							record.ReviewJobID = jobID
							record.FinishedAt = ""
							// Don't update backlog — not terminal yet
						}
					} else {
						_ = s.updateBacklogStatus(item, "completed")
					}
				} else if nextStatus == StatusFailed {
					_ = s.updateBacklogStatus(item, "failed")
				} else if nextStatus == StatusCanceled {
					_ = s.updateBacklogStatus(item, restoreBacklogStatus(*record))
				}
			}
		}
		changed = true
	}

	if changed {
		return s.store.Save(records)
	}
	return nil
}

// handleSpecSyncComplete performs the archive after a successful spec-sync run.
func (s *Service) handleSpecSyncComplete(ctx context.Context, record *Record) {
	if s.archiver == nil {
		log.Printf("[execution] spec-sync completed but no archiver configured for %s", record.BacklogName)
		record.FailureReason = "archiver not configured"
		record.Status = StatusFailed
		return
	}

	ac := record.ArchiveContext
	if _, err := os.Stat(ac.ScenarioPath); err != nil {
		log.Printf("[execution] spec-sync completed but scenario dir missing: %s", ac.ScenarioPath)
		record.FailureReason = "scenario directory no longer exists"
		record.Status = StatusFailed
		return
	}

	if err := s.archiver.ArchiveScenario(ctx, *ac); err != nil {
		log.Printf("[execution] post-spec-sync archive failed for %s: %v", ac.ScenarioName, err)
		record.FailureReason = "archive failed after spec-sync: " + err.Error()
		record.Status = StatusFailed
		return
	}

	// Delete the scenario directory after successful archive
	if err := os.RemoveAll(ac.ScenarioPath); err != nil {
		log.Printf("[execution] post-archive scenario deletion failed for %s: %v", ac.ScenarioName, err)
		record.FailureReason = "scenario deletion failed after archive: " + err.Error()
		record.Status = StatusFailed
		return
	}

	log.Printf("[execution] spec-sync-archive completed for %s", ac.ScenarioName)
}

func (s *Service) spawnFixupRun(ctx context.Context, record *Record, item backlogItem) {
	now := nowRFC3339()

	// Build fixup prompt from review result
	var reviewSummary string
	if record.ReviewResult != nil {
		reviewSummary = record.ReviewResult.Summary
		for _, dim := range record.ReviewResult.Dimensions {
			if dim.Status != "green" {
				reviewSummary += fmt.Sprintf("\n- %s (%s): %s", dim.Name, dim.Status, dim.Details)
			}
		}
	}

	prompt := fmt.Sprintf(
		"Post-execution review found issues. Fix the following problems:\n\n%s\n\n"+
			"The original plan is in plan.md. Focus only on fixing the identified issues.",
		reviewSummary,
	)

	fixupRecord := Record{
		ExecutionID:       idgen.Generate(),
		BacklogKind:       record.BacklogKind,
		BacklogName:       record.BacklogName,
		PreviousStatus:    "queued",
		Status:            StatusPending,
		Mode:              ModeYOLO,
		StartedBy:         "swarm-manager:fixup",
		Operation:         "fixup",
		ParentExecutionID: record.ExecutionID,
		FixupAttempt:      record.FixupAttempt + 1,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	records, loadErr := s.store.Load()
	if loadErr != nil {
		log.Printf("[execution] failed to load records for fixup: %v", loadErr)
		return
	}
	records = append(records, fixupRecord)

	scopePath := item.Scope
	if scopePath != "" && !filepath.IsAbs(scopePath) {
		scopePath = filepath.Join(s.rootDir, scopePath)
	}
	if scopePath == "" {
		scopePath = s.itemDir(item.Kind, item.Name)
	}

	runResult, err := s.agentService.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
		Kind:            item.Kind,
		Name:            item.Name,
		Title:           fmt.Sprintf("Fix-up: %s/%s (attempt %d)", item.Kind, item.Name, fixupRecord.FixupAttempt),
		Description:     prompt,
		Prompt:          prompt,
		ScopePath:       scopePath,
		ProjectRoot:     ".",
		CreatedBy:       "swarm-manager:fixup",
		Purpose:         "fixup",
		AcceptanceAllow: item.AcceptanceAllow,
		AcceptanceDeny:  item.AcceptanceDeny,
	})
	if err != nil {
		log.Printf("[execution] failed to spawn fixup run: %v", err)
		// Update fixup record to failed
		for i := range records {
			if records[i].ExecutionID == fixupRecord.ExecutionID {
				records[i].Status = StatusFailed
				records[i].FailureReason = fmt.Sprintf("spawn failed: %v", err)
				records[i].FinishedAt = now
				break
			}
		}
		_ = s.store.Save(records)
		return
	}

	// Update fixup record with run info
	for i := range records {
		if records[i].ExecutionID == fixupRecord.ExecutionID {
			records[i].TaskID = runResult.TaskID
			records[i].RunID = runResult.RunID
			records[i].Status = StatusStarting
			records[i].StartedAt = now
			break
		}
	}

	// Mark original record as failed with reference to fixup
	record.Status = StatusFailed
	record.FailureReason = fmt.Sprintf("fix-up spawned: %s", fixupRecord.ExecutionID)
	record.FinishedAt = now
	record.UpdatedAt = now

	_ = s.store.Save(records)
}

// FollowUp creates a follow-up execution from a completed/failed/needs_fixup execution.
func (s *Service) FollowUp(ctx context.Context, req FollowUpRequest) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.agentService == nil || !s.agentService.IsEnabled() {
		return Record{}, agentmanager.ErrNotAvailable
	}

	records, idx, err := s.loadRecordLocked(req.ExecutionID)
	if err != nil {
		return Record{}, err
	}
	parent := &records[idx]

	// Only allow follow-up from terminal or needs_fixup states.
	switch parent.Status {
	case StatusCompleted, StatusFailed, StatusNeedsFixup:
		// OK
	default:
		return Record{}, fmt.Errorf("cannot follow up execution in %q state", parent.Status)
	}

	// Build the follow-up prompt/message.
	message := s.buildFollowUpMessage(parent, req)

	now := nowRFC3339()
	followUpRecord := Record{
		ExecutionID:       idgen.Generate(),
		BacklogKind:       parent.BacklogKind,
		BacklogName:       parent.BacklogName,
		PreviousStatus:    string(parent.Status),
		Status:            StatusPending,
		Mode:              ModeYOLO,
		StartedBy:         "swarm-manager:follow-up",
		Operation:         req.FollowUpType,
		ParentExecutionID: parent.ExecutionID,
		CreatedAt:         now,
		UpdatedAt:         now,
	}
	if req.FollowUpType == "fixup" {
		followUpRecord.FixupAttempt = parent.FixupAttempt + 1
	}

	if req.RunMode == "continue" && strings.TrimSpace(parent.RunID) != "" {
		// Continue existing agent-manager session.
		if s.continuer == nil {
			return Record{}, fmt.Errorf("cannot follow up: run continuation not available")
		}
		if err := s.continuer.ContinueRun(ctx, parent.RunID, message); err != nil {
			if strings.Contains(err.Error(), "session_expired") || strings.Contains(err.Error(), "continuation_not_supported") {
				return Record{}, fmt.Errorf("%w: %v", errSessionExpired, err)
			}
			return Record{}, fmt.Errorf("continue run failed: %w", err)
		}
		followUpRecord.RunID = parent.RunID
		followUpRecord.TaskID = parent.TaskID
		followUpRecord.Status = StatusRunning
		followUpRecord.StartedAt = now
	} else {
		// Spawn a fresh run.
		item, loadErr := s.loadBacklogItemByRecord(parent)
		if loadErr != nil {
			return Record{}, fmt.Errorf("cannot follow up: %w", loadErr)
		}

		scopePath := item.Scope
		if scopePath != "" && !filepath.IsAbs(scopePath) {
			scopePath = filepath.Join(s.rootDir, scopePath)
		}
		if scopePath == "" {
			scopePath = s.itemDir(item.Kind, item.Name)
		}

		runResult, spawnErr := s.agentService.SpawnBacklog(ctx, agentmanager.BacklogSpawnRequest{
			Kind:            item.Kind,
			Name:            item.Name,
			Title:           fmt.Sprintf("Follow-up: %s/%s", item.Kind, item.Name),
			Description:     message,
			Prompt:          message,
			ScopePath:       scopePath,
			ProjectRoot:     ".",
			CreatedBy:       "swarm-manager:follow-up",
			Purpose:         req.FollowUpType,
			AcceptanceAllow: item.AcceptanceAllow,
			AcceptanceDeny:  item.AcceptanceDeny,
		})
		if spawnErr != nil {
			return Record{}, fmt.Errorf("spawn follow-up failed: %w", spawnErr)
		}
		followUpRecord.TaskID = runResult.TaskID
		followUpRecord.RunID = runResult.RunID
		followUpRecord.Status = StatusStarting
		followUpRecord.StartedAt = now
	}

	records = append(records, followUpRecord)
	if err := s.store.Save(records); err != nil {
		return Record{}, fmt.Errorf("failed to save follow-up record: %w", err)
	}

	return followUpRecord, nil
}

func (s *Service) buildFollowUpMessage(parent *Record, req FollowUpRequest) string {
	switch req.FollowUpType {
	case "fixup":
		var reviewSummary string
		if parent.ReviewResult != nil {
			reviewSummary = parent.ReviewResult.Summary
			for _, dim := range parent.ReviewResult.Dimensions {
				if dim.Status != "green" {
					reviewSummary += fmt.Sprintf("\n- %s (%s): %s", dim.Name, dim.Status, dim.Details)
				}
			}
		}
		prompt := fmt.Sprintf(
			"Post-execution review found issues. Fix the following problems:\n\n%s\n\n"+
				"The original plan is in plan.md. Focus only on fixing the identified issues.",
			reviewSummary,
		)
		if strings.TrimSpace(req.Context) != "" {
			prompt += "\n\nAdditional context:\n" + req.Context
		}
		return prompt

	case "followup":
		prompt := "Continue working on this backlog item. The original plan is in plan.md."
		if strings.TrimSpace(req.Context) != "" {
			prompt += "\n\nAdditional context:\n" + req.Context
		}
		return prompt

	default: // custom
		if strings.TrimSpace(req.Context) != "" {
			return req.Context
		}
		return "Continue working on this backlog item."
	}
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
	case ModeScheduled:
		return ModeScheduled
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

func validateModeDelayInputs(mode Mode, delaySeconds int64) error {
	if delaySeconds < 0 {
		return fmt.Errorf("delay_seconds must be >= 0")
	}
	if mode != ModeScheduled && delaySeconds > 0 {
		return fmt.Errorf("delay_seconds is only supported for scheduled mode")
	}
	return nil
}

func validatePolicyInputs(policy Policy) error {
	mode := normalizeMode(policy.DefaultMode)
	if mode == "" {
		return fmt.Errorf("default_mode must be manual, scheduled, or yolo")
	}
	if policy.DefaultDelaySeconds < 0 {
		return fmt.Errorf("default_delay_seconds must be >= 0")
	}
	if mode == ModeScheduled && policy.DefaultDelaySeconds <= 0 {
		return fmt.Errorf("scheduled default_mode requires default_delay_seconds > 0")
	}
	return nil
}

func plannedSchedule(mode Mode, delaySeconds int64) (string, Status) {
	switch mode {
	case ModeScheduled:
		delay := time.Duration(delaySeconds) * time.Second
		if delay < 0 {
			delay = 0
		}
		return time.Now().UTC().Add(delay).Format(time.RFC3339), StatusScheduled
	case ModeManual:
		return "", StatusPending
	default:
		return "", StatusPending
	}
}

func mapRunStatus(status, errorMsg string) (Status, string) {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pending", "starting":
		return StatusStarting, ""
	case "running":
		return StatusRunning, ""
	case "needs_review":
		return StatusNeedsReview, ""
	case "complete":
		return StatusCompleted, ""
	case "failed":
		reason := strings.TrimSpace(errorMsg)
		if reason == "" {
			reason = "agent-manager run failed"
		}
		return StatusFailed, reason
	case "cancelled":
		return StatusCanceled, ""
	default:
		return StatusRunning, ""
	}
}

func matchesFilters(record Record, filters ListFilters) bool {
	if strings.TrimSpace(filters.Status) != "" && string(record.Status) != strings.TrimSpace(filters.Status) {
		return false
	}
	if strings.TrimSpace(filters.Mode) != "" && string(record.Mode) != strings.TrimSpace(filters.Mode) {
		return false
	}
	if strings.TrimSpace(filters.BacklogKind) != "" && record.BacklogKind != strings.TrimSpace(filters.BacklogKind) {
		return false
	}
	if strings.TrimSpace(filters.BacklogName) != "" && record.BacklogName != strings.TrimSpace(filters.BacklogName) {
		return false
	}
	if strings.TrimSpace(filters.StartedBy) != "" && record.StartedBy != strings.TrimSpace(filters.StartedBy) {
		return false
	}
	if strings.TrimSpace(filters.CreatedFrom) != "" {
		from, err := time.Parse(time.RFC3339, strings.TrimSpace(filters.CreatedFrom))
		if err == nil {
			createdAt, createdErr := time.Parse(time.RFC3339, strings.TrimSpace(record.CreatedAt))
			if createdErr == nil && createdAt.Before(from) {
				return false
			}
		}
	}
	if strings.TrimSpace(filters.CreatedTo) != "" {
		to, err := time.Parse(time.RFC3339, strings.TrimSpace(filters.CreatedTo))
		if err == nil {
			createdAt, createdErr := time.Parse(time.RFC3339, strings.TrimSpace(record.CreatedAt))
			if createdErr == nil && createdAt.After(to) {
				return false
			}
		}
	}
	return true
}

func restoreBacklogStatus(record Record) string {
	previous := strings.ToLower(strings.TrimSpace(record.PreviousStatus))
	switch previous {
	case "archived":
		if strings.TrimSpace(record.BacklogKind) == "idea" {
			return "archived"
		}
	case "backlog", "researching", "ready":
		return previous
	}
	return "ready"
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
	ResearchTarget     string   `json:"research_target,omitempty"`
	SourceScenarioName string   `json:"sourceScenarioName,omitempty"`
	Scope              string   `json:"scope,omitempty"`
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
	if strings.TrimSpace(item.ResearchTarget) != "" && item.Kind == "research" {
		merged["research_target"] = item.ResearchTarget
	} else {
		delete(merged, "research_target")
	}
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
	case "backlog", "researching", "ready":
		return true
	case "archived":
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

// processingSkillIDs maps backlog kind to prompt-manager skill IDs.
var processingSkillIDs = map[string]string{
	"idea":     "swarm-manager-process-idea",
	"fix":      "swarm-manager-process-fix",
	"execute":  "swarm-manager-process-execute",
	"research": "swarm-manager-process-execute",
	"chore":    "swarm-manager-process-execute",
}

// fetchProcessingPrompt loads a processing prompt from prompt-manager.
func (s *Service) fetchProcessingPrompt(ctx context.Context, item backlogItem, operation string) (promptSelection, error) {
	skillID := processingSkillIDs[item.Kind]
	if skillID == "" {
		skillID = "swarm-manager-process-execute"
	}
	targetScenarioID, archivedRevival := resolveTargetScenario(item)

	itemFolder := s.itemDir(item.Kind, item.Name)
	vars := map[string]string{
		"ITEM_NAME":            item.Name,
		"ITEM_TITLE":           item.Title,
		"ITEM_DESCRIPTION":     item.Description,
		"ITEM_KIND":            item.Kind,
		"ITEM_STATUS":          item.Status,
		"ITEM_PRIORITY":        fmt.Sprintf("%d", item.Priority),
		"ITEM_TAGS":            strings.Join(item.Tags, ", "),
		"ITEM_FOLDER":          itemFolder,
		"TARGET_SCENARIO_ID":   targetScenarioID,
		"ARCHIVED_REVIVAL":     fmt.Sprintf("%t", archivedRevival),
		"SOURCE_SCENARIO_NAME": strings.TrimSpace(item.SourceScenarioName),
		"PLAN_DRAFT":           workshop.LoadPlanContent(itemFolder),
	}

	prompt, err := s.promptClient.ReadSkill(ctx, skillID, vars, false)
	if err != nil {
		return promptSelection{
			SkillID:   skillID,
			Variables: vars,
		}, err
	}

	if strings.TrimSpace(operation) == "improver" {
		prompt = prompt + "\n\nOperation hint: improver (focus on improving an existing scenario).\n"
	}
	return promptSelection{
		SkillID:   skillID,
		Variables: vars,
		Prompt:    prompt,
	}, nil
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

	if strings.EqualFold(strings.TrimSpace(item.Kind), "research") {
		preflight.BlockingReasons = append(preflight.BlockingReasons, "research items must be converted before processing")
	}

	// Check workshop readiness instead of clarify questions.
	itemDir := s.itemDir(item.Kind, item.Name)
	if !workshop.HasPlan(itemDir) {
		preflight.BlockingReasons = append(preflight.BlockingReasons, "no implementation plan (plan.md) exists — run workshop first")
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
	} else if workshop.HasPlan(itemDir) {
		// Plan exists but no workshop rounds — allow execution (manually created plan)
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
	return strings.TrimSpace(item.Name), strings.EqualFold(strings.TrimSpace(item.Status), "archived")
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
