package codingjobs

import "testing"

func TestJobAdvanceUsesOneTransitionOwner(t *testing.T) {
	const digest = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	tests := []struct {
		name    string
		state   State
		stage   string
		actor   string
		digest  string
		outcome string
		want    State
	}{
		{name: "solver starts", state: StateReady, stage: "solver_started", actor: "agent-manager-solver", want: StateSolving},
		{name: "solver completes", state: StateSolving, stage: "solver_completed", actor: "agent-manager-solver", digest: digest, want: StateScoring},
		{name: "scoring starts", state: StateScoring, stage: "scoring_started", actor: "agent-manager-scorer", want: StateScoring},
		{name: "strict evaluation", state: StateScoring, stage: "evaluation_recorded", digest: digest, outcome: "strict_pass", want: StateReviewable},
		{name: "non-pass evaluation", state: StateScoring, stage: "evaluation_recorded", digest: digest, outcome: "non_pass", want: StateFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := Job{State: test.state}
			if err := job.advance(test.stage, test.actor, "evidence", test.digest, test.outcome); err != nil {
				t.Fatalf("advance: %v", err)
			}
			if job.State != test.want {
				t.Fatalf("state = %q, want %q", job.State, test.want)
			}
		})
	}
}

func TestJobAdvanceRejectsInvalidEdgesWithoutChangingState(t *testing.T) {
	tests := []struct {
		name    string
		state   State
		stage   string
		actor   string
		digest  string
		outcome string
	}{
		{name: "skip solver", state: StateReady, stage: "solver_completed", actor: "agent-manager-solver"},
		{name: "wrong actor", state: StateReady, stage: "solver_started", actor: "agent-manager-scorer"},
		{name: "terminal reversal", state: StateReviewable, stage: "solver_started", actor: "agent-manager-solver"},
		{name: "evaluation before scoring", state: StateSolving, stage: "evaluation_recorded", digest: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", outcome: "strict_pass"},
		{name: "unclassified evaluation", state: StateScoring, stage: "evaluation_recorded", outcome: ""},
		{name: "unknown evaluation outcome", state: StateScoring, stage: "evaluation_recorded", outcome: "unknown"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			job := Job{State: test.state}
			if err := job.advance(test.stage, test.actor, "evidence", test.digest, test.outcome); err == nil {
				t.Fatal("invalid transition was accepted")
			}
			if job.State != test.state {
				t.Fatalf("rejected transition changed state to %q", job.State)
			}
		})
	}
}
