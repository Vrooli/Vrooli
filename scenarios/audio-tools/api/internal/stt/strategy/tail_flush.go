package strategy

import (
	"context"
	"time"
)

// canceledTailFlushBudget bounds the one best-effort transcription a strategy
// performs for audio the caller already handed it when the session context is
// cancelled. Cancellation is a stream end, not permission to drop a
// committed-but-unemitted tail, but teardown must also stay bounded: a wedged
// provider may not hold the transport open indefinitely. Providers honour the
// supplied context, so the timeout is the real upper bound on the attempt.
const canceledTailFlushBudget = 1500 * time.Millisecond

// bestEffortTailContext builds the context for that final attempt. It keeps
// the caller's values (credentials, trace ids) but detaches from the already
// cancelled Done channel, then applies the shorter of the production budget and
// whatever remains of the caller's own deadline. ok is false when the caller's
// deadline is already spent: there is no time left to finish a provider call, so
// the caller should emit its terminal Done and return immediately.
func bestEffortTailContext(ctx context.Context) (context.Context, context.CancelFunc, bool) {
	budget := canceledTailFlushBudget
	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return nil, func() {}, false
		}
		if remaining < budget {
			budget = remaining
		}
	}
	flushCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), budget)
	return flushCtx, cancel, true
}
