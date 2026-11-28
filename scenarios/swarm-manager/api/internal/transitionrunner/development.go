package transitionrunner

import (
	"context"
	"errors"
	"fmt"
	"google.golang.org/protobuf/proto"
	"math"
	"strings"

	domainpb "github.com/vrooli/vrooli/packages/proto/gen/go/agent-manager/v1/domain"
	"swarm-manager/internal/agentmanager"
	"swarm-manager/internal/workflowcontract"
)

// DevelopmentInvoker adapts the transport-neutral development boundary to
// Agent Manager at the transition-runner edge. Keeping this conversion here
// prevents subject domains from importing the provider client or bypassing the
// runner's dispatch journal and routing policy.
type DevelopmentInvoker struct {
	invoker agentmanager.WorkflowInvoker
}

func NewDevelopmentInvoker(invoker agentmanager.WorkflowInvoker) *DevelopmentInvoker {
	return &DevelopmentInvoker{invoker: invoker}
}

func (a *DevelopmentInvoker) Start(ctx context.Context, invocation workflowcontract.Invocation) (workflowcontract.Start, error) {
	if a == nil || a.invoker == nil {
		return workflowcontract.Start{}, agentmanager.ErrNotAvailable
	}
	grant, err := toAgentGrant(invocation.Grant)
	if err != nil {
		return workflowcontract.Start{}, err
	}
	var activity *agentmanager.WorkflowActivity
	if invocation.Activity != nil {
		activity = &agentmanager.WorkflowActivity{
			OwnerType:  invocation.Activity.OwnerType,
			OwnerKind:  invocation.Activity.OwnerKind,
			OwnerName:  invocation.Activity.OwnerName,
			OwnerTitle: invocation.Activity.OwnerTitle,
			Purpose:    invocation.Activity.Purpose,
		}
	}
	started, err := a.invoker.StartWorkflow(ctx, agentmanager.Invocation{
		Owner: invocation.Owner, WorkflowKey: invocation.WorkflowKey, ApprovalDigest: invocation.ApprovalDigest, WorkflowDigest: invocation.WorkflowDigest, GrantDigest: invocation.GrantDigest, Input: invocation.Input,
		IdempotencyKey: invocation.IdempotencyKey, FirstRunNodeID: invocation.FirstRunNodeID,
		Activity: activity, EngagementGrant: grant,
	})
	if err != nil {
		return workflowcontract.Start{}, err
	}
	return workflowcontract.Start{
		ExecutionID: started.ExecutionID, RunID: started.RunID,
		WorkflowDigest: started.WorkflowDigest, DefinitionDigest: started.DefinitionDigest, ApprovalDigest: started.ApprovalDigest, GrantDigest: started.GrantDigest, Status: started.Status.String(),
	}, nil
}

// ReconcileStart is deliberately optional on the Agent Manager transport
// boundary. It reads the durable owner execution by the original start key;
// it never retries workflow creation.
func (a *DevelopmentInvoker) ReconcileStart(ctx context.Context, idempotencyKey string) (workflowcontract.Start, error) {
	if a == nil || a.invoker == nil {
		return workflowcontract.Start{}, agentmanager.ErrNotAvailable
	}
	reconciler, ok := a.invoker.(interface {
		GetWorkflowExecutionByIdempotencyKey(context.Context, string) (agentmanager.WorkflowStart, error)
	})
	if !ok {
		return workflowcontract.Start{}, fmt.Errorf("workflow start reconciliation is not supported")
	}
	started, err := reconciler.GetWorkflowExecutionByIdempotencyKey(ctx, idempotencyKey)
	if err != nil {
		if errors.Is(err, agentmanager.ErrWorkflowNotFound) {
			return workflowcontract.Start{}, workflowcontract.ErrStartNotFound
		}
		return workflowcontract.Start{}, err
	}
	return workflowcontract.Start{ExecutionID: started.ExecutionID, RunID: started.RunID, WorkflowDigest: started.WorkflowDigest, DefinitionDigest: started.DefinitionDigest, ApprovalDigest: started.ApprovalDigest, GrantDigest: started.GrantDigest, Status: started.Status.String()}, nil
}

func (a *DevelopmentInvoker) Collect(ctx context.Context, executionID string) (workflowcontract.Completion, error) {
	if a == nil || a.invoker == nil {
		return workflowcontract.Completion{}, agentmanager.ErrNotAvailable
	}
	completion, err := a.invoker.CollectWorkflow(ctx, strings.TrimSpace(executionID))
	if err != nil {
		return workflowcontract.Completion{}, err
	}
	result := workflowcontract.Completion{
		ExecutionID: completion.ExecutionID, WorkflowDigest: completion.WorkflowDigest, DefinitionDigest: completion.DefinitionDigest, ApprovalDigest: completion.ApprovalDigest, GrantDigest: completion.GrantDigest,
		Status: completion.Status.String(), TerminalCode: completion.TerminalCode,
		BudgetName: completion.BudgetName, Input: completion.Input, Output: completion.Output,
	}
	if completion.BudgetUsage != nil {
		result.Usage = &workflowcontract.Usage{Tokens: int64(completion.BudgetUsage.GetTokens()), WallSeconds: completion.WallSeconds, TokensKnown: true}
	}
	if receipt := completion.ChargeReceipt; receipt != nil {
		if result.Usage == nil {
			result.Usage = &workflowcontract.Usage{WallSeconds: completion.WallSeconds}
		}
		// CostUsd is a readable historical estimate. Only an explicitly
		// measured receipt with an amount can settle charge accounting.
		if receipt.Measured && receipt.AmountMicroUsd != nil && *receipt.AmountMicroUsd >= 0 {
			result.Usage.ChargeMicroUSD = *receipt.AmountMicroUsd
			result.Usage.ChargeMeasured = true
		}
	}
	return result, nil
}

func (a *DevelopmentInvoker) Cancel(ctx context.Context, executionID, idempotencyKey, reason string) error {
	if a == nil || a.invoker == nil {
		return agentmanager.ErrNotAvailable
	}
	canceller, ok := a.invoker.(interface {
		CancelWorkflow(context.Context, string, string, string) error
	})
	if !ok {
		return fmt.Errorf("workflow cancellation is not supported")
	}
	return canceller.CancelWorkflow(ctx, executionID, idempotencyKey, reason)
}

func toAgentGrant(grant *workflowcontract.Grant) (*domainpb.WorkflowEngagementGrant, error) {
	if grant == nil {
		return nil, nil
	}
	if grant.MaxTokens <= 0 || grant.MaxWallTimeSeconds <= 0 || grant.MaxTokens > math.MaxInt32 || grant.MaxWallTimeSeconds > math.MaxInt32 {
		return nil, fmt.Errorf("workflow engagement grant exceeds Agent Manager limits: %w", agentmanager.ErrRequestFailed)
	}
	out := &domainpb.WorkflowEngagementGrant{
		MaxTurns: int32(grant.MaxTurns), MaxTokens: int32(grant.MaxTokens), MaxChargeMicroUsd: grant.MaxChargeMicroUSD, MaxWallTimeSeconds: int32(grant.MaxWallTimeSeconds), MaxNodeAttempts: int32(grant.MaxNodeAttempts), MaxChildren: int32(grant.MaxChildren), MaxConcurrency: int32(grant.MaxConcurrency), MaxRecursion: int32(grant.MaxRecursion), MaxWaitSeconds: int32(grant.MaxWaitSeconds),
		AllowedEffects: append([]string(nil), grant.AllowedEffects...),
	}
	if grant.RetryLimitSet || grant.MaxRetries > 0 {
		out.MaxRetries = proto.Int32(int32(grant.MaxRetries))
	}
	return out, nil
}

var (
	_ workflowcontract.Invoker   = (*DevelopmentInvoker)(nil)
	_ workflowcontract.Canceller = (*DevelopmentInvoker)(nil)
)
