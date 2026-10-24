// This file implements out-of-band operator notification for runs that enter
// needs_review. When a sandboxed run finishes under manualReview, it publishes
// a *.decision_requested.v1 event onto Vrooli Events. notification-hub turns
// that generic decision event into an operator Ask (push/desktop/email) with
// Approve/Reject options and a review link — no per-scenario code in the hub.
// The operator's answer returns as notification_hub.ask.resolved.v1 and drives
// ApproveRun/RejectRun (see handlers/decision_events.go + resolveRunApproval).
package orchestration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	"agent-manager/internal/adapters/sandbox"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"
	"agent-manager/internal/supervision"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
	"github.com/vrooli/api-core/operatordecision"
)

// RunApprovalDecisionEventType is the decision event a run raises when it needs
// operator approval. The `.decision_requested.v1` suffix is the generic
// contract notification-hub already subscribes to (**.decision_requested.v1),
// so no hub change is needed to notify the operator.
const RunApprovalDecisionEventType = "agent_manager.run.decision_requested.v1"

// RunApprovalWithdrawalEventType retracts a previously published request when
// the exact run review is no longer needed. It is not an operator answer.
const RunApprovalWithdrawalEventType = "agent_manager.run.decision_withdrawn.v1"

const (
	reviewNoticeRequestedMarkerPrefix = "decision-review-requested:v1:"
	reviewNoticeWithdrawnMarkerPrefix = "decision-review-withdrawn:v1:"
	verifiedEmptyReviewNoticeSentinel = "\x00verified-empty-review"
)

// runApprovalSource is the Vrooli Events source scenario for these events.
const runApprovalSource = "agent-manager"

// ApprovalEventPublisher publishes a decision event onto Vrooli Events. The
// api-core eventbus.Client satisfies it in production. Nil leaves runs in
// needs_review silently (no regression from today's behavior). It is an alias
// for the shared operatordecision.Publisher so every raise path and the
// substrate speak one interface.
type ApprovalEventPublisher = operatordecision.Publisher

// WithApprovalEventPublisher wires the Vrooli Events publisher used to raise a
// run's approval decision event. Without one, runs still reach needs_review;
// they just do not notify the operator out-of-band.
func WithApprovalEventPublisher(p ApprovalEventPublisher) Option {
	return func(o *Orchestrator) { o.approvalPublisher = p }
}

// NotifyRunNeedsReview satisfies phases.ApprovalNotifier. It is invoked from
// the single run-end seam (phases.ApplyAtRunEnd) the moment a run enters
// needs_review under manualReview. Delivery is best-effort: a publish failure
// is logged and never blocks the run's terminal transition.
func (o *Orchestrator) NotifyRunNeedsReview(ctx context.Context, run *domain.Run) {
	if o == nil || o.approvalPublisher == nil || run == nil {
		return
	}
	attemptID := o.beginRunReviewNotice(run.ID)
	eventID, err := o.publishRunNeedsReview(ctx, run)
	o.recordRunReviewNotice(run.ID, eventID, attemptID, err)
	if err != nil {
		approvalLog().Warn("run approval decision event publication failed",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

// publishRunNeedsReview reconstructs the event from durable run state and the
// current sandbox diff. The event ID is deterministic, so a retry after a lost
// acknowledgement is safe at Events and notification-hub.
func (o *Orchestrator) publishRunNeedsReview(ctx context.Context, run *domain.Run) (string, error) {
	if o == nil || o.approvalPublisher == nil || run == nil {
		return "", errors.New("run approval decision publisher is unavailable")
	}
	event, verifiedEmpty, err := o.currentRunApprovalDecision(ctx, run)
	if err != nil {
		return "", fmt.Errorf("build current run approval event: %w", err)
	}
	if verifiedEmpty {
		// Initial finalization normally handled this before entering needs_review.
		// A diff can become empty between that check and publication; do not
		// manufacture a zero-file Ask. The sentinel is internal to the
		// reconciler and triggers terminal reconciliation without masquerading
		// as an Events event ID.
		return verifiedEmptyReviewNoticeSentinel, nil
	}
	return o.publishCurrentRunApprovalDecision(ctx, run, event)
}

// publishCurrentRunApprovalDecision is the shared durable publication seam for
// initial/retried notices and stale-answer replacement notices. Every current
// generation is marked before publication and supersedes only older exact refs
// recorded by this producer.
func (o *Orchestrator) publishCurrentRunApprovalDecision(ctx context.Context, run *domain.Run, event eventbus.DomainEvent) (string, error) {
	if o == nil || o.approvalPublisher == nil || run == nil || run.ID == uuid.Nil {
		return "", errors.New("run approval decision publisher is unavailable")
	}
	eventID := operatordecision.EventID(event)
	if o.events != nil {
		// Retain the exact requester reference before external publication. A
		// crash between this append and Events acceptance is safe: withdrawing
		// an Ask that was never created is a no-op at the Hub.
		if err := o.events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "debug", reviewNoticeRequestedMarkerPrefix+eventID)); err != nil {
			return "", fmt.Errorf("retain run review request reference: %w", err)
		}
	}
	if err := o.publishRunApprovalEvent(ctx, event); err != nil {
		return eventID, err
	}
	// A newly published request supersedes prior generations for this run.
	// Keep the exact current reference and withdraw only older references that
	// were durably recorded by this producer. A failed withdrawal is retried
	// when this same deterministic request is delivered again.
	if o.events != nil && o.reconciler != nil {
		refs, complete, err := o.reconciler.reviewNoticeRefsFromEvents(ctx, run.ID)
		if err != nil {
			approvalLog().Warn("run approval superseded reference scan failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
			return eventID, fmt.Errorf("scan exact superseded run review references: %w", err)
		}
		if !complete {
			approvalLog().Warn("run approval superseded reference scan is incomplete", obs.KeyRunID, run.ID.String())
			return eventID, errors.New("superseded run review references are incomplete")
		}
		older := refs[:0]
		for _, ref := range refs {
			if ref != eventID {
				older = append(older, ref)
			}
		}
		if !o.withdrawRunApprovalRefs(ctx, run, older, "superseded_by_new_review") {
			return eventID, errors.New("superseded run review references were not fully withdrawn")
		}
	}
	return eventID, nil
}

// WithdrawRunNeedsReview is invoked only after a verified empty diff proves
// that no operator decision is needed. It publishes producer withdrawals for
// active exact durable request references, never synthetic human answers.
func (o *Orchestrator) WithdrawRunNeedsReview(ctx context.Context, run *domain.Run, reason string) error {
	if o == nil || run == nil || run.ID == uuid.Nil {
		return errors.New("run approval withdrawal requires an orchestrator and run")
	}
	if reason != "verified_empty_diff" {
		return fmt.Errorf("unsupported run approval withdrawal reason %q", reason)
	}
	if o.approvalPublisher == nil || o.reconciler == nil {
		return errors.New("run approval withdrawal publisher or reconciler is unavailable")
	}
	requesterRefs, complete, err := o.reconciler.reviewNoticeRefsFromEvents(ctx, run.ID)
	if err != nil {
		approvalLog().Warn("run approval withdrawal reference unavailable", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
		return fmt.Errorf("read exact run approval withdrawal references: %w", err)
	}
	withdrawn := o.withdrawRunApprovalRefs(ctx, run, requesterRefs, reason)
	if !complete {
		approvalLog().Warn("run approval withdrawal reference scan is incomplete", obs.KeyRunID, run.ID.String())
		return errors.New("run approval withdrawal reference scan is incomplete")
	}
	if !withdrawn {
		return errors.New("run approval withdrawal was not fully acknowledged")
	}
	return nil
}

func (o *Orchestrator) withdrawRunApprovalRefs(ctx context.Context, run *domain.Run, requesterRefs []string, reason string) bool {
	if o == nil || o.approvalPublisher == nil || run == nil || run.ID == uuid.Nil || len(requesterRefs) == 0 {
		return len(requesterRefs) == 0
	}
	if reason != "verified_empty_diff" && reason != "superseded_by_new_review" {
		return false
	}
	complete := true
	const maxWithdrawalsPerFinalization = 50
	if len(requesterRefs) > maxWithdrawalsPerFinalization {
		approvalLog().Warn("run approval withdrawal marker tail exceeds per-finalization bound", obs.KeyRunID, run.ID.String(), "pendingReferences", len(requesterRefs), "limit", maxWithdrawalsPerFinalization)
		requesterRefs = requesterRefs[:maxWithdrawalsPerFinalization]
		complete = false
	}
	for _, requesterRef := range requesterRefs {
		event := eventbus.DomainEvent{
			Source:    runApprovalSource,
			EventType: RunApprovalWithdrawalEventType,
			Payload: map[string]any{
				"requester_ref":     requesterRef,
				"source_event_type": RunApprovalDecisionEventType,
				"reason":            reason,
				"correlation":       map[string]any{"run_id": run.ID.String()},
			},
		}
		if err := o.publishRunApprovalEvent(ctx, event); err != nil {
			approvalLog().Warn("run approval withdrawal publication failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
			complete = false
			continue
		}
		if o.events != nil {
			if err := o.events.Append(ctx, run.ID, domain.NewLogEvent(run.ID, "debug", reviewNoticeWithdrawnMarkerPrefix+requesterRef)); err != nil {
				approvalLog().Warn("run approval withdrawal marker append failed", obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
				complete = false
			}
		} else {
			complete = false
		}
	}
	return complete
}

// settleVerifiedEmptyRunReview runs only after the current sandbox owner
// provided strict verified-empty evidence. It reloads and checks the run's
// lifecycle fence before cleanup and again before writing the outcome; run-row
// updates remain fenced by LifecycleVersion. Workspace Sandbox does not expose
// a transaction spanning its diff snapshot and the Agent Manager row, so a
// sandbox change after the final verified-empty read remains a bounded TOCTOU
// limitation. The exact durable Ask references must all be withdrawn before
// the run leaves pending review; failed or incomplete cleanup remains retryable.
func (o *Orchestrator) settleVerifiedEmptyRunReview(ctx context.Context, run *domain.Run) (bool, error) {
	if o == nil || run == nil || run.ID == uuid.Nil || run.Status != domain.RunStatusNeedsReview || run.ApprovalState != domain.ApprovalStatePending {
		return false, nil
	}
	if o.reconciler == nil {
		return false, errors.New("run review reconciler is unavailable for exact Ask cleanup")
	}
	if o.runs == nil {
		return false, errors.New("run repository is unavailable for verified-empty review resolution")
	}
	classified := run
	current, err := o.runs.Get(ctx, run.ID)
	if err != nil {
		return false, fmt.Errorf("reload run before verified-empty cleanup: %w", err)
	}
	if current == nil || current.LifecycleVersion != classified.LifecycleVersion || current.Status != domain.RunStatusNeedsReview || current.ApprovalState != domain.ApprovalStatePending {
		return false, errors.New("run review lifecycle changed before verified-empty cleanup")
	}
	run = current
	refs, complete, err := o.reconciler.reviewNoticeRefsFromEvents(ctx, run.ID)
	if err != nil {
		return false, fmt.Errorf("read exact review request references: %w", err)
	}
	withdrawn := true
	if len(refs) > 0 {
		withdrawn = o.withdrawRunApprovalRefs(ctx, run, refs, "verified_empty_diff")
	}
	if !complete || !withdrawn {
		return false, errors.New("exact review request references are not fully withdrawn")
	}

	if o.runs == nil {
		return false, errors.New("run repository is unavailable for verified-empty review resolution")
	}
	latest, err := o.runs.Get(ctx, run.ID)
	if err != nil {
		return false, fmt.Errorf("reload run before verified-empty resolution: %w", err)
	}
	if latest == nil || latest.LifecycleVersion != run.LifecycleVersion || latest.Status != domain.RunStatusNeedsReview || latest.ApprovalState != domain.ApprovalStatePending {
		return false, errors.New("run review lifecycle changed during verified-empty cleanup")
	}
	run = latest
	status, known := verifiedEmptyReviewReturnStatus(run)
	run.ApprovalState = domain.ApprovalStateNone
	run.ApprovedBy = ""
	run.ApprovedAt = nil
	now := o.now()
	if known {
		markVerifiedEmptyReviewSkipped(run, "manualReview=true; verified empty diff; no approval needed", now)
		_, err := o.applyRunStatusTransition(ctx, RunStatusTransitionInput{
			Run:       run,
			NewStatus: status,
			Reason:    "verified empty sandbox diff resolved manual review",
		})
		if err != nil {
			return false, fmt.Errorf("persist verified-empty terminal outcome: %w", err)
		}
		return true, nil
	}

	// Historical rows can lack sufficient persisted terminal evidence. Clear
	// the now-unnecessary approval state but retain NeedsReview so no success,
	// failure, or cancellation is fabricated and no stale answer can apply.
	markVerifiedEmptyReviewSkipped(run, "manualReview=true; verified empty diff; terminal outcome unknown; no approval applied", now)
	if err := o.runs.Update(ctx, run); err != nil {
		return false, fmt.Errorf("persist unknown-outcome empty review resolution: %w", err)
	}
	if o.broadcaster != nil {
		o.broadcaster.BroadcastRunStatus(o.attachRunActions(ctx, run))
	}
	return true, nil
}

// markVerifiedEmptyReviewSkipped preserves an earlier finalization failure as
// durable evidence. The empty diff means approval is unnecessary; it does not
// erase a distinct failure already recorded by the finalization owner.
func markVerifiedEmptyReviewSkipped(run *domain.Run, reason string, now time.Time) {
	if run == nil || run.FinalizationStatus == domain.RunFinalizationStatusFailed {
		return
	}
	domain.MarkFinalizationSkipped(run, reason, now)
}

// verifiedEmptyReviewReturnStatus restores a terminal outcome only from
// explicit, internally consistent runner/status evidence. Missing or
// contradictory evidence stays a NeedsReview row with ApprovalStateNone.
func verifiedEmptyReviewReturnStatus(run *domain.Run) (domain.RunStatus, bool) {
	if run == nil || run.EndedAt == nil || run.Result == nil {
		return "", false
	}
	result := run.Result
	if domain.TerminalClassForStopReason(run.StopReason) == "" || run.TerminalClass != domain.TerminalClassForStopReason(run.StopReason) {
		return "", false
	}
	switch result.TerminalReason {
	case "completed", "goal_complete":
		if result.Success && result.ExitCode == 0 && run.StopReason == domain.RunStopReasonComplete && run.TerminalClass == domain.RunTerminalClassVerdict {
			return domain.RunStatusComplete, true
		}
	case "goal_blocked":
		if result.Success && result.ExitCode == 0 && run.StopReason == domain.RunStopReasonBlocked && run.TerminalClass == domain.RunTerminalClassVerdict {
			return domain.RunStatusComplete, true
		}
	case "failed":
		if !result.Success && result.ExitCode != 0 && run.TerminalClass == domain.RunTerminalClassInterruption &&
			(run.StopReason == domain.RunStopReasonTimeout || run.StopReason == domain.RunStopReasonCrash || run.StopReason == domain.RunStopReasonSessionLost || run.StopReason == domain.RunStopReasonUsageWindow) {
			return domain.RunStatusFailed, true
		}
	case "timeout":
		if !result.Success && result.ExitCode != 0 && run.StopReason == domain.RunStopReasonTimeout && run.TerminalClass == domain.RunTerminalClassInterruption {
			return domain.RunStatusFailed, true
		}
	case "goal_usage_limited":
		if !result.Success && result.ExitCode != 0 && run.StopReason == domain.RunStopReasonUsageWindow && run.TerminalClass == domain.RunTerminalClassInterruption {
			return domain.RunStatusFailed, true
		}
	case "goal_budget_limited", "cancelled":
		// Token-cap termination is the only persisted typed cancellation reason.
		// A cancellation request timestamp by itself is intent, not proof that
		// the runner actually ended in the cancelled state.
		if !result.Success && result.ExitCode != 0 && run.StopReason == domain.RunStopReasonTokenCap && run.TerminalClass == domain.RunTerminalClassInterruption {
			return domain.RunStatusCancelled, true
		}
	}
	return "", false
}

func (o *Orchestrator) publishRunApprovalEvent(ctx context.Context, event eventbus.DomainEvent) error {
	if o == nil || o.approvalPublisher == nil {
		return errors.New("run approval decision publisher is unavailable")
	}
	return o.approvalPublisher.PublishDomainEvent(ctx, event)
}

func (o *Orchestrator) beginRunReviewNotice(runID uuid.UUID) uint64 {
	if o == nil || o.reconciler == nil {
		return 0
	}
	return o.reconciler.beginReviewNoticeAttempt(runID)
}

func (o *Orchestrator) recordRunReviewNotice(runID uuid.UUID, eventID string, attemptID uint64, err error) {
	if o == nil || o.reconciler == nil || attemptID == 0 {
		return
	}
	if err != nil {
		o.reconciler.recordReviewNoticeFailure(runID, eventID, attemptID, o.reconciler.now())
		return
	}
	if eventID == verifiedEmptyReviewNoticeSentinel {
		o.reconciler.recordReviewNoticeEmptyCheck(runID, attemptID)
		return
	}
	o.reconciler.recordReviewNoticePublished(runID, eventID, attemptID)
}

// currentRunApprovalEvent reads the current sandbox diff and builds the exact
// decision generation. The matching resolution recomputes this snapshot before
// effects, so a changed diff cannot reuse an older answer.
func (o *Orchestrator) currentRunApprovalEvent(ctx context.Context, run *domain.Run) (eventbus.DomainEvent, error) {
	event, _, err := o.currentRunApprovalDecision(ctx, run)
	return event, err
}

// currentRunApprovalDecision is the single read/classification seam for
// notification retries and answers. Only the sandbox owner's complete,
// identity-bound evidence can classify a review as verified empty; missing or
// unavailable evidence continues through the ordinary review path.
func (o *Orchestrator) currentRunApprovalDecision(ctx context.Context, run *domain.Run) (eventbus.DomainEvent, bool, error) {
	if run == nil || run.SandboxID == nil || o.sandbox == nil {
		return eventbus.DomainEvent{}, false, errors.New("run approval requires a sandboxed run with a readable diff")
	}
	diff, err := o.sandbox.GetDiff(ctx, *run.SandboxID)
	if err != nil {
		return eventbus.DomainEvent{}, false, fmt.Errorf("read current sandbox diff: %w", err)
	}
	if diff == nil || diff.SandboxID != *run.SandboxID {
		return eventbus.DomainEvent{}, false, errors.New("sandbox diff does not match the run sandbox")
	}
	if sandbox.IsVerifiedEmptyDiff(diff, *run.SandboxID) {
		return eventbus.DomainEvent{}, true, nil
	}
	event, err := o.runApprovalEvent(ctx, run, diff)
	if err != nil {
		return eventbus.DomainEvent{}, false, err
	}
	return event, false, nil
}

func (o *Orchestrator) reconcilePublishedEmptyReview(ctx context.Context, run *domain.Run) (bool, error) {
	_, verifiedEmpty, err := o.currentRunApprovalDecision(ctx, run)
	if err != nil || !verifiedEmpty {
		return false, err
	}
	if _, err := o.settleVerifiedEmptyRunReview(ctx, run); err != nil {
		return true, err
	}
	return true, nil
}

// runApprovalEvent builds the decision_requested event for a run awaiting
// review via the shared operatordecision substrate. The facts are the facts-only
// producer contract notification-hub parses (see
// scenarios/notification-hub/.../decision_events.go: decisionFacts). It emits the
// established RunApprovalDecisionEventType (underscore source token) through
// BuildEventTyped so the Source stays "agent-manager" (hyphen) and the bytes are
// compatibility of the established source and event contract.
func (o *Orchestrator) runApprovalEvent(ctx context.Context, run *domain.Run, diff *sandbox.DiffResult) (eventbus.DomainEvent, error) {
	generation, err := runApprovalGeneration(run, diff)
	if err != nil {
		return eventbus.DomainEvent{}, err
	}
	changedFiles := diff.Stats.FilesChanged
	if changedFiles == 0 && len(diff.Files) > 0 {
		changedFiles = len(diff.Files)
	}
	changeCount := pluralizeFiles(changedFiles)
	question := fmt.Sprintf("Apply the changes from this run? %s changed.", changeCount)
	technicalDetails := fmt.Sprintf("This sandboxed run currently shows %s in its diff. Review the diff and run results before choosing. No changes are applied without an explicit approval, and this decision is not broader task or goal acceptance.", changeCount)
	if changedFiles <= 0 {
		// An incomplete empty-looking payload is not verified proof that no
		// files changed. Keep the review visible without claiming a zero diff.
		question = "Apply the changes from this run? The complete file count is unavailable."
		technicalDetails = "This sandboxed run currently has an empty-looking diff payload whose complete file count is unavailable. Review the diff and run results before choosing. No changes are applied without an explicit approval, and this decision is not broader task or goal acceptance."
	}
	label := approvalRunLabel(run)
	subject := "Run " + shortRunID(run.ID.String()) + " needs review"
	if label != "" {
		subject += ": " + label
	}
	bigPicture := "Review the proposed sandbox changes before they are applied. Approving applies them to the canonical repository; rejecting discards the sandbox changes."
	if label != "" {
		bigPicture = fmt.Sprintf("Review the proposed sandbox changes for %q before they are applied. Approving applies them to the canonical repository; rejecting discards the sandbox changes.", label)
		if changedFiles <= 0 {
			technicalDetails = fmt.Sprintf("Run %q currently has an empty-looking diff payload whose complete file count is unavailable. Review the diff and run results before choosing. No changes are applied without an explicit approval, and this decision is not broader task or goal acceptance.", label)
		} else {
			technicalDetails = fmt.Sprintf("Run %q currently shows %s in its sandbox diff. Review the diff and run results before choosing. No changes are applied without an explicit approval, and this decision is not broader task or goal acceptance.", label, changeCount)
		}
	}
	facts := operatordecision.Facts{
		Subject:  subject,
		Question: question,
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Approve & apply"},
			{Key: "reject", Label: "Reject & discard"},
		},
		Recommended: "approve",
		// reversible=true: a reviewed run can be re-run; no silent default is set so
		// nothing is ever applied without an explicit operator answer.
		Reversible:       true,
		Urgency:          "normal",
		BigPicture:       bigPicture,
		TechnicalDetails: technicalDetails,
		ContextURL:       reviewURLForBase(o.resolveWebConsoleNoticeBase(ctx), run),
		Correlation: map[string]string{
			"run_id":              run.ID.String(),
			"review_generation":   generation,
			"review_patch_sha256": diff.PatchSHA256,
		},
	}

	// A stable occurrence time keeps the event id (SHA-256 of the event)
	// deterministic so a redelivery (or a re-raise) returns the SAME ask instead
	// of a duplicate operator notification. A needs_review run has no ApprovedAt
	// yet, so prefer EndedAt — the terminal transition time set at the run-end
	// seam that raised this notification (phases.ApplyAtRunEnd). Fall back to
	// ApprovedAt (re-notify after approval), then to now() bucketed to the ask
	// window, matching the authorization path's deterministic truncation.
	var occurred time.Time
	switch {
	case run.EndedAt != nil:
		occurred = run.EndedAt.UTC()
	case run.ApprovedAt != nil:
		occurred = run.ApprovedAt.UTC()
	default:
		occurred = o.now().UTC().Truncate(pendingAuthorizationWindow)
	}

	return operatordecision.BuildEventTyped(runApprovalSource, RunApprovalDecisionEventType, occurred, facts), nil
}

// runApprovalGeneration binds the out-of-band answer to the persisted run
// lifecycle/artifact references and canonical current diff contents. DiffResult.Generated
// is intentionally excluded because the provider stamps each read independently.
func runApprovalGeneration(run *domain.Run, diff *sandbox.DiffResult) (string, error) {
	if run == nil || run.SandboxID == nil || diff == nil || diff.SandboxID != *run.SandboxID {
		return "", errors.New("run approval generation requires a matching sandbox diff")
	}
	files := append([]sandbox.FileChange(nil), diff.Files...)
	sort.Slice(files, func(i, j int) bool {
		left, _ := json.Marshal(files[i])
		right, _ := json.Marshal(files[j])
		return string(left) < string(right)
	})
	digestBytes, digestErr := hex.DecodeString(diff.PatchSHA256)
	if digestErr != nil || len(digestBytes) != sha256.Size || strings.ToLower(diff.PatchSHA256) != diff.PatchSHA256 {
		return "", errors.New("sandbox diff has no canonical patch SHA-256")
	}
	diffSnapshot := struct {
		SandboxID    uuid.UUID
		PatchSHA256  string
		Files        []sandbox.FileChange
		Stats        sandbox.DiffStats
		ArchiveState sandbox.ArchiveState
	}{diff.SandboxID, diff.PatchSHA256, files, diff.Stats, generationArchiveState(diff.ArchiveState)}
	diffBytes, err := json.Marshal(diffSnapshot)
	if err != nil {
		return "", fmt.Errorf("encode sandbox diff snapshot: %w", err)
	}
	diffHash := sha256.Sum256(diffBytes)
	endedAt := ""
	if run.EndedAt != nil {
		endedAt = run.EndedAt.UTC().Format(time.RFC3339Nano)
	}
	approvedAt := ""
	if run.ApprovedAt != nil {
		approvedAt = run.ApprovedAt.UTC().Format(time.RFC3339Nano)
	}
	terms := struct {
		RunID          string
		EndedAt        string
		ApprovedAt     string
		SandboxID      string
		DiffPath       string
		ChangedFiles   int
		TotalSizeBytes int64
		CommitHash     string
		DiffSHA256     string
	}{run.ID.String(), endedAt, approvedAt, run.SandboxID.String(), run.DiffPath, run.ChangedFiles, run.TotalSizeBytes, run.CommitHash, hex.EncodeToString(diffHash[:])}
	termsBytes, err := json.Marshal(terms)
	if err != nil {
		return "", fmt.Errorf("encode run approval generation: %w", err)
	}
	sum := sha256.Sum256(termsBytes)
	return hex.EncodeToString(sum[:]), nil
}

// A live diff and its later durable archive may describe exactly the same
// patch. Archive materialization is provenance availability, not a change to
// decision terms, so canonicalize only complete to the live marker. Every
// other diff field remains included in the digest; not_captured and unknown
// states remain distinct.
func generationArchiveState(state sandbox.ArchiveState) sandbox.ArchiveState {
	if state == sandbox.ArchiveStateComplete {
		return ""
	}
	return state
}

// approvalRunLabel uses the label copied onto the run at admission. It avoids
// a task lookup during notification and keeps this context stable across retries.
func approvalRunLabel(run *domain.Run) string {
	if run == nil {
		return ""
	}
	value := strings.Join(strings.Fields(run.Label), " ")
	runes := []rune(value)
	if len(runes) > 140 {
		return string(runes[:137]) + "…"
	}
	return value
}

// reviewURLForBase builds a browser link to the run's review page in
// web-console. Empty when base is empty (the notification then omits the link
// and the operator opens the AI Changes queue directly).
func reviewURLForBase(base string, run *domain.Run) string {
	if base == "" || run == nil {
		return ""
	}
	return strings.TrimRight(base, "/") + "/?run=" + url.QueryEscape(run.ID.String())
}

func shortRunID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

func pluralizeFiles(n int) string {
	if n == 1 {
		return "1 file"
	}
	return fmt.Sprintf("%d files", n)
}

// ResolveRunApprovalFromHub applies a notification-hub ask resolution that
// answers a run-approval decision this run raised. It reports handled=true when
// the resolution belongs to a run approval (so the caller stops routing it to
// the effort-decision resolver), and false when it is some other decision.
//
// Idempotent: a redelivered resolution for a run that already left needs_review
// is a benign no-op. Only a concrete "answered" state acts; expired/defaulted
// resolutions leave the run in needs_review for manual handling.
func (o *Orchestrator) ResolveRunApprovalFromHub(ctx context.Context, res supervision.DecisionResolution) (bool, error) {
	if o == nil || res.SourceEventType != RunApprovalDecisionEventType {
		return false, nil
	}
	if res.State != "answered" {
		approvalLog().Info("run approval resolution ignored (no operator answer)",
			"state", res.State, "ask_id", res.AskID)
		return true, nil
	}
	runID, err := uuid.Parse(strings.TrimSpace(res.Correlation["run_id"]))
	if err != nil {
		return true, fmt.Errorf("run approval resolution lacks a valid run_id correlation: %w", err)
	}

	// Idempotency: if the run already left needs_review, a duplicate delivery
	// must not error or re-apply. Reading state first also avoids a confusing
	// state-error log on the common redelivery path.
	run, err := o.GetRun(ctx, runID)
	if err != nil {
		// A resolution for a run that no longer exists is benign, not a server
		// error: returning 500 would make vrooli-events retry a poison message
		// forever. Acknowledge and drop it.
		var notFound *domain.NotFoundError
		if errors.As(err, &notFound) {
			approvalLog().Info("run approval resolution references an unknown run; ignoring",
				obs.KeyRunID, runID.String())
			return true, nil
		}
		return true, err
	}
	if run.Status != domain.RunStatusNeedsReview || run.ApprovalState != domain.ApprovalStatePending {
		approvalLog().Info("run approval resolution is a no-op; run has no pending manual approval",
			obs.KeyRunID, runID.String(), "status", string(run.Status), "approval_state", string(run.ApprovalState))
		return true, nil
	}

	// Bind the answer to the exact current decision event. A run ID alone is not
	// a generation: the run may re-enter needs_review with different terms. The
	// event hash includes the run lifecycle and current sandbox patch digest,
	// plus the resolved review URL. Changed content or URL fails closed. Missing
	// refs from legacy resolutions are deliberately not treated as approval.
	// Reserve before reading the snapshot: a slow older read must not be
	// assigned a newer ticket than a later generation's publication.
	attemptID := o.beginRunReviewNotice(runID)
	currentEvent, verifiedEmpty, err := o.currentRunApprovalDecision(ctx, run)
	if err != nil {
		return true, fmt.Errorf("read current run approval generation: %w", err)
	}
	if verifiedEmpty {
		// The resolution is consumed without applying it. The reconciler retries
		// any exact-reference withdrawal that is not yet durably acknowledged.
		if _, settleErr := o.settleVerifiedEmptyRunReview(ctx, run); settleErr != nil {
			return true, fmt.Errorf("settle verified-empty run review: %w", settleErr)
		}
		return true, nil
	}
	currentEventID := operatordecision.EventID(currentEvent)
	actor := strings.TrimSpace(res.AnsweredBy)
	if actor == "" {
		return true, errors.New("run approval resolution carries no answer attribution from Notification Hub")
	}
	if res.RequesterRef == "" {
		approvalLog().Info("run approval resolution has no request reference; ignoring",
			obs.KeyRunID, runID.String(), "ask_id", res.AskID)
		return true, nil
	}
	if res.RequesterRef != currentEventID {
		// The old answer cannot authorize the current diff. Publish the current
		// generation before acknowledging it, so the operator retains a live
		// decision request after the stale Ask is consumed.
		publishedID, publishErr := o.publishCurrentRunApprovalDecision(ctx, run, currentEvent)
		if publishErr != nil {
			o.recordRunReviewNotice(runID, publishedID, attemptID, publishErr)
			return true, fmt.Errorf("republish current run approval generation: %w", publishErr)
		}
		o.recordRunReviewNotice(runID, publishedID, attemptID, nil)
		approvalLog().Info("stale run approval answer ignored; current generation republished",
			obs.KeyRunID, runID.String(), "ask_id", res.AskID)
		return true, nil
	}
	correlation, _ := currentEvent.Payload["correlation"].(map[string]any)
	currentPatchSHA256, _ := correlation["review_patch_sha256"].(string)
	if strings.TrimSpace(currentPatchSHA256) == "" {
		return true, errors.New("current run approval event has no reviewed patch digest")
	}
	switch res.Answer {
	case "approve":
		_, err = o.ApproveRun(ctx, ApproveRequest{
			RunID:               runID,
			Actor:               actor,
			CommitMsg:           "Approved via notification-hub by " + actor,
			ExpectedPatchSHA256: currentPatchSHA256,
		})
	case "reject":
		reason := strings.TrimSpace(res.Note)
		if reason == "" {
			reason = "rejected via notification-hub by " + actor
		}
		err = o.RejectRunWithExpectedPatch(ctx, runID, actor, reason, currentPatchSHA256)
	default:
		return true, fmt.Errorf("run approval resolution has an unknown answer %q", res.Answer)
	}
	if err != nil {
		return true, fmt.Errorf("apply run approval (%s): %w", res.Answer, err)
	}
	approvalLog().Info("run approval applied from notification-hub",
		obs.KeyRunID, runID.String(), "answer", res.Answer, "actor", actor)
	return true, nil
}
