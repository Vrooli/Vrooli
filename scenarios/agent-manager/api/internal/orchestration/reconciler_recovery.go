// Responsibility: reconcile retained run and sandbox lifecycle state.
package orchestration

import (
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

func (r *Reconciler) syncReviewRuns(ctx context.Context, stats *ReconcileStats) {
	if r.sandbox == nil {
		return
	}

	needsReview := domain.RunStatusNeedsReview
	reviewRuns, err := r.runs.List(ctx, repository.RunListFilter{
		Status: &needsReview,
	})
	if err != nil {
		stats.Errors = append(stats.Errors, "failed to list needs_review runs: "+err.Error())
		return
	}
	stats.ReviewChecked = len(reviewRuns)

	for _, run := range reviewRuns {
		// List deliberately omits sandbox_id and other heavy fields. Review
		// synchronization needs that durable association, so reload the full row
		// instead of broadening the read-side list contract for every caller.
		full, getErr := r.runs.Get(ctx, run.ID)
		if getErr != nil || full == nil {
			if getErr != nil {
				stats.Errors = append(stats.Errors, "failed to reload needs_review run "+run.ID.String()+": "+getErr.Error())
			}
			continue
		}
		run = full
		if run.SandboxID == nil {
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
		}
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

	// Use the hydrated heartbeat. A list snapshot can be stale even though
	// the active executor renewed its lease before this check.
	if run.LastHeartbeat != nil {
		heartbeatAge = r.now().Sub(*run.LastHeartbeat)
	}
	result, recoveryErr := r.recoverRun(ctx, run, true)
	if recoveryErr != nil && r.isProcessAlive(ctx, run) {
		// Artifact failure is not executor death. In particular, neither an
		// old heartbeat nor an old attempt deadline authorizes terminating a
		// positively identified process when recovery evidence is unavailable.
		stats.Errors = append(stats.Errors, recoveryErr.Error())
		r.log().Warn("live executor retained for agent-manager owner diagnosis", obs.KeyRunID, run.ID.String(), obs.KeyError, recoveryErr.Error())
		return
	}
	if recoveryErr == nil && result != nil {
		if result.Recovered {
			stats.RunsRecovered++
		}
		if run.Status == domain.RunStatusComplete || run.Status == domain.RunStatusFailed || run.Status == domain.RunStatusCancelled {
			return
		}
		if result.Recovered {
			// Recovery attached an owner to the surviving executor. The old
			// process heartbeat is not a timeout for that new owner. In
			// particular a long tool call survives an AM restart unchanged.
			return
		}
	}

	// Interactive runs live in a web-console tmux pane, not a local tagged child.
	// recoverRun already verified the session (GetSession) and reattached the
	// tailer or finalized the run, so the pgid scan / MaxRecoveryAge kill below
	// must not run — it would falsely fail a healthy interactive run.
	if run.ExecutionMode.Normalized() == domain.ExecutionModeInteractive {
		return
	}

	// First, check if the process is actually still running
	processAlive := r.isProcessAlive(ctx, run)

	if !processAlive {
		// Process died but DB wasn't updated - mark as failed
		r.log().Warn("run process not found, marking failed",
			obs.KeyRunID, run.ID.String(),
			"tag", tag,
			"heartbeatAge", heartbeatAge.Round(time.Second).String(),
		)
		r.markRunFailed(ctx, run, fmt.Sprintf("process terminated unexpectedly (detected by reconciler after %v without heartbeat, tag=%s)",
			heartbeatAge.Round(time.Second), tag))
		return
	}

	// Process is alive but heartbeat is stale - could be legitimate slow work,
	// or the executor is gone (e.g., agent-manager restarted) and nobody is
	// managing this process anymore.
	r.log().Info("run stale but process alive",
		obs.KeyRunID, run.ID.String(),
		"lastHeartbeat", formatTimePtr(run.LastHeartbeat),
	)

	// MaxRecoveryAge is an observation threshold, not permission to terminate
	// a positively identified detached runner. A control-plane restart can
	// leave a healthy executor quiet while it is in a long provider call;
	// killing it here would turn temporary ownership loss into data loss.
	if r.config.MaxRecoveryAge > 0 && heartbeatAge > r.config.MaxRecoveryAge {
		r.log().Warn("run exceeded recovery observation age; retaining live process for owner diagnosis",
			obs.KeyRunID, run.ID.String(),
			"tag", tag,
			"heartbeatAge", heartbeatAge.Round(time.Second).String(),
			"maxRecoveryAge", r.config.MaxRecoveryAge.String(),
		)
		return
	}

	if r.config.AutoRecover {
		// The process is alive but the executor heartbeat loop isn't updating.
		// Don't reset LastHeartbeat here: it remains useful evidence for the
		// current owner diagnosis. Count the retained executor as recovered.
		stats.RunsRecovered++
	}
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
