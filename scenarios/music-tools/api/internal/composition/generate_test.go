package composition

import (
	"math"
	"testing"
)

func TestPlannedSeedsAreRepeatableEffectiveUint32(t *testing.T) {
	for _, base := range []int64{0, 7, -1, math.MaxInt64, 424242} {
		req := BatchRequest{JobID: "seed-test", StyleID: "song", Caption: "original", Takes: 100, Duration: 30, Seed: base}
		if base == 424242 {
			req.JobID = "v5-proof"
			req.StyleID = "v5"
			req.Takes = 1
			req.Duration = 48
		}
		first, err := PlanBatch(req)
		if err != nil {
			t.Fatal(err)
		}
		second, err := PlanBatch(req)
		if err != nil {
			t.Fatal(err)
		}
		if base == 424242 {
			t.Logf("canonical PlanBatch base=%d takes=%d duration=%d first effective uint32 seed=%d", base, req.Takes, req.Duration, first[0].Provenance.Seed)
		}
		for i, take := range first {
			if take.Provenance.Seed < 0 || take.Provenance.Seed > math.MaxUint32 {
				t.Fatalf("base=%d take=%d seed=%d is not representable without rounding", base, i, take.Provenance.Seed)
			}
			if second[i].Provenance.Seed != take.Provenance.Seed {
				t.Fatal("effective seeds changed for the same base")
			}
		}
	}
}

func TestPlanBatchPreservesAuthoredCaptionAndProvenance(t *testing.T) {
	takes, err := PlanBatch(BatchRequest{Duration: 45, JobID: "job-1", StyleID: "launch-trap", Caption: "cold detuned metallic lead", Takes: 2, Seed: 7})
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
		if take.Provenance.ModelID != "" || take.Provenance.LicenseLane != "" || take.Provenance.AppliedRung != "" {
			t.Fatalf("planning fabricated execution provenance: %#v", take.Provenance)
		}
	}
}

func TestPlanBatchRejectsInvalidRequests(t *testing.T) {
	if _, err := PlanBatch(BatchRequest{Duration: 45, JobID: "job", StyleID: "style", Caption: "x", Takes: 0}); err != ErrInvalidBatch {
		t.Fatalf("expected invalid batch, got %v", err)
	}
}
