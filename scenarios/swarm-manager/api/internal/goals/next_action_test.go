package goals

import (
	"testing"

	"swarm-manager/internal/backlog"
)

func TestResolveNextActionGoalFunnel(t *testing.T) {
	delivered := "2026-07-24T00:00:00Z"
	chain := backlog.NextActionProjection{ID: backlog.NextActionRun, Enabled: true}
	for _, tt := range []struct {
		name  string
		goal  Goal
		input NextActionInput
		want  backlog.NextActionID
	}{
		{"archived has none", Goal{Status: StatusArchived}, NextActionInput{}, backlog.NextActionNone},
		{"proposal decides first", Goal{Status: StatusActive}, NextActionInput{ReadyProposalCount: 1}, backlog.NextActionDecide},
		{"review precedes planning", Goal{Status: StatusActive}, NextActionInput{ReviewMilestone: "evidence"}, backlog.NextActionReview},
		{"empty goal plans", Goal{Status: StatusActive}, NextActionInput{}, backlog.NextActionPlanGoal},
		{"delivered goal closes", Goal{Status: StatusActive, Milestones: []Milestone{{Name: "done", VerifiedDeliveredAt: &delivered}}}, NextActionInput{}, backlog.NextActionCloseOut},
		{"member action chains", Goal{Status: StatusActive, Targets: []string{"idea/member"}}, NextActionInput{ChainedRef: "idea/member", ChainedAction: chain}, backlog.NextActionRun},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := ResolveNextAction(tt.goal, tt.input)
			if got.ID != tt.want || (tt.want != backlog.NextActionNone && !got.Enabled) {
				t.Fatalf("action = %+v, want %s enabled", got, tt.want)
			}
		})
	}
}

func TestReviewableMilestoneRequiresTerminalEvidenceBearingMembers(t *testing.T) {
	goal := Goal{Milestones: []Milestone{{Name: "proof", Items: []string{"idea/member"}, AcceptanceCriteria: []string{"passes"}}}}
	items := map[string]backlog.BacklogItem{"idea/member": {Kind: backlog.KindIdea, Name: "member", Status: backlog.StatusCompleted}}
	if got := ReviewableMilestone(goal, items); got != "proof" {
		t.Fatalf("reviewable milestone = %q", got)
	}
	goal.Milestones[0].AcceptanceCriteria = nil
	if got := ReviewableMilestone(goal, items); got != "" {
		t.Fatalf("milestone without evidence criteria = %q", got)
	}
}
