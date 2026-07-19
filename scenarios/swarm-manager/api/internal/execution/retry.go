package execution

import (
	"context"
	"fmt"
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

	preflight := s.processPreflightForItem(item, false)
	if !preflight.Ready && !parent.Force && hasNonForceableExecutionReasons(preflight.BlockingReasons) {
		return Record{}, apierr.BadRequest("process preflight failed: %s", strings.Join(preflight.BlockingReasons, "; "))
	}

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
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	// Retry creates a new declared workflow execution with its own immutable
	// intent, never an operation-runner retry. Plan-backed work uses the same
	// phase-drain declaration as a first attempt; planless research uses its
	// declared conclusion workflow. The parent link remains Swarm domain data.
	if hasExecutionPlanRef(item) {
		records = append(records, retryRecord)
		return s.startPlanOperationLocked(ctx, records, len(records)-1, retryRecord, item)
	} else {
		res, snapshot, startErr := s.startConclusionWorkflow(ctx, item, retryRecord, req.Note)
		if startErr != nil {
			return Record{}, wrapAgentError(fmt.Errorf("start retry conclusion workflow failed: %w", startErr))
		}
		if strings.TrimSpace(res.ExecutionID) == "" {
			return Record{}, apierr.BadGateway("research-conclude workflow started but returned no execution id")
		}
		retryRecord.RunID = res.RunID
		retryRecord.TaskID = res.ExecutionID
		retryRecord.AgentWorkflowExecutionID = res.ExecutionID
		retryRecord.AgentWorkflowKey = snapshot.WorkflowKey
		retryRecord.AgentWorkflowDefinition = res.DefinitionDigest
		retryRecord.AgentWorkflowEntityVersion = snapshot.EntityVersion
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
