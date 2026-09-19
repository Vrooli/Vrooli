package composition

import "testing"

func TestPlanBatchPreservesAuthoredCaptionAndProvenance(t *testing.T) {
	takes, err := PlanBatch(BatchRequest{JobID: "job-1", StyleID: "launch-trap", Caption: "cold detuned metallic lead", Takes: 2, Seed: 7})
	if err != nil {
		t.Fatal(err)
	}
	if len(takes) != 2 {
		t.Fatalf("got %d takes", len(takes))
	}
	for _, take := range takes {
		if take.Provenance.CaptionAsAuthored != "cold detuned metallic lead" || take.Provenance.CaptionAsSent != take.Provenance.CaptionAsAuthored {
			t.Fatalf("caption violated: %#v", take.Provenance)
		}
		if take.Provenance.ModelID == "" || take.Provenance.LicenseLane == "" || take.Provenance.AppliedRung == "" {
			t.Fatalf("incomplete provenance: %#v", take.Provenance)
		}
	}
}

func TestPlanBatchRejectsInvalidRequests(t *testing.T) {
	if _, err := PlanBatch(BatchRequest{JobID: "job", StyleID: "style", Caption: "x", Takes: 0}); err != ErrInvalidBatch {
		t.Fatalf("expected invalid batch, got %v", err)
	}
}
