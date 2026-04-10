package execution

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/workshop"
)

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
		return Record{}, apierr.BadRequest("cannot start canceled execution")
	}

	// Concurrency gate.
	if gov, govErr := s.governanceProvider.LoadGovernance(); govErr == nil {
		active := countActiveExecutions(records)
		if active >= gov.MaxConcurrentExecutions {
			return Record{}, apierr.Wrap(errAtCapacity, http.StatusConflict, "concurrency limit reached")
		}
	}

	if s.agentService == nil || !s.agentService.IsEnabled() {
		return Record{}, apierr.Unavailable("agent-manager is not available")
	}

	item, err := s.loadBacklogItem(record.BacklogKind, record.BacklogName)
	if err != nil {
		return Record{}, err
	}
	preflight := s.processPreflightForItem(item, false)
	if !preflight.Ready && (!record.Force || hasNonForceableExecutionReasons(preflight.BlockingReasons)) {
		return Record{}, apierr.BadRequest("process preflight failed: %s", strings.Join(preflight.BlockingReasons, "; "))
	}

	itemDir := s.itemDir(item.Kind, item.Name)
	deliverablePath := deliverableForKind(item.Kind)
	deliverableContent := workshop.LoadPlanContentByName(itemDir, deliverablePath)
	usedFallback := strings.TrimSpace(deliverableContent) == ""
	if usedFallback {
		slog.Warn("deliverable empty or missing", "path", deliverablePath, "dir", itemDir)
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
		SuggestedSkills:    item.SuggestedSkills,
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
		return Record{}, wrapAgentError(err)
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
			return Record{}, apierr.BadRequest("cancel is not supported by current agent service")
		}
		if strings.TrimSpace(record.RunID) == "" {
			return Record{}, apierr.BadRequest("execution has no run id")
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
		// Also mark finalization as failed so the UI stops showing the progress indicator.
		if record.Finalization != nil {
			record.Finalization.Status = FinalizationStatusFailed
			record.Finalization.Phase = FinalizationPhaseFailed
			record.Finalization.CompletedAt = nowRFC3339()
		}
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
		return Record{}, apierr.BadRequest("only pending/starting/running/needs_review/validating/needs_fixup executions can be canceled")
	}
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
		return Record{}, apierr.BadRequest("cannot trigger post-run checks for execution in %q status", record.Status)
	}

	if _, loadErr := s.loadBacklogItem(record.BacklogKind, record.BacklogName); loadErr != nil {
		return Record{}, apierr.NotFound("backlog item not found for post-run checks")
	}
	if !isFinalizationEligible(*record) {
		return Record{}, apierr.BadRequest("execution type %q does not support post-run checks", record.effectiveRunType())
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
		return Record{}, apierr.BadRequest("only failed executions can be retried")
	}
	return s.startLocked(ctx, executionID)
}
