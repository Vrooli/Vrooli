package execution

import (
	"context"
	"swarm-manager/internal/apierr"
	"swarm-manager/internal/workflowcontract"
)

// prepareExecutionGrantLocked derives the next reservation from the reviewed
// allowance and terminal owner receipts. The existing execution store and
// mutex serialize reservation with dispatch; retries do not create a budget.
func (s *Service) prepareExecutionGrantLocked(ctx context.Context, records []Record, record *Record, item backlogItem) error {
	if record.ExecutionLimits == nil {
		return nil
	}
	if err := record.ExecutionLimits.Validate(); err != nil {
		return apierr.BadRequest("%s", err)
	}
	if item.PlanAcceptance == nil || record.ApprovalDigest != digestStrings(item.PlanAcceptance.SubjectVersion, item.PlanAcceptance.PlanContentHash) {
		return apierr.Conflict("execution limits are not bound to the currently accepted work contract")
	}
	if record.WorkflowGrant != nil {
		return nil
	} // Replay the retained reservation.
	limits := record.ExecutionLimits
	remaining := workflowcontract.Grant{MaxTurns: limits.MaxTurns, MaxTokens: limits.MaxTokens, MaxWallTimeSeconds: limits.MaxWallSeconds, MaxChargeMicroUSD: limits.MaxChargeMicroUSD, MaxChildren: limits.MaxChildren, MaxNodeAttempts: limits.MaxNodeAttempts, MaxRetries: limits.MaxRetries, MaxConcurrency: 1, MaxRecursion: 3, MaxWaitSeconds: 3600}
	remaining.RetryLimitSet = true
	priorReservations := 0
	slices := limits.MaxSlices
	for i := range records {
		prior := &records[i]
		if prior.ExecutionID == record.ExecutionID || prior.BacklogKind != record.BacklogKind || prior.BacklogName != record.BacklogName || prior.ApprovalDigest != record.ApprovalDigest || prior.WorkflowGrant == nil {
			continue
		}
		priorReservations++
		if prior.SettledUsage == nil || !prior.SettledUsage.TokensKnown || !prior.SettledUsage.ChargeMeasured {
			correlation, err := s.transitionCorrelation(*prior)
			if err != nil {
				return apierr.Conflict("prior execution %s has an unresolved dispatch reservation", prior.ExecutionID)
			}
			usage, err := s.transitionRunner.CollectUsage(ctx, correlation.ExecutionID, prior.ApprovalDigest, workflowcontract.GrantDigest(prior.WorkflowGrant))
			if err != nil {
				return apierr.Conflict("prior execution %s accounting remains unresolved: %s", prior.ExecutionID, err)
			}
			prior.SettledUsage = usage
		}
		usage := prior.SettledUsage
		if usage == nil || !usage.TokensKnown || !usage.ChargeMeasured || usage.WallSeconds <= 0 || usage.Tokens < 0 || usage.Turns < 0 || usage.ChargeMicroUSD < 0 || usage.Children < 0 || usage.NodeAttempts < 0 || usage.Retries < 0 || usage.Slices < 0 {
			return apierr.Conflict("prior execution %s lacks terminal token, wall-time, or charge accounting; its reservation remains held", prior.ExecutionID)
		}
		remaining.MaxTokens -= usage.Tokens
		remaining.MaxTurns -= int(usage.Turns)
		remaining.MaxWallTimeSeconds -= usage.WallSeconds
		remaining.MaxChargeMicroUSD -= usage.ChargeMicroUSD
		remaining.MaxChildren -= int(usage.Children)
		remaining.MaxNodeAttempts -= int(usage.NodeAttempts)
		remaining.MaxRetries -= int(usage.Retries)
		slices -= int(usage.Slices)
	}
	// Every additional reservation under this acceptance is a retry, including
	// a fresh Queue/Start whose operation label is not "retry". One original
	// attempt plus N prior reservations consumes N outer retries here.
	remaining.MaxRetries -= priorReservations
	if remaining.MaxTokens <= 0 || remaining.MaxTurns <= 0 || remaining.MaxWallTimeSeconds <= 0 || remaining.MaxChargeMicroUSD <= 0 || remaining.MaxChildren <= 0 || remaining.MaxNodeAttempts <= 0 || remaining.MaxRetries < 0 || slices <= 0 {
		return apierr.Conflict("the accepted aggregate execution allowance is exhausted; a new reviewed allowance is required")
	}
	if record.MaxSlices <= 0 || record.MaxSlices > slices {
		record.MaxSlices = slices
	}
	record.WorkflowGrant = &remaining
	return nil
}
