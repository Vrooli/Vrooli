package effortdecision

import (
	"strings"
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
	// 2026-11-14 and carry a page and a subject in words from 2026-11-15, whose
	// options end with how to answer (also 2026-11-15); this pins that shape.
	if got := EventID(event); got != "bd5369fe098b230498490d5570d265740000bd3bc6e0bdb4251f8b12376f0086" {
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

// Every effort decision reaches the operator as a page: the agent's own page
// when it sent one, otherwise one laid out from the decision's texts.
func TestRequestedEventCarriesTheAgentsPageOrAGeneratedOne(t *testing.T) {
	deadline := time.Date(2026, 11, 16, 12, 0, 0, 0, time.UTC)
	decision := &pb.EffortDecision{
		GoalHome: "/home/op/.vrooli/research/experience-engineering/cohorts/ux-redesign-20261115", DecisionId: "D-7",
		Question:    "Which inbox layout should the redesign build?",
		BigPicture:  "Two layouts were captured from the live app and redrawn.\n\nThe chosen one becomes the target for the next build.",
		Options:     []*pb.EffortDecisionOption{{Key: "cards", Label: "Build the card layout"}, {Key: "list", Label: "Build the dense list"}},
		Recommended: "cards", RecommendationReason: "It tested better on a phone.", DefaultAnswer: "cards", Reversible: true, Deadline: timestamppb.New(deadline),
		CreatedAt: timestamppb.New(time.Date(2026, 11, 15, 6, 0, 0, 0, time.UTC)),
	}
	event := RequestedEvent(decision)
	if subject := event.Payload["subject"]; subject != "Ux redesign" {
		t.Errorf("the subject names the work in words, not the folder: %v", subject)
	}
	page, _ := event.Payload["explainer_html"].(string)
	for _, want := range []string{
		"<h1>Which inbox layout should the redesign build?</h1>", "<p>The chosen one becomes the target for the next build.</p>",
		"The agent working on Ux redesign", "Build the card layout", "It tested better on a phone.",
		"&#34;Build the card layout&#34; applies by itself at Mon Nov 16 12:00 UTC",
	} {
		if !strings.Contains(page, want) {
			t.Errorf("generated page lacks %q", want)
		}
	}
	decision.ExplainerHtml = "<!doctype html><title>Layouts</title><p>side by side</p>"
	if got := RequestedEvent(decision).Payload["explainer_html"]; got != decision.ExplainerHtml {
		t.Fatalf("the agent's own page was replaced: %v", got)
	}
}
