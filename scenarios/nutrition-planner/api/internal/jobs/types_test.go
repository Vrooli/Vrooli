package jobs

import "testing"

func TestJobStateMachineRequiresReviewBeforeApply(t *testing.T) {
	job, err := New("j1", "w1", "recipe_extract", "w1:recipe:r1:2")
	if err != nil {
		t.Fatal(err)
	}
	job, err = job.Transition(Running)
	if err != nil {
		t.Fatal(err)
	}
	job, err = job.Transition(WaitingForReview)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = job.Transition(Succeeded); err == nil {
		t.Fatal("review job must not skip explicit apply/retry state")
	}
	if _, err = job.Transition(Canceled); err != nil {
		t.Fatal(err)
	}
}
