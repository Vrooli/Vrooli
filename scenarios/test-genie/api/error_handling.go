package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"
)

// Circuit Breaker States
type CircuitBreakerState int

const (
	StateClosed CircuitBreakerState = iota
	StateOpen
	StateHalfOpen
)

// Circuit Breaker Configuration
type CircuitBreakerConfig struct {
	Name            string
	MaxFailures     int
	ResetTimeout    time.Duration
	TimeoutDuration time.Duration
	OnStateChange   func(name string, from, to CircuitBreakerState)
}

// Circuit Breaker Implementation
type CircuitBreaker struct {
	config       CircuitBreakerConfig
	state        CircuitBreakerState
	failures     int
	lastFailTime time.Time
	mutex        sync.Mutex
}

// NewCircuitBreaker creates a new circuit breaker instance
func NewCircuitBreaker(config CircuitBreakerConfig) *CircuitBreaker {
	return &CircuitBreaker{
		config:       config,
		state:        StateClosed,
		failures:     0,
		lastFailTime: time.Time{},
	}
}

// Execute runs the given function with circuit breaker protection
func (cb *CircuitBreaker) Execute(ctx context.Context, fn func() (interface{}, error)) (interface{}, error) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	// Check if circuit should be opened
	if cb.shouldOpen() {
		cb.setState(StateOpen)
	}

	// Check if circuit should move from open to half-open
	if cb.state == StateOpen && cb.shouldAttemptReset() {
		cb.setState(StateHalfOpen)
	}

	// If circuit is open, fail fast
	if cb.state == StateOpen {
		return nil, fmt.Errorf("circuit breaker '%s' is open", cb.config.Name)
	}

	// Execute the function with timeout
	resultChan := make(chan struct {
		result interface{}
		err    error
	}, 1)

	go func() {
		result, err := fn()
		resultChan <- struct {
			result interface{}
			err    error
		}{result, err}
	}()

	select {
	case res := <-resultChan:
		if res.err != nil {
			cb.recordFailure()
			return nil, res.err
		}
		cb.recordSuccess()
		return res.result, nil
	case <-ctx.Done():
		cb.recordFailure()
		return nil, fmt.Errorf("circuit breaker '%s' timeout: %w", cb.config.Name, ctx.Err())
	case <-time.After(cb.config.TimeoutDuration):
		cb.recordFailure()
		return nil, fmt.Errorf("circuit breaker '%s' operation timeout", cb.config.Name)
	}
}

func (cb *CircuitBreaker) shouldOpen() bool {
	return cb.failures >= cb.config.MaxFailures
}

func (cb *CircuitBreaker) shouldAttemptReset() bool {
	return time.Since(cb.lastFailTime) > cb.config.ResetTimeout
}

func (cb *CircuitBreaker) recordSuccess() {
	if cb.state == StateHalfOpen {
		cb.setState(StateClosed)
	}
	cb.failures = 0
}

func (cb *CircuitBreaker) recordFailure() {
	cb.failures++
	cb.lastFailTime = time.Now()
	log.Printf("⚠️ Circuit breaker '%s' failure recorded: %d/%d",
		cb.config.Name, cb.failures, cb.config.MaxFailures)
}

func (cb *CircuitBreaker) setState(newState CircuitBreakerState) {
	if cb.config.OnStateChange != nil {
		cb.config.OnStateChange(cb.config.Name, cb.state, newState)
	}

	oldState := cb.state
	cb.state = newState

	log.Printf("🔄 Circuit breaker '%s' state changed: %s -> %s",
		cb.config.Name, stateToString(oldState), stateToString(newState))
}

func stateToString(state CircuitBreakerState) string {
	switch state {
	case StateClosed:
		return "CLOSED"
	case StateOpen:
		return "OPEN"
	case StateHalfOpen:
		return "HALF_OPEN"
	default:
		return "UNKNOWN"
	}
}

// Retry Logic with Exponential Backoff
type RetryConfig struct {
	MaxAttempts int
	BaseDelay   time.Duration
	MaxDelay    time.Duration
	BackoffFunc func(attempt int, baseDelay time.Duration) time.Duration
	ShouldRetry func(error) bool
}

// DefaultRetryConfig provides sensible defaults for retry configuration
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts: 3,
		BaseDelay:   100 * time.Millisecond,
		MaxDelay:    5 * time.Second,
		BackoffFunc: ExponentialBackoff,
		ShouldRetry: IsRetryableError,
	}
}

// ExponentialBackoff calculates delay using exponential backoff
func ExponentialBackoff(attempt int, baseDelay time.Duration) time.Duration {
	delay := time.Duration(float64(baseDelay) * (1.5 * float64(attempt)))
	return delay
}

// IsRetryableError determines if an error should trigger a retry
func IsRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Network timeout errors
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// Database connection errors
	if errors.Is(err, sql.ErrConnDone) {
		return true
	}

	// Check for temporary network errors
	errorStr := err.Error()
	temporaryErrors := []string{
		"connection refused",
		"timeout",
		"temporary failure",
		"network unreachable",
		"host unreachable",
		"no such host",
		"connection reset",
	}

	for _, tempErr := range temporaryErrors {
		if contains(errorStr, tempErr) {
			return true
		}
	}

	return false
}

func contains(str, substr string) bool {
	return len(str) >= len(substr) &&
		(str == substr ||
			str[0:len(substr)] == substr ||
			str[len(str)-len(substr):] == substr ||
			findSubstring(str, substr))
}

func findSubstring(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// RetryWithBackoff executes a function with retry logic and exponential backoff
func RetryWithBackoff(ctx context.Context, config RetryConfig, fn func() error) error {
	var lastErr error

	for attempt := 0; attempt < config.MaxAttempts; attempt++ {
		if attempt > 0 {
			// Calculate delay
			delay := config.BackoffFunc(attempt, config.BaseDelay)
			if delay > config.MaxDelay {
				delay = config.MaxDelay
			}

			log.Printf("🔄 Retry attempt %d/%d after %v", attempt+1, config.MaxAttempts, delay)

			select {
			case <-time.After(delay):
			case <-ctx.Done():
				return fmt.Errorf("retry cancelled: %w", ctx.Err())
			}
		}

		err := fn()
		if err == nil {
			if attempt > 0 {
				log.Printf("✅ Operation succeeded after %d attempts", attempt+1)
			}
			return nil
		}

		lastErr = err

		if !config.ShouldRetry(err) {
			log.Printf("❌ Error not retryable: %v", err)
			return err
		}

		log.Printf("⚠️ Attempt %d/%d failed: %v", attempt+1, config.MaxAttempts, err)
	}

	return fmt.Errorf("operation failed after %d attempts: %w", config.MaxAttempts, lastErr)
}

// Fallback Handler
type FallbackHandler struct {
	primary   func() (interface{}, error)
	fallback  func() (interface{}, error)
	predicate func(error) bool
}

// NewFallbackHandler creates a new fallback handler
func NewFallbackHandler(
	primary func() (interface{}, error),
	fallback func() (interface{}, error),
	predicate func(error) bool,
) *FallbackHandler {
	if predicate == nil {
		predicate = func(error) bool { return true }
	}

	return &FallbackHandler{
		primary:   primary,
		fallback:  fallback,
		predicate: predicate,
	}
}

// Execute runs the primary function, falling back on error
func (fh *FallbackHandler) Execute(ctx context.Context) (interface{}, error) {
	result, err := fh.primary()
	if err != nil && fh.predicate(err) {
		log.Printf("🔀 Primary function failed (%v), falling back...", err)
		return fh.fallback()
	}
	return result, err
}

// Global circuit breakers and configurations
var (
	openCodeCircuitBreaker *CircuitBreaker
	dbCircuitBreaker       *CircuitBreaker
	circuitBreakerInit     sync.Once
)

// InitializeCircuitBreakers sets up global circuit breakers
func InitializeCircuitBreakers() {
	circuitBreakerInit.Do(func() {
		// OpenCode circuit breaker
		openCodeCircuitBreaker = NewCircuitBreaker(CircuitBreakerConfig{
			Name:            "opencode",
			MaxFailures:     5,
			ResetTimeout:    60 * time.Second,
			TimeoutDuration: 30 * time.Second,
			OnStateChange:   logCircuitBreakerStateChange,
		})

		// Database circuit breaker
		dbCircuitBreaker = NewCircuitBreaker(CircuitBreakerConfig{
			Name:            "database",
			MaxFailures:     3,
			ResetTimeout:    30 * time.Second,
			TimeoutDuration: 10 * time.Second,
			OnStateChange:   logCircuitBreakerStateChange,
		})

		log.Println("🔧 Circuit breakers initialized successfully")
	})
}

func logCircuitBreakerStateChange(name string, from, to CircuitBreakerState) {
	log.Printf("🚨 CIRCUIT BREAKER ALERT: %s changed from %s to %s",
		name, stateToString(from), stateToString(to))
}

// Health Check with Circuit Breaker
type HealthChecker struct {
	name           string
	checkFunc      func() error
	circuitBreaker *CircuitBreaker
	lastCheck      time.Time
	lastResult     error
	checkInterval  time.Duration
	mutex          sync.RWMutex
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(name string, checkFunc func() error, cb *CircuitBreaker) *HealthChecker {
	return &HealthChecker{
		name:           name,
		checkFunc:      checkFunc,
		circuitBreaker: cb,
		checkInterval:  30 * time.Second,
	}
}

// Check performs a health check with circuit breaker protection
func (hc *HealthChecker) Check(ctx context.Context) error {
	hc.mutex.RLock()
	if time.Since(hc.lastCheck) < hc.checkInterval {
		defer hc.mutex.RUnlock()
		return hc.lastResult
	}
	hc.mutex.RUnlock()

	hc.mutex.Lock()
	defer hc.mutex.Unlock()

	// Double-check after acquiring write lock
	if time.Since(hc.lastCheck) < hc.checkInterval {
		return hc.lastResult
	}

	result, err := hc.circuitBreaker.Execute(ctx, func() (interface{}, error) {
		return nil, hc.checkFunc()
	})

	hc.lastCheck = time.Now()
	if err != nil {
		hc.lastResult = err
		log.Printf("❌ Health check failed for %s: %v", hc.name, err)
	} else {
		hc.lastResult = nil
		log.Printf("✅ Health check passed for %s", hc.name)
	}

	_ = result // Unused but needed for interface compliance
	return hc.lastResult
}

// Graceful degradation response types
type DegradationLevel int

const (
	FullService DegradationLevel = iota
	PartialService
	MinimalService
	ServiceUnavailable
)

func (dl DegradationLevel) String() string {
	switch dl {
	case FullService:
		return "FULL_SERVICE"
	case PartialService:
		return "PARTIAL_SERVICE"
	case MinimalService:
		return "MINIMAL_SERVICE"
	case ServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		return "UNKNOWN"
	}
}

// ServiceManager manages service availability and degradation
type ServiceManager struct {
	services map[string]*HealthChecker
	mutex    sync.RWMutex
}

// NewServiceManager creates a new service manager
func NewServiceManager() *ServiceManager {
	return &ServiceManager{
		services: make(map[string]*HealthChecker),
	}
}

// RegisterService registers a service with health checking
func (sm *ServiceManager) RegisterService(name string, checker *HealthChecker) {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()
	sm.services[name] = checker
}

// GetServiceLevel determines the current service degradation level
func (sm *ServiceManager) GetServiceLevel(ctx context.Context) DegradationLevel {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	healthyServices := 0
	totalServices := len(sm.services)

	for name, checker := range sm.services {
		if err := checker.Check(ctx); err == nil {
			healthyServices++
		} else {
			log.Printf("⚠️ Service %s is unhealthy: %v", name, err)
		}
	}

	if totalServices == 0 {
		return ServiceUnavailable
	}

	healthRatio := float64(healthyServices) / float64(totalServices)

	switch {
	case healthRatio >= 1.0:
		return FullService
	case healthRatio >= 0.75:
		return PartialService
	case healthRatio >= 0.5:
		return MinimalService
	default:
		return ServiceUnavailable
	}
}
