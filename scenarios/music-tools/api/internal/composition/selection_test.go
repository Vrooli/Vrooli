package composition

import "testing"

func TestSelectionIsNotInterpretedAsRanking(t *testing.T) {
	takes, err := PlanBatch(BatchRequest{JobID: "job-5", StyleID: "style", Caption: "sound", Takes: 2, Seed: 3})
	if err != nil {
		t.Fatal(err)
	}
	if takes[0].ID == takes[1].ID {
		t.Fatal("peer takes must remain addressable")
	}
}
