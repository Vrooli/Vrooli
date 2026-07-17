package execution

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"strings"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/idgen"
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
	deliverable, deliverableErr := s.resolveExecutionDeliverable(ctx, item, itemDir)
	if deliverableErr != nil {
		record.Status = StatusFailed
		record.FailureReason = "failed to render linked plan for fixup: " + deliverableErr.Error()
		slog.Warn("failed to render linked plan for fixup", "kind", item.Kind, "name", item.Name, "err", deliverableErr)
		return
	}
	ideaHandoff, handoffErr := s.buildIdeaHandoffPackage(item, itemDir, s.processPreflightForItem(item, false), deliverable.Path)
	if handoffErr != nil {
		slog.Warn("failed to build idea handoff for fixup", "kind", item.Kind, "name", item.Name, "err", handoffErr)
	}

	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               item.Kind,
		Name:               item.Name,
		Title:              item.Title,
		ItemFolder:         itemDir,
		RunType:            "fixup",
		DeliverablePath:    deliverable.Path,
		DeliverableContent: deliverable.Markdown,
		ReviewFeedback:     buildFinalizationFeedback(record.Finalization),
		IdeaHandoff:        ideaHandoff,
		SuggestedSkills:    item.SuggestedSkills,
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
		// No per-record engagement inheritance (plan P-b): engagements are owned by
		// the backlog item (ownerKeyFor), and a fixup shares the parent's
		// kind/name, so it transparently sees the same EngagementStore set. The
		// fixup's own pre-merge hold expands the set if it touches new scenarios.
		PromptTrace: &PromptTrace{
			Purpose:        "fixup",
			Prompt:         prompt,
			PromptRevision: promptRevision(prompt),
			UsedFallback:   false,
			// Reconstructed caller-context provenance; the agent runs the
			// bound operation's mode prompt (see PromptTrace doc).
			Synthetic:  true,
			CapturedAt: now,
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

	// Reroute through the operation runner: a fix-up is the execution-fixup
	// operation on the backlog-item target. The bound mode (backlog-fixup) owns the
	// prompt; the review feedback + type ride as caller inputs. The completion
	// bridge's commit-execution-round handler drives this record to terminal
	// (execution.opshandlers), so the poller defers it (OpExecutionID set).
	if s.operationStarter == nil {
		slog.Error("fixup start skipped: execution operation runner unavailable", "kind", item.Kind, "name", item.Name)
		for i := range records {
			if records[i].ExecutionID == fixupRecord.ExecutionID {
				records[i].Status = StatusFailed
				records[i].FailureReason = "execution operation runner is not available"
				records[i].FinishedAt = now
				break
			}
		}
		_ = s.store.Save(records)
		s.dispatchStatusUpdate(fixupRecord)
		return
	}
	res, opErr := s.operationStarter.StartOperation(ctx, OperationStartRequest{
		Operation:        operationExecutionFixup,
		OperationVersion: operationVersionPinned,
		TargetKind:       targetKindBacklogItem,
		TargetID:         item.Kind + "/" + item.Name,
		// execution-fixup declares only OPERATOR_NOTE; the review feedback rides
		// there (routed to the mode's operator-note channel).
		CallerInputs: map[string]string{
			"OPERATOR_NOTE": buildFinalizationFeedback(record.Finalization),
		},
		IdempotencyKey: "exec-" + fixupRecord.ExecutionID,
		RequestedBy:    fixupRecord.StartedBy,
	})
	if opErr != nil {
		if errors.Is(opErr, agentactivity.ErrBacklogItemBusy) {
			slog.Warn("fixup start skipped: agent already active", "kind", item.Kind, "name", item.Name)
		} else {
			slog.Error("failed to start fixup operation", "err", opErr)
		}
		for i := range records {
			if records[i].ExecutionID == fixupRecord.ExecutionID {
				records[i].Status = StatusFailed
				records[i].FailureReason = fmt.Sprintf("start failed: %v", opErr)
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
			records[i].RunID = res.RunID
			records[i].OpWorkflowID = res.WorkflowID
			records[i].OpExecutionID = res.ExecutionID
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

// FollowUp creates a new execution run from a completed/failed/needs_fixup
// parent run. User-initiated only: invoked through
// `POST /api/v1/execution/{id}/follow-up`.
//
// NOT to be confused with the backlog.StatusNeedsFollowup terminal status:
//   - StatusNeedsFollowup (item level) is a signal from review-decide that
//     the *item* needs more work. It does not auto-open any execution.
//   - FollowUp (run level) spawns another *run* against the same item when
//     the user asks for one, regardless of the item's status. It may be
//     invoked against a `needs_followup` item, a `failed` item, or any
//     other item whose last execution reached completed / failed /
//     needs_fixup.
//
// There is no auto-FollowUp path — production code never calls this from
// finalization or polling. If you add one, reconsider: the W1 plan routed
// post-run state through `in_review` / `review_pending` specifically so
// the user decides whether a follow-up is warranted.
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
	deliverable, deliverableErr := s.resolveExecutionDeliverable(ctx, item, itemDir)
	if deliverableErr != nil {
		return Record{}, apierr.BadRequest("%s", deliverableErr.Error())
	}
	ideaHandoff, handoffErr := s.buildIdeaHandoffPackage(item, itemDir, s.processPreflightForItem(item, false), deliverable.Path)
	if handoffErr != nil {
		slog.Warn("failed to build idea handoff for follow-up", "kind", item.Kind, "name", item.Name, "err", handoffErr)
	}
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               item.Kind,
		Name:               item.Name,
		Title:              item.Title,
		ItemFolder:         itemDir,
		RunType:            runType,
		DeliverablePath:    deliverable.Path,
		DeliverableContent: deliverable.Markdown,
		ReviewFeedback:     buildFinalizationFeedback(parent.Finalization),
		FollowUpNote:       strings.TrimSpace(req.Context),
		IdeaHandoff:        ideaHandoff,
		SuggestedSkills:    item.SuggestedSkills,
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
			// Reconstructed caller-context provenance; the agent runs the
			// bound operation's mode prompt (see PromptTrace doc).
			Synthetic:  true,
			CapturedAt: now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}
	if req.FollowUpType == "fixup" {
		followUpRecord.FixupAttempt = parent.FixupAttempt + 1
	}

	// Reroute through the operation runner. A follow-up starts a fresh
	// execution-followup operation (or execution-fixup when the caller asked for a
	// fix-up flavor) on the backlog-item target; the bound mode owns the prompt and
	// the note/type ride as caller inputs. Agent-session continuation
	// (run_mode=continue) is not part of the declarative-operations model — a fresh
	// run reads the same completed deliverable + feedback — so it collapses to a
	// fresh start. The completion bridge drives this record to terminal, so the
	// poller defers it (OpExecutionID set).
	if s.operationStarter == nil {
		return Record{}, apierr.Unavailable("execution operation runner is not available")
	}
	// Route by operation, matching each contract's declared caller inputs:
	// execution-followup accepts FOLLOWUP_NOTE + FOLLOWUP_TYPE; execution-fixup
	// accepts only OPERATOR_NOTE (LivePreparer rejects any undeclared input).
	followUpOp := operationExecutionFollowup
	callerInputs := map[string]string{
		"FOLLOWUP_TYPE": runType,
		"FOLLOWUP_NOTE": strings.TrimSpace(req.Context),
	}
	if req.FollowUpType == "fixup" {
		followUpOp = operationExecutionFixup
		callerInputs = map[string]string{"OPERATOR_NOTE": strings.TrimSpace(req.Context)}
	}
	res, opErr := s.operationStarter.StartOperation(ctx, OperationStartRequest{
		Operation:        followUpOp,
		OperationVersion: operationVersionPinned,
		TargetKind:       targetKindBacklogItem,
		TargetID:         item.Kind + "/" + item.Name,
		CallerInputs:     callerInputs,
		IdempotencyKey:   "exec-" + followUpRecord.ExecutionID,
		RequestedBy:      followUpRecord.StartedBy,
	})
	if opErr != nil {
		return Record{}, wrapAgentError(fmt.Errorf("start follow-up operation failed: %w", opErr))
	}
	followUpRecord.RunID = res.RunID
	followUpRecord.OpWorkflowID = res.WorkflowID
	followUpRecord.OpExecutionID = res.ExecutionID
	followUpRecord.Status = StatusStarting
	followUpRecord.StartedAt = now

	records = append(records, followUpRecord)
	if err := s.store.Save(records); err != nil {
		return Record{}, fmt.Errorf("failed to save follow-up record: %w", err)
	}

	s.dispatchStatusUpdate(followUpRecord)
	return followUpRecord, nil
}
