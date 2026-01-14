package queue

import (
	"log"
	"sync"
	"time"
)

// RateLimiter manages rate limiting logic for the queue processor.
// It provides pure logic for checking and managing rate limit state.
type RateLimiter struct {
	mu         sync.Mutex
	paused     bool
	pauseUntil time.Time
	broadcast  chan<- any
}

// NewRateLimiter creates a new RateLimiter with optional broadcast channel.
func NewRateLimiter(broadcast chan<- any) *RateLimiter {
	return &RateLimiter{
		broadcast: broadcast,
	}
}

// RateLimitStatus represents the current rate limit state
type RateLimitStatus struct {
	IsPaused      bool
	PauseUntil    time.Time
	RemainingTime time.Duration
	JustResumed   bool // True if pause just expired on this check
}

// CheckLimit checks if processing is rate limited.
// Returns status indicating if processing should continue.
// Also handles auto-expiration and broadcasts appropriately.
func (r *RateLimiter) CheckLimit() RateLimitStatus {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.paused {
		return RateLimitStatus{IsPaused: false}
	}

	now := time.Now()
	if now.Before(r.pauseUntil) {
		remaining := r.pauseUntil.Sub(now)
		log.Printf("⏸️ Queue paused due to rate limit. Resuming in %v", remaining.Round(time.Second))

		// Broadcast pause status
		r.broadcastUpdate("rate_limit_pause", map[string]any{
			"paused":         true,
			"pause_until":    r.pauseUntil.Format(time.RFC3339),
			"remaining_secs": int(remaining.Seconds()),
		})

		return RateLimitStatus{
			IsPaused:      true,
			PauseUntil:    r.pauseUntil,
			RemainingTime: remaining,
		}
	}

	// Pause has expired, auto-resume
	r.paused = false
	r.pauseUntil = time.Time{}
	log.Printf("✅ Rate limit pause expired. Resuming queue processing.")

	// Broadcast resume
	r.broadcastUpdate("rate_limit_resume", map[string]any{
		"paused": false,
	})

	return RateLimitStatus{IsPaused: false, JustResumed: true}
}

// HandlePause pauses processing due to rate limiting for the specified duration.
// Duration is capped at 4 hours and has a minimum of 5 minutes.
func (r *RateLimiter) HandlePause(retryAfterSeconds int) {
	r.mu.Lock()
	defer r.mu.Unlock()

	// Cap the pause duration at 4 hours
	if retryAfterSeconds > 14400 {
		retryAfterSeconds = 14400
	}

	// Minimum pause of 5 minutes
	if retryAfterSeconds < 300 {
		retryAfterSeconds = 300
	}

	pauseDuration := time.Duration(retryAfterSeconds) * time.Second
	r.paused = true
	r.pauseUntil = time.Now().Add(pauseDuration)

	log.Printf("🛑 RATE LIMIT HIT: Pausing queue processor for %v (until %s)",
		pauseDuration, r.pauseUntil.Format(time.RFC3339))

	// Broadcast the pause event
	r.broadcastUpdate("rate_limit_pause_started", map[string]any{
		"pause_duration": retryAfterSeconds,
		"pause_until":    r.pauseUntil.Format(time.RFC3339),
		"reason":         "API rate limit reached",
	})
}

// Reset manually clears any active rate limit pause.
func (r *RateLimiter) Reset() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	wasRateLimited := r.paused
	r.paused = false
	r.pauseUntil = time.Time{}

	if wasRateLimited {
		log.Printf("✅ Rate limit pause manually reset. Queue processing resumed.")

		// Broadcast the resume event
		r.broadcastUpdate("rate_limit_manual_reset", map[string]any{
			"paused": false,
			"manual": true,
		})
	}

	return wasRateLimited
}

// IsPaused returns (isPaused, pauseUntil) without triggering auto-expiration.
// Use this for status queries without side effects.
func (r *RateLimiter) IsPaused() (bool, time.Time) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.paused && time.Now().Before(r.pauseUntil) {
		return true, r.pauseUntil
	}
	return false, time.Time{}
}

// broadcastUpdate sends an event to the broadcast channel if configured.
func (r *RateLimiter) broadcastUpdate(event string, data map[string]any) {
	if r.broadcast == nil {
		return
	}

	select {
	case r.broadcast <- map[string]any{
		"type": event,
		"data": data,
	}:
	default:
		// Channel full, skip broadcast
	}
}
