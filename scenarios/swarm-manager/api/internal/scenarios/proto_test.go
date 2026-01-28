package scenarios

import "testing"

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
	})

	if proto.CompletenessScore == nil {
		t.Fatalf("expected completeness score to be set")
	}
	if *proto.CompletenessScore != int32(score) {
		t.Fatalf("expected completeness %d, got %d", score, *proto.CompletenessScore)
	}
}
