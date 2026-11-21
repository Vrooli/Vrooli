package runs

import (
	"context"
	"fmt"
	"time"
)

// An automatic retry exists for a failure that looks transient. When the retry
// itself ends the same way, trying again a minute later cannot change the
// answer, so each further retry of the same plan waits longer: 5 minutes, then
// 15, 45, 2h15m, and never more than 6 hours between retries. A run that
// succeeds, a different failure, or any scheduled or manual run starts the
// count again, and an operator's own trigger is never held back.
const (
	retryBackoffBase = 5 * time.Minute
	retryBackoffCap  = 6 * time.Hour
	// retryStreakWindow bounds the history read; the cap is reached well before.
	retryStreakWindow = 8
)

// ErrRetryBackoff is returned instead of starting an automatic retry that would
// repeat the failure the previous retries already ended with.
type ErrRetryBackoff struct {
	PlanID string
	// Until is the earliest time the next automatic retry is allowed.
	Until time.Time
	// Failed is how many automatic retries in a row have failed this way.
	Failed int
}

func (e ErrRetryBackoff) Error() string {
	return fmt.Sprintf("plan %q automatic retry held until %s after %d failed retries",
		e.PlanID, e.Until.Format(time.RFC3339), e.Failed)
}

// retryDelay is how long to wait after the nth consecutive failed retry.
func retryDelay(failedRetries int) time.Duration {
	delay := retryBackoffBase
	for i := 1; i < failedRetries; i++ {
		delay *= 3
		if delay >= retryBackoffCap {
			return retryBackoffCap
		}
	}
	return delay
}

// retryHeldUntil reports when the next automatic retry of a plan is allowed.
// newestFirst is the plan's recent run history. The zero time means go ahead.
func retryHeldUntil(newestFirst []Run) (until time.Time, failed int) {
	var code string
	for i, r := range newestFirst {
		if r.Trigger != TriggerRetry || r.Status != RunFailed {
			break
		}
		if i == 0 {
			code = string(r.FailureCode)
		} else if string(r.FailureCode) != code {
			break
		}
		failed++
	}
	if failed == 0 {
		return time.Time{}, 0
	}
	return newestFirst[0].FinishedAt.Add(retryDelay(failed)), failed
}

func (s *service) checkRetryAllowed(ctx context.Context, planID string, now time.Time) error {
	history, err := s.deps.Repo.ListRuns(ctx, planID, retryStreakWindow)
	if err != nil {
		return fmt.Errorf("read run history for retry: %w", err)
	}
	if until, failed := retryHeldUntil(history); now.Before(until) {
		return ErrRetryBackoff{PlanID: planID, Until: until, Failed: failed}
	}
	return nil
}
