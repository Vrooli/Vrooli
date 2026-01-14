package queue

import "time"

// handleRateLimitPause pauses the queue processor due to rate limiting.
// Delegates to the RateLimiter component.
func (qp *Processor) handleRateLimitPause(retryAfterSeconds int) {
	qp.rateLimiter.HandlePause(retryAfterSeconds)
}

// IsRateLimitPaused checks if the processor is currently paused due to rate limits.
// Delegates to the RateLimiter component.
func (qp *Processor) IsRateLimitPaused() (bool, time.Time) {
	return qp.rateLimiter.IsPaused()
}

// ResetRateLimitPause manually resets the rate limit pause.
// Delegates to the RateLimiter component.
func (qp *Processor) ResetRateLimitPause() {
	qp.rateLimiter.Reset()
}
