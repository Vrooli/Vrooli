// Responsibility: expose workflow ownership only after checking retained issuer evidence.
package orchestration

import (
	"context"
	"strings"

	"agent-manager/internal/domain"
	"agent-manager/internal/identity"
	"github.com/google/uuid"
)

// projectVerifiedWorkflowIdentity preserves valid baseline authority while
// removing unproven workflow labels. A signature proves issuer origin, not
// membership: historical public environments could contain these labels.
func (o *Orchestrator) projectVerifiedWorkflowIdentity(ctx context.Context, run *domain.Run, claims *identity.Claims) error {
	executionText := strings.TrimSpace(claims.Meta["workflowExecutionId"])
	node := strings.TrimSpace(claims.Meta["workflowNodeId"])
	attemptText := strings.TrimSpace(claims.Meta["workflowAttemptId"])
	for _, key := range []string{"workflowExecutionId", "workflowNodeId", "workflowAttemptId"} {
		delete(claims.Meta, key)
	}
	if executionText == "" || node == "" || attemptText == "" || o.workflowExecutions == nil {
		return nil
	}
	execution, err := uuid.Parse(executionText)
	if err != nil {
		return nil
	}
	attemptID, err := uuid.Parse(attemptText)
	if err != nil {
		return nil
	}
	attempts, err := o.workflowExecutions.ListAttempts(ctx, execution)
	if err != nil {
		return err
	}
	for _, attempt := range attempts {
		if attempt.ID != attemptID || attempt.ExecutionID != execution || attempt.NodeID != node {
			continue
		}
		bound := attempt.RunID != nil && *attempt.RunID == run.ID
		if !bound && attempt.RunID == nil && run.ResolvedConfig != nil && run.ResolvedConfig.Admission != nil {
			proof := run.ResolvedConfig.Admission.WorkflowIdentity
			// The launcher records this private receipt in the run row before any
			// execution. The interpreter acknowledges RunID later; exact task/key and
			// retained attempt checks close that window without trusting environment.
			bound = proof != nil && proof.ExecutionID == execution && proof.AttemptID == attempt.ID && proof.NodeID == node && run.IdempotencyKey == attempt.IdempotencyKey && run.TaskID == uuid.NewSHA1(attempt.ID, []byte("workflow-node-task"))
			if bound {
				parent, err := o.workflowExecutions.Get(ctx, execution)
				if err != nil {
					return err
				}
				bound = parent != nil && !parent.Status.Terminal()
			}
		}
		if bound {
			claims.Meta["workflowExecutionId"] = execution.String()
			claims.Meta["workflowNodeId"] = node
			claims.Meta["workflowAttemptId"] = attemptID.String()
		}
		return nil
	}
	return nil
}
