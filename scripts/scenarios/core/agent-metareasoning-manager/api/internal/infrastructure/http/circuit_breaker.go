package http

import (
	"fmt"
	"sync"
	"time"

	"metareasoning-api/internal/pkg/interfaces"
)

// CircuitBreakerManager implements interfaces.CircuitBreakerManager
type CircuitBreakerManager struct {
	breakers map[string]interfaces.CircuitBreaker
	mutex    sync.RWMutex
}

// NewCircuitBreakerManager creates a new circuit breaker manager
func NewCircuitBreakerManager() interfaces.CircuitBreakerManager {
	return &CircuitBreakerManager{
		breakers: make(map[string]interfaces.CircuitBreaker),
	}
}

// GetBreaker gets or creates a circuit breaker for a service
func (m *CircuitBreakerManager) GetBreaker(name string) interfaces.CircuitBreaker {
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
	
	config := getCircuitBreakerConfig(name)
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

// CircuitBreakerConfig holds configuration for circuit breaker
type CircuitBreakerConfig struct {
	MaxFailures      int
	ResetTimeout     time.Duration
	SuccessThreshold int
}

// getCircuitBreakerConfig returns configuration based on service name
func getCircuitBreakerConfig(serviceName string) CircuitBreakerConfig {
	base := CircuitBreakerConfig{
		MaxFailures:      3,
		ResetTimeout:     30 * time.Second,
		SuccessThreshold: 2,
	}
	
	// Customize config based on service
	switch serviceName {
	case "ollama":
		base.ResetTimeout = 60 * time.Second // AI services may need longer recovery
	case "n8n", "windmill":
		base.ResetTimeout = 30 * time.Second
	}
	
	return base
}

// CircuitBreaker implements interfaces.CircuitBreaker
type CircuitBreaker struct {
	config          CircuitBreakerConfig
	state           interfaces.CircuitBreakerState
	failureCount    int
	successCount    int
	lastFailureTime time.Time
	mutex           sync.RWMutex
	name            string
}

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(name string, config CircuitBreakerConfig) interfaces.CircuitBreaker {
	return &CircuitBreaker{
		config: config,
		state:  interfaces.Closed,
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

// GetState returns the current state of the circuit breaker
func (cb *CircuitBreaker) GetState() interfaces.CircuitBreakerState {
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

// shouldAllowCall determines if a call should be allowed
func (cb *CircuitBreaker) shouldAllowCall() bool {
	switch cb.state {
	case interfaces.Closed:
		return true
	case interfaces.Open:
		// Check if enough time has passed to try again
		if time.Since(cb.lastFailureTime) > cb.config.ResetTimeout {
			cb.state = interfaces.HalfOpen
			cb.successCount = 0
			return true
		}
		return false
	case interfaces.HalfOpen:
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
	case interfaces.Closed:
		if cb.failureCount >= cb.config.MaxFailures {
			cb.state = interfaces.Open
			fmt.Printf("Circuit breaker %s OPENED after %d failures\n", cb.name, cb.failureCount)
		}
	case interfaces.HalfOpen:
		cb.state = interfaces.Open
		fmt.Printf("Circuit breaker %s returned to OPEN from half-open\n", cb.name)
	}
}

// recordSuccess records a success and potentially closes the circuit
func (cb *CircuitBreaker) recordSuccess() {
	cb.failureCount = 0 // Reset failure count on success
	
	switch cb.state {
	case interfaces.HalfOpen:
		cb.successCount++
		if cb.successCount >= cb.config.SuccessThreshold {
			cb.state = interfaces.Closed
			fmt.Printf("Circuit breaker %s CLOSED after %d successes\n", cb.name, cb.successCount)
		}
	}
}