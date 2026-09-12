package transitionrunner

import (
	"testing"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/transitionrun"
	"swarm-manager/internal/workflowcontract"
)

func TestOwnerUsageRequiresCompleteAccountingAndExactGrant(t *testing.T) {
	amount := int64(25)
	completion := agentmanager.InvocationCompletion{ExecutionID: "owner-1", ApprovalDigest: "approval", GrantDigest: "grant", BudgetUsage: &domainpb.WorkflowBudgetUsage{Tokens: 500}, ChargeReceipt: &domainpb.ChargeReceipt{Measured: true, AmountMicroUsd: &amount}, WallSeconds: 20}
	usage := workflowUsage(completion)
	if usage.TokensKnown || usage.ChargeMeasured {
		t.Fatal("positive partial aggregate was treated as a complete receipt")
	}
	completion.BudgetUsage.AccountingComplete = true
	usage = workflowUsage(completion)
	if !usage.TokensKnown || !usage.ChargeMeasured || usage.Tokens != 500 || usage.ChargeMicroUSD != 25 {
		t.Fatalf("complete owner receipt lost: %+v", usage)
	}
	completion.BudgetUsage.Turns = -1
	if invalid := workflowUsage(completion); invalid.TokensKnown || invalid.ChargeMeasured {
		t.Fatal("negative accounting could replenish a reservation")
	}
	completion.BudgetUsage.Turns = 0
	runner, owner, _ := testRunner(t)
	owner.completion = completion
	if _, err := runner.CollectUsage(t.Context(), "owner-1", "approval", "other-grant"); err == nil {
		t.Fatal("mismatched grant released a reservation")
	}
	if _, err := runner.CollectUsage(t.Context(), "owner-1", "approval", "grant"); err != nil {
		t.Fatal(err)
	}
	_, _, err := completionOutcome(completion, transitionrun.Correlation{}, Snapshot{ApprovalDigest: "approval", Grant: &workflowcontract.Grant{MaxTokens: 1000}})
	if err == nil {
		t.Fatal("mismatched completion grant reached consumer apply")
	}
}
