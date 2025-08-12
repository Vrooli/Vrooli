package main

import (
	"fmt"
	"sync"
	"time"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState int

const (
	Closed CircuitBreakerState = iota
	Open
	HalfOpen
)

func (s CircuitBreakerState) String() string {
	switch s {
	case Closed:
		return "CLOSED"
	case Open:
		return "OPEN"
	case HalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	MaxFailures     int           // Number of failures before opening
	ResetTimeout    time.Duration // Time before trying again
	SuccessThreshold int          // Successes needed in half-open to close
}

// DefaultCircuitBreakerConfig returns sensible defaults
func DefaultCircuitBreakerConfig() CircuitBreakerConfig {
	return CircuitBreakerConfig{
		MaxFailures:     3,                // Open after 3 failures
		ResetTimeout:    30 * time.Second, // Try again after 30 seconds
		SuccessThreshold: 2,               // Need 2 successes to close
	}
}

// CircuitBreaker implements the circuit breaker pattern
type CircuitBreaker struct {
	config          CircuitBreakerConfig
	state           CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	mutex           sync.RWMutex
	name            string // For logging
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  Closed,
		name:   name,
	}
}

// Call executes a function with circuit breaker protection
func (cb *CircuitBreaker) Call(fn func() error) error {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check if we should allow the call
	if !cb.shouldAllowCall() {
		return fmt.Errorf("circuit breaker %s is OPEN - service unavailable", cb.name)
	}

	// Execute the function
	err := fn()
	
	// Record the result
	if err != nil {
		cb.recordFailure()
		return fmt.Errorf("circuit breaker %s recorded failure: %w", cb.name, err)
	} else {
		cb.recordSuccess()
		return nil
	}
}

// shouldAllowCall determines if a call should be allowed based on circuit breaker state
func (cb *CircuitBreaker) shouldAllowCall() bool {
	switch cb.state {
	case Closed:
		return true
	case Open:
		// Check if enough time has passed to try again
		if time.Since(cb.lastFailureTime) > cb.config.ResetTimeout {
			cb.state = HalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case HalfOpen:
		return true
	default:
		return false
	}
}

// recordFailure records a failure and potentially opens the circuit
func (cb *CircuitBreaker) recordFailure() {
	cb.failureCount++
	cb.lastFailureTime = time.Now()
	
	switch cb.state {
	case Closed:
		if cb.failureCount >= cb.config.MaxFailures {
			cb.state = Open
			fmt.Printf("Circuit breaker %s OPENED after %d failures\n", cb.name, cb.failureCount)
		}
	case HalfOpen:
		cb.state = Open
		fmt.Printf("Circuit breaker %s returned to OPEN from half-open\n", cb.name)
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	cb.failureCount = 0 // Reset failure count on success
	
	switch cb.state {
	case HalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = Closed
			fmt.Printf("Circuit breaker %s CLOSED after %d successes\n", cb.name, cb.successCount)
		}
	}
}

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() CircuitBreakerState {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	return cb.state
}

// GetStats returns current statistics
func (cb *CircuitBreaker) GetStats() map[string]interface{} {
	cb.mutex.RLock()
	defer cb.mutex.RUnlock()
	
	return map[string]interface{}{
		"name":          cb.name,
		"state":         cb.state.String(),
		"failure_count": cb.failureCount,
		"success_count": cb.successCount,
		"last_failure":  cb.lastFailureTime,
	}
}

// CircuitBreakerManager manages multiple circuit breakers
type CircuitBreakerManager struct {
	breakers map[string]*CircuitBreaker
	mutex    sync.RWMutex
}

// NewCircuitBreakerManager creates a new manager
func NewCircuitBreakerManager() *CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]*CircuitBreaker),
	}
}

// GetBreaker gets or creates a circuit breaker for a service
func (m *CircuitBreakerManager) GetBreaker(name string) *CircuitBreaker {
	m.mutex.RLock()
	if breaker, exists := m.breakers[name]; exists {
		m.mutex.RUnlock()
		return breaker
	}
	m.mutex.RUnlock()

	// Create new breaker
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	// Double-check after acquiring write lock
	if breaker, exists := m.breakers[name]; exists {
		return breaker
	}
	
	config := DefaultCircuitBreakerConfig()
	// Customize config based on service
	switch name {
	case "ollama":
		config.ResetTimeout = 60 * time.Second // AI services may need longer recovery
	case "n8n", "windmill":
		config.ResetTimeout = 30 * time.Second
	}
	
	breaker := NewCircuitBreaker(name, config)
	m.breakers[name] = breaker
	return breaker
}

// GetAllStats returns stats for all circuit breakers
func (m *CircuitBreakerManager) GetAllStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	for name, breaker := range m.breakers {
		stats[name] = breaker.GetStats()
	}
	
	return stats
}

// Global circuit breaker manager
var circuitBreakerManager = NewCircuitBreakerManager()