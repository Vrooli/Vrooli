package queue

import (
	"log"
	"time"
)

// handleRateLimitPause pauses the queue processor due to rate limiting
func (qp *Processor) handleRateLimitPause(retryAfterSeconds int) {
	qp.pauseMutex.Lock()
	defer qp.pauseMutex.Unlock()

	// Cap the pause duration at 4 hours
	if retryAfterSeconds > 14400 {
		retryAfterSeconds = 14400
	}

	// Minimum pause of 5 minutes
	if retryAfterSeconds < 300 {
		retryAfterSeconds = 300
	}

	pauseDuration := time.Duration(retryAfterSeconds) * time.Second
	qp.rateLimitPaused = true
	qp.pauseUntil = time.Now().Add(pauseDuration)

	log.Printf("🛑 RATE LIMIT HIT: Pausing queue processor for %v (until %s)",
		pauseDuration, qp.pauseUntil.Format(time.RFC3339))

	// Broadcast the pause event
	qp.broadcastUpdate("rate_limit_pause_started", map[string]interface{}{
		"pause_duration": retryAfterSeconds,
		"pause_until":    qp.pauseUntil.Format(time.RFC3339),
		"reason":         "API rate limit reached",
	})
}

// IsRateLimitPaused checks if the processor is currently paused due to rate limits
func (qp *Processor) IsRateLimitPaused() (bool, time.Time) {
	qp.pauseMutex.Lock()
	defer qp.pauseMutex.Unlock()

	if qp.rateLimitPaused && time.Now().Before(qp.pauseUntil) {
		return true, qp.pauseUntil
	}
	return false, time.Time{}
}

// ResetRateLimitPause manually resets the rate limit pause
func (qp *Processor) ResetRateLimitPause() {
	qp.pauseMutex.Lock()
	defer qp.pauseMutex.Unlock()

	wasRateLimited := qp.rateLimitPaused
	qp.rateLimitPaused = false
	qp.pauseUntil = time.Time{}

	if wasRateLimited {
		log.Printf("✅ Rate limit pause manually reset. Queue processing resumed.")

		// Broadcast the resume event
		qp.broadcastUpdate("rate_limit_manual_reset", map[string]interface{}{
			"paused": false,
			"manual": true,
		})
	}
}
