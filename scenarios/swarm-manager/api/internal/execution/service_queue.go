package execution

import (
	"context"
	"errors"
	"log/slog"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/idgen"
)

// QueueBacklog creates an execution record and optionally starts it.
func (s *Service) QueueBacklog(ctx context.Context, req CreateRequest) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	mode, item, preflight, err := s.validateAndLoadQueueRequest(req)
	if err != nil {
		return Record{}, err
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
		return Record{}, apierr.Conflict("circuit breaker tripped: %s (cooldown remaining: %s)", itemKey, remaining.Truncate(time.Second))
	}

	records, err := s.store.Load()
	if err != nil {
		return Record{}, err
	}

	// Drop pending records whose backlog item has disappeared — they can
	// never be started and otherwise consume the queue-depth budget.
	if filtered, pruned := pruneOrphanedPendingRecords(records, s.itemDir); pruned > 0 {
		if saveErr := s.store.Save(filtered); saveErr != nil {
			return Record{}, saveErr
		}
		records = filtered
	}

	if err := enforceQueueGovernance(records, gov, req.Force); err != nil {
		return Record{}, err
	}

	record := buildNewQueueRecord(ctx, req, item, mode, preflight)

	if err := s.updateBacklogStatus(item, backlogStatusQueued); err != nil {
		return Record{}, err
	}

	records = append(records, record)
	if err := s.store.Save(records); err != nil {
		return Record{}, err
	}
	s.logExecutionEvent(record, "")

	if mode == ModeYOLO {
		return s.startQueuedYOLO(ctx, record, item, itemKey)
	}

	s.dispatchStatusUpdate(record)
	return record, nil
}

// validateAndLoadQueueRequest validates the create request, resolves the effective
// mode, loads the backlog item, and runs preflight checks. Returns the resolved
// mode, loaded item, preflight result, and any validation error.
func (s *Service) validateAndLoadQueueRequest(req CreateRequest) (Mode, backlogItem, ProcessPreflight, error) {
	if strings.TrimSpace(req.BacklogKind) == "" {
		return "", backlogItem{}, ProcessPreflight{}, apierr.BadRequest("backlog_kind is required")
	}
	if strings.TrimSpace(req.BacklogName) == "" {
		return "", backlogItem{}, ProcessPreflight{}, apierr.BadRequest("backlog_name is required")
	}

	mode := normalizeMode(req.Mode)
	if mode == "" {
		policy, err := s.policyProvider.LoadPolicy()
		if err != nil {
			return "", backlogItem{}, ProcessPreflight{}, err
		}
		mode = policy.DefaultMode
	}
	if mode == "" {
		return "", backlogItem{}, ProcessPreflight{}, apierr.BadRequest("mode must be manual or yolo")
	}

	item, err := s.loadBacklogItem(req.BacklogKind, req.BacklogName)
	if err != nil {
		return "", backlogItem{}, ProcessPreflight{}, apierr.NotFound("backlog item not found: %s/%s", req.BacklogKind, req.BacklogName)
	}
	isArchived := item.ArchivedAt != nil
	if !isQueueableStatus(item.Kind, item.Status) && !(isArchived && strings.ToLower(strings.TrimSpace(item.Kind)) == "idea") {
		return "", backlogItem{}, ProcessPreflight{}, apierr.BadRequest("backlog item cannot be queued from current status: %s", item.Status)
	}

	preflight := s.processPreflightForItem(item, true)
	if !preflight.Ready && (!req.Force || hasNonForceableExecutionReasons(preflight.BlockingReasons)) {
		return "", backlogItem{}, ProcessPreflight{}, apierr.BadRequest("process preflight failed: %s", strings.Join(allBlockingReasons(preflight), "; "))
	}

	return mode, item, preflight, nil
}

// buildNewQueueRecord constructs a pending Record from the validated request,
// resolved item, and mode. It resolves StartedBy from the context when not
// provided and defaults Operation to "generator".
func buildNewQueueRecord(ctx context.Context, req CreateRequest, item backlogItem, mode Mode, _ ProcessPreflight) Record {
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
		QueuedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}
	if record.StartedBy == "" {
		prov := identity.FromContext(ctx)
		if prov.IsAgent() {
			record.StartedBy = prov.FormatStartedBy()
		} else {
			record.StartedBy = "swarm-manager"
		}
	}
	if record.Operation == "" {
		record.Operation = "generator"
	}
	return record
}

// enforceQueueGovernance applies queue-depth and cost-cap limits before a
// record is admitted to the queue.
func enforceQueueGovernance(records []Record, gov GovernanceSettings, force bool) error {
	// Queue depth enforcement.
	if gov.MaxQueueDepth > 0 {
		queued := countQueuedExecutions(records)
		if queued >= gov.MaxQueueDepth {
			return apierr.Conflict("queue depth limit exceeded (%d/%d)", queued, gov.MaxQueueDepth)
		}
	}

	// Cost cap enforcement.
	if gov.ExecutionCostCapPerRun > 0 && gov.CostPerTurnEstimate > 0 {
		agentMaxTurns := gov.AgentMaxTurns
		if agentMaxTurns <= 0 {
			agentMaxTurns = 600
		}
		estimatedCost := gov.CostPerTurnEstimate * float64(agentMaxTurns)
		if estimatedCost > gov.ExecutionCostCapPerRun && !force {
			return apierr.Conflict("estimated cost $%.2f exceeds cap $%.2f (use force to override)", estimatedCost, gov.ExecutionCostCapPerRun)
		}
	}
	return nil
}

// startQueuedYOLO attempts an immediate start for a YOLO-mode record, leaving
// it pending at capacity and rolling back queue side-effects on other failures.
func (s *Service) startQueuedYOLO(ctx context.Context, record Record, item backlogItem, itemKey string) (Record, error) {
	started, startErr := s.startLocked(ctx, record.ExecutionID)
	if startErr == nil {
		return started, nil
	}

	// At capacity: leave as pending for the poller to drain later.
	if errors.Is(startErr, errAtCapacity) {
		slog.Info("at capacity, leaving as pending", "execution_id", record.ExecutionID, "item_key", itemKey)
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

// QueueSpecSyncArchive creates an execution that runs spec-sync, then archives on completion.
func (s *Service) QueueSpecSyncArchive(ctx context.Context, ac ArchiveContext) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if strings.TrimSpace(ac.ScenarioName) == "" {
		return Record{}, apierr.BadRequest("scenario_name is required")
	}
	if strings.TrimSpace(ac.ScenarioPath) == "" {
		return Record{}, apierr.BadRequest("scenario_path is required")
	}
	if _, err := os.Stat(ac.ScenarioPath); err != nil {
		return Record{}, apierr.BadRequest("scenario path does not exist: %s", ac.ScenarioPath)
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
		QueuedAt:       now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// The declared workflow owns the spec-sync agent work. Swarm retains the
	// archive capability and applies a matching typed terminal result explicitly.
	res, snapshot, err := s.startSpecSyncWorkflow(ctx, record)
	if err != nil {
		return Record{}, wrapAgentError(err)
	}
	if strings.TrimSpace(res.ExecutionID) == "" {
		return Record{}, apierr.BadGateway("scenario spec-sync workflow started but returned no execution id")
	}
	record.RunID = res.RunID
	record.TaskID = res.ExecutionID
	record.AgentWorkflowExecutionID = res.ExecutionID
	record.AgentWorkflowKey = snapshot.WorkflowKey
	record.AgentWorkflowDefinition = res.DefinitionDigest
	record.AgentWorkflowEntityVersion = snapshot.EntityVersion
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
