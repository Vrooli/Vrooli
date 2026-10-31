package transitionrunner

import (
	"context"
	"fmt"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/workflowcontract"
)

// CollectUsage reads the terminal receipt from the existing workflow owner.
// Missing accounting never releases a consumer's outstanding reservation.
func (r *Runner) CollectUsage(ctx context.Context, executionID, approvalDigest, grantDigest string) (*workflowcontract.Usage, error) {
	completion, err := r.workflows.CollectWorkflow(ctx, executionID)
	if err != nil {
		return nil, err
	}
	if completion.ExecutionID != executionID || completion.ApprovalDigest != approvalDigest || completion.GrantDigest != grantDigest {
		return nil, fmt.Errorf("workflow accounting receipt does not match the retained execution grant")
	}
	return workflowUsage(completion), nil
}

func workflowUsage(completion agentmanager.InvocationCompletion) *workflowcontract.Usage {
	if completion.BudgetUsage == nil {
		return nil
	}
	b := completion.BudgetUsage
	complete := b.GetAccountingComplete() && b.GetTokens() >= 0 && b.GetTurns() >= 0 && b.GetChildren() >= 0 && b.GetNodeAttempts() >= 0 && b.GetRetries() >= 0 && completion.WallSeconds >= 0
	usage := &workflowcontract.Usage{Tokens: int64(b.GetTokens()), Turns: int64(b.GetTurns()), Children: int64(b.GetChildren()), NodeAttempts: int64(b.GetNodeAttempts()), Retries: int64(b.GetRetries()), WallSeconds: completion.WallSeconds, TokensKnown: complete}
	for _, attempt := range completion.Attempts {
		if attempt != nil && attempt.NodeId == "slice" {
			usage.Slices++
		}
	}
	if receipt := completion.ChargeReceipt; receipt != nil && complete && receipt.Measured && receipt.AmountMicroUsd != nil && *receipt.AmountMicroUsd >= 0 {
		usage.ChargeMeasured = true
		usage.ChargeMicroUSD = *receipt.AmountMicroUsd
	}
	return usage
}
