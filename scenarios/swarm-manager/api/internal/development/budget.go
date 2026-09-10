package development

import "fmt"

const (
	BudgetMetered = "metered-cancellation"
	BudgetHard    = "hard-ceiling"
)

// Default only NEW proposals. An absent policy on a retained historical
// snapshot must never acquire overshoot authority without a new approval.
func defaultBudgetPolicy(policy string) string {
	if policy == "" {
		return BudgetMetered
	}
	return policy
}

func validateBudgetPolicy(policy string) error {
	if policy != BudgetMetered && policy != BudgetHard {
		return fmt.Errorf("unsupported development budget policy: %w", ErrInvalid)
	}
	return nil
}

func renderBudgetPolicy(policy string) string {
	if defaultBudgetPolicy(policy) == BudgetMetered {
		return "Budget policy: metered cancellation. The execution owner requests cancellation when reported usage reaches the allowance. Provider reporting and cancellation can lag; in-flight usage may overshoot. All observed usage remains charged to this engagement. This is not a hard token ceiling and does not authorize another attempt after exhaustion.\n"
	}
	return "Budget policy: hard ceiling. Admit only an execution path that proves enforcement of the selected ceiling across its children and continuations. Refuse unsupported paths; do not silently substitute metered cancellation.\n"
}
