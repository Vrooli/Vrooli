package effortdecision

import (
	"testing"
	"time"

	pb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRequestedEventCarriesExplicitContextAndKeepsAnswerNoteSeparate(t *testing.T) {
	decision := &pb.EffortDecision{
		GoalHome: "scenarios/demo/GOAL.md", DecisionId: "decision-1",
		Question: "Continue?", Options: []*pb.EffortDecisionOption{{Key: "yes"}, {Key: "no"}},
		Recommended: "yes", BigPicture: "Why this decision matters.",
		TechnicalDetails: "Two checks remain; exact limits are known.", Note: "Legacy context.", AnswerNote: "Operator reply",
		CreatedAt: timestamppb.New(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)),
	}

	event := RequestedEvent(decision)
	if got := event.Payload["big_picture"]; got != "Why this decision matters." {
		t.Fatalf("big picture = %#v", got)
	}
	if got := event.Payload["technical_details"]; got != "Two checks remain; exact limits are known." {
		t.Fatalf("explicit technical details must override legacy Note: %#v", got)
	}
	if _, found := event.Payload["answer_note"]; found {
		t.Fatalf("operator answer note leaked into request: %#v", event.Payload)
	}
}

func TestRequestedEventFallsBackToProducerNoteWhenTechnicalDetailsAreAbsent(t *testing.T) {
	decision := &pb.EffortDecision{
		GoalHome: "scenarios/demo/GOAL.md", DecisionId: "decision-1",
		Question: "Continue?", Options: []*pb.EffortDecisionOption{{Key: "yes"}, {Key: "no"}},
		Recommended: "yes", Note: "Two checks remain.", AnswerNote: "Operator reply",
		CreatedAt: timestamppb.New(time.Date(2026, 10, 16, 12, 0, 0, 0, time.UTC)),
	}

	event := RequestedEvent(decision)
	if got := event.Payload["technical_details"]; got != "Two checks remain." {
		t.Fatalf("legacy Note fallback = %#v", got)
	}
	if _, found := event.Payload["big_picture"]; found {
		t.Fatalf("empty purpose should remain omitted: %#v", event.Payload)
	}
	if _, found := event.Payload["answer_note"]; found {
		t.Fatalf("operator answer note leaked into request: %#v", event.Payload)
	}
	// The event is rebuilt from the stored record on every retry, so its bytes, and
	// therefore its id, must not change. Decisions publish under a subject key from
	// 2026-11-14; this pins that shape (a deliberate change from the key-less one).
	if got := EventID(event); got != "34d89c2494da6d75651987f145274243a730bcadca289c0f133cd567ca571477" {
		t.Fatalf("event identity changed: %s", got)
	}
}

func TestRequestedEventIsKeyedByTheDecisionAndDeclaresContextRequestsOnlyForAVerifiedRaiser(t *testing.T) {
	decision := &pb.EffortDecision{
		GoalHome: "scenarios/demo", DecisionId: "D-1", Question: "Continue?",
		Options:     []*pb.EffortDecisionOption{{Key: "yes"}, {Key: "no"}},
		Recommended: "yes", CreatedAt: timestamppb.New(time.Date(2026, 11, 14, 12, 0, 0, 0, time.UTC)),
	}
	event := RequestedEvent(decision)
	if got := event.Payload["subject_key"]; got != "effort-decision:scenarios/demo:D-1" || got != SubjectKey("scenarios/demo", "D-1") {
		t.Fatalf("subject_key = %#v", got)
	}
	if _, declared := event.Payload["accepts_context_requests"]; declared {
		t.Fatalf("a decision nobody raised from a run has no one to answer the operator, so it must not offer the question: %#v", event.Payload)
	}
	decision.RaisedByRunId = "run-1"
	if got := RequestedEvent(decision).Payload["accepts_context_requests"]; got != true {
		t.Fatalf("a decision raised by a verified run answers the operator's questions: %#v", got)
	}
	other := &pb.EffortDecision{GoalHome: "scenarios/demo", DecisionId: "D-2"}
	if SubjectKey(decision.GoalHome, decision.DecisionId) == SubjectKey(other.GoalHome, other.DecisionId) {
		t.Fatal("two decisions never share a subject")
	}
}

func TestAnExternalGoalHomeIsAKeyAndAnEscapingOneIsRefused(t *testing.T) {
	for raw, want := range map[string]string{
		"scenarios/demo":                                 "scenarios/demo",
		"path:scenarios/demo/GOAL.md":                    "scenarios/demo",
		"/home/me/.vrooli/research/cohorts/c1":           "/home/me/.vrooli/research/cohorts/c1",
		"/home/me/.vrooli/research/cohorts/c1/":          "/home/me/.vrooli/research/cohorts/c1",
		"path:/home/me/.vrooli/research/cohorts/GOAL.md": "/home/me/.vrooli/research/cohorts",
	} {
		got, err := NormalizeDecisionHome(raw)
		if err != nil || got != want {
			t.Errorf("NormalizeDecisionHome(%q) = %q, %v, want %q", raw, got, err, want)
		}
	}
	for _, raw := range []string{"", "/", "../outside", "/home/me/../other", "/home/me/./x", "//double"} {
		if got, err := NormalizeDecisionHome(raw); err == nil {
			t.Errorf("NormalizeDecisionHome(%q) = %q, want it refused", raw, got)
		}
	}
	if _, err := NormalizeGoalHome("/etc/x"); err == nil {
		t.Error("an effort's goal home stays repository-relative")
	}
}
