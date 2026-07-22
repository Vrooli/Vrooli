package execution

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"

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
		CreatedAt: now,
		UpdatedAt: now,
	}

	records, loadErr := s.store.Load()
	if loadErr != nil {
		slog.Error("failed to load records for fixup", "err", loadErr)
		return
	}
	records = append(records, fixupRecord)

	res, snapshot, startErr := s.startWorkWorkflow(ctx, item, fixupRecord, *record, "fixup", buildFinalizationFeedback(record.Finalization))
	if startErr != nil || strings.TrimSpace(res.ExecutionID) == "" {
		failureReason := "fixup workflow returned no execution id"
		if startErr != nil {
			slog.Error("failed to start fixup workflow", "err", startErr)
			failureReason = fmt.Sprintf("start failed: %v", startErr)
		}
		for i := range records {
			if records[i].ExecutionID == fixupRecord.ExecutionID {
				records[i].Status = StatusFailed
				records[i].FailureReason = failureReason
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
			records[i].TaskID = res.ExecutionID
			records[i].AgentWorkflowExecutionID = res.ExecutionID
			records[i].AgentWorkflowKey = snapshot.WorkflowKey
			records[i].AgentWorkflowDefinition = res.DefinitionDigest
			records[i].AgentWorkflowFrontier = snapshot.FrontierDigest
			records[i].AgentWorkflowEntityVersion = snapshot.EntityVersion
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
	// A policy or observer retry must not start a second run for the same
	// evidence-backed proposal. Manual follow-ups intentionally leave this key
	// empty and retain their existing repeatable behavior.
	if sourceProposalID := strings.TrimSpace(req.SourceProposalID); sourceProposalID != "" {
		for _, record := range records {
			if record.ParentExecutionID == parent.ExecutionID && record.FollowUpSourceProposalID == sourceProposalID {
				return record, nil
			}
		}
	}

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

	runType := req.FollowUpType
	if runType == "" {
		runType = "followup"
	}

	now := nowRFC3339()
	followUpRecord := Record{
		ExecutionID:              idgen.Generate(),
		BacklogKind:              parent.BacklogKind,
		BacklogName:              parent.BacklogName,
		PreviousStatus:           string(parent.Status),
		Status:                   StatusPending,
		Mode:                     ModeYOLO,
		StartedBy:                "swarm-manager:follow-up",
		Operation:                req.FollowUpType,
		ParentExecutionID:        parent.ExecutionID,
		FollowUpSourceProposalID: strings.TrimSpace(req.SourceProposalID),
		FollowUpSourceReviewRef:  strings.TrimSpace(req.SourceReviewRef),
		CreatedAt:                now,
		UpdatedAt:                now,
	}
	if req.FollowUpType == "fixup" {
		followUpRecord.FixupAttempt = parent.FixupAttempt + 1
	}

	// The declared workflow owns the prompt, run lifecycle, and structured
	// result. This adapter supplies only an immutable domain snapshot.
	res, snapshot, startErr := s.startWorkWorkflow(ctx, item, followUpRecord, *parent, runType, req.Context)
	if startErr != nil {
		return Record{}, wrapAgentError(startErr)
	}
	if strings.TrimSpace(res.ExecutionID) == "" {
		return Record{}, apierr.BadGateway("work workflow started but returned no execution id")
	}
	followUpRecord.RunID = res.RunID
	followUpRecord.TaskID = res.ExecutionID
	followUpRecord.AgentWorkflowExecutionID = res.ExecutionID
	followUpRecord.AgentWorkflowKey = snapshot.WorkflowKey
	followUpRecord.AgentWorkflowDefinition = res.DefinitionDigest
	followUpRecord.AgentWorkflowFrontier = snapshot.FrontierDigest
	followUpRecord.AgentWorkflowEntityVersion = snapshot.EntityVersion
	followUpRecord.Status = StatusStarting
	followUpRecord.StartedAt = now

	records = append(records, followUpRecord)
	if err := s.store.Save(records); err != nil {
		return Record{}, fmt.Errorf("failed to save follow-up record: %w", err)
	}

	s.dispatchStatusUpdate(followUpRecord)
	return followUpRecord, nil
}
