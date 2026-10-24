package main

import (
	"context"

	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/goals"
)

// goalTransitionProposalRecorder records the operator-facing proposal side
// effect after the shared transition runner has validated a terminal result.
type goalTransitionProposalRecorder struct{ sessions *agentsessions.Service }

func (r goalTransitionProposalRecorder) RecordGoalWorkflowProposals(ctx context.Context, proposal goals.GoalWorkflowProposal) (goals.GoalWorkflowProposalReceipt, error) {
	session, recorded, err := r.sessions.RecordWorkflowMutationProposals(ctx, proposal.Title, proposal.Summary, proposal.GoalVersion, agentsessions.ProposalTarget{Type: agentsessions.ContextGoal, Ref: proposal.GoalName, Name: proposal.GoalName}, proposal.Payloads, agentsessions.Attribution{Type: agentsessions.AttributionAgent, RunID: proposal.ExecutionID, ProfileKey: "swarm-manager/deep-work", Source: "workflow/" + proposal.WorkflowKey})
	if err != nil {
		return goals.GoalWorkflowProposalReceipt{}, err
	}
	ids := make([]string, 0, len(recorded))
	for _, entry := range recorded {
		ids = append(ids, entry.ID)
	}
	return goals.GoalWorkflowProposalReceipt{SessionID: session.ID, ProposalIDs: ids}, nil
}
