// Package circuitbreaker implements the circuit breaker pattern for collector resilience
// [REQ:SCS-CB-001] Auto-disable failing collectors
// [REQ:SCS-CB-002] Failure tracking
// [REQ:SCS-CB-003] Periodic retry
// [REQ:SCS-CB-004] Circuit breaker reset API
package circuitbreaker

import (
	"sync"
	"time"
)

// State represents the current state of a circuit breaker
type State int

const (
	// Closed means the circuit is working normally
	Closed State = iota
	// Open means the circuit has tripped and requests will fast-fail
	Open
	// HalfOpen means the circuit is testing if the underlying service has recovered
	HalfOpen
)

// String returns the string representation of the state
func (s State) String() string {
	switch s {
	case Closed:
		return "closed"
	case Open:
		return "open"
	case HalfOpen:
		return "half_open"
	default:
		return "unknown"
	}
}

// Config holds circuit breaker configuration
// [REQ:SCS-CB-001] Configurable failure threshold
type Config struct {
	// FailureThreshold is the number of consecutive failures before opening the circuit
	FailureThreshold int `json:"failure_threshold"`
	// RetryInterval is how long to wait before attempting a half-open test
	RetryInterval time.Duration `json:"retry_interval"`
	// Timeout is how long to wait for an operation before considering it failed
	Timeout time.Duration `json:"timeout"`
}

// DefaultConfig returns the default circuit breaker configuration
func DefaultConfig() Config {
	return Config{
		FailureThreshold: 3,
		RetryInterval:    30 * time.Second,
		Timeout:          10 * time.Second,
	}
}

// CircuitBreaker implements the circuit breaker pattern
// [REQ:SCS-CB-002] Tracks consecutive failures per collector
type CircuitBreaker struct {
	mu sync.RWMutex

	name           string
	state          State
	failureCount   int
	lastFailure    time.Time
	lastSuccess    time.Time
	lastStateChange time.Time
	config         Config
}

// New creates a new circuit breaker with the given name and config
func New(name string, cfg Config) *CircuitBreaker {
	return &CircuitBreaker{
		name:            name,
		state:           Closed,
		config:          cfg,
		lastStateChange: time.Now(),
	}
}

// Name returns the circuit breaker's name
func (cb *CircuitBreaker) Name() string {
	return cb.name
}

// State returns the current state of the circuit breaker
func (cb *CircuitBreaker) State() State {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.state
}

// FailureCount returns the current failure count
func (cb *CircuitBreaker) FailureCount() int {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.failureCount
}

// LastFailure returns the time of the last failure
func (cb *CircuitBreaker) LastFailure() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastFailure
}

// LastSuccess returns the time of the last success
func (cb *CircuitBreaker) LastSuccess() time.Time {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.lastSuccess
}

// Config returns the circuit breaker's configuration
func (cb *CircuitBreaker) Config() Config {
	cb.mu.RLock()
	defer cb.mu.RUnlock()
	return cb.config
}

// Allow checks if a request should be allowed through the circuit breaker
// Returns true if the request is allowed, false if it should fast-fail
// [REQ:SCS-CB-003] Periodic retry via half-open state
func (cb *CircuitBreaker) Allow() bool {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	switch cb.state {
	case Closed:
		return true
	case Open:
		// Check if we should transition to half-open
		if time.Since(cb.lastStateChange) >= cb.config.RetryInterval {
			cb.state = HalfOpen
			cb.lastStateChange = time.Now()
			return true
		}
		return false
	case HalfOpen:
		// In half-open, allow exactly one request through to test
		return true
	default:
		return false
	}
}

// RecordSuccess records a successful operation
// [REQ:SCS-CB-003] Recovery after successful retry
func (cb *CircuitBreaker) RecordSuccess() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastSuccess = time.Now()
	cb.failureCount = 0

	// If we're in half-open and succeeded, close the circuit
	if cb.state == HalfOpen {
		cb.state = Closed
		cb.lastStateChange = time.Now()
	}
}

// RecordFailure records a failed operation
// [REQ:SCS-CB-001] Auto-disable after threshold failures
func (cb *CircuitBreaker) RecordFailure() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.lastFailure = time.Now()
	cb.failureCount++

	switch cb.state {
	case Closed:
		// Check if we should trip the breaker
		if cb.failureCount >= cb.config.FailureThreshold {
			cb.state = Open
			cb.lastStateChange = time.Now()
		}
	case HalfOpen:
		// Failed during half-open test, go back to open
		cb.state = Open
		cb.lastStateChange = time.Now()
		// Don't reset failure count, keep it to show history
	}
}

// Reset manually resets the circuit breaker to closed state
// [REQ:SCS-CB-004] Manual reset capability
func (cb *CircuitBreaker) Reset() {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	cb.state = Closed
	cb.failureCount = 0
	cb.lastStateChange = time.Now()
}

// Status returns a snapshot of the circuit breaker's current status
type Status struct {
	Name            string    `json:"name"`
	State           string    `json:"state"`
	FailureCount    int       `json:"failure_count"`
	FailureThreshold int      `json:"failure_threshold"`
	LastFailure     *string   `json:"last_failure,omitempty"`
	LastSuccess     *string   `json:"last_success,omitempty"`
	LastStateChange string    `json:"last_state_change"`
	RetryInterval   string    `json:"retry_interval"`
}

// Status returns the current status of the circuit breaker
func (cb *CircuitBreaker) Status() Status {
	cb.mu.RLock()
	defer cb.mu.RUnlock()

	status := Status{
		Name:             cb.name,
		State:            cb.state.String(),
		FailureCount:     cb.failureCount,
		FailureThreshold: cb.config.FailureThreshold,
		LastStateChange:  cb.lastStateChange.Format(time.RFC3339),
		RetryInterval:    cb.config.RetryInterval.String(),
	}

	if !cb.lastFailure.IsZero() {
		t := cb.lastFailure.Format(time.RFC3339)
		status.LastFailure = &t
	}
	if !cb.lastSuccess.IsZero() {
		t := cb.lastSuccess.Format(time.RFC3339)
		status.LastSuccess = &t
	}

	return status
}
