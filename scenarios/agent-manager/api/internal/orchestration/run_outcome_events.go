// This file implements out-of-band operator notification for runs that reach a
// terminal, non-review outcome. When a run completes or fails without manual
// review, it publishes a generic *.notification.v1 event onto Vrooli Events.
// notification-hub turns that neutral notificationFacts payload into a "Run
// completed" / "Run failed" operator notification per the operator's catalog
// preferences (see scenarios/agent-manager/.vrooli/notifications.json) — no
// per-scenario code in the hub.
//
// A needs_review run is NOT notified here; it already raises an Approve/Reject
// Ask via the decision path (run_approval_events.go). The ApplyAtRunEnd seam
// skips the needs_review status before invoking NotifyRunOutcome, so a run is
// never double-notified.
package orchestration

import (
	"context"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/obs"

	"github.com/vrooli/api-core/eventbus"
)

// RunOutcomeNotificationEventType is the generic notification event a run raises
// when it reaches a terminal, non-review outcome. The `.notification.v1` suffix
// is the generic contract notification-hub subscribes to (**.notification.v1),
// so no hub change is needed to surface the notification.
const RunOutcomeNotificationEventType = "agent_manager.run.notification.v1"

// Notification categories. These MUST match the category keys declared in
// scenarios/agent-manager/.vrooli/notifications.json so the operator's catalog
// preferences apply.
const (
	runOutcomeCategoryCompleted = "run-completed"
	runOutcomeCategoryFailed    = "run-failed"
)

// NotifyRunOutcome satisfies phases.ApprovalNotifier. It is invoked from the
// single run-end seam (phases.ApplyAtRunEnd) once a run reaches a terminal,
// non-review status (Completed or Failed). Delivery is best-effort: a publish
// failure is logged and never blocks the run's terminal transition.
func (o *Orchestrator) NotifyRunOutcome(ctx context.Context, run *domain.Run) {
	if o == nil || o.approvalPublisher == nil || run == nil {
		return
	}
	if run.Status != domain.RunStatusComplete && run.Status != domain.RunStatusFailed {
		return
	}
	event := o.runOutcomeEvent(ctx, run)
	if err := o.approvalPublisher.PublishDomainEvent(ctx, event); err != nil {
		approvalLog().Warn("run outcome notification event publish failed",
			obs.KeyRunID, run.ID.String(), obs.KeyError, err.Error())
	}
}

// runOutcomeEvent builds the *.notification.v1 event for a terminal, non-review
// run. The payload is the neutral notificationFacts producer contract
// notification-hub parses (title, body, category, urgency, sensitivity,
// context_url, dedupe_key, correlation).
func (o *Orchestrator) runOutcomeEvent(ctx context.Context, run *domain.Run) eventbus.DomainEvent {
	short := shortRunID(run.ID.String())

	var title, body, category, urgency string
	if run.Status == domain.RunStatusFailed {
		category = runOutcomeCategoryFailed
		urgency = "high"
		title = "Run " + short + " failed"
		body = runFailureBody(run)
	} else {
		category = runOutcomeCategoryCompleted
		urgency = "low"
		title = "Run " + short + " completed"
		body = pluralizeFiles(run.ChangedFiles) + " changed."
	}

	payload := map[string]any{
		"title":       title,
		"body":        body,
		"category":    category,
		"urgency":     urgency,
		"sensitivity": "private",
		"dedupe_key":  run.ID.String(),
		"correlation": map[string]any{"run_id": run.ID.String()},
	}
	if link := reviewURLForBase(o.resolveWebConsoleNoticeBase(ctx), run); link != "" {
		payload["context_url"] = link
	}

	// A stable occurrence time keeps the event id (SHA-256 of the event)
	// deterministic for the common single-finalization path, so a redelivery
	// returns the same notification instead of a duplicate.
	occurred := o.now().UTC()
	if run.ApprovedAt != nil {
		occurred = run.ApprovedAt.UTC()
	}

	return eventbus.DomainEvent{
		Source:    runApprovalSource,
		EventType: RunOutcomeNotificationEventType,
		Occurred:  occurred,
		Payload:   payload,
	}
}

// runFailureBody derives a concise failure summary for the notification body,
// preferring the run's captured error message and falling back to a generic
// line when none is recorded.
func runFailureBody(run *domain.Run) string {
	msg := strings.TrimSpace(run.ErrorMsg)
	if msg == "" {
		return "The run ended in error."
	}
	const max = 240
	if len(msg) > max {
		msg = msg[:max] + "…"
	}
	return msg
}
