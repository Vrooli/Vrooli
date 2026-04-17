package scenarios

import (
	"swarm-manager/internal/execution"
	"testing"
	"time"
)

func makeReviewRecord(scenarioName, classification, reviewedAt string) execution.Record {
	return execution.Record{
		CreatedAt: reviewedAt,
		Finalization: &execution.Finalization{
			Scenarios: []execution.ScenarioFinalization{
				{
					ScenarioName: scenarioName,
					Review: execution.ScenarioReviewStep{
						Result: &execution.ReviewResult{
							Classification: classification,
							ReviewedAt:     reviewedAt,
						},
					},
				},
			},
		},
	}
}

func TestComputeReviewSummaries_Empty(t *testing.T) {
	result := ComputeReviewSummaries(nil)
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(result))
	}

	result = ComputeReviewSummaries([]execution.Record{})
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(result))
	}
}

func TestComputeReviewSummaries_SingleRecord(t *testing.T) {
	ts := "2026-04-01T12:00:00Z"
	records := []execution.Record{makeReviewRecord("my-scenario", "ready", ts)}
	result := ComputeReviewSummaries(records)

	summary, ok := result["my-scenario"]
	if !ok {
		t.Fatalf("expected summary for my-scenario")
	}
	if summary.LastReviewClassification != "ready" {
		t.Fatalf("expected ready, got %s", summary.LastReviewClassification)
	}
	expected, _ := time.Parse(time.RFC3339, ts)
	if !summary.LastReviewAt.Equal(expected) {
		t.Fatalf("expected %v, got %v", expected, summary.LastReviewAt)
	}
}

func TestComputeReviewSummaries_LatestWins(t *testing.T) {
	records := []execution.Record{
		makeReviewRecord("app", "needs_work", "2026-03-01T10:00:00Z"),
		makeReviewRecord("app", "ready", "2026-04-01T10:00:00Z"),
		makeReviewRecord("app", "ready_with_notes", "2026-03-15T10:00:00Z"),
	}
	result := ComputeReviewSummaries(records)

	summary := result["app"]
	if summary.LastReviewClassification != "ready" {
		t.Fatalf("expected ready (latest), got %s", summary.LastReviewClassification)
	}
}

func TestComputeReviewSummaries_NilFinalization(t *testing.T) {
	records := []execution.Record{
		{CreatedAt: "2026-04-01T12:00:00Z", Finalization: nil},
		makeReviewRecord("valid", "ready", "2026-04-01T12:00:00Z"),
	}
	result := ComputeReviewSummaries(records)

	if len(result) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(result))
	}
	if _, ok := result["valid"]; !ok {
		t.Fatalf("expected valid scenario in results")
	}
}

func TestComputeReviewSummaries_InvalidTimestamp(t *testing.T) {
	records := []execution.Record{
		makeReviewRecord("bad-ts", "ready", "not-a-timestamp"),
		makeReviewRecord("good-ts", "ready", "2026-04-01T12:00:00Z"),
	}
	result := ComputeReviewSummaries(records)

	if _, ok := result["bad-ts"]; ok {
		t.Fatalf("expected bad-ts to be skipped")
	}
	if _, ok := result["good-ts"]; !ok {
		t.Fatalf("expected good-ts in results")
	}
}

func TestComputeReviewSummaries_MultipleScenarios(t *testing.T) {
	records := []execution.Record{
		makeReviewRecord("alpha", "ready", "2026-04-01T12:00:00Z"),
		makeReviewRecord("beta", "needs_work", "2026-04-02T12:00:00Z"),
	}
	result := ComputeReviewSummaries(records)

	if len(result) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(result))
	}
	if result["alpha"].LastReviewClassification != "ready" {
		t.Fatalf("expected alpha=ready, got %s", result["alpha"].LastReviewClassification)
	}
	if result["beta"].LastReviewClassification != "needs_work" {
		t.Fatalf("expected beta=needs_work, got %s", result["beta"].LastReviewClassification)
	}
}

func TestComputeReviewSummaries_NilReviewResult(t *testing.T) {
	records := []execution.Record{
		{
			CreatedAt: "2026-04-01T12:00:00Z",
			Finalization: &execution.Finalization{
				Scenarios: []execution.ScenarioFinalization{
					{
						ScenarioName: "no-review",
						Review:       execution.ScenarioReviewStep{Result: nil},
					},
				},
			},
		},
	}
	result := ComputeReviewSummaries(records)
	if len(result) != 0 {
		t.Fatalf("expected empty map for nil review result, got %d", len(result))
	}
}
