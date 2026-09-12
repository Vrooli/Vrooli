package execution

import (
	"context"
	"testing"

	"swarm-manager/internal/review"
)

type recordingReviewService struct {
	description string
	criteria    any
	evidence    []review.EvidenceItem
}

func (s *recordingReviewService) StartReviewForExecution(_ context.Context, _, _, _, _, description, _ string, criteria any, evidence []review.EvidenceItem, _ []string, _ map[string][]string, _, _ string) error {
	s.description = description
	s.criteria = criteria
	s.evidence = evidence
	return nil
}

func (*recordingReviewService) RecordUnavailableReview(string, string, string, string) error {
	return nil
}

func TestTriggerReviewAgent_ForwardsTypedReviewContract(t *testing.T) {
	service := NewService(ServiceConfig{DataRoot: t.TempDir()})
	reviewService := &recordingReviewService{}
	service.SetReviewService(reviewService)
	criteria := []backlogCriterion{{ID: "criterion-1", Gherkin: "Given a completed run When reviewed Then proof is available."}}
	item := backlogItem{Name: "review-contract", Kind: "execute", Title: "Review contract", Description: "Outcome statement", AcceptanceCriteria: criteria}
	if err := service.triggerReviewAgent(context.Background(), "execution-1", finalizationScope{}, item); err != nil {
		t.Fatalf("triggerReviewAgent: %v", err)
	}
	if reviewService.description != item.Description {
		t.Fatalf("description = %q, want %q", reviewService.description, item.Description)
	}
	got, ok := reviewService.criteria.([]backlogCriterion)
	if !ok || len(got) != 1 || got[0].ID != "criterion-1" {
		t.Fatalf("criteria = %#v, want typed criterion", reviewService.criteria)
	}
}

func TestTriggerReviewAgent_ForwardsMachineSettledEvidence(t *testing.T) {
	service := NewService(ServiceConfig{DataRoot: t.TempDir()})
	reviewService := &recordingReviewService{}
	service.SetReviewService(reviewService)
	item := backlogItem{
		Name:  "machine-evidence",
		Kind:  "execute",
		Title: "Machine evidence",
		AcceptanceCriteria: []backlogCriterion{{
			ID:      "criterion-1",
			Gherkin: "Given a command check When it passes Then the criterion is settled.",
			Check:   &backlogCriterionCheck{Kind: "command", Argv: []string{"true"}},
		}},
	}
	if err := service.triggerReviewAgent(context.Background(), "execution-1", finalizationScope{}, item); err != nil {
		t.Fatalf("triggerReviewAgent: %v", err)
	}
	if len(reviewService.evidence) != 1 || reviewService.evidence[0].CriterionID != "criterion-1" || reviewService.evidence[0].Settlement != "settled" {
		t.Fatalf("machine evidence = %#v", reviewService.evidence)
	}
}
