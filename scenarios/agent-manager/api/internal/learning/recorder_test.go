package learning

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/investigation"
)

func TestInvestigationEvidenceRefsAreStableAndScoped(t *testing.T) {
	item := &investigation.Lifecycle{ID: "inv-1", Request: investigation.Request{Subject: investigation.Subject{Owner: "agent-manager", Kind: "run-set", Ref: "subject-1"}}, Result: &investigation.Result{
		EvidenceRefs:  []investigation.EvidenceReference{{Owner: "agent-manager", Kind: "workflow", Ref: "wf-1"}},
		Coverage:      []investigation.CoveragePlane{{Plane: "events", ThroughRef: "event-9"}},
		EvidenceCut:   investigation.EvidenceCut{CapturedAt: "2026-09-06T12:00:00Z"},
		Applicability: investigation.Applicability{CheckedAt: "2026-09-06T12:00:01Z"},
		Learning:      investigation.Learning{AttemptID: "inv-1/attempt-1", CaptureState: "pending", AdviceVerdict: "unknown"},
	}}
	got := investigationEvidenceRefs(item)
	if len(got) != 2 || got[0] != "agent-manager:workflow:wf-1" || got[1] != "coverage:events:event-9" {
		t.Fatalf("refs=%v", got)
	}
	_, started, finished := investigation.StableLearningAttempt(item)
	if !started.Equal(time.Date(2026, 9, 6, 12, 0, 0, 0, time.UTC)) || !finished.Equal(time.Date(2026, 9, 6, 12, 0, 1, 0, time.UTC)) {
		t.Fatalf("timestamps=%s %s", started, finished)
	}
	_ = context.Background()
}
