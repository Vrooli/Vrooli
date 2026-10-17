package orchestration

import (
	"context"
	"strings"
	"testing"
	"time"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
	"github.com/vrooli/api-core/operatordecision"
)

// TestRunApprovalEventShape pins the emitted event to notification-hub's
// *.decision_requested.v1 producer contract (decisionFacts): a non-empty
// question, >=2 options, a recommended option key, and a run_id correlation.
func TestRunApprovalEventShape(t *testing.T) {
	runID := uuid.MustParse("11111111-2222-3333-4444-555555555555")
	o := &Orchestrator{
		webConsoleUIBase: "http://localhost:21238/",
		clock:            func() time.Time { return time.Unix(1000, 0).UTC() },
	}
	run := &domain.Run{ID: runID, ChangedFiles: 3}

	ev := o.runApprovalEvent(context.Background(), run)

	if ev.EventType != RunApprovalDecisionEventType {
		t.Fatalf("event type = %q, want %q", ev.EventType, RunApprovalDecisionEventType)
	}
	if ev.Source != "agent-manager" {
		t.Fatalf("source = %q, want agent-manager", ev.Source)
	}
	if !strings.HasSuffix(ev.EventType, ".decision_requested.v1") {
		t.Fatalf("event type %q must ride the generic decision suffix", ev.EventType)
	}

	p := ev.Payload
	if q, _ := p["question"].(string); strings.TrimSpace(q) == "" || !strings.Contains(q, "3 files") {
		t.Fatalf("question missing or wrong file count: %q", p["question"])
	}
	if p["recommended"] != "approve" {
		t.Fatalf("recommended = %v, want approve", p["recommended"])
	}
	if p["reversible"] != true {
		t.Fatalf("reversible = %v, want true", p["reversible"])
	}
	if _, hasDefault := p["default_answer"]; hasDefault {
		t.Fatalf("run approval must not set a default_answer (never auto-apply)")
	}

	opts, ok := p["options"].([]any)
	if !ok || len(opts) != 2 {
		t.Fatalf("options = %v, want 2 entries", p["options"])
	}
	keys := map[string]bool{}
	for _, o := range opts {
		m, _ := o.(map[string]any)
		key, _ := m["key"].(string)
		if key == "" || m["label"] == "" {
			t.Fatalf("option missing key/label: %v", o)
		}
		keys[key] = true
	}
	if !keys["approve"] || !keys["reject"] {
		t.Fatalf("options must include approve and reject, got %v", keys)
	}

	corr, ok := p["correlation"].(map[string]any)
	if !ok || corr["run_id"] != runID.String() {
		t.Fatalf("correlation.run_id = %v, want %s", p["correlation"], runID.String())
	}
	if url, _ := p["context_url"].(string); !strings.Contains(url, runID.String()) {
		t.Fatalf("context_url %q must reference the run id", p["context_url"])
	}
}

// TestRunApprovalEventOmitsLinkWithoutUIBase ensures a missing web-console base
// drops context_url rather than emitting an empty link.
func TestRunApprovalEventOmitsLinkWithoutUIBase(t *testing.T) {
	// No override and no resolvable base: context_url must be omitted.
	restore := resolveExternalURLForNotice
	resolveExternalURLForNotice = func(context.Context, string) (string, error) { return "", nil }
	t.Cleanup(func() { resolveExternalURLForNotice = restore })

	o := &Orchestrator{clock: func() time.Time { return time.Unix(1000, 0).UTC() }}
	ev := o.runApprovalEvent(context.Background(), &domain.Run{ID: uuid.New(), ChangedFiles: 1})
	if _, ok := ev.Payload["context_url"]; ok {
		t.Fatalf("context_url must be omitted when no UI base is resolved")
	}
	if q, _ := ev.Payload["question"].(string); !strings.Contains(q, "1 file") || strings.Contains(q, "1 files") {
		t.Fatalf("singular file count wrong: %q", q)
	}
}

// TestRunApprovalEventOccurredIsDeterministicForNeedsReview pins that a
// needs_review run (no ApprovedAt yet) derives Occurred from the stable EndedAt
// transition time, so two builds across a redelivery produce the SAME EventID —
// what makes the hub deduplicate instead of raising a duplicate ask. It also
// pins that a moving wall clock does NOT change the EventID.
func TestRunApprovalEventOccurredIsDeterministicForNeedsReview(t *testing.T) {
	runID := uuid.MustParse("22222222-3333-4444-5555-666666666666")
	ended := time.Date(2026, 10, 16, 8, 0, 0, 0, time.UTC)
	run := &domain.Run{ID: runID, ChangedFiles: 2, EndedAt: &ended}

	// Two Orchestrators with DIFFERENT wall clocks — EventID must be identical.
	o1 := &Orchestrator{webConsoleUIBase: "http://localhost:21238/", clock: func() time.Time { return time.Unix(1000, 0).UTC() }}
	o2 := &Orchestrator{webConsoleUIBase: "http://localhost:21238/", clock: func() time.Time { return time.Unix(9999, 0).UTC() }}
	ev1 := o1.runApprovalEvent(context.Background(), run)
	ev2 := o2.runApprovalEvent(context.Background(), run)
	if !ev1.Occurred.Equal(ended) {
		t.Fatalf("Occurred must be the stable EndedAt, got %v want %v", ev1.Occurred, ended)
	}
	if operatordecision.EventID(ev1) != operatordecision.EventID(ev2) {
		t.Fatalf("EventID must be deterministic across wall clocks for a needs_review run: %s != %s", operatordecision.EventID(ev1), operatordecision.EventID(ev2))
	}
}
