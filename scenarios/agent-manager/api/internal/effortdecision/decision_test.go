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
	if got := EventID(event); got != "41b3777abdaf7aa92addeb66fa06a68a0e508502fb26918812f7449c59467dfc" {
		t.Fatalf("legacy event identity changed: %s", got)
	}
}
