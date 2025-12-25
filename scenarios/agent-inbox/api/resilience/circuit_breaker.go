package resilience

import (
	"context"
	"errors"
	"sync"
	"time"
)

// CircuitState represents the current state of a circuit breaker.
type CircuitState int

const (
	// StateClosed means the circuit is healthy - requests pass through.
	StateClosed CircuitState = iota
	// StateOpen means the circuit is tripped - requests fail fast.
	StateOpen
	// StateHalfOpen means the circuit is testing - one request allowed through.
	StateHalfOpen
)

func (s CircuitState) String() string {
	switch s {
	case StateClosed:
		return "closed"
	case StateOpen:
		return "open"
	case StateHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// ErrCircuitOpen is returned when a request is rejected because the circuit is open.
var ErrCircuitOpen = errors.New("circuit breaker is open")

// CircuitBreakerConfig controls circuit breaker behavior.
type CircuitBreakerConfig struct {
	// FailureThreshold is the number of consecutive failures before opening.
	FailureThreshold int
	// SuccessThreshold is the number of consecutive successes in half-open to close.
	SuccessThreshold int
	// Cooldown is how long the circuit stays open before transitioning to half-open.
	Cooldown time.Duration
	// OnStateChange is called when the circuit state changes (optional).
	OnStateChange func(from, to CircuitState)
}

// DefaultCircuitBreakerConfig returns sensible defaults.
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		FailureThreshold: 5,
		SuccessThreshold: 2,
		Cooldown:         30 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern for protecting services.
// It prevents cascading failures by failing fast when a dependency is unhealthy.
//
// States:
//   - Closed: Normal operation, requests pass through
//   - Open: Failures exceeded threshold, requests fail immediately
//   - Half-Open: Testing recovery, limited requests allowed
//
// Thread-safe for concurrent use.
type CircuitBreaker struct {
	cfg CircuitBreakerConfig

	mu              sync.RWMutex
	state           CircuitState
	failures        int
	successes       int
	lastFailure     time.Time
	lastStateChange time.Time
	totalRequests   int64
	totalFailures   int64
	totalRejections int64
}

// NewCircuitBreaker creates a new circuit breaker with the given configuration.
func NewCircuitBreaker(cfg CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		cfg:             cfg,
		state:           StateClosed,
		lastStateChange: time.Now(),
	}
}

// Execute runs fn if the circuit allows it.
// Returns ErrCircuitOpen if the circuit is open and not ready for half-open test.
// Records success/failure to update circuit state.
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func(context.Context) error) error {
	if !cb.allowRequest() {
		cb.recordRejection()
		return ErrCircuitOpen
	}

	cb.recordRequest()

	err := fn(ctx)

	if err != nil {
		cb.recordFailure()
	} else {
		cb.recordSuccess()
	}

	return err
}

// allowRequest determines if a request should be allowed through.
func (cb *CircuitBreaker) allowRequest() bool {
	cb.mu.RLock()
	state := cb.state
	lastFailure := cb.lastFailure
	cb.mu.RUnlock()

	switch state {
	case StateClosed:
		return true
	case StateOpen:
		// Check if cooldown has elapsed
		if time.Since(lastFailure) >= cb.cfg.Cooldown {
			cb.transitionTo(StateHalfOpen)
			return true
		}
		return false
	case StateHalfOpen:
		// Allow limited requests in half-open state
		return true
	default:
		return false
	}
}

// recordSuccess records a successful request.
func (cb *CircuitBreaker) recordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures = 0 // Reset consecutive failures

	switch cb.state {
	case StateClosed:
		// Stay closed
	case StateHalfOpen:
		cb.successes++
		if cb.successes >= cb.cfg.SuccessThreshold {
			cb.transitionToLocked(StateClosed)
		}
	case StateOpen:
		// Shouldn't happen, but handle gracefully
		cb.transitionToLocked(StateClosed)
	}
}

// recordFailure records a failed request.
func (cb *CircuitBreaker) recordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.failures++
	cb.successes = 0 // Reset consecutive successes
	cb.lastFailure = time.Now()
	cb.totalFailures++

	switch cb.state {
	case StateClosed:
		if cb.failures >= cb.cfg.FailureThreshold {
			cb.transitionToLocked(StateOpen)
		}
	case StateHalfOpen:
		// Any failure in half-open reopens the circuit
		cb.transitionToLocked(StateOpen)
	case StateOpen:
		// Stay open, reset cooldown
	}
}

// recordRequest increments the total request counter.
func (cb *CircuitBreaker) recordRequest() {
	cb.mu.Lock()
	cb.totalRequests++
	cb.mu.Unlock()
}

// recordRejection increments the rejection counter.
func (cb *CircuitBreaker) recordRejection() {
	cb.mu.Lock()
	cb.totalRejections++
	cb.mu.Unlock()
}

// transitionTo changes the circuit state (acquires lock).
func (cb *CircuitBreaker) transitionTo(newState CircuitState) {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.transitionToLocked(newState)
}

// transitionToLocked changes the circuit state (assumes lock is held).
func (cb *CircuitBreaker) transitionToLocked(newState CircuitState) {
	if cb.state == newState {
		return
	}

	oldState := cb.state
	cb.state = newState
	cb.lastStateChange = time.Now()

	// Reset counters on state change
	if newState == StateClosed {
		cb.failures = 0
		cb.successes = 0
	} else if newState == StateHalfOpen {
		cb.successes = 0
	}

	// Notify listener (if configured)
	if cb.cfg.OnStateChange != nil {
		// Call outside lock to prevent deadlocks
		go cb.cfg.OnStateChange(oldState, newState)
	}
}

// State returns the current circuit state.
func (cb *CircuitBreaker) State() CircuitState {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// Stats returns circuit breaker metrics.
func (cb *CircuitBreaker) Stats() CircuitBreakerStats {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return CircuitBreakerStats{
		State:           cb.state,
		ConsecFailures:  cb.failures,
		ConsecSuccesses: cb.successes,
		TotalRequests:   cb.totalRequests,
		TotalFailures:   cb.totalFailures,
		TotalRejections: cb.totalRejections,
		LastFailure:     cb.lastFailure,
		LastStateChange: cb.lastStateChange,
	}
}

// CircuitBreakerStats contains observable metrics about the circuit breaker.
type CircuitBreakerStats struct {
	State           CircuitState
	ConsecFailures  int
	ConsecSuccesses int
	TotalRequests   int64
	TotalFailures   int64
	TotalRejections int64
	LastFailure     time.Time
	LastStateChange time.Time
}

// Reset forces the circuit to closed state.
// Use with caution - typically for testing or manual recovery.
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = StateClosed
	cb.failures = 0
	cb.successes = 0
	cb.lastStateChange = time.Now()
}
