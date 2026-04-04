package execution

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/promptcatalog"
)

// QueueBacklog creates an execution record and optionally starts it.
func (s *Service) QueueBacklog(ctx context.Context, req CreateRequest) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	policy, err := s.policyProvider.LoadPolicy()
	if err != nil {
		return Record{}, err
	}

	if strings.TrimSpace(req.BacklogKind) == "" {
		return Record{}, apierr.BadRequest("backlog_kind is required")
	}
	if strings.TrimSpace(req.BacklogName) == "" {
		return Record{}, apierr.BadRequest("backlog_name is required")
	}
	mode := normalizeMode(req.Mode)
	if mode == "" {
		mode = policy.DefaultMode
	}
	if mode == "" {
		return Record{}, apierr.BadRequest("mode must be manual or yolo")
	}

	item, err := s.loadBacklogItem(req.BacklogKind, req.BacklogName)
	if err != nil {
		return Record{}, apierr.NotFound("backlog item not found: %s/%s", req.BacklogKind, req.BacklogName)
	}
	if !isQueueableStatus(item.Kind, item.Status) {
		return Record{}, apierr.BadRequest("backlog item cannot be queued from current status: %s", item.Status)
	}
	preflight := s.processPreflightForItem(item, true)
	if !preflight.Ready && (!req.Force || hasNonForceableExecutionReasons(preflight.BlockingReasons)) {
		return Record{}, apierr.BadRequest("process preflight failed: %s", strings.Join(preflight.BlockingReasons, "; "))
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

	// Queue depth enforcement.
	if gov.MaxQueueDepth > 0 {
		queued := countQueuedExecutions(records)
		if queued >= gov.MaxQueueDepth {
			return Record{}, apierr.Conflict("queue depth limit exceeded (%d/%d)", queued, gov.MaxQueueDepth)
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
			return Record{}, apierr.Conflict("estimated cost $%.2f exceeds cap $%.2f (use force to override)", estimatedCost, gov.ExecutionCostCapPerRun)
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

	s.dispatchStatusUpdate(record)
	return record, nil
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

	if s.agentService == nil || !s.agentService.IsEnabled() {
		return Record{}, apierr.Unavailable("agent-manager is not available")
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
		slog.Warn("spec-sync prompt fetch failed, using fallback", "err", promptErr)
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
