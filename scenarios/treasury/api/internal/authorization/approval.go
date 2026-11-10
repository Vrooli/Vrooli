package authorization

import (
	"context"
	"fmt"

	"github.com/vrooli/api-core/schedule"
	"treasury/internal/budget"
)

// ApprovalController revalidates the durable grant, freezes and funding when
// an operator acts on a queued request. Approval never enlarges a mandate or
// the budget's total/period limits. It is a one-charge exception to the
// automatic per-transaction cap only.
type ApprovalController struct {
	Repository
	mandates Mandates
	budgets  Budgets
	clock    schedule.Clock
}

func NewApprovalController(repo Repository, mandates Mandates, budgets Budgets, clock schedule.Clock) *ApprovalController {
	return &ApprovalController{Repository: repo, mandates: mandates, budgets: budgets, clock: clock}
}

func (c *ApprovalController) Approve(ctx context.Context, id string) (Record, error) {
	current, err := c.Repository.Get(ctx, id)
	if err != nil {
		return Record{}, err
	}
	if current.Verdict == VerdictApproved {
		return current, nil
	}
	now := c.clock.Now().UTC()
	if current.Verdict != VerdictPending || !now.Before(current.ExpiresAt) {
		return Record{}, fmt.Errorf("%w: queued authorization expired", ErrInvalid)
	}
	grant, err := c.mandates.RequireLive(ctx, current.MandateID)
	if err != nil {
		return Record{}, err
	}
	policy, err := c.budgets.Get(ctx, current.BudgetID)
	if err != nil {
		return Record{}, err
	}
	if grant.BookID != current.BookID || grant.BudgetID != current.BudgetID || grant.Currency != current.Currency || policy.Currency != current.Currency || !grant.AllowsCounterparty(current.Counterparty) || !policy.AllowsCounterparty(current.Counterparty) {
		return Record{}, fmt.Errorf("%w: queued authorization scope changed", ErrInvalid)
	}
	if freezes, ok := c.budgets.(FreezeReader); ok {
		frozen, _, err := freezes.IsFrozen(ctx, current.BookID, current.BudgetID)
		if err != nil {
			return Record{}, err
		}
		if frozen {
			return Record{}, fmt.Errorf("%w: budget is frozen", ErrInvalid)
		}
	} else if policy.Frozen {
		return Record{}, fmt.Errorf("%w: budget is frozen", ErrInvalid)
	}
	usage, err := c.Repository.Usage(ctx, current.BudgetID, current.MandateID, periodBoundary(now, policy.Period), now)
	if err != nil {
		return Record{}, err
	}
	// Pending requests already hold their exact amount in all usage totals.
	if usage.BudgetTotalMinor > policy.TotalCapMinor || usage.BudgetPeriodMinor > policy.PeriodicCapMinor || usage.MandateTotalMinor > grant.CapMinor {
		return Record{}, fmt.Errorf("%w: queued authorization exceeds current headroom", ErrInvalid)
	}
	if funded, ok := c.budgets.(budget.FundingReader); ok {
		funding, err := funded.Funding(ctx, current.BudgetID)
		if err != nil {
			return Record{}, err
		}
		if funding.Required && usage.BudgetTotalMinor > funding.CreditedMinor {
			return Record{}, fmt.Errorf("%w: receipt funding no longer covers the held charge", ErrInvalid)
		}
	}
	return c.Repository.Approve(ctx, id)
}
