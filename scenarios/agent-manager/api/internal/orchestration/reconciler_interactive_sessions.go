// Responsibility: bound how long web-console sessions outlive their interactive runs.
package orchestration

import (
	"context"
	"fmt"
	"time"

	"agent-manager/internal/adapters/webconsole"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
)

// releaseRetainedInteractiveSessions is the owner-inventory sweep that ends the
// continuation window of interactive runs. Finalize deliberately leaves an
// ended run's web-console session (and its idle agent CLI) alive so
// `run continue` can type into it; this sweep is what bounds that retention.
//
// Only sessions agent-manager created (owner agent-manager, programmatic
// origin) are examined. One is released (archived: web-console ends its process
// but keeps metadata and transcript; its archive retention owns disposal) when
// every run referencing it is
// terminal and the latest ended at least InteractiveSessionRetention ago, or
// when no run references it and the session itself is that old (a failed
// launch or a deleted run). Pending, running, parked, and needs_review runs
// keep their session. Web-console errors are recorded and skipped; they never
// abort the cycle.
func (r *Reconciler) releaseRetainedInteractiveSessions(ctx context.Context, stats *ReconcileStats) {
	retention := r.config.InteractiveSessionRetention
	if r.sessions == nil || r.runs == nil || retention <= 0 {
		return
	}

	listCtx, cancel := context.WithTimeout(ctx, orphanSandboxOperationTimeout)
	inventory, err := r.sessions.ListSessions(listCtx)
	cancel()
	if err != nil {
		r.log().Warn("interactive session retention skipped: web-console inventory unavailable", obs.KeyError, err.Error())
		stats.Errors = append(stats.Errors, "failed to list web-console sessions: "+err.Error())
		return
	}

	owned := make([]webconsole.SessionInfo, 0, len(inventory))
	ids := make([]string, 0, len(inventory))
	for _, info := range inventory {
		if info.ID == "" || info.Owner != webconsole.OwnerAgentManager || info.Origin != webconsole.OriginProgrammatic {
			continue
		}
		owned = append(owned, info)
		ids = append(ids, info.ID)
	}
	if len(owned) == 0 {
		return
	}
	referencing, err := r.runs.ListByWebConsoleSessionIDs(ctx, ids)
	if err != nil {
		stats.Errors = append(stats.Errors, "failed to resolve web-console session runs: "+err.Error())
		return
	}
	bySession := make(map[string][]*domain.Run, len(referencing))
	for _, run := range referencing {
		bySession[run.WebConsoleSessionID] = append(bySession[run.WebConsoleSessionID], run)
	}

	for _, info := range owned {
		if stats.InteractiveSessionsReleased >= interactiveSessionReleaseLimit {
			return
		}
		if ok, _ := interactiveSessionReleasable(info, bySession[info.ID], retention, r.now()); !ok {
			continue
		}
		if !r.releaseInteractiveSession(ctx, info, retention, stats) {
			// A continuation or recovery owner holds the lock; retry next cycle.
			return
		}
	}
}

// interactiveSessionReleasable decides from durable state alone whether the
// retention window for a session has closed. reason names the rule that fired.
func interactiveSessionReleasable(info webconsole.SessionInfo, runs []*domain.Run, retention time.Duration, now time.Time) (ok bool, reason string) {
	if len(runs) == 0 {
		// An unknown creation time is never evidence of age.
		if info.CreatedAt.IsZero() || now.Sub(info.CreatedAt) < retention {
			return false, ""
		}
		return true, "no run references it"
	}
	var latestEnd time.Time
	for _, run := range runs {
		if run == nil || !run.Status.IsTerminal() || run.EndedAt == nil {
			return false, ""
		}
		if run.EndedAt.After(latestEnd) {
			latestEnd = *run.EndedAt
		}
	}
	if now.Sub(latestEnd) < retention {
		return false, ""
	}
	return true, "run ended " + now.Sub(latestEnd).Round(time.Minute).String() + " ago"
}

// releaseInteractiveSession archives one session while holding the continuation
// owner lock, so a racing `run continue` either finishes admitting first (and
// the re-read below sees the run running again) or finds the session gone and
// reports that a new run is required. It returns false without acting when the
// lock is busy.
func (r *Reconciler) releaseInteractiveSession(ctx context.Context, info webconsole.SessionInfo, retention time.Duration, stats *ReconcileStats) bool {
	if !r.interactiveRecoveryMu.TryLock() {
		return false
	}
	defer r.interactiveRecoveryMu.Unlock()

	runs, err := r.runs.ListByWebConsoleSessionIDs(ctx, []string{info.ID})
	if err != nil {
		stats.Errors = append(stats.Errors, fmt.Sprintf("failed to re-read runs for web-console session %s: %v", info.ID, err))
		return true
	}
	ok, reason := interactiveSessionReleasable(info, runs, retention, r.now())
	if !ok {
		return true
	}
	for _, run := range runs {
		if (r.interactiveLiveDrivers != nil && r.interactiveLiveDrivers.has(run.ID)) || r.hasTailer(run.ID) {
			return true
		}
	}

	archiveCtx, cancel := context.WithTimeout(ctx, orphanSandboxOperationTimeout)
	err = r.sessions.ArchiveSession(archiveCtx, info.ID)
	cancel()
	if err != nil {
		r.log().Warn("interactive session archive failed", "sessionId", info.ID, "label", info.DisplayLabel, obs.KeyError, err.Error())
		stats.Errors = append(stats.Errors, fmt.Sprintf("failed to archive web-console session %s: %v", info.ID, err))
		return true
	}
	stats.InteractiveSessionsReleased++

	if len(runs) == 0 {
		r.log().Info("archived unreferenced agent-manager web-console session",
			"sessionId", info.ID, "label", info.DisplayLabel, "createdAt", info.CreatedAt.Format(time.RFC3339))
		return true
	}
	detail := fmt.Sprintf("interactive session %s archived after retention (%s; window %s); start a new run to continue", info.ID, reason, retention)
	for _, run := range runs {
		r.log().Info("interactive session archived after retention",
			obs.KeyRunID, run.ID.String(), "sessionId", info.ID, "label", info.DisplayLabel, "reason", reason)
		if r.events == nil {
			continue
		}
		if err := r.events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "info", detail)); err != nil {
			r.log().Warn("interactive session archive event append failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		}
	}
	return true
}
