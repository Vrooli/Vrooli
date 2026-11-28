package budgets

import (
	"testing"
	"time"
)

func TestProposeDeadlineRequiresQualifiedEvidenceAndDoesNotApplyCandidate(t *testing.T) {
	samples := make([]LatencyObservation, 0, 80)
	for _, status := range []string{"success", "failure"} {
		for _, thermal := range []string{"cold", "warm"} {
			for n := 1; n <= 20; n++ {
				total := time.Duration(n) * time.Second
				if thermal == "cold" {
					total += 5 * time.Second
				}
				samples = append(samples, LatencyObservation{Status: status, ThermalState: thermal, Total: total, BindingPhase: time.Second, ModelPhase: 2 * time.Second})
			}
		}
	}

	proposal, err := ProposeDeadline(samples, DefaultLatencyEvidencePolicy(), 3*time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if proposal.SlowestP95 != 24*time.Second || proposal.Candidate != 27*time.Second || proposal.Headroom != 3*time.Second {
		t.Fatalf("proposal=%+v", proposal)
	}
	if len(proposal.Evidence.P95) != 4 {
		t.Fatalf("proposal omitted strata: %+v", proposal.Evidence)
	}
	// The proposal API is intentionally detached from the serving ladder.
	if SyncSubmit != 2*time.Minute {
		t.Fatalf("proposal changed serving budget: sync_submit=%s", SyncSubmit)
	}

	if _, err := ProposeDeadline(samples[:60], DefaultLatencyEvidencePolicy(), time.Second); err == nil {
		t.Fatal("incomplete strata produced a deadline candidate")
	}
	if _, err := ProposeDeadline(samples, DefaultLatencyEvidencePolicy(), -time.Second); err == nil {
		t.Fatal("negative headroom produced a deadline candidate")
	}
}
