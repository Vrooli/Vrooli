package execution

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/idgen"
)

// Retry creates a new execution run from a terminal parent run, copying the
// scope verbatim. User-initiated only: invoked through
// `POST /api/v1/execution/{id}/retry`, or indirectly via the backlog/initiative
// retry routes.
//
// Retry differs from FollowUp in that it does NOT carry review feedback
// forward — the user is asserting that the prior attempt was an environmental
// no-op (e.g., a fixed agent-manager bug, an upstream dependency repaired)
// and the work itself does not need to change. The optional Note flows
// through as informational context only.
//
// The parent Record is never mutated. Its logs, finalization, status,
// timestamps, and outcome remain intact for audit and stats. The new Record
// has ParentExecutionID pointing at the parent and Operation="retry" so stats
// can slice retries from process/fixup/followup runs.
//
// Idempotency: if a retry of this parent is already in flight (any non-terminal
// status with the same ParentExecutionID), Retry returns the existing in-flight
// record instead of creating a duplicate. This dedups double-clicks and racing
// HTTP retries without persistent idempotency keys.
//
// There is no auto-Retry path. Like FollowUp, Retry is strictly user-initiated.
func (s *Service) Retry(ctx context.Context, req RetryRequest) (Record, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.agentService == nil || !s.agentService.IsEnabled() {
		return Record{}, apierr.Unavailable("agent-manager is not available")
	}

	records, idx, err := s.loadRecordLocked(req.ExecutionID)
	if err != nil {
		return Record{}, err
	}
	parent := records[idx]

	// Eligible parent statuses: terminal or effectively-terminal. Same set as
	// FollowUp; pending/running/validating must reach a stable state before
	// the user can decide to retry.
	switch parent.Status {
	case StatusCompleted, StatusFailed, StatusCanceled, StatusNeedsFixup:
		// OK
	default:
		return Record{}, apierr.BadRequest("cannot retry execution in %q state", parent.Status)
	}

	// Idempotency: if a retry of this parent is already in flight, return it.
	for i := range records {
		if records[i].ParentExecutionID != parent.ExecutionID {
			continue
		}
		if records[i].Operation != "retry" {
			continue
		}
		if isInFlightStatus(records[i].Status) {
			return records[i], nil
		}
	}

	// Load backlog item for context.
	item, loadErr := s.loadBacklogItemByRecord(&parent)
	if loadErr != nil {
		return Record{}, fmt.Errorf("cannot retry: %w", loadErr)
	}

	// Build the unified execution prompt with run_type="retry". Critically:
	// no ReviewFeedback (we are not iterating on prior feedback) — the only
	// channel for user context is the optional Note, surfaced via the
	// FollowUpNote slot in the prompt template.
	itemDir := s.itemDir(item.Kind, item.Name)
	preflight := s.processPreflightForItem(item, false)
	if !preflight.Ready && !parent.Force && hasNonForceableExecutionReasons(preflight.BlockingReasons) {
		return Record{}, apierr.BadRequest("process preflight failed: %s", strings.Join(preflight.BlockingReasons, "; "))
	}
	deliverable, err := s.resolveExecutionDeliverable(ctx, item, itemDir)
	if err != nil {
		return Record{}, apierr.BadRequest("%s", err.Error())
	}
	ideaHandoff, handoffErr := s.buildIdeaHandoffPackage(item, itemDir, preflight, deliverable.Path)
	if handoffErr != nil {
		slog.Warn("failed to build idea handoff for retry", "kind", item.Kind, "name", item.Name, "err", handoffErr)
	}
	prompt := buildExecutionPrompt(executionPromptParams{
		Kind:               item.Kind,
		Name:               item.Name,
		Title:              item.Title,
		ItemFolder:         itemDir,
		RunType:            "retry",
		DeliverablePath:    deliverable.Path,
		DeliverableContent: deliverable.Markdown,
		FollowUpNote:       strings.TrimSpace(req.Note),
		IdeaHandoff:        ideaHandoff,
		SuggestedSkills:    item.SuggestedSkills,
	})

	now := nowRFC3339()
	retryRecord := Record{
		ExecutionID:       idgen.Generate(),
		BacklogKind:       parent.BacklogKind,
		BacklogName:       parent.BacklogName,
		PreviousStatus:    string(parent.Status),
		Status:            StatusPending,
		Mode:              parent.Mode,
		StartedBy:         "swarm-manager:retry",
		Operation:         "retry",
		ParentExecutionID: parent.ExecutionID,
		// FixupAttempt intentionally not carried forward: a retry is a user
		// action, not an auto-fixup. Conflating would consume the fixup-attempt
		// budget for unrelated reasons.
		PromptTrace: &PromptTrace{
			Purpose:        "retry",
			Prompt:         prompt,
			PromptRevision: promptRevision(prompt),
			UsedFallback:   false,
			CapturedAt:     now,
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Retry re-drains as a brand-new attempt. Plan-backed items start the
	// execution-retry operation against their plan-execution target (mirroring the
	// primary start; Retry-as-New-Attempt lineage is preserved on the record).
	// Research-conclusion items have no plan_ref and keep the legacy direct spawn
	// (slice-B exception, note 8828b096). "Continue existing session" semantics
	// belong to FollowUp, never here.
	if hasExecutionPlanRef(item) {
		if s.operationStarter == nil {
			return Record{}, apierr.Unavailable("execution operation runner is not available")
		}
		planHandle, herr := executionPlanHandle(item)
		if herr != nil {
			return Record{}, apierr.BadRequest("%s", herr.Error())
		}
		var inputs map[string]string
		if note := strings.TrimSpace(req.Note); note != "" {
			inputs = map[string]string{"RETRY_NOTE": note}
		}
		res, opErr := s.operationStarter.StartOperation(ctx, OperationStartRequest{
			Operation:        operationExecutionRetry,
			OperationVersion: operationVersionPinned,
			TargetKind:       targetKindPlanExecution,
			TargetID:         planHandle,
			CallerInputs:     inputs,
			IdempotencyKey:   "exec-" + retryRecord.ExecutionID,
			RequestedBy:      retryRecord.StartedBy,
		})
		if opErr != nil {
			return Record{}, wrapAgentError(fmt.Errorf("start retry operation failed: %w", opErr))
		}
		retryRecord.RunID = res.RunID
		retryRecord.OpWorkflowID = res.WorkflowID
		retryRecord.OpExecutionID = res.ExecutionID
		retryRecord.Status = StatusStarting
		retryRecord.StartedAt = now
	} else {
		// Research-conclusion items (no plan_ref) retry as a fresh research-conclude
		// operation against their backlog-item target — the same operation their
		// primary execution uses (retry-as-new-attempt lineage preserved on the
		// record). The optional note rides as OPERATOR_NOTE.
		if s.operationStarter == nil {
			return Record{}, apierr.Unavailable("execution operation runner is not available")
		}
		var inputs map[string]string
		if note := strings.TrimSpace(req.Note); note != "" {
			inputs = map[string]string{"OPERATOR_NOTE": note}
		}
		res, opErr := s.operationStarter.StartOperation(ctx, OperationStartRequest{
			Operation:        operationResearchConclude,
			OperationVersion: operationVersionPinned,
			TargetKind:       targetKindBacklogItem,
			TargetID:         item.Kind + "/" + item.Name,
			CallerInputs:     inputs,
			IdempotencyKey:   "exec-" + retryRecord.ExecutionID,
			RequestedBy:      retryRecord.StartedBy,
		})
		if opErr != nil {
			return Record{}, wrapAgentError(fmt.Errorf("start retry conclusion operation failed: %w", opErr))
		}
		retryRecord.RunID = res.RunID
		retryRecord.OpWorkflowID = res.WorkflowID
		retryRecord.OpExecutionID = res.ExecutionID
		retryRecord.Status = StatusStarting
		retryRecord.StartedAt = now
	}

	records = append(records, retryRecord)
	if err := s.store.Save(records); err != nil {
		return Record{}, fmt.Errorf("failed to save retry record: %w", err)
	}

	s.dispatchStatusUpdate(retryRecord)
	return retryRecord, nil
}

// isInFlightStatus reports whether a record is in a non-terminal state for
// the purpose of retry idempotency. Used to dedup concurrent retry calls.
func isInFlightStatus(status Status) bool {
	switch status {
	case StatusPending, StatusStarting, StatusRunning, StatusNeedsReview, StatusValidating:
		return true
	}
	return false
}

// RetryLatestForBacklog finds the most recent retry-eligible execution for
// the given backlog item and creates a new attempt parented to it. Used by
// the backlog-level and initiative-level retry routes so callers do not have
// to discover the latest execution id themselves.
//
// Returns (Record{}, false, nil) when the item has no executions at all
// (callers map this to a 400). A non-nil error indicates a real failure.
//
// Latest = greatest CreatedAt among the item's executions whose status is in
// the retry-eligible set. If the most recent execution is currently in
// flight (pending/starting/running), the most recent terminal one is used.
// This matches the "user fixed something, retry the last attempt" intent.
func (s *Service) RetryLatestForBacklog(ctx context.Context, backlogKind, backlogName, note string) (Record, bool, error) {
	s.mu.Lock()
	records, err := s.store.Load()
	s.mu.Unlock()
	if err != nil {
		return Record{}, false, err
	}

	var (
		latestEligibleIdx = -1
		anyMatch          = false
	)
	for i := range records {
		r := &records[i]
		if r.BacklogKind != backlogKind || r.BacklogName != backlogName {
			continue
		}
		anyMatch = true
		switch r.Status {
		case StatusCompleted, StatusFailed, StatusCanceled, StatusNeedsFixup:
			// eligible
		default:
			continue
		}
		if latestEligibleIdx == -1 || r.CreatedAt > records[latestEligibleIdx].CreatedAt {
			latestEligibleIdx = i
		}
	}

	if !anyMatch {
		return Record{}, false, nil
	}
	if latestEligibleIdx == -1 {
		return Record{}, false, apierr.BadRequest("no terminal execution to retry for %s/%s; all attempts are in flight", backlogKind, backlogName)
	}

	parent := records[latestEligibleIdx]
	newRecord, err := s.Retry(ctx, RetryRequest{ExecutionID: parent.ExecutionID, Note: note})
	if err != nil {
		return Record{}, true, err
	}
	return newRecord, true, nil
}
