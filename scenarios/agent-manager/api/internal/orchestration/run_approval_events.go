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
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

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
	event := o.runApprovalEvent(ctx, run)
	if err := o.approvalPublisher.PublishDomainEvent(ctx, event); err != nil {
		approvalLog().Warn("run approval decision event publish failed",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

// runApprovalEvent builds the decision_requested event for a run awaiting
// review via the shared operatordecision substrate. The facts are the facts-only
// producer contract notification-hub parses (see
// scenarios/notification-hub/.../decision_events.go: decisionFacts). It emits the
// established RunApprovalDecisionEventType (underscore source token) through
// BuildEventTyped so the Source stays "agent-manager" (hyphen) and the bytes are
// unchanged from the pre-substrate hand-built payload.
func (o *Orchestrator) runApprovalEvent(ctx context.Context, run *domain.Run) eventbus.DomainEvent {
	facts := operatordecision.Facts{
		Subject:  "Run " + shortRunID(run.ID.String()) + " needs review",
		Question: fmt.Sprintf("Apply the changes from this run? %s changed.", pluralizeFiles(run.ChangedFiles)),
		Options: []operatordecision.Option{
			{Key: "approve", Label: "Approve & apply"},
			{Key: "reject", Label: "Reject & discard"},
		},
		Recommended: "approve",
		// reversible=true: a reviewed run can be re-run; no silent default is set so
		// nothing is ever applied without an explicit operator answer.
		Reversible:  true,
		Urgency:     "normal",
		ContextURL:  reviewURLForBase(o.resolveWebConsoleNoticeBase(ctx), run),
		Correlation: map[string]string{"run_id": run.ID.String()},
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

	return operatordecision.BuildEventTyped(runApprovalSource, RunApprovalDecisionEventType, occurred, facts)
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
	if run.Status != domain.RunStatusNeedsReview {
		approvalLog().Info("run approval resolution is a no-op; run already resolved",
			obs.KeyRunID, runID.String(), "status", string(run.Status))
		return true, nil
	}

	actor := strings.TrimSpace(res.AnsweredBy)
	if actor == "" {
		actor = "operator"
	}
	switch res.Answer {
	case "approve":
		_, err = o.ApproveRun(ctx, ApproveRequest{
			RunID:     runID,
			Actor:     actor,
			CommitMsg: "Approved via notification-hub by " + actor,
		})
	case "reject":
		reason := strings.TrimSpace(res.Note)
		if reason == "" {
			reason = "rejected via notification-hub by " + actor
		}
		err = o.RejectRun(ctx, runID, actor, reason)
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
