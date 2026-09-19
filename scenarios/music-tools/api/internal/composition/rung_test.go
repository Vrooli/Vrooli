package composition

import "testing"

func TestAppliedRungTravelsWithTake(t *testing.T) {
	takes, err := PlanBatch(BatchRequest{JobID: "job-4", StyleID: "style", Caption: "sound", Takes: 1, Seed: 2})
	if err != nil {
		t.Fatal(err)
	}
	if takes[0].Provenance.AppliedRung != "full" {
		t.Fatalf("missing applied rung: %#v", takes[0].Provenance)
	}
}
