// Responsibility: reconcile retained run and sandbox lifecycle state.
package orchestration

import (
	"agent-manager/internal/adapters/event"
	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/repository"
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// attachedRunProcessAlive checks only process presence. A supplied PID is
// authoritative for harnesses that can report one. Otherwise, the launcher
// provides the signed token through the standard environment variable and the
// sweep matches its hash in /proc. Reading /proc cannot terminate or alter a
// process, which keeps this safety net independent from process-control code.
func (r *Reconciler) attachedRunProcessAlive(run *domain.Run) bool {
	if run == nil {
		return false
	}
	if run.RunnerPID > 0 {
		_, err := os.Stat(filepath.Join("/proc", strconv.Itoa(run.RunnerPID)))
		return err == nil
	}
	if run.IdentityTokenHash == "" {
		return false
	}
	entries, err := os.ReadDir("/proc")
	if err != nil {
		return false
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		if _, err := strconv.Atoi(entry.Name()); err != nil {
			continue
		}
		data, err := os.ReadFile(filepath.Join("/proc", entry.Name(), "environ"))
		if err != nil {
			continue
		}
		for _, value := range strings.Split(string(data), "\x00") {
			if !strings.HasPrefix(value, "VROOLI_AGENT_IDENTITY_TOKEN=") {
				continue
			}
			token := strings.TrimPrefix(value, "VROOLI_AGENT_IDENTITY_TOKEN=")
			if token != "" && identity.HashToken(token) == run.IdentityTokenHash {
				return true
			}
		}
	}
	return false
}

const eventRetentionBatchSize = 1_000

func (r *Reconciler) reapPendingRun(ctx context.Context, run *domain.Run) {
	age := time.Since(run.CreatedAt).Round(time.Second)
	reason := fmt.Sprintf("run remained pending for %v without dispatcher execution", age)
	r.log().Warn("reaping aged pending run", obs.KeyRunID, run.ID.String(), "pendingAge", age.String())
	r.markRunFailed(ctx, run, reason)
	if r.events == nil {
		return
	}
	message := fmt.Sprintf("reconciler failed stranded pending run after %v: %s", age, reason)
	if err := r.events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "error", message)); err != nil {
		r.log().Warn("pending reap event append failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

const (
	reviewNoticeRetryLimit = 10
	reviewNoticeScanLimit  = 100
	reviewNoticeBackoffMin = 30 * time.Second
	reviewNoticeBackoffMax = 10 * time.Minute
)

type reviewNoticeState struct {
	eventID          string
	publishedEventID string
	failures         int
	nextAttempt      time.Time
	attemptID        uint64
}

// reviewNoticeRefFromEvents recovers the durable requester reference from the
// append-only run event stream. The bounded tail keeps withdrawal lookup
// independent of long transcript history; if retention has removed the marker,
// callers fail closed and leave the Ask unchanged.
func (r *Reconciler) reviewNoticeRefsFromEvents(ctx context.Context, runID uuid.UUID) ([]string, bool, error) {
	if r == nil || r.events == nil || runID == uuid.Nil {
		return nil, false, nil
	}
	count, err := r.events.Count(ctx, runID)
	if err != nil {
		return nil, false, err
	}
	const markerTailLimit = 2_000
	offset := int(count) - markerTailLimit
	if offset < 0 {
		offset = 0
	}
	events, err := r.events.Get(ctx, runID, event.GetOptions{AfterSequence: -1, Limit: markerTailLimit, Offset: offset})
	if err != nil {
		return nil, false, err
	}
	active := make(map[string]bool)
	order := make([]string, 0)
	for _, item := range events {
		data, ok := item.Data.(*domain.LogEventData)
		if !ok {
			continue
		}
		switch {
		case strings.HasPrefix(data.Message, reviewNoticeRequestedMarkerPrefix):
			ref := strings.TrimPrefix(data.Message, reviewNoticeRequestedMarkerPrefix)
			if ref != "" {
				if _, seen := active[ref]; !seen {
					order = append(order, ref)
				}
				active[ref] = true
			}
		case strings.HasPrefix(data.Message, reviewNoticeWithdrawnMarkerPrefix):
			withdrawnRef := strings.TrimPrefix(data.Message, reviewNoticeWithdrawnMarkerPrefix)
			if withdrawnRef != "" {
				active[withdrawnRef] = false
			}
		}
	}
	refs := make([]string, 0, len(active))
	for _, ref := range order {
		if active[ref] {
			refs = append(refs, ref)
		}
	}
	// This proves only that the retained event stream was fully scanned, not
	// that every historical Hub Ask has a producer marker. Legacy requests
	// published before marker recording existed are not discoverable here; do
	// not claim this scan establishes their absence or withdraw them by guess.
	// A bounded tail or age-based event retention may also omit an earlier
	// still-open marked request. Return known refs for safe cleanup, but report
	// any observed prefix gap or truncated tail as partial.
	complete := count <= int64(len(events))
	if len(events) > 0 && events[0].Sequence > 1 {
		complete = false
	}
	return refs, complete, nil
}

func (r *Reconciler) syncReviewRuns(ctx context.Context, stats *ReconcileStats) {
	if r.sandbox == nil {
		return
	}
	needsReview := domain.RunStatusNeedsReview
	reviewRuns, err := r.runs.List(ctx, repository.RunListFilter{Status: &needsReview})
	if err != nil {
		stats.Errors = append(stats.Errors, "list review runs failed: "+err.Error())
		return
	}
	stats.ReviewChecked = len(reviewRuns)

	// Keep state for rows whose full read temporarily failed; a transient storage
	// error must not erase their retry backoff and cause a burst next cycle.
	knownReviewRuns := make(map[uuid.UUID]struct{}, len(reviewRuns))
	for _, run := range reviewRuns {
		knownReviewRuns[run.ID] = struct{}{}
	}
	var notificationCandidates []*domain.Run
	for _, run := range reviewRuns {
		// List deliberately omits sandbox_id and other heavy fields. Review
		// synchronization needs that durable association, so reload the full row
		// before touching the workspace-sandbox owner.
		full, err := r.runs.Get(ctx, run.ID)
		if err != nil || full == nil {
			if err != nil {
				stats.Errors = append(stats.Errors, "load review run "+run.ID.String()+" failed: "+err.Error())
			}
			continue
		}
		run = full
		if run.SandboxID == nil {
			delete(knownReviewRuns, run.ID)
			continue
		}
		sb, err := r.sandbox.Get(ctx, *run.SandboxID)
		if err != nil {
			continue
		}
		switch sb.Status {
		case sandbox.SandboxStatusApproved:
			if run.Status == domain.RunStatusComplete && run.ApprovalState == domain.ApprovalStateApproved {
				continue
			}
			r.markRunApprovedFromSandbox(ctx, run, "workspace-sandbox-sync")
			stats.ReviewSynced++
		case sandbox.SandboxStatusRejected:
			if run.Status == domain.RunStatusFailed && run.ApprovalState == domain.ApprovalStateRejected {
				continue
			}
			r.markRunRejectedFromSandbox(ctx, run, "workspace-sandbox-sync")
			stats.ReviewSynced++
		case sandbox.SandboxStatusActive:
			if isPendingManualReview(run) {
				// The in-memory success cache avoids repeated marker scans, but a
				// process restart loses it. Durable request markers therefore also
				// trigger a verified-empty check before retrying a previously raised
				// Ask. An incomplete marker scan fails closed and keeps the run pending.
				needsEmptyCheck := r.reviewNoticeNeedsEmptyCheck(run.ID)
				if !needsEmptyCheck && r.events != nil {
					refs, complete, markerErr := r.reviewNoticeRefsFromEvents(ctx, run.ID)
					if markerErr != nil {
						stats.Errors = append(stats.Errors, "run review request marker scan failed for "+run.ID.String()+": "+markerErr.Error())
						continue
					}
					needsEmptyCheck = len(refs) > 0 || !complete
				}
				if needsEmptyCheck && r.reviewNoticeEmptyReconciler != nil {
					empty, reconcileErr := r.reviewNoticeEmptyReconciler(ctx, run)
					if reconcileErr != nil {
						stats.Errors = append(stats.Errors, "run verified-empty review reconciliation failed for "+run.ID.String()+": "+reconcileErr.Error())
						continue
					}
					if empty {
						continue
					}
				}
				notificationCandidates = append(notificationCandidates, run)
			} else {
				delete(knownReviewRuns, run.ID)
			}
		}
	}
	r.pruneReviewNoticeStates(knownReviewRuns)
	r.retryReviewNoticeBatch(ctx, notificationCandidates, stats)
}

func (r *Reconciler) reviewNoticeNeedsEmptyCheck(runID uuid.UUID) bool {
	if r == nil || runID == uuid.Nil {
		return false
	}
	r.reviewNoticeMu.Lock()
	defer r.reviewNoticeMu.Unlock()
	state := r.reviewNoticeStates[runID]
	return state.publishedEventID != "" || state.eventID == verifiedEmptyReviewNoticeSentinel
}

func isPendingManualReview(run *domain.Run) bool {
	return run != nil && run.Status == domain.RunStatusNeedsReview &&
		run.ApprovalState == domain.ApprovalStatePending && run.SandboxID != nil &&
		run.TerminalClass != domain.RunTerminalClassInterruption &&
		run.StopReason != domain.RunStopReasonCrash
}

func (r *Reconciler) retryReviewNoticeBatch(ctx context.Context, candidates []*domain.Run, stats *ReconcileStats) {
	if r.reviewNoticePublisher == nil || len(candidates) == 0 {
		return
	}
	selected := r.selectReviewNoticeBatch(candidates, r.now())
	for _, run := range selected {
		attemptID := r.beginReviewNoticeAttempt(run.ID)
		eventID, err := r.reviewNoticePublisher(ctx, run)
		if err != nil {
			r.recordReviewNoticeFailure(run.ID, eventID, attemptID, r.now())
			stats.Errors = append(stats.Errors, "run approval notice retry failed for "+run.ID.String()+": "+err.Error())
			continue
		}
		if eventID == verifiedEmptyReviewNoticeSentinel {
			if r.reviewNoticeEmptyReconciler == nil {
				r.recordReviewNoticeFailure(run.ID, "", attemptID, r.now())
				stats.Errors = append(stats.Errors, "run verified-empty review reconciler is unavailable for "+run.ID.String())
				continue
			}
			empty, reconcileErr := r.reviewNoticeEmptyReconciler(ctx, run)
			if reconcileErr != nil {
				r.recordReviewNoticeFailure(run.ID, "", attemptID, r.now())
				stats.Errors = append(stats.Errors, "run verified-empty review reconciliation failed for "+run.ID.String()+": "+reconcileErr.Error())
				continue
			}
			if empty {
				continue
			}
			// The diff changed between the publisher's read and this settlement
			// check. Do not emit an event from stale terms; retry from a fresh read.
			r.recordReviewNoticeFailure(run.ID, "", attemptID, r.now())
			continue
		}
		r.recordReviewNoticePublished(run.ID, eventID, attemptID)
	}
}

func (r *Reconciler) selectReviewNoticeBatch(candidates []*domain.Run, now time.Time) []*domain.Run {
	ordered := append([]*domain.Run(nil), candidates...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].CreatedAt.Equal(ordered[j].CreatedAt) {
			return ordered[i].ID.String() < ordered[j].ID.String()
		}
		return ordered[i].CreatedAt.Before(ordered[j].CreatedAt)
	})
	if len(ordered) == 0 {
		return nil
	}

	r.reviewNoticeMu.Lock()
	defer r.reviewNoticeMu.Unlock()
	if r.reviewNoticeStates == nil {
		r.reviewNoticeStates = make(map[uuid.UUID]reviewNoticeState)
	}
	start := 0
	if r.reviewNoticeCursor != uuid.Nil {
		for i, run := range ordered {
			if run.ID == r.reviewNoticeCursor {
				start = (i + 1) % len(ordered)
				break
			}
		}
	}
	scan := len(ordered)
	if scan > reviewNoticeScanLimit {
		scan = reviewNoticeScanLimit
	}
	selected := make([]*domain.Run, 0, reviewNoticeRetryLimit)
	for offset := 0; offset < scan && len(selected) < reviewNoticeRetryLimit; offset++ {
		run := ordered[(start+offset)%len(ordered)]
		r.reviewNoticeCursor = run.ID
		state := r.reviewNoticeStates[run.ID]
		if state.publishedEventID != "" || (!state.nextAttempt.IsZero() && now.Before(state.nextAttempt)) {
			continue
		}
		selected = append(selected, run)
	}
	return selected
}

// beginReviewNoticeAttempt reserves a process-wide monotonic ticket before an
// initial, retry, or stale-generation publication starts. The counter is not
// pruned with per-run state, so late completions cannot match a recreated row.
func (r *Reconciler) beginReviewNoticeAttempt(runID uuid.UUID) uint64 {
	if r == nil || runID == uuid.Nil {
		return 0
	}
	r.reviewNoticeMu.Lock()
	defer r.reviewNoticeMu.Unlock()
	if r.reviewNoticeStates == nil {
		r.reviewNoticeStates = make(map[uuid.UUID]reviewNoticeState)
	}
	r.reviewNoticeAttemptID++
	if r.reviewNoticeAttemptID == 0 {
		r.reviewNoticeAttemptID++
	}
	state := r.reviewNoticeStates[runID]
	state.attemptID = r.reviewNoticeAttemptID
	r.reviewNoticeStates[runID] = state
	return state.attemptID
}

func (r *Reconciler) recordReviewNoticeFailure(runID uuid.UUID, eventID string, attemptID uint64, now time.Time) {
	if r == nil || runID == uuid.Nil || attemptID == 0 {
		return
	}
	r.reviewNoticeMu.Lock()
	defer r.reviewNoticeMu.Unlock()
	if r.reviewNoticeStates == nil {
		r.reviewNoticeStates = make(map[uuid.UUID]reviewNoticeState)
	}
	state := r.reviewNoticeStates[runID]
	if state.attemptID != attemptID {
		return
	}
	// A failed duplicate must not undo a successful publication of the same
	// deterministic generation. For different generations, only the latest
	// started attempt is allowed to update retry state.
	if eventID != "" && state.publishedEventID == eventID {
		return
	}
	if state.eventID != eventID {
		state = reviewNoticeState{attemptID: attemptID}
	}
	state.eventID = eventID
	state.publishedEventID = ""
	state.failures++
	delay := reviewNoticeBackoffMin
	for i := 1; i < state.failures && delay < reviewNoticeBackoffMax; i++ {
		delay *= 2
	}
	if delay > reviewNoticeBackoffMax {
		delay = reviewNoticeBackoffMax
	}
	state.nextAttempt = now.Add(delay)
	r.reviewNoticeStates[runID] = state
}

func (r *Reconciler) recordReviewNoticeEmptyCheck(runID uuid.UUID, attemptID uint64) {
	if r == nil || runID == uuid.Nil || attemptID == 0 {
		return
	}
	r.reviewNoticeMu.Lock()
	defer r.reviewNoticeMu.Unlock()
	if r.reviewNoticeStates == nil {
		r.reviewNoticeStates = make(map[uuid.UUID]reviewNoticeState)
	}
	state := r.reviewNoticeStates[runID]
	if state.attemptID != attemptID {
		return
	}
	r.reviewNoticeStates[runID] = reviewNoticeState{eventID: verifiedEmptyReviewNoticeSentinel, attemptID: attemptID}
}

func (r *Reconciler) recordReviewNoticePublished(runID uuid.UUID, eventID string, attemptID uint64) {
	if r == nil || runID == uuid.Nil || attemptID == 0 || strings.TrimSpace(eventID) == "" {
		return
	}
	r.reviewNoticeMu.Lock()
	defer r.reviewNoticeMu.Unlock()
	if r.reviewNoticeStates == nil {
		r.reviewNoticeStates = make(map[uuid.UUID]reviewNoticeState)
	}
	state := r.reviewNoticeStates[runID]
	if state.attemptID != attemptID {
		return
	}
	r.reviewNoticeStates[runID] = reviewNoticeState{eventID: eventID, publishedEventID: eventID, attemptID: attemptID}
}

func (r *Reconciler) pruneReviewNoticeStates(known map[uuid.UUID]struct{}) {
	r.reviewNoticeMu.Lock()
	defer r.reviewNoticeMu.Unlock()
	for id := range r.reviewNoticeStates {
		if _, ok := known[id]; !ok {
			delete(r.reviewNoticeStates, id)
		}
	}
	if _, ok := known[r.reviewNoticeCursor]; !ok {
		r.reviewNoticeCursor = uuid.Nil
	}
}

// reconcileOrphanSandboxes is a conservative owner-recovery sweep. A sandbox
// is not an orphan merely because a run is terminal: terminal runs remain the
// authoritative owner while their lifecycle/finalization is recoverable. The
// sweep acts only when the run row is genuinely absent, the sandbox is old
// enough to outlive create/dispatch races, and a bounded diff proves there
// are no changes worth preserving. Provider errors and non-empty diffs remain
// owner-review work and are never converted into deletion.
func (r *Reconciler) reconcileOrphanSandboxes(ctx context.Context, stats *ReconcileStats) {
	if r.sandbox == nil || r.runs == nil || r.config.OrphanGracePeriod <= 0 {
		return
	}
	inventory, ok := r.sandbox.(sandbox.Inventory)
	if !ok {
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, orphanSandboxOperationTimeout)
	sandboxes, err := inventory.List(listCtx, string(sandbox.SandboxStatusActive))
	cancel()
	if err != nil {
		stats.Errors = append(stats.Errors, "failed to list active run sandboxes: "+err.Error())
		return
	}

	checked := 0
	for _, sb := range sandboxes {
		if checked >= orphanSandboxSweepLimit {
			break
		}
		if sb == nil || sb.Status != sandbox.SandboxStatusActive {
			continue
		}
		runIDText := strings.TrimSpace(sb.Metadata[sandbox.AgentManagerRunIDMetadataKey])
		if runIDText == "" {
			continue
		}
		runID, parseErr := uuid.Parse(runIDText)
		if parseErr != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("active sandbox %s has invalid run owner %q: %v", sb.ID, runIDText, parseErr))
			continue
		}
		if sb.CreatedAt.IsZero() || r.now().Sub(sb.CreatedAt) < r.config.OrphanGracePeriod {
			continue
		}
		checked++
		stats.SandboxOrphansChecked++

		run, runErr := r.runs.Get(ctx, runID)
		if runErr != nil {
			var notFound *domain.NotFoundError
			if !errors.As(runErr, &notFound) {
				stats.Errors = append(stats.Errors, fmt.Sprintf("failed to resolve sandbox %s owner run %s: %v", sb.ID, runID, runErr))
				stats.SandboxOrphansPreserved++
				continue
			}
			// A typed NotFoundError continues through the same proof path as
			// the SQL repository's nil,nil missing-row result.
		} else if run != nil {
			continue
		}

		diffCtx, diffCancel := context.WithTimeout(ctx, orphanSandboxOperationTimeout)
		diff, diffErr := r.sandbox.GetDiff(diffCtx, sb.ID)
		diffCancel()
		if diffErr != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("failed to inspect orphan sandbox %s: %v", sb.ID, diffErr))
			stats.SandboxOrphansPreserved++
			continue
		}
		if !sandboxDiffEmpty(diff) {
			stats.SandboxOrphansPreserved++
			continue
		}

		deleteCtx, deleteCancel := context.WithTimeout(ctx, orphanSandboxOperationTimeout)
		deleteErr := r.sandbox.Delete(deleteCtx, sb.ID)
		deleteCancel()
		if deleteErr != nil {
			stats.Errors = append(stats.Errors, fmt.Sprintf("failed to delete empty orphan sandbox %s: %v", sb.ID, deleteErr))
			stats.SandboxOrphansPreserved++
			continue
		}
		stats.SandboxOrphansReclaimed++
		r.log().Info("reclaimed empty sandbox with missing run owner", "sandboxId", sb.ID.String(), "runId", runID.String())
	}
}

func (r *Reconciler) markRunApprovedFromSandbox(ctx context.Context, run *domain.Run, actor string) {
	now := r.now()
	run.ApprovalState = domain.ApprovalStateApproved
	run.ApprovedBy = actor
	run.ApprovedAt = &now
	run.Status = domain.RunStatusComplete
	run.Phase = domain.RunPhaseCompleted
	run.EndedAt = &now
	run.UpdatedAt = now

	if err := r.runs.Update(ctx, run); err != nil {
		r.log().Warn("approved run sync failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		return
	}
	if r.broadcaster != nil {
		r.broadcaster.BroadcastRunStatus(run)
	}
}

func (r *Reconciler) markRunRejectedFromSandbox(ctx context.Context, run *domain.Run, actor string) {
	now := r.now()
	run.ApprovalState = domain.ApprovalStateRejected
	run.ApprovedBy = actor
	run.ApprovedAt = &now
	run.Status = domain.RunStatusFailed
	run.Phase = domain.RunPhaseCompleted
	run.EndedAt = &now
	run.UpdatedAt = now

	if err := r.runs.Update(ctx, run); err != nil {
		r.log().Warn("rejected run sync failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		return
	}
	if r.broadcaster != nil {
		r.broadcaster.BroadcastRunStatus(run)
	}
}

// handleStaleRun handles a run that appears to have stalled.
func (r *Reconciler) handleStaleRun(ctx context.Context, run *domain.Run, stats *ReconcileStats) {
	tag := run.GetTag()
	var heartbeatAge time.Duration
	if run.LastHeartbeat != nil {
		heartbeatAge = time.Since(*run.LastHeartbeat)
	} else {
		heartbeatAge = time.Since(run.CreatedAt)
	}

	r.log().Debug("checking stale run",
		obs.KeyRunID, run.ID.String(),
		"tag", tag,
		"status", string(run.Status),
		"heartbeatAge", heartbeatAge.Round(time.Second).String(),
		"staleThreshold", r.config.StaleThreshold.String(),
	)

	// reconcile() supplies pruned-column runs (no ResolvedConfig) — re-fetch
	// with Get so recoverRun has everything recoveryParser needs. See the
	// matching note in RecoverInFlightRuns for the production bug this guards.
	if full, err := r.runs.Get(ctx, run.ID); err == nil && full != nil {
		run = full
	}

	result, recoveryErr := r.recoverRun(ctx, run, true)
	if recoveryErr != nil {
		// Failed, missing, or unpersistable recovery evidence is not a terminal
		// runner verdict. Keep the durable run state available for a later retry
		// or owner diagnosis. Do not project Failed from process absence; a
		// positively identified live process remains under the existing recovery
		// owner and a failed recovery write remains visible as an error.
		stats.Errors = append(stats.Errors, recoveryErr.Error())
		r.log().Warn("stale run recovery remains unresolved", obs.KeyRunID, run.ID.String(), obs.KeyError, recoveryErr.Error())
		return
	}
	if result != nil {
		if result.Recovered {
			stats.RunsRecovered++
		}
		// Recovery owns terminal, review, and reattachment outcomes. Do not
		// reinterpret a second process scan as a different terminal verdict.
		return
	}
	stats.Errors = append(stats.Errors, "stale run recovery returned no owner outcome for run "+run.ID.String())
	r.log().Warn("stale run recovery returned no owner outcome", obs.KeyRunID, run.ID.String())
}

// markRunFailed marks a run as failed due to unexpected termination.
func (r *Reconciler) markRunFailed(ctx context.Context, run *domain.Run, reason string) {
	now := r.now()
	run.Status = domain.RunStatusFailed
	run.ErrorMsg = reason
	run.EndedAt = &now
	run.UpdatedAt = now

	if err := r.runs.Update(ctx, run); err != nil {
		r.log().Warn("run status update failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		return // Do not project a status that lost its atomic writer fence.
	}

	// Broadcast status change
	if r.broadcaster != nil {
		r.broadcaster.BroadcastRunStatus(run)
	}
}

func (r *Reconciler) appendAttachedLifecycleEvent(ctx context.Context, id uuid.UUID, state, detail string) {
	if r.events == nil {
		return
	}
	if err := r.events.Append(ctx, id, domain.NewLogEvent(id, "attached_run_"+state, detail)); err != nil {
		r.log().Warn("attached run lifecycle event append failed", obs.KeyRunID, id.String(), obs.KeyError, err.Error())
	}
}
