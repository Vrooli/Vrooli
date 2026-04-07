package execution

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/idgen"
	"swarm-manager/internal/workshop"
)

// handleSpecSyncComplete performs the archive after a successful spec-sync run.
func (s *Service) handleSpecSyncComplete(ctx context.Context, record *Record) {
	if s.archiver == nil {
		slog.Error("spec-sync completed but no archiver configured", "backlog_name", record.BacklogName)
		record.FailureReason = "archiver not configured"
		record.Status = StatusFailed
		return
	}

	ac := record.ArchiveContext
	if _, err := os.Stat(ac.ScenarioPath); err != nil {
		slog.Error("spec-sync completed but scenario dir missing", "scenario_path", ac.ScenarioPath)
		record.FailureReason = "scenario directory no longer exists"
		record.Status = StatusFailed
		return
	}

	if err := s.archiver.ArchiveScenario(ctx, *ac); err != nil {
		slog.Error("post-spec-sync archive failed", "scenario_name", ac.ScenarioName, "err", err)
		record.FailureReason = "archive failed after spec-sync: " + err.Error()
		record.Status = StatusFailed
		return
	}

	// Delete the scenario directory after successful archive
	if err := os.RemoveAll(ac.ScenarioPath); err != nil {
		slog.Error("post-archive scenario deletion failed", "scenario_name", ac.ScenarioName, "err", err)
		record.FailureReason = "scenario deletion failed after archive: " + err.Error()
		record.Status = StatusFailed
		return
	}

	slog.Info("spec-sync-archive completed", "scenario_name", ac.ScenarioName)
}

func (s *Service) spawnFixupRun(ctx context.Context, record *Record, item backlogItem) {
	now := nowRFC3339()
	itemDir := s.itemDir(item.Kind, item.Name)
	deliverablePath := deliverableForKind(item.Kind)
	ideaHandoff, handoffErr := s.buildIdeaHandoffPackage(item, itemDir, s.processPreflightForItem(item, false))
	if handoffErr != nil {
		slog.Warn("failed to build idea handoff for fixup", "kind", item.Kind, "name", item.Name, "err", handoffErr)
	}

	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               item.Kind,
		Name:               item.Name,
		Title:              item.Title,
		ItemFolder:         itemDir,
		RunType:            "fixup",
		DeliverablePath:    deliverablePath,
		DeliverableContent: workshop.LoadPlanContentByName(itemDir, deliverablePath),
		ReviewFeedback:     buildFinalizationFeedback(effectiveFinalization(*record)),
		IdeaHandoff:        ideaHandoff,
	})

	fixupRecord := Record{
		ExecutionID:       idgen.Generate(),
		BacklogKind:       record.BacklogKind,
		BacklogName:       record.BacklogName,
		PreviousStatus:    backlogStatusQueued,
		Status:            StatusPending,
		Mode:              ModeYOLO,
		StartedBy:         "swarm-manager:fixup",
		Operation:         "fixup",
		ParentExecutionID: record.ExecutionID,
		FixupAttempt:      record.FixupAttempt + 1,
		PromptTrace: &PromptTrace{
			Purpose:        "fixup",
			Prompt:         prompt,
			PromptRevision: promptRevision(prompt),
			UsedFallback:   false,
			CapturedAt:     now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	records, loadErr := s.store.Load()
	if loadErr != nil {
		slog.Error("failed to load records for fixup", "err", loadErr)
		return
	}
	records = append(records, fixupRecord)

	activityCtx := agentactivity.WithSpec(ctx, backlogActivitySpec(
		item,
		fixupRecord.ExecutionID,
		agentactivity.PurposeFixup,
		"swarm-manager:fixup",
		map[string]string{
			"entrypoint": "execution.fixup",
			"attempt":    fmt.Sprintf("%d", fixupRecord.FixupAttempt),
		},
	))

	runResult, err := s.agentService.SpawnBacklog(activityCtx, agentmanager.BacklogSpawnRequest{
		Kind:            item.Kind,
		Name:            item.Name,
		Title:           fmt.Sprintf("Fix-up: %s/%s (attempt %d)", item.Kind, item.Name, fixupRecord.FixupAttempt),
		Description:     prompt,
		Prompt:          prompt,
		ScopePath:       ".",
		ProjectRoot:     ".",
		CreatedBy:       "swarm-manager:fixup",
		Purpose:         "fixup",
		AcceptanceAllow: item.AcceptanceAllow,
		AcceptanceDeny:  item.AcceptanceDeny,
		Environment:     map[string]string{"VROOLI_SPAWN_SOURCE": item.Kind + "/" + item.Name},
	})
	if err != nil {
		if errors.Is(err, agentactivity.ErrBacklogItemBusy) {
			slog.Warn("fixup spawn skipped: agent already active", "kind", item.Kind, "name", item.Name)
		} else {
			slog.Error("failed to spawn fixup run", "err", err)
		}
		for i := range records {
			if records[i].ExecutionID == fixupRecord.ExecutionID {
				records[i].Status = StatusFailed
				records[i].FailureReason = fmt.Sprintf("spawn failed: %v", err)
				records[i].FinishedAt = now
				break
			}
		}
		_ = s.store.Save(records)
		for _, candidate := range records {
			if candidate.ExecutionID == fixupRecord.ExecutionID {
				s.dispatchStatusUpdate(candidate)
				break
			}
		}
		return
	}

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
	for _, candidate := range records {
		if candidate.ExecutionID == fixupRecord.ExecutionID {
			s.dispatchStatusUpdate(candidate)
			continue
		}
		if candidate.ExecutionID == record.ExecutionID {
			s.dispatchStatusUpdate(candidate)
		}
	}
}

// FollowUp creates a follow-up execution from a completed/failed/needs_fixup execution.
func (s *Service) FollowUp(ctx context.Context, req FollowUpRequest) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.agentService == nil || !s.agentService.IsEnabled() {
		return Record{}, apierr.Unavailable("agent-manager is not available")
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
		return Record{}, apierr.BadRequest("cannot follow up execution in %q state", parent.Status)
	}

	// Load backlog item for context.
	item, loadErr := s.loadBacklogItemByRecord(parent)
	if loadErr != nil {
		return Record{}, fmt.Errorf("cannot follow up: %w", loadErr)
	}

	// Build the unified execution prompt.
	itemDir := s.itemDir(item.Kind, item.Name)
	runType := req.FollowUpType
	if runType == "" {
		runType = "followup"
	}
	deliverablePath := deliverableForKind(item.Kind)
	ideaHandoff, handoffErr := s.buildIdeaHandoffPackage(item, itemDir, s.processPreflightForItem(item, false))
	if handoffErr != nil {
		slog.Warn("failed to build idea handoff for follow-up", "kind", item.Kind, "name", item.Name, "err", handoffErr)
	}
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               item.Kind,
		Name:               item.Name,
		Title:              item.Title,
		ItemFolder:         itemDir,
		RunType:            runType,
		DeliverablePath:    deliverablePath,
		DeliverableContent: workshop.LoadPlanContentByName(itemDir, deliverablePath),
		ReviewFeedback:     buildFinalizationFeedback(effectiveFinalization(*parent)),
		FollowUpNote:       strings.TrimSpace(req.Context),
		IdeaHandoff:        ideaHandoff,
	})

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
		PromptTrace: &PromptTrace{
			Purpose:        runType,
			Prompt:         prompt,
			PromptRevision: promptRevision(prompt),
			UsedFallback:   false,
			CapturedAt:     now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if req.FollowUpType == "fixup" {
		followUpRecord.FixupAttempt = parent.FixupAttempt + 1
	}

	if req.RunMode == "continue" && strings.TrimSpace(parent.RunID) != "" {
		// Continue existing agent-manager session.
		if s.continuer == nil {
			return Record{}, fmt.Errorf("cannot follow up: run continuation not available")
		}
		continueCtx := agentactivity.WithSpec(ctx, backlogActivitySpec(
			item,
			followUpRecord.ExecutionID,
			executionActivityPurpose(runType),
			followUpRecord.StartedBy,
			map[string]string{
				"entrypoint":       "execution.follow_up",
				"follow_up_type":   runType,
				"run_mode":         req.RunMode,
				"parent_execution": parent.ExecutionID,
			},
		))
		if err := s.continuer.ContinueRun(continueCtx, parent.RunID, prompt); err != nil {
			if strings.Contains(err.Error(), "session_expired") || strings.Contains(err.Error(), "continuation_not_supported") {
				return Record{}, apierr.Wrap(apierr.ErrSessionExpired, http.StatusConflict, "agent session expired; retry with run_mode=new")
			}
			return Record{}, fmt.Errorf("continue run failed: %w", err)
		}
		followUpRecord.RunID = parent.RunID
		followUpRecord.TaskID = parent.TaskID
		followUpRecord.Status = StatusRunning
		followUpRecord.StartedAt = now
	} else {
		// Spawn a fresh run.
		spawnCtx := agentactivity.WithSpec(ctx, backlogActivitySpec(
			item,
			followUpRecord.ExecutionID,
			executionActivityPurpose(runType),
			followUpRecord.StartedBy,
			map[string]string{
				"entrypoint":       "execution.follow_up",
				"follow_up_type":   runType,
				"run_mode":         req.RunMode,
				"parent_execution": parent.ExecutionID,
			},
		))
		runResult, spawnErr := s.agentService.SpawnBacklog(spawnCtx, agentmanager.BacklogSpawnRequest{
			Kind:            item.Kind,
			Name:            item.Name,
			Title:           fmt.Sprintf("Follow-up: %s/%s", item.Kind, item.Name),
			Description:     prompt,
			Prompt:          prompt,
			ScopePath:       ".",
			ProjectRoot:     ".",
			CreatedBy:       "swarm-manager:follow-up",
			Purpose:         req.FollowUpType,
			AcceptanceAllow: item.AcceptanceAllow,
			AcceptanceDeny:  item.AcceptanceDeny,
			Environment:     map[string]string{"VROOLI_SPAWN_SOURCE": item.Kind + "/" + item.Name},
		})
		if spawnErr != nil {
			return Record{}, wrapAgentError(fmt.Errorf("spawn follow-up failed: %w", spawnErr))
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

	s.dispatchStatusUpdate(followUpRecord)
	return followUpRecord, nil
}
