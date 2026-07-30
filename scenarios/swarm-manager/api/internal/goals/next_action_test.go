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
		{"missing criteria is actionable", Goal{Status: StatusActive, Milestones: []Milestone{{Name: "proof"}}}, NextActionInput{}, backlog.NextActionDefineCriteria},
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

func TestMilestoneMissingCriteriaIgnoresArchivedAndVerifiedMilestones(t *testing.T) {
	archived, delivered := "2026-07-24T00:00:00Z", "2026-07-25T00:00:00Z"
	goal := Goal{Milestones: []Milestone{{Name: "old", ArchivedAt: &archived}, {Name: "done", VerifiedDeliveredAt: &delivered}, {Name: "needs-definition"}}}
	if got := MilestoneMissingCriteria(goal); got != "needs-definition" {
		t.Fatalf("missing criteria milestone = %q", got)
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

func TestGoalFunnelNeverHidesMilestoneWithoutCriteria(t *testing.T) {
	statuses := []backlog.BacklogStatus{backlog.StatusBacklog, backlog.StatusCompleted}
	for _, status := range statuses {
		t.Run(string(status), func(t *testing.T) {
			goal := Goal{Status: StatusActive, Milestones: []Milestone{{Name: "missing", Items: []string{"idea/member"}}}}
			items := map[string]backlog.BacklogItem{"idea/member": {Kind: backlog.KindIdea, Name: "member", Status: status}}
			action, _ := ResolveNextAction(goal, NextActionInput{ReviewMilestone: ReviewableMilestone(goal, items)})
			if action.ID != backlog.NextActionDefineCriteria || !action.Enabled {
				t.Fatalf("status %q action = %+v, want enabled define_criteria", status, action)
			}
		})
	}
}
