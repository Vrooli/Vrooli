package composition

import "testing"

func TestPlanningDoesNotInventAppliedRung(t *testing.T) {
	takes, err := PlanBatch(BatchRequest{Duration: 45, JobID: "job-4", StyleID: "style", Caption: "sound", Takes: 1, Seed: 2})
	if err != nil {
		t.Fatal(err)
	}
	if takes[0].Provenance.AppliedRung != "" {
		t.Fatalf("planning fabricated applied rung: %#v", takes[0].Provenance)
	}
}
