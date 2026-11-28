package orchestration

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/eventbus"
)

// TestRunOutcomeEventCompletedShape pins a successful run's emitted event to
// notification-hub's *.notification.v1 producer contract (notificationFacts):
// run-completed category, low urgency, a title/body, a run_id correlation, and
// a context_url that references the run when a UI base is resolved.
func TestRunOutcomeEventCompletedShape(t *testing.T) {
	runID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	o := &Orchestrator{
		webConsoleUIBase: "http://localhost:21238/",
		clock:            func() time.Time { return time.Unix(1000, 0).UTC() },
	}
	run := &domain.Run{ID: runID, Status: domain.RunStatusComplete, ChangedFiles: 3}

	ev := o.runOutcomeEvent(context.Background(), run)

	if ev.EventType != RunOutcomeNotificationEventType {
		t.Fatalf("event type = %q, want %q", ev.EventType, RunOutcomeNotificationEventType)
	}
	if !strings.HasSuffix(ev.EventType, ".notification.v1") {
		t.Fatalf("event type %q must ride the generic notification suffix", ev.EventType)
	}
	if ev.Source != "agent-manager" {
		t.Fatalf("source = %q, want agent-manager", ev.Source)
	}

	p := ev.Payload
	if p["category"] != runOutcomeCategoryCompleted {
		t.Fatalf("category = %v, want %s", p["category"], runOutcomeCategoryCompleted)
	}
	if p["urgency"] != "low" {
		t.Fatalf("urgency = %v, want low", p["urgency"])
	}
	if p["sensitivity"] != "private" {
		t.Fatalf("sensitivity = %v, want private", p["sensitivity"])
	}
	if title, _ := p["title"].(string); strings.TrimSpace(title) == "" || !strings.Contains(title, "completed") {
		t.Fatalf("title missing/wrong: %q", p["title"])
	}
	if body, _ := p["body"].(string); strings.TrimSpace(body) == "" || !strings.Contains(body, "3 files") {
		t.Fatalf("body missing/wrong file count: %q", p["body"])
	}
	if p["dedupe_key"] != runID.String() {
		t.Fatalf("dedupe_key = %v, want %s", p["dedupe_key"], runID.String())
	}
	corr, ok := p["correlation"].(map[string]any)
	if !ok || corr["run_id"] != runID.String() {
		t.Fatalf("correlation.run_id = %v, want %s", p["correlation"], runID.String())
	}
	if url, _ := p["context_url"].(string); !strings.Contains(url, runID.String()) {
		t.Fatalf("context_url %q must reference the run id", p["context_url"])
	}
}

// TestRunOutcomeEventFailedShape pins a failed run to the run-failed category
// with high urgency and the failure reason surfaced in the body.
func TestRunOutcomeEventFailedShape(t *testing.T) {
	runID := uuid.New()
	// No override and no resolvable base: context_url must be omitted.
	restore := resolveExternalURLForNotice
	resolveExternalURLForNotice = func(context.Context, string) (string, error) { return "", nil }
	t.Cleanup(func() { resolveExternalURLForNotice = restore })

	o := &Orchestrator{clock: func() time.Time { return time.Unix(1000, 0).UTC() }}
	run := &domain.Run{ID: runID, Status: domain.RunStatusFailed, ErrorMsg: "agent exited non-zero"}

	ev := o.runOutcomeEvent(context.Background(), run)

	if !strings.HasSuffix(ev.EventType, ".notification.v1") {
		t.Fatalf("event type %q must ride the generic notification suffix", ev.EventType)
	}
	p := ev.Payload
	if p["category"] != runOutcomeCategoryFailed {
		t.Fatalf("category = %v, want %s", p["category"], runOutcomeCategoryFailed)
	}
	if p["urgency"] != "high" {
		t.Fatalf("urgency = %v, want high", p["urgency"])
	}
	if title, _ := p["title"].(string); !strings.Contains(title, "failed") {
		t.Fatalf("title %q must mark the run failed", p["title"])
	}
	if body, _ := p["body"].(string); !strings.Contains(body, "agent exited non-zero") {
		t.Fatalf("body %q must surface the failure reason", p["body"])
	}
	// No UI base resolved -> no context_url.
	if _, ok := p["context_url"]; ok {
		t.Fatalf("context_url must be omitted when no UI base is resolved")
	}
	corr, _ := p["correlation"].(map[string]any)
	if corr["run_id"] != runID.String() {
		t.Fatalf("correlation.run_id = %v, want %s", corr["run_id"], runID.String())
	}
}

// TestNotifyRunOutcomeSkipsNonTerminal ensures NotifyRunOutcome is a no-op for
// a needs_review run (notified via the decision path) so no event is published.
func TestNotifyRunOutcomeSkipsNonTerminal(t *testing.T) {
	pub := &capturingPublisher{}
	o := &Orchestrator{approvalPublisher: pub, clock: func() time.Time { return time.Unix(1000, 0).UTC() }}

	o.NotifyRunOutcome(t.Context(), &domain.Run{ID: uuid.New(), Status: domain.RunStatusNeedsReview})
	if pub.count != 0 {
		t.Fatalf("needs_review must not raise an outcome notification; published %d", pub.count)
	}

	o.NotifyRunOutcome(t.Context(), &domain.Run{ID: uuid.New(), Status: domain.RunStatusComplete})
	if pub.count != 1 {
		t.Fatalf("a completed run must raise exactly one notification; published %d", pub.count)
	}
}

type capturingPublisher struct {
	count int
}

func (c *capturingPublisher) PublishDomainEvent(_ context.Context, _ eventbus.DomainEvent) error {
	c.count++
	return nil
}
