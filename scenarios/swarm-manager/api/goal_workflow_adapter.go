package main

import (
	"context"
	"errors"
	"fmt"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/agentsessions"
	"swarm-manager/internal/goals"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type goalWorkflowAdapter struct{ invoker agentmanager.WorkflowInvoker }

func (a goalWorkflowAdapter) StartWorkflow(ctx context.Context, in goals.WorkflowInvocation) (goals.WorkflowStart, error) {
	invocation := agentmanager.Invocation{Owner: in.Owner, WorkflowKey: in.WorkflowKey, Input: in.Input, IdempotencyKey: in.IdempotencyKey, FirstRunNodeID: in.FirstRunNodeID}
	if in.ActivityOwnerName != "" {
		invocation.Activity = &agentmanager.WorkflowActivity{OwnerType: in.ActivityOwnerType, OwnerKind: in.ActivityOwnerKind, OwnerName: in.ActivityOwnerName, OwnerTitle: in.ActivityOwnerTitle, Purpose: in.ActivityPurpose}
	}
	start, err := a.invoker.StartWorkflow(ctx, invocation)
	return goals.WorkflowStart{ExecutionID: start.ExecutionID, RunID: start.RunID, DefinitionDigest: start.DefinitionDigest}, err
}

func (a goalWorkflowAdapter) CollectWorkflow(ctx context.Context, executionID string) (goals.WorkflowCompletion, error) {
	completion, err := a.invoker.CollectWorkflow(ctx, executionID)
	// Translate the transport sentinels into the goal domain's own so goals
	// does not have to depend on the agent-manager client package. Both mean
	// "ask again later": the run has not finished, or agent-manager could not
	// be reached at all. Neither is a defect in the correlation record, so
	// neither may be recorded as an apply failure against it.
	switch {
	case err == nil:
	case errors.Is(err, agentmanager.ErrWorkflowNotReady):
		err = fmt.Errorf("%w: %v", goals.ErrWorkflowNotReady, err)
	case errors.Is(err, agentmanager.ErrNotAvailable):
		err = fmt.Errorf("%w: %v", goals.ErrWorkflowUnavailable, err)
	}
	return goals.WorkflowCompletion{ExecutionID: completion.ExecutionID, DefinitionDigest: completion.DefinitionDigest, Succeeded: completion.Status == domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED, Input: completion.Input, Output: completion.Output}, err
}

type goalWorkflowProposalRecorder struct{ sessions *agentsessions.Service }

func (r goalWorkflowProposalRecorder) RecordGoalWorkflowProposals(ctx context.Context, proposal goals.GoalWorkflowProposal) (goals.GoalWorkflowProposalReceipt, error) {
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
