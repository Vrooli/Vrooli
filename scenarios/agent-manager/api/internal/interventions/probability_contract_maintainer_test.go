package interventions

import (
	"context"
	"database/sql"
	"testing"
	"time"

	inferencev1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/inference"
)

func TestMalformedDistributionStaysOutOfStoredClassificationSummary(t *testing.T) {
	response := &inferencev1.DecideResponse{Answers: []*inferencev1.DecisionAnswer{{
		QuestionId: questionID,
		Choice:     "quality_miss",
		Probabilities: []*inferencev1.DecisionProbability{
			{Option: "early_stop", Probability: 0.125},
			{Option: "false_inability", Probability: 0.125},
			{Option: "wrong_claim", Probability: 0.125},
			{Option: "quality_miss", Probability: 0.15},
			{Option: "too_dense", Probability: 0.125},
			{Option: "instruction_ignored", Probability: 0.125},
			{Option: "scope_change", Probability: 0.125},
			{Option: "approval_or_info", Probability: 0.125},
		},
	}}}
	answer := answerFrom(response)
	if answer.Status != StatusUnclassified || answer.Category != "" || answer.Probability() != nil || len(answer.Probabilities) != 0 {
		t.Fatalf("decoded answer = %+v, want unclassified with no category or probability values", answer)
	}

	db := newTestDB(t)
	store := NewStore(db)
	at := time.Date(2026, 11, 1, 12, 0, 0, 0, time.UTC)
	if err := store.Save(context.Background(), []Record{{
		Candidate: Candidate{EventID: "event-1", RunID: "run-1", OccurredAt: "2026-11-01T11:00:00Z"},
		Version: "v1", Answer: answer,
	}}, at); err != nil {
		t.Fatal(err)
	}
	var status, category, distribution string
	var probability sql.NullFloat64
	if err := db.QueryRow(`SELECT status, category, probability, probabilities_json FROM operator_interventions WHERE event_id = 'event-1'`).Scan(&status, &category, &probability, &distribution); err != nil {
		t.Fatal(err)
	}
	if status != "unclassified" || category != "" || probability.Valid || distribution != "{}" {
		t.Fatalf("stored status/category/probability/distribution = %q/%q/%v/%q, want unclassified/empty/NULL/{}", status, category, probability, distribution)
	}

	got, err := store.Summarize(context.Background(), "v1", Gate{}, at.Add(-time.Hour), at.Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	if got.Unclassified != 1 || got.Classified != 0 || got.Pooled.Shown != 0 {
		t.Fatalf("summary counts = unclassified %d, classified %d, pooled shown %d; want 1, 0, 0", got.Unclassified, got.Classified, got.Pooled.Shown)
	}
	for _, category := range got.Categories {
		if category.Classified != 0 || category.Shown != 0 {
			t.Fatalf("category %q summary = %+v, want no stored classification", category.Category, category)
		}
	}
}
