package composition

import "testing"

func TestPlanBatchProducesPeerTakesWithDistinctSeeds(t *testing.T) {
	takes, err := PlanBatch(BatchRequest{JobID: "job-2", StyleID: "style", Caption: "sound", Takes: 10, Seed: 99})
	if err != nil {
		t.Fatal(err)
	}
	seen := map[int64]bool{}
	for _, take := range takes {
		if seen[take.Provenance.Seed] {
			t.Fatalf("duplicate seed: %d", take.Provenance.Seed)
		}
		seen[take.Provenance.Seed] = true
	}
}
