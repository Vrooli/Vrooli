package transitionrunner

import (
	"context"
	"testing"

	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/workflowcontract"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
)

type developmentWorkflowStub struct {
	invocation agentmanager.Invocation
	started    agentmanager.WorkflowStart
	completion agentmanager.InvocationCompletion
	cancelled  bool
}

func (s *developmentWorkflowStub) StartWorkflow(_ context.Context, invocation agentmanager.Invocation) (agentmanager.WorkflowStart, error) {
	s.invocation = invocation
	return s.started, nil
}

func (s *developmentWorkflowStub) CollectWorkflow(context.Context, string) (agentmanager.InvocationCompletion, error) {
	return s.completion, nil
}

func (s *developmentWorkflowStub) CancelWorkflow(context.Context, string, string, string) error {
	s.cancelled = true
	return nil
}

func TestDevelopmentInvokerConvertsGrantAndMeasuredCompletion(t *testing.T) {
	stub := &developmentWorkflowStub{
		started: agentmanager.WorkflowStart{ExecutionID: "exec-1", RunID: "run-1", DefinitionDigest: "sha256:one", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_RUNNING},
		completion: agentmanager.InvocationCompletion{
			ExecutionID: "exec-1", DefinitionDigest: "sha256:one",
			Status:      domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
			BudgetUsage: &domainpb.WorkflowBudgetUsage{Tokens: 42, CostUsd: 0.00125}, WallSeconds: 7,
			ChargeReceipt: &domainpb.ChargeReceipt{AmountMicroUsd: ptrInt64(1250), Currency: "USD", Measured: true},
		},
	}
	adapter := NewDevelopmentInvoker(stub)
	started, err := adapter.Start(context.Background(), workflowcontract.Invocation{
		Owner: "swarm-manager", WorkflowKey: "development", Grant: &workflowcontract.Grant{MaxTurns: 4, MaxTokens: 500, MaxChargeMicroUSD: 9000, MaxWallTimeSeconds: 60, MaxNodeAttempts: 5, MaxChildren: 2, MaxConcurrency: 1, MaxRecursion: 1, MaxRetries: 2, MaxWaitSeconds: 30},
		Activity: &workflowcontract.Activity{OwnerType: "scenario", OwnerName: "audio-tools"},
	})
	if err != nil || started.ExecutionID != "exec-1" || started.RunID != "run-1" {
		t.Fatalf("started=%+v err=%v", started, err)
	}
	if stub.invocation.EngagementGrant == nil || stub.invocation.EngagementGrant.GetMaxTokens() != 500 || stub.invocation.EngagementGrant.GetMaxTurns() != 4 || stub.invocation.EngagementGrant.GetMaxChargeMicroUsd() != 9000 || stub.invocation.EngagementGrant.GetMaxChildren() != 2 || stub.invocation.Activity == nil || stub.invocation.Activity.OwnerName != "audio-tools" {
		t.Fatalf("provider invocation=%+v", stub.invocation)
	}
	completion, err := adapter.Collect(context.Background(), "exec-1")
	if err != nil || completion.Usage == nil || !completion.Usage.TokensKnown || completion.Usage.Tokens != 42 || completion.Usage.WallSeconds != 7 || !completion.Usage.ChargeMeasured {
		t.Fatalf("completion=%+v err=%v", completion, err)
	}
	if err := adapter.Cancel(context.Background(), "exec-1", "cancel-1", "operator stop"); err != nil || !stub.cancelled {
		t.Fatalf("cancel err=%v cancelled=%t", err, stub.cancelled)
	}
}

func TestDevelopmentInvokerKeepsUnmeasuredChargeDistinctFromZero(t *testing.T) {
	adapter := NewDevelopmentInvoker(&developmentWorkflowStub{completion: agentmanager.InvocationCompletion{
		ExecutionID: "exec-unknown", Status: domainpb.WorkflowExecutionStatus_WORKFLOW_EXECUTION_STATUS_SUCCEEDED,
		BudgetUsage: &domainpb.WorkflowBudgetUsage{Tokens: 1}, ChargeReceipt: &domainpb.ChargeReceipt{Currency: "USD", Measured: false},
	}})
	completion, err := adapter.Collect(context.Background(), "exec-unknown")
	if err != nil || completion.Usage == nil || completion.Usage.ChargeMeasured || completion.Usage.ChargeMicroUSD != 0 {
		t.Fatalf("unmeasured charge became zero: %+v err=%v", completion.Usage, err)
	}
}

func ptrInt64(value int64) *int64 { return &value }

func TestExplicitZeroRetryAllowanceSurvivesOwnerTransport(t *testing.T) {
	grant := &workflowcontract.Grant{MaxTokens: 500, MaxWallTimeSeconds: 60}
	legacyDigest := workflowcontract.GrantDigest(grant)
	legacy, err := toAgentGrant(grant)
	if err != nil || legacy.MaxRetries != nil {
		t.Fatalf("legacy omitted retry limit changed: %+v %v", legacy, err)
	}
	grant.RetryLimitSet = true
	explicit, err := toAgentGrant(grant)
	if err != nil || explicit.MaxRetries == nil || explicit.GetMaxRetries() != 0 {
		t.Fatalf("exhausted allowance became unspecified: %+v %v", explicit, err)
	}
	if workflowcontract.GrantDigest(grant) == legacyDigest {
		t.Fatal("explicit zero and unspecified retry authority share a digest")
	}
}

func TestDevelopmentInvokerRejectsGrantOutsideProviderRange(t *testing.T) {
	adapter := NewDevelopmentInvoker(&developmentWorkflowStub{})
	if _, err := adapter.Start(context.Background(), workflowcontract.Invocation{Grant: &workflowcontract.Grant{MaxTokens: 1 << 40, MaxWallTimeSeconds: 1}}); err == nil {
		t.Fatal("oversized grant was accepted")
	}
}
