package scenarios

import (
	"testing"
	"time"
)

func TestScenarioToProtoCompleteness(t *testing.T) {
	score := 88
	proto := scenarioToProto(Scenario{
		Name:              "demo",
		DisplayName:       "Demo",
		Description:       "Test",
		Status:            StatusRunning,
		Priority:          2,
		CompletenessScore: &score,
		IsGreenfield:      true,
		Tags:              []string{"tag"},
	}, nil)

	if proto.CompletenessScore == nil {
		t.Fatalf("expected completeness score to be set")
	}
	if *proto.CompletenessScore != int32(score) {
		t.Fatalf("expected completeness %d, got %d", score, *proto.CompletenessScore)
	}
	if proto.LastReviewClassification != nil {
		t.Fatalf("expected nil review classification when no review summary")
	}
}

func TestScenarioToProtoWithReview(t *testing.T) {
	reviewTime := time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC)
	review := &ScenarioReviewSummary{
		LastReviewClassification: "needs_work",
		LastReviewAt:             reviewTime,
	}
	proto := scenarioToProto(Scenario{
		Name:     "test",
		Status:   StatusRunning,
		Priority: 1,
	}, review)

	if proto.LastReviewClassification == nil {
		t.Fatalf("expected review classification to be set")
	}
	if *proto.LastReviewClassification != "needs_work" {
		t.Fatalf("expected needs_work, got %s", *proto.LastReviewClassification)
	}
	if proto.LastReviewAt == nil {
		t.Fatalf("expected review timestamp to be set")
	}
	if *proto.LastReviewAt != reviewTime.Format(time.RFC3339) {
		t.Fatalf("expected %s, got %s", reviewTime.Format(time.RFC3339), *proto.LastReviewAt)
	}
}

func TestScenarioToProtoPreservesHealthProjection(t *testing.T) {
	proto := scenarioToProto(Scenario{
		Name: "test", Status: StatusRunning, Priority: 1,
		Health: &ScenarioHealthSnapshot{
			EvidenceState: HealthEvidenceStale,
			Reason:        "The latest comparable run is older than the freshness window.",
			SourceRunID:   "run-42",
			Phases: []ScenarioHealthPhase{{
				Phase: "unit", CurrentRung: "L1", NextRung: "L2", PriorityCapabilityID: "coverage",
			}},
		},
	}, nil)
	if proto.Health == nil || proto.Health.EvidenceState != string(HealthEvidenceStale) {
		t.Fatalf("health projection = %#v", proto.Health)
	}
	if proto.Health.SourceRunId == nil || *proto.Health.SourceRunId != "run-42" {
		t.Fatalf("source run id = %#v", proto.Health.SourceRunId)
	}
	if len(proto.Health.Phases) != 1 || proto.Health.Phases[0].PriorityCapabilityId == nil || *proto.Health.Phases[0].PriorityCapabilityId != "coverage" {
		t.Fatalf("phase projection = %#v", proto.Health.Phases)
	}
}
