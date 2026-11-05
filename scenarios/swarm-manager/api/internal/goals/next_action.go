package goals

import "swarm-manager/internal/backlog"

// NextActionInput supplies the already-resolved cross-domain facts needed by
// the goal funnel. Goals own the precedence; the feed owns loading members.
type NextActionInput struct {
	ReadyProposalCount int
	ReviewMilestone    string
	ChainedRef         string
	ChainedAction      backlog.NextActionProjection
}

func ReviewableMilestone(goal Goal, items map[string]backlog.BacklogItem) string {
	for _, milestone := range goal.Milestones {
		if milestone.ArchivedAt != nil || milestone.VerifiedDeliveredAt != nil || len(milestone.AcceptanceCriteria) == 0 || len(milestone.Items) == 0 {
			continue
		}
		ready := true
		for _, ref := range milestone.Items {
			item, ok := items[ref]
			if !ok || !backlog.IsTerminalStatus(item.Status) {
				ready = false
				break
			}
		}
		if ready {
			return milestone.Name
		}
	}
	return ""
}

// MilestoneMissingCriteria identifies the first active milestone whose
// definition of done is absent. It is intentionally separate from
// ReviewableMilestone: missing criteria is an operator action, not a reason to
// silently suppress the milestone from the funnel.
func MilestoneMissingCriteria(goal Goal) string {
	for _, milestone := range goal.Milestones {
		if milestone.ArchivedAt == nil && milestone.VerifiedDeliveredAt == nil && len(milestone.AcceptanceCriteria) == 0 {
			return milestone.Name
		}
	}
	return ""
}

// ResolveNextAction is the goal-domain half of the server-owned operator
// inbox. It never creates a second action for a member: chain preserves the
// member's resolved action and records its ref for the caller.
func ResolveNextAction(goal Goal, input NextActionInput) (backlog.NextActionProjection, string) {
	if goal.Status != StatusActive {
		return backlog.NextActionProjection{ID: backlog.NextActionNone, Enabled: false, Effect: backlog.EffectForNextAction(backlog.NextActionNone)}, ""
	}
	if input.ReadyProposalCount > 0 {
		return goalAction(backlog.NextActionDecide, "Decide", "Review goal proposal", "A proposed goal change is waiting for an operator decision.", "proposal_decision"), ""
	}
	if input.ReviewMilestone != "" {
		// Wording matters here: no evidence exists yet. This action is derived
		// purely from state (every member item terminal, criteria present, not
		// yet verified), and the operator's move is to *start* a review that
		// gathers the evidence. "Awaiting evidence review" read as though a
		// gathered packet were already sitting somewhere to be found.
		return goalAction(
			backlog.NextActionReview,
			"Start review",
			"Start milestone review",
			"Every item in this milestone is complete. Starting a review dispatches an agent to gather evidence against its acceptance criteria; you decide on that evidence afterwards.",
			"milestone_review:"+input.ReviewMilestone,
		), ""
	}
	if milestone := MilestoneMissingCriteria(goal); milestone != "" {
		return goalAction(backlog.NextActionDefineCriteria, "Define criteria", "Define milestone criteria", "This milestone needs acceptance criteria before it can be independently reviewed.", "milestone_criteria:"+milestone), ""
	}
	if len(goal.Milestones) == 0 && len(goal.Targets) == 0 {
		return goalAction(backlog.NextActionPlanGoal, "Plan goal", "Create goal structure", "This goal has no milestones or target items.", "goal_plan"), ""
	}
	if IsCloseOutReady(goal) {
		return goalAction(backlog.NextActionCloseOut, "Close out", "Mark goal achieved", "Every milestone is verified delivered.", "goal_close_out"), ""
	}
	if input.ChainedAction.Enabled && input.ChainedAction.ID != backlog.NextActionNone {
		return input.ChainedAction, input.ChainedRef
	}
	return backlog.NextActionProjection{ID: backlog.NextActionNone, Enabled: false}, ""
}

func goalAction(id backlog.NextActionID, compact, expanded, reason, target string) backlog.NextActionProjection {
	return backlog.NextActionProjection{ID: id, CompactLabel: compact, ExpandedLabel: expanded, Enabled: true, Reason: reason, Target: target, TransitionKey: backlog.TransitionKeyForNextAction(id), Effect: backlog.EffectForNextAction(id), Destructive: backlog.NextActionIsDestructive(id)}
}
