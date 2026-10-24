package execution

import (
	"context"
	"strings"

	"swarm-manager/internal/apierr"
	"swarm-manager/internal/workflowcontract"
)

// Documented child-workflow defaults. They apply only when no admitted effort
// aggregate grant references the item's canonical plan. An admitted effort
// binds these values to its reviewed aggregate allowance instead, so they are
// never the sole authority for a goal-directed effort.
const (
	defaultChildConcurrency = 1
	defaultChildRecursion   = 3
	defaultChildWaitSeconds = 3600
)

// childGrantLimits resolves the effort-wide child limits for one item. The
// projection is additive and non-weakening: an unset, unowned or unreachable
// aggregate grant leaves the documented defaults in force, while an admitted
// effort overrides only the fields it explicitly bounds.
func (s *Service) childGrantLimits(ctx context.Context, item backlogItem) (concurrency, recursion, waitSeconds int) {
	concurrency, recursion, waitSeconds = defaultChildConcurrency, defaultChildRecursion, defaultChildWaitSeconds
	if s == nil || s.aggregateGrants == nil || item.PlanRef == nil {
		return concurrency, recursion, waitSeconds
	}
	planID := strings.TrimSpace(item.PlanRef.PlanID)
	if planID == "" {
		planID = strings.TrimSpace(item.PlanRef.Slug)
	}
	if planID == "" {
		return concurrency, recursion, waitSeconds
	}
	grant, ok, err := s.aggregateGrants.AggregateGrantForPlan(ctx, planID)
	if err != nil || !ok {
		return concurrency, recursion, waitSeconds
	}
	if grant.MaxConcurrency > 0 {
		concurrency = grant.MaxConcurrency
	}
	if grant.MaxRecursion > 0 {
		recursion = grant.MaxRecursion
	}
	if grant.MaxWaitSeconds > 0 {
		waitSeconds = grant.MaxWaitSeconds
	}
	return concurrency, recursion, waitSeconds
}

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
	childConcurrency, childRecursion, childWaitSeconds := s.childGrantLimits(ctx, item)
	remaining := workflowcontract.Grant{MaxTurns: limits.MaxTurns, MaxTokens: limits.MaxTokens, MaxWallTimeSeconds: limits.MaxWallSeconds, MaxChargeMicroUSD: limits.MaxChargeMicroUSD, MaxChildren: limits.MaxChildren, MaxNodeAttempts: limits.MaxNodeAttempts, MaxRetries: limits.MaxRetries, MaxConcurrency: childConcurrency, MaxRecursion: childRecursion, MaxWaitSeconds: childWaitSeconds}
	remaining.RetryLimitSet = true
	priorReservations := 0
	slices := limits.MaxSlices
	for i := range records {
		prior := &records[i]
		if prior.ExecutionID == record.ExecutionID || prior.BacklogKind != record.BacklogKind || prior.BacklogName != record.BacklogName || prior.ApprovalDigest != record.ApprovalDigest || prior.WorkflowGrant == nil {
			continue
		}
		priorReservations++
		if isGoalRecord(*prior) && (prior.SettledUsage == nil || !prior.SettledUsage.TokensKnown || !prior.SettledUsage.ChargeMeasured) {
			// A goal run has no workflow receipt; its own owner metering is the
			// only source that can settle its reservation.
			usage, err := s.settledGoalUsage(ctx, *prior)
			if err != nil {
				return apierr.Conflict("prior execution %s accounting remains unresolved: %s", prior.ExecutionID, err)
			}
			if usage == nil {
				return apierr.Conflict("prior execution %s lacks terminal token, wall-time, or charge accounting; its reservation remains held", prior.ExecutionID)
			}
			prior.SettledUsage = usage
		}
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
