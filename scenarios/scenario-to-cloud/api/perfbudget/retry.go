package perfbudget

import (
	"fmt"
	"math"
	"time"
)

// Backoff returns the delay before attempt number `attempt` (1-based; the
// first attempt has no delay) and whether the attempt is admitted at all.
// Beyond MaxAttempts it returns (0, false): the caller parks the operation
// with a typed next action instead of retrying.
func (r RetryBudget) Backoff(attempt int) (time.Duration, bool) {
	if attempt < 1 || attempt > r.MaxAttempts {
		return 0, false
	}
	if attempt == 1 {
		return 0, true
	}
	seconds := r.InitialBackoffSeconds * math.Pow(r.BackoffMultiplier, float64(attempt-2))
	if r.BackoffCapSeconds > 0 && seconds > r.BackoffCapSeconds {
		seconds = r.BackoffCapSeconds
	}
	return time.Duration(seconds * float64(time.Second)), true
}

// TotalBackoff is the worst-case wall clock a class can spend waiting over
// all admitted attempts. It is finite for every valid budget.
func (r RetryBudget) TotalBackoff() time.Duration {
	var total time.Duration
	for attempt := 1; ; attempt++ {
		d, ok := r.Backoff(attempt)
		if !ok {
			return total
		}
		total += d
	}
}

// RetryFor returns the budget of a retry class or an error naming the
// missing class; callers never fall back to an unbounded default.
func (b *Budgets) RetryFor(class string) (RetryBudget, error) {
	rb, ok := b.Phase23.Retries[class]
	if !ok {
		return RetryBudget{}, fmt.Errorf("retry class %q has no frozen budget", class)
	}
	return rb, nil
}
