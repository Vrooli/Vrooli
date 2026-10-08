package workflowruntime

import (
	"time"

	"agent-manager/internal/domain"
)

// Exhaustion prevents additional work; exceeding a limit also prevents a
// successful terminal result. Keep those predicates distinct so completing
// exactly at a limit can succeed without authorizing another child or repair.
func exhaustedAgentBudget(usage domain.WorkflowBudgetUsage, budgets domain.WorkflowBudgets) string {
	switch {
	case usage.Turns >= budgets.MaxTurns:
		return "turns"
	case usage.Tokens >= budgets.MaxTokens:
		return "tokens"
	case usage.ChargeMicroUSD >= budgets.MaxChargeMicroUSD:
		return "charge"
	default:
		return ""
	}
}

// These are the limits the child API can enforce today. Token/charge admission
// uses reported aggregate usage; it is NOT a hard in-flight token/charge cap,
// nor a reservation across concurrently running children.
func (e *Engine) boundAgentRequest(request *ChildRequest, x *domain.WorkflowExecution, budgets domain.WorkflowBudgets, journal []*domain.WorkflowJournalEntry) string {
	if budget := exhaustedAgentBudget(x.BudgetUsage, budgets); budget != "" {
		return budget
	}
	remainingTurns := budgets.MaxTurns - x.BudgetUsage.Turns
	if request.MaxTurns <= 0 || request.MaxTurns > remainingTurns {
		request.MaxTurns = remainingTurns
	}
	now := e.now()
	remainingTime := time.Duration(budgets.WallTimeSeconds)*time.Second - (now.Sub(x.CreatedAt) - waitedDuration(journal, now))
	if remainingTime <= 0 {
		return "wall_time"
	}
	if request.Timeout <= 0 || request.Timeout > remainingTime {
		request.Timeout = remainingTime
	}
	return ""
}
