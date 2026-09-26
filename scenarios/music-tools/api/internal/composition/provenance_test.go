package composition

import "testing"

func TestRewriteIsVisibleInProvenance(t *testing.T) {
	takes, err := PlanBatch(BatchRequest{JobID: "job-3", StyleID: "style", Caption: "authored", Takes: 1, Seed: 1})
	if err != nil {
		t.Fatal(err)
	}
	takes[0].Provenance.CaptionAsSent = "rewritten"
	if takes[0].Provenance.CaptionAsSent == takes[0].Provenance.CaptionAsAuthored {
		t.Fatal("test setup failed")
	}
}
